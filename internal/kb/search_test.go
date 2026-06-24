//go:build fts5

package kb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// newTestDB creates a temporary kb database for use in tests.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSearchResultHasRelatedField(t *testing.T) {
	r := SearchResult{
		ID:    "wiki/source-notes/test.md",
		Title: "Test",
	}
	r.Related = []RelatedDoc{
		{ID: "wiki/concepts/test-concept.md", Title: "Test Concept", Kind: "concept"},
	}
	if len(r.Related) != 1 {
		t.Fatalf("expected 1 related doc, got %d", len(r.Related))
	}
	if r.Related[0].Kind != "concept" {
		t.Fatalf("expected kind 'concept', got %q", r.Related[0].Kind)
	}
}

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

func TestSearchLayeredSeparatesKinds(t *testing.T) {
	db := newTestDB(t)
	// Insert 6 source-notes and 4 concept pages all matching "RAG"
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("wiki/source-notes/rag-%d.md", i)
		db.Exec(`INSERT INTO documents(id,path,layer,kind,title,content,content_hash,updated_at,wiki_priority) VALUES(?,?,?,?,?,?,?,?,?)`,
			id, id, "wiki", "source-note", fmt.Sprintf("RAG source %d", i), "RAG content", fmt.Sprintf("h%d", i), int64(1700000000+i), 1.0)
	}
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("wiki/concepts/rag-%d.md", i)
		db.Exec(`INSERT INTO documents(id,path,layer,kind,title,content,content_hash,updated_at,wiki_priority) VALUES(?,?,?,?,?,?,?,?,?)`,
			id, id, "wiki", "concept", fmt.Sprintf("RAG concept %d", i), "RAG content", fmt.Sprintf("c%d", i), int64(1700000000+i), 1.0)
	}
	// rebuild FTS
	db.Exec(`INSERT INTO document_fts(document_fts) VALUES('rebuild')`)

	results, _, err := SearchLayered(db, "", "RAG", nil, nil, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	var noteCount, synthCount int
	for _, r := range results {
		if r.Kind == "source-note" {
			noteCount++
		} else {
			synthCount++
		}
	}
	if noteCount > 5 {
		t.Errorf("source-notes should be capped at 5, got %d", noteCount)
	}
	if synthCount > 3 {
		t.Errorf("synthesized pages should be capped at 3, got %d", synthCount)
	}
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

func TestHybridRankSynthesizedBoostIsMultiplicative(t *testing.T) {
	fts := []SearchResult{
		{ID: "a", Kind: "source-note", FTSRank: -1.0},
		{ID: "b", Kind: "concept", FTSRank: -1.0},
	}
	result := HybridRank(fts, nil, nil, nil)
	var aScore, bScore float64
	for _, r := range result {
		if r.ID == "a" {
			aScore = r.HybridScore
		}
		if r.ID == "b" {
			bScore = r.HybridScore
		}
	}
	// concept should score higher than source-note due to 1.3x multiplier
	if bScore <= aScore {
		t.Errorf("concept score (%f) should be > source-note score (%f)", bScore, aScore)
	}
	// verify multiplicative: bScore should be ~1.3x aScore (before authority/graph adjustments)
	ratio := bScore / aScore
	if ratio < 1.2 || ratio > 1.5 {
		t.Errorf("expected multiplicative ratio ~1.3, got %f", ratio)
	}
}
