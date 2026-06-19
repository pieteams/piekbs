//go:build fts5

package kb

import (
	"testing"
)

func TestHybridRank_FTSOnly(t *testing.T) {
	// When no vec results, HybridRank should return FTS results ranked by fts_score + wiki_priority.
	fts := []SearchResult{
		{ID: "a", Layer: "wiki", Title: "A", FTSRank: -1.0},
		{ID: "b", Layer: "raw", Title: "B", FTSRank: -2.0},
	}
	results := HybridRank(fts, nil, nil)
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// wiki layer should rank higher due to wikiPriority
	if results[0].ID != "a" {
		t.Errorf("expected wiki result first, got %q", results[0].ID)
	}
	// All results should have a non-negative hybrid score.
	// Note: the FTS-worst result will score 0 after normalization — that is correct.
	for _, r := range results {
		if r.HybridScore < 0 {
			t.Errorf("result %q has negative hybrid score %f", r.ID, r.HybridScore)
		}
	}
	// The wiki result should score strictly higher than the raw result.
	if results[0].HybridScore <= results[1].HybridScore {
		t.Errorf("wiki result should outrank raw result: %f <= %f", results[0].HybridScore, results[1].HybridScore)
	}
}

func TestHybridRank_MergesFTSAndVec(t *testing.T) {
	// Overlapping results should be merged and deduplicated.
	fts := []SearchResult{
		{ID: "a", Layer: "wiki", Title: "A", FTSRank: -1.0},
		{ID: "b", Layer: "raw", Title: "B", FTSRank: -2.0},
	}
	vec := []SearchResult{
		{ID: "b", Layer: "raw", Title: "B", VecScore: 0.9},
		{ID: "c", Layer: "raw", Title: "C", VecScore: 0.7},
	}
	results := HybridRank(fts, vec, nil)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3 (a, b, c)", len(results))
	}
	// b should have both fts and vec scores contributing
	var bResult *SearchResult
	for i := range results {
		if results[i].ID == "b" {
			bResult = &results[i]
			break
		}
	}
	if bResult == nil {
		t.Fatal("result 'b' not found")
	}
	if bResult.VecScore == 0 {
		t.Error("merged result 'b' should have vec_score")
	}
	// b appears in both FTS (rank 2) and vec (rank 1), so RRF score should be
	// higher than c which only appears in vec (rank 2).
	var cScore float64
	for _, r := range results {
		if r.ID == "c" {
			cScore = r.HybridScore
			break
		}
	}
	if bResult.HybridScore <= cScore {
		t.Errorf("b (in both FTS+vec) should outscore c (vec-only): b=%f c=%f",
			bResult.HybridScore, cScore)
	}
}

func TestHybridRank_WikiBoost(t *testing.T) {
	// wiki-layer docs should score higher than raw-layer docs with equal FTS rank.
	fts := []SearchResult{
		{ID: "wiki-doc", Layer: "wiki", Title: "W", FTSRank: -1.0},
		{ID: "raw-doc", Layer: "raw", Title: "R", FTSRank: -1.0},
	}
	results := HybridRank(fts, nil, nil)
	var wScore, rScore float64
	for _, r := range results {
		if r.ID == "wiki-doc" {
			wScore = r.HybridScore
		}
		if r.ID == "raw-doc" {
			rScore = r.HybridScore
		}
	}
	if wScore <= rScore {
		t.Errorf("wiki doc should outscore raw doc: wiki=%f raw=%f", wScore, rScore)
	}
}

func TestSearch_NoEmbedder(t *testing.T) {
	// Search without embedder should use FTS only.
	dir, db := setupSearchTest(t)

	results, neighbors, conflicts, err := Search(db, dir, "programming", nil, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Errorf("got %d results, want ≥2", len(results))
	}
	// neighbors and conflicts may be nil/empty on a simple test KB; just verify no panic.
	_ = neighbors
	_ = conflicts
}

func TestSearch_EmptyQuery(t *testing.T) {
	dir, db := setupSearchTest(t)

	results, neighbors, conflicts, err := Search(db, dir, "", nil, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("empty query should return 0 results, got %d", len(results))
	}
	_ = neighbors
	_ = conflicts
}
