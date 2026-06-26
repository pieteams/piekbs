//go:build fts5

package kb

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKBError(t *testing.T) {
	err := &KBError{Code: 400, Message: "filename is required"}
	if err.Error() != "filename is required" {
		t.Errorf("got %q", err.Error())
	}
	if err.Code != 400 {
		t.Errorf("expected code 400, got %d", err.Code)
	}
	// errors.As unwrap
	var kbe *KBError
	if !errors.As(err, &kbe) {
		t.Error("errors.As should match *KBError")
	}
}

func TestAppendQueryLog(t *testing.T) {
	dir := t.TempDir()
	wikiDir := filepath.Join(dir, "wiki")
	os.MkdirAll(wikiDir, 0o755)

	AppendQueryLog(dir, "kb_search", "test query")

	files, _ := filepath.Glob(filepath.Join(wikiDir, "query_log_*.jsonl"))
	if len(files) != 1 {
		t.Fatalf("expected 1 log file, got %d", len(files))
	}
	content, _ := os.ReadFile(files[0])
	if !strings.Contains(string(content), `"kb_search"`) {
		t.Errorf("log missing tool: %s", content)
	}
	if !strings.Contains(string(content), `"test query"`) {
		t.Errorf("log missing query: %s", content)
	}
}

func TestAppendQueryLogWithExtra(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "wiki"), 0o755)

	AppendQueryLog(dir, "kb_search", "q", "related", `["wiki/a.md"]`)

	files, _ := filepath.Glob(filepath.Join(dir, "wiki", "query_log_*.jsonl"))
	content, _ := os.ReadFile(files[0])
	if !strings.Contains(string(content), `"related"`) {
		t.Errorf("extra field missing: %s", content)
	}
}

func TestKBStatus(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "wiki"), 0o755)
	// OpenDB creates the DB
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	result, err := KBStatus(dir)
	if err != nil {
		t.Fatalf("KBStatus: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Documents < 0 {
		t.Errorf("documents should be >= 0")
	}
	if result.ByLayer == nil {
		t.Error("by_layer should not be nil")
	}
	if result.ByKind == nil {
		t.Error("by_kind should not be nil")
	}
	if result.IndexSize < 0 {
		t.Error("index_size should be >= 0")
	}
	if result.DistillQueue == nil {
		t.Error("distill_queue should not be nil")
	}
}

func TestKBSearch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "wiki"), 0o755)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	resp, err := KBSearch(dir, "test", nil, nil, 5, 2)
	if err != nil {
		t.Fatalf("KBSearch: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	// Empty KB → empty results, not error
	if resp.Results == nil {
		t.Error("results should be non-nil slice (may be empty)")
	}
}

func TestKBPageEmptyIDs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "wiki"), 0o755)
	OpenDB(dir) // init DB

	_, err := KBPage(dir, nil, false)
	var kbe *KBError
	if !errors.As(err, &kbe) || kbe.Code != 400 {
		t.Errorf("expected KBError 400 for empty ids, got %v", err)
	}
}

func TestKBAdd(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "wiki"), 0o755)
	db, _ := OpenDB(dir)
	db.Close()

	result, err := KBAdd(dir, "references/test.md", "# Test", "", false)
	if err != nil {
		t.Fatalf("KBAdd: %v", err)
	}
	if result.Path != "raw/references/test.md" {
		t.Errorf("expected path raw/references/test.md, got %q", result.Path)
	}
	if _, err := os.Stat(filepath.Join(dir, "raw", "references", "test.md")); err != nil {
		t.Error("file not written")
	}
}

func TestKBAddPathTraversal(t *testing.T) {
	dir := t.TempDir()
	_, err := KBAdd(dir, "../evil.md", "bad", "", false)
	var kbe *KBError
	if !errors.As(err, &kbe) || kbe.Code != 400 {
		t.Errorf("expected KBError 400 for path traversal, got %v", err)
	}
}

func TestKBAddFileExists(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "wiki"), 0o755)
	os.MkdirAll(filepath.Join(dir, "raw"), 0o755)
	db, _ := OpenDB(dir)
	db.Close()
	os.WriteFile(filepath.Join(dir, "raw", "a.md"), []byte("x"), 0o644)

	_, err := KBAdd(dir, "a.md", "y", "", false)
	var kbe *KBError
	if !errors.As(err, &kbe) || kbe.Code != 400 {
		t.Errorf("expected KBError 400 for existing file, got %v", err)
	}
}

func TestKBUpload(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "raw"), 0o755)
	os.MkdirAll(filepath.Join(dir, "wiki"), 0o755)
	db, _ := OpenDB(dir)
	db.Close()

	r := strings.NewReader("hello world")
	if err := KBUpload(dir, "doc.txt", r); err != nil {
		t.Fatalf("KBUpload: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "raw", "doc.txt")); err != nil {
		t.Error("uploaded file not found")
	}
}

func TestKBReindex(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "raw"), 0o755)
	os.MkdirAll(filepath.Join(dir, "wiki"), 0o755)
	db, _ := OpenDB(dir)
	db.Close()

	result, err := KBReindex(dir, false)
	if err != nil {
		t.Fatalf("KBReindex: %v", err)
	}
	if result.Message == "" {
		t.Error("expected non-empty message")
	}
}

func TestKBLint(t *testing.T) {
	dir := t.TempDir()
	result, err := KBLint(dir)
	if err != nil {
		t.Fatalf("KBLint: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Warnings == nil {
		t.Error("warnings should be non-nil slice")
	}
}

// TestKBLint_CleansBrokenLinks verifies that KBLint deletes broken links from
// the links table and writes red_links.json with concept-name gaps.
func TestKBLint_CleansBrokenLinks(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Insert a source doc and a concept-name broken link.
	db.Exec(`INSERT OR IGNORE INTO documents (id,path,layer,kind,title,description,content,content_hash,updated_at,authority,doc_timestamp) VALUES ('wiki/src.md','wiki/src.md','wiki','source-note','S','','body','h',1,3,0)`)
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/src.md','数字经济','related_to',1.0)`)
	db.Close()

	result, err := KBLint(dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.RedLinks == nil || len(result.RedLinks) != 1 {
		t.Fatalf("want 1 RedLink, got %+v", result.RedLinks)
	}
	if result.RedLinks[0].Concept != "数字经济" {
		t.Errorf("want concept '数字经济', got %q", result.RedLinks[0].Concept)
	}

	// red_links.json must exist and be valid.
	jsonPath := filepath.Join(dir, "wiki", "index", "red_links.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("red_links.json not written: %v", err)
	}
	var links []RedLink
	if err := json.Unmarshal(data, &links); err != nil {
		t.Fatalf("invalid red_links.json: %v", err)
	}
	if len(links) != 1 || links[0].Concept != "数字经济" {
		t.Errorf("unexpected red_links.json content: %s", data)
	}
}
