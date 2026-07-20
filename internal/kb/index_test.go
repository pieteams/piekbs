//go:build fts5

package kb

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func setupTestKB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, d := range []string{"raw", "raw/converted", "wiki/source-notes", "schema", "index"} {
		os.MkdirAll(filepath.Join(dir, d), 0755)
	}
	return dir
}

func TestUpsertDocument_New(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	rawPath := filepath.Join(dir, "raw", "test.md")
	os.WriteFile(rawPath, []byte("---\ntitle: Hello\nkind: source-note\n---\nBody text"), 0644)

	n, err := IndexFiles(db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("indexed %d files, want 1", n)
	}

	var title string
	err = db.QueryRow("SELECT title FROM documents WHERE id = ?", "raw/test.md").Scan(&title)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Hello" {
		t.Errorf("title = %q, want 'Hello'", title)
	}
}

func TestIndexFiles_TabularFiles(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tsv := "社区昵称\t创意帖链接\t创意帖标题\nu1725\thttps://forum.example/t/26353\t水质参数光谱预测系统\n"
	os.WriteFile(filepath.Join(dir, "raw", "table-01.snapshot.tsv"), []byte(tsv), 0644)
	os.WriteFile(filepath.Join(dir, "raw", "records.csv"), []byte("name,title\nalice,project-x\n"), 0644)

	n, err := IndexFiles(db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("indexed %d files, want 2", n)
	}

	for _, id := range []string{"raw/table-01.snapshot.tsv", "raw/records.csv"} {
		var layer string
		err = db.QueryRow("SELECT layer FROM documents WHERE id = ?", id).Scan(&layer)
		if err != nil {
			t.Fatalf("%s not indexed: %v", id, err)
		}
		if layer != "raw" {
			t.Errorf("%s layer = %q, want 'raw'", id, layer)
		}
	}

	// Cell values must be reachable through FTS.
	var hits int
	err = db.QueryRow("SELECT COUNT(*) FROM document_fts WHERE document_fts MATCH ?", "水质参数光谱预测系统").Scan(&hits)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Errorf("FTS hits for TSV cell value = %d, want 1", hits)
	}
}

func TestUpsertDocument_SkipUnchanged(t *testing.T) {
	dir := setupTestKB(t)
	db, _ := OpenDB(dir)
	defer db.Close()

	rawPath := filepath.Join(dir, "raw", "test.md")
	os.WriteFile(rawPath, []byte("---\ntitle: Hello\n---\nBody"), 0644)

	n1, _ := IndexFiles(db, dir)
	n2, _ := IndexFiles(db, dir)

	if n1 != 1 {
		t.Errorf("first index: %d, want 1", n1)
	}
	if n2 != 0 {
		t.Errorf("second index: %d, want 0 (unchanged)", n2)
	}
}

// TestIndexFilesFull_ReindexesUnchanged verifies that full reindex rewrites
// documents whose content is unchanged — the behavior kb_reindex(full=true)
// promises. If full silently degraded to incremental, this would return 0 and
// fail, which is exactly the regression we want to catch.
func TestIndexFilesFull_ReindexesUnchanged(t *testing.T) {
	dir := setupTestKB(t)
	db, _ := OpenDB(dir)
	defer db.Close()

	rawPath := filepath.Join(dir, "raw", "test.md")
	os.WriteFile(rawPath, []byte("---\ntitle: Hello\n---\nBody"), 0644)

	IndexFiles(db, dir) // initial index

	// Incremental would skip (hash unchanged); full must still rewrite it.
	n, err := IndexFilesFull(db, dir)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("full reindex wrote %d, want 1 (must rewrite unchanged docs)", n)
	}
}

func TestPurgeDeletedDocuments(t *testing.T) {
	dir := setupTestKB(t)
	db, _ := OpenDB(dir)
	defer db.Close()

	rawPath := filepath.Join(dir, "raw", "test.md")
	os.WriteFile(rawPath, []byte("---\ntitle: Hello\n---\nBody"), 0644)
	IndexFiles(db, dir)

	os.Remove(rawPath)
	IndexFiles(db, dir)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count)
	if count != 0 {
		t.Errorf("document count = %d after purge, want 0", count)
	}
}

func TestUpsertDocumentTagsFromClaims(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO documents
		(id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp)
		VALUES ('wiki/a.md','wiki/a.md','wiki','source-note','title','','content','abc',1,3,0)`)
	if err != nil {
		t.Fatal(err)
	}

	claims := []string{
		"【Karpathy|人物】提出【LLM Wiki|概念】，由【Anthropic|组织】验证",
		"基于【Obsidian|产品】实现知识管理",
	}
	upsertDocumentTags(db, "wiki/a.md", claims)

	rows, err := db.Query("SELECT tag, source FROM document_tags WHERE doc_id='wiki/a.md' ORDER BY tag")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type row struct{ tag, source string }
	var got []row
	for rows.Next() {
		var r row
		rows.Scan(&r.tag, &r.source)
		got = append(got, r)
	}

	// 期望：Anthropic, Karpathy, LLM Wiki, Obsidian（4个 claim 实体）
	if len(got) != 4 {
		t.Fatalf("expected 4 claim tags, got %d: %v", len(got), got)
	}
	for _, r := range got {
		if r.source != "claim" {
			t.Errorf("expected source=claim, got %q for tag %q", r.source, r.tag)
		}
	}
}

func TestUpsertDocumentTagsFiltersNoise(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO documents
		(id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp)
		VALUES ('wiki/b.md','wiki/b.md','wiki','source-note','title','','content','def',1,3,0)`)
	if err != nil {
		t.Fatal(err)
	}

	// 【技术|库】：name="技术" 是 type 词，应过滤
	// 【A|产品】：name 长度 1，应过滤
	// 【OpenAI|组织】：有效
	claims := []string{"【技术|库】做了【A|产品】，基于【OpenAI|组织】"}
	upsertDocumentTags(db, "wiki/b.md", claims)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM document_tags WHERE doc_id='wiki/b.md'").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 valid claim entity (OpenAI), got %d", count)
	}
}

func TestUpsertDocumentTagsNilClaims(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`INSERT INTO documents
		(id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp)
		VALUES ('wiki/c.md','wiki/c.md','wiki','source-note','title','','content','ghi',1,3,0)`)
	if err != nil {
		t.Fatal(err)
	}

	// 没有 key_claims 的文档 → 不写入任何 tag
	upsertDocumentTags(db, "wiki/c.md", nil)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM document_tags WHERE doc_id='wiki/c.md'").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 tags for nil claims, got %d", count)
	}
}

func TestUpsertDocument_StoresDistillVersion(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	content := "---\ntitle: Versioned Note\ndistill_version: 0.4.7\n---\nBody"
	os.WriteFile(filepath.Join(dir, "raw", "ver.md"), []byte(content), 0644)

	_, err = IndexFiles(db, dir)
	if err != nil {
		t.Fatal(err)
	}

	var v sql.NullString
	err = db.QueryRow("SELECT distill_version FROM documents WHERE id = ?", "raw/ver.md").Scan(&v)
	if err != nil {
		t.Fatal(err)
	}
	if !v.Valid || v.String != "0.4.7" {
		t.Errorf("expected 0.4.7, got %v", v)
	}
}
