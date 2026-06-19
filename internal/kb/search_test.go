//go:build fts5

package kb

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func setupSearchTest(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := setupTestKB(t)

	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	docs := []struct {
		path    string
		content string
	}{
		{
			filepath.Join(dir, "raw", "go-intro.md"),
			"---\ntitle: Go Introduction\nkind: source-note\n---\nGo is a programming language.",
		},
		{
			filepath.Join(dir, "raw", "python-intro.md"),
			"---\ntitle: Python Introduction\nkind: source-note\n---\nPython is a programming language.",
		},
		{
			filepath.Join(dir, "wiki", "concepts", "go.md"),
			"---\ntitle: Go Language\nkind: concept\n---\nGo is fast and compiled.",
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

	return dir, db
}

func TestFTSSearch_BasicMatch(t *testing.T) {
	_, db := setupSearchTest(t)

	results, err := FTSSearch(db, "programming", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Errorf("got %d results, want ≥2", len(results))
	}
}

func TestFTSSearch_LayerFilter(t *testing.T) {
	_, db := setupSearchTest(t)

	layer := "wiki"
	results, err := FTSSearch(db, "Go", &layer, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Layer != "wiki" {
			t.Errorf("result %q has layer %q, want wiki", r.ID, r.Layer)
		}
	}
}

func TestFTSSearch_NoResults(t *testing.T) {
	_, db := setupSearchTest(t)

	results, err := FTSSearch(db, "nonexistent_term_xyz", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestFTSSearch_CommaKeywords(t *testing.T) {
	_, db := setupSearchTest(t)

	results, err := FTSSearch(db, "Go, Python", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Errorf("got %d results, want ≥2", len(results))
	}
}
