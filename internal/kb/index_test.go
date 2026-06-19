//go:build fts5

package kb

import (
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
