//go:build fts5

package kb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildContext(t *testing.T) {
	dir := setupTestKB(t)

	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create a raw source and a wiki page that cites it.
	docs := []struct {
		path    string
		content string
	}{
		{
			filepath.Join(dir, "raw", "gopher.md"),
			"---\ntitle: Gopher Source\nkind: source-note\n---\nGopher is the mascot of the Go programming language.",
		},
		{
			filepath.Join(dir, "wiki", "concepts", "gopher.md"),
			"---\ntitle: Gopher\nkind: concept\nsources:\n  - raw/gopher.md\n---\nGopher represents the Go community.",
		},
	}

	for _, d := range docs {
		if err := os.MkdirAll(filepath.Dir(d.path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(d.path, []byte(d.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := IndexFiles(db, dir); err != nil {
		t.Fatal(err)
	}

	bundle := BuildContext(db, dir, "Gopher", nil, 10)

	if bundle.Question != "Gopher" {
		t.Errorf("Question = %q, want %q", bundle.Question, "Gopher")
	}
	if len(bundle.WikiPages) == 0 {
		t.Errorf("WikiPages is empty, want at least one result")
	}
}
