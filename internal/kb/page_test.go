//go:build fts5

package kb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchPagesSingleFull(t *testing.T) {
	dir := t.TempDir()
	wikiDir := filepath.Join(dir, "wiki", "source-notes")
	os.MkdirAll(wikiDir, 0o755)
	content := "---\ntype: source-note\ntitle: Test\n---\n\n" + strings.Repeat("x", 6000)
	docPath := filepath.Join(wikiDir, "test.md")
	os.WriteFile(docPath, []byte(content), 0o644)

	db := newTestDB(t)
	db.Exec(`INSERT INTO documents(id,path,layer,kind,title,content,content_hash,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		"wiki/source-notes/test.md", "wiki/source-notes/test.md", "wiki", "source-note", "Test", "", "0000000000000000", 0)

	// full=true → no truncation
	pages, err := FetchPages(db, dir, []string{"wiki/source-notes/test.md"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if pages[0].Truncated {
		t.Error("expected Truncated=false for full=true single page")
	}
	if len(pages[0].Content) < 6000 {
		t.Errorf("expected full content (>=6000 chars), got %d", len(pages[0].Content))
	}
}

func TestFetchPagesMultiTruncates(t *testing.T) {
	dir := t.TempDir()
	wikiDir := filepath.Join(dir, "wiki", "source-notes")
	os.MkdirAll(wikiDir, 0o755)
	longContent := "---\ntype: source-note\ntitle: Long\n---\n\n" + strings.Repeat("y", 6000)
	for _, name := range []string{"a.md", "b.md"} {
		os.WriteFile(filepath.Join(wikiDir, name), []byte(longContent), 0o644)
	}

	db := newTestDB(t)
	for _, name := range []string{"a.md", "b.md"} {
		id := "wiki/source-notes/" + name
		db.Exec(`INSERT INTO documents(id,path,layer,kind,title,content,content_hash,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
			id, id, "wiki", "source-note", "Long", "", "0000000000000000", 0)
	}

	// multi ids → always truncate regardless of full param
	pages, err := FetchPages(db, dir, []string{"wiki/source-notes/a.md", "wiki/source-notes/b.md"}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pages {
		if len(p.Content) > 5000 && !p.Truncated {
			t.Errorf("multi-id page should be truncated at 5000, got %d chars, Truncated=%v", len(p.Content), p.Truncated)
		}
	}
}
