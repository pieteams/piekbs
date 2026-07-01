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

// TestSortResults_MatchPhaseBeatsHigherFTSRank verifies that AND results (MatchPhase=0)
// rank above OR results (MatchPhase=1) even when the OR result has a better FTSRank,
// which was the root bug this feature is fixing.
func TestSortResults_MatchPhaseBeatsHigherFTSRank(t *testing.T) {
	// OR result has a better (lower) FTSRank than AND result.
	// Without MatchPhase sorting, the OR result would wrongly win.
	results := []SearchResult{
		{ID: "or-doc", Title: "Go Python match", Snippet: "Go Python", WikiPriority: 1.0, FTSRank: -2.0, MatchPhase: 1},
		{ID: "and-doc", Title: "Go Python full match", Snippet: "Go Python", WikiPriority: 1.0, FTSRank: -1.0, MatchPhase: 0, Coverage: 2},
	}
	sortResults(results, []string{"Go", "Python"})
	if results[0].ID != "and-doc" {
		t.Errorf("AND result should rank first, but got %q first", results[0].ID)
	}
}

// TestSortResults_WikiPriorityBeatsMatchPhase verifies that WikiPriority still wins
// over MatchPhase: a wiki-layer OR result beats a raw-layer AND result.
func TestSortResults_WikiPriorityBeatsMatchPhase(t *testing.T) {
	results := []SearchResult{
		{ID: "wiki-or", WikiPriority: 1.0, FTSRank: -1.0, MatchPhase: 1},
		{ID: "raw-and", WikiPriority: 0.0, FTSRank: -2.0, MatchPhase: 0, Coverage: 2},
	}
	sortResults(results, []string{"Go"})
	if results[0].ID != "wiki-or" {
		t.Errorf("wiki-layer OR should rank first, but got %q first", results[0].ID)
	}
}

// TestSortResults_CoverageTiebreaker verifies that among OR results at the same phase,
// higher Coverage (more keyword hits) ranks first.
func TestSortResults_CoverageTiebreaker(t *testing.T) {
	results := []SearchResult{
		{ID: "low-cov", Title: "Go article", Snippet: "Go language", WikiPriority: 1.0, FTSRank: -2.0, MatchPhase: 1},
		{ID: "high-cov", Title: "Go Python article", Snippet: "Go Python together", WikiPriority: 1.0, FTSRank: -1.0, MatchPhase: 1},
	}
	sortResults(results, []string{"Go", "Python"})
	// high-cov matches both "go" and "python" in title+snippet; low-cov only matches "go"
	if results[0].ID != "high-cov" {
		t.Errorf("higher Coverage should rank first, but got %q first", results[0].ID)
	}
}

// TestSortResults_FTSRankFinalTiebreaker verifies that FTSRank (lower=better) is the
// final tiebreaker when WikiPriority, MatchPhase, and Coverage are all equal.
func TestSortResults_FTSRankFinalTiebreaker(t *testing.T) {
	results := []SearchResult{
		{ID: "worse-rank", WikiPriority: 1.0, FTSRank: -1.0, MatchPhase: 0, Coverage: 2},
		{ID: "better-rank", WikiPriority: 1.0, FTSRank: -2.0, MatchPhase: 0, Coverage: 2},
	}
	sortResults(results, []string{"Go", "Python"})
	if results[0].ID != "better-rank" {
		t.Errorf("lower FTSRank (better) should rank first, but got %q first", results[0].ID)
	}
}

// TestSearchResultMatchPhaseFields verifies the new struct fields exist and are zero-valued by default.
func TestSearchResultMatchPhaseFields(t *testing.T) {
	r := SearchResult{ID: "test", Title: "Test"}
	if r.MatchPhase != 0 {
		t.Errorf("default MatchPhase should be 0 (AND), got %d", r.MatchPhase)
	}
	if r.Coverage != 0 {
		t.Errorf("default Coverage should be 0, got %d", r.Coverage)
	}
}

func TestMergeRelated(t *testing.T) {
	a := []RelatedDoc{
		{ID: "wiki/a.md", Title: "A", Kind: "source-note"},
		{ID: "wiki/b.md", Title: "B", Kind: "source-note"},
	}
	b := []RelatedDoc{
		{ID: "wiki/b.md", Title: "B", Kind: "source-note"}, // duplicate
		{ID: "wiki/c.md", Title: "C", Kind: "source-note"},
	}
	merged := mergeRelated(a, b, 8)
	if len(merged) != 3 {
		t.Errorf("expected 3 after dedup, got %d: %v", len(merged), merged)
	}
	// Cap test
	many := make([]RelatedDoc, 10)
	for i := range many {
		many[i] = RelatedDoc{ID: fmt.Sprintf("wiki/%d.md", i)}
	}
	capped := mergeRelated(many, nil, 5)
	if len(capped) != 5 {
		t.Errorf("expected cap at 5, got %d", len(capped))
	}
}
