//go:build fts5

package kb

import (
	"database/sql"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Embedder computes a vector embedding for a text string.
// Kept for interface compatibility; passing nil disables vector search.
type Embedder interface {
	Encode(text string) ([]float32, error)
}

// RelatedDoc is a lightweight reference to a related wiki document,
// embedded in SearchResult for graph navigation.
type RelatedDoc struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
}

// SearchResult holds a single result from FTS or hybrid search.
type SearchResult struct {
	ID           string       `json:"id"`
	DocID        string       `json:"doc_id,omitempty"` // parent doc ID when result is a chunk
	Path         string       `json:"path"`
	Layer        string       `json:"layer"`
	Kind         string       `json:"kind"`
	Title        string       `json:"title"`
	Description  string       `json:"description,omitempty"`
	Snippet      string       `json:"snippet,omitempty"`
	WikiPriority float64      `json:"wiki_priority"`
	Authority    int          `json:"authority,omitempty"`
	FTSRank      float64      `json:"fts_rank,omitempty"`
	FTSScore     float64      `json:"fts_score,omitempty"`
	VecScore     float64      `json:"vec_score,omitempty"`
	HybridScore  float64      `json:"hybrid_score,omitempty"`
	MatchPhase   int          `json:"match_phase,omitempty"` // 0=AND, 1=OR, 2=LIKE
	Coverage     int          `json:"coverage,omitempty"`    // 命中查询词条数（AND 结果=总词数，OR/LIKE 由 sortResults 补全）
	GraphBoost   float64      `json:"graph_boost,omitempty"`
	Related      []RelatedDoc `json:"related,omitempty"`
	Conflicts    []string     `json:"conflicts,omitempty"`
	DocTimestamp int64        `json:"doc_timestamp,omitempty"`
}

// minTrigramLen is the minimum token length for FTS5 trigram tokenizer.
const minTrigramLen = 3

// FTSSearch performs a full-text search over the documents table.
// query supports comma-separated keywords: "Go, Python" → per-keyword search.
// layer optionally filters results to a specific layer (raw/wiki/schema).
// kind optionally filters by page kind (source-note, concept, comparison, decision).
// Returns nil (not error) for empty queries.
//
// FTS5 trigram tokenizer requires tokens ≥ 3 chars. Keywords shorter than
// minTrigramLen fall back to SQL LIKE matching on title and content.
// Results are deduplicated by ID before being returned.
func FTSSearch(db *sql.DB, query string, layer *string, limit int) ([]SearchResult, error) {
	return FTSSearchFiltered(db, query, layer, nil, limit)
}

// FTSSearchFiltered is like FTSSearch but also accepts an optional kind filter.
func FTSSearchFiltered(db *sql.DB, query string, layer, kind *string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	keywords := splitKeywords(query)

	// Separate long keywords (FTS5) from short keywords (LIKE fallback).
	var ftsKws, likeKws []string
	for _, kw := range keywords {
		if utf8.RuneCountInString(kw) >= minTrigramLen {
			ftsKws = append(ftsKws, kw)
		} else {
			likeKws = append(likeKws, kw)
		}
	}

	// Expand limit for multi-keyword OR queries.
	effectiveLimit := limit * len(keywords)

	seen := make(map[string]bool)
	var results []SearchResult

	// FTS5 two-phase strategy: AND first (precision), OR fallback (recall).
	if len(ftsKws) > 0 {
		// Phase 1: AND query — all keywords must appear in the document.
		// This eliminates cross-context false positives (e.g. "recall" in
		// Agent memory articles when searching for RAG recall rate).
		if len(ftsKws) >= 2 {
			andQuery := buildFTSAndQuery(ftsKws)
			res, err := ftsQuery_(db, andQuery, layer, kind, effectiveLimit)
			if err != nil {
				return nil, err
			}
			for _, r := range res {
				if !seen[r.ID] {
					seen[r.ID] = true
					r.MatchPhase = 0
					r.Coverage = len(keywords)
					results = append(results, r)
				}
			}
		}

		// Phase 2: OR query — any keyword matches (fallback for coverage).
		// Only adds documents not already found by AND query.
		orQuery := buildFTSQuery(ftsKws)
		res, err := ftsQuery_(db, orQuery, layer, kind, effectiveLimit)
		if err != nil {
			return nil, err
		}
		for _, r := range res {
			if !seen[r.ID] {
				seen[r.ID] = true
				r.MatchPhase = 1
				results = append(results, r)
			}
		}
	}

	// LIKE fallback for short keywords (e.g., "Go", "C").
	for _, kw := range likeKws {
		res, err := likeSearch(db, kw, layer, kind, effectiveLimit)
		if err != nil {
			return nil, err
		}
		for _, r := range res {
			if !seen[r.ID] {
				seen[r.ID] = true
				r.MatchPhase = 2
				results = append(results, r)
			}
		}
	}

	// Sort: WikiPriority → MatchPhase → Coverage → FTSRank.
	sortResults(results, keywords)

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// ftsQuery_ executes a FTS5 MATCH query and returns SearchResult rows.
func ftsQuery_(db *sql.DB, ftsQuery string, layer, kind *string, limit int) ([]SearchResult, error) {
	var args []interface{}
	args = append(args, ftsQuery)

	layerFilter := ""
	if layer != nil {
		layerFilter = "AND d.layer = ?"
		args = append(args, *layer)
	}
	kindFilter := ""
	if kind != nil {
		kindFilter = "AND d.kind = ?"
		args = append(args, *kind)
	}
	args = append(args, limit)

	sqlStr := `
SELECT
    d.id,
    d.path,
    d.layer,
    COALESCE(d.kind, '') AS kind,
    COALESCE(d.title, '') AS title,
    COALESCE(d.description, '') AS description,
    snippet(document_fts, 2, '[', ']', '...', 10) AS snippet,
    CASE d.layer WHEN 'wiki' THEN 1.0 ELSE 0.0 END AS wiki_priority,
    COALESCE(d.authority, 3) AS authority,
    rank AS fts_rank,
    COALESCE(d.doc_timestamp, 0) AS doc_timestamp
FROM document_fts
JOIN documents d ON d.id = document_fts.id
WHERE document_fts MATCH ?
` + layerFilter + kindFilter + `
ORDER BY wiki_priority DESC, rank
LIMIT ?`

	return scanResults(db, sqlStr, args...)
}

// likeSearch performs LIKE-based fallback search for short keywords (< 3 chars).
func likeSearch(db *sql.DB, kw string, layer, kind *string, limit int) ([]SearchResult, error) {
	pattern := "%" + kw + "%"
	var args []interface{}
	args = append(args, pattern, pattern)

	layerFilter := ""
	if layer != nil {
		layerFilter = "AND layer = ?"
		args = append(args, *layer)
	}
	kindFilter := ""
	if kind != nil {
		kindFilter = "AND kind = ?"
		args = append(args, *kind)
	}
	args = append(args, limit)

	sqlStr := `
SELECT
    id,
    path,
    layer,
    COALESCE(kind, '') AS kind,
    COALESCE(title, '') AS title,
    COALESCE(description, '') AS description,
    '' AS snippet,
    CASE layer WHEN 'wiki' THEN 1.0 ELSE 0.0 END AS wiki_priority,
    COALESCE(authority, 3) AS authority,
    0.0 AS fts_rank,
    COALESCE(doc_timestamp, 0) AS doc_timestamp
FROM documents
WHERE (title LIKE ? OR content LIKE ?)
` + layerFilter + kindFilter + `
ORDER BY wiki_priority DESC
LIMIT ?`

	return scanResults(db, sqlStr, args...)
}

// scanResults executes a query and scans rows into SearchResult slice.
func scanResults(db *sql.DB, sqlStr string, args ...interface{}) ([]SearchResult, error) {
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(
			&r.ID, &r.Path, &r.Layer, &r.Kind, &r.Title,
			&r.Description, &r.Snippet, &r.WikiPriority, &r.Authority, &r.FTSRank, &r.DocTimestamp,
		); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// sortResults sorts results by: WikiPriority → MatchPhase → Coverage → FTSRank.
// keywords is used to compute Coverage for OR/LIKE results (AND results already have Coverage set).
func sortResults(results []SearchResult, keywords []string) {
	// Compute Coverage for OR/LIKE results by counting keyword hits in Title+Snippet.
	for i := range results {
		if results[i].MatchPhase > 0 && len(keywords) > 0 {
			text := strings.ToLower(results[i].Title + " " + results[i].Snippet)
			count := 0
			for _, kw := range keywords {
				if strings.Contains(text, strings.ToLower(kw)) {
					count++
				}
			}
			results[i].Coverage = count
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.WikiPriority != b.WikiPriority {
			return a.WikiPriority > b.WikiPriority
		}
		if a.MatchPhase != b.MatchPhase {
			return a.MatchPhase < b.MatchPhase
		}
		if a.Coverage != b.Coverage {
			return a.Coverage > b.Coverage
		}
		return a.FTSRank < b.FTSRank
	})
}

// splitRe splits on commas (ASCII/fullwidth) or whitespace (one or more).
var splitRe = regexp.MustCompile(`[,，]+|\s+`)

// splitKeywords splits a query on commas and whitespace into trimmed keywords,
// filtering out empty entries.
func splitKeywords(query string) []string {
	parts := splitRe.Split(query, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// buildFTSQuery converts keywords into a FTS5 OR expression with quoted phrases.
// Single keyword: `"Go"` → `"Go"`.
// Multiple: `"Go" OR "Python"`.
func buildFTSQuery(keywords []string) string {
	quoted := make([]string, len(keywords))
	for i, kw := range keywords {
		// Escape any existing double-quotes inside keyword.
		kw = strings.ReplaceAll(kw, `"`, `""`)
		quoted[i] = `"` + kw + `"`
	}
	return strings.Join(quoted, " OR ")
}

// buildFTSAndQuery converts keywords into a FTS5 AND expression.
// All keywords must appear in the document.
// Example: ["RAG", "召回率"] → `"RAG" AND "召回率"`
func buildFTSAndQuery(keywords []string) string {
	quoted := make([]string, len(keywords))
	for i, kw := range keywords {
		kw = strings.ReplaceAll(kw, `"`, `""`)
		quoted[i] = `"` + kw + `"`
	}
	return strings.Join(quoted, " AND ")
}

// rrfK is the constant in Reciprocal Rank Fusion: score = 1/(k + rank).
// k=60 is the standard value from the original RRF paper (Cormack et al., 2009).
const rrfK = 60.0

// multiKindFTS runs FTS queries for each wiki kind in parallel and merges via RRF.
// This gives each kind (source-note, concept, comparison, decision) an equal chance
// to surface top results, preventing high-volume kinds from drowning out others.
// When kind is non-nil, falls back to a single FTSSearchFiltered call.
func multiKindFTS(db *sql.DB, query string, layer, kind *string, limit int) ([]SearchResult, error) {
	if kind != nil {
		return FTSSearchFiltered(db, query, layer, kind, limit)
	}
	wikiLayer := "wiki"
	kinds := []string{"source-note", "concept", "comparison", "decision"}

	type kindResult struct {
		results []SearchResult
		err     error
	}
	ch := make(chan kindResult, len(kinds)+1)

	for _, k := range kinds {
		k := k
		go func() {
			// Use a large limit for source-notes (high volume kind) so lower-ranked
			// but relevant results aren't truncated before the RRF merge.
			perLimit := limit * 3
			if k == "source-note" {
				perLimit = limit * 10
			}
			res, err := FTSSearchFiltered(db, query, &wikiLayer, strPtr(k), perLimit)
			ch <- kindResult{res, err}
		}()
	}

	// Also run an unrestricted FTS to catch raw-layer and unclassified docs.
	go func() {
		res, err := FTSSearchFiltered(db, query, layer, nil, limit*3)
		ch <- kindResult{res, err}
	}()

	// Collect results from all goroutines.
	total := len(kinds) + 1
	allLists := make([][]SearchResult, 0, total)
	for i := 0; i < total; i++ {
		kr := <-ch
		if kr.err != nil {
			continue
		}
		if len(kr.results) > 0 {
			allLists = append(allLists, kr.results)
		}
	}

	// RRF merge across all lists.
	type entry struct {
		r    SearchResult
		score float64
	}
	merged := make(map[string]*entry)
	for _, list := range allLists {
		for rank, r := range list {
			score := 1.0 / (float64(rrfK) + float64(rank+1))
			if e, ok := merged[r.ID]; ok {
				e.score += score
			} else {
				merged[r.ID] = &entry{r: r, score: score}
			}
		}
	}

	results := make([]SearchResult, 0, len(merged))
	for _, e := range merged {
		r := e.r
		r.HybridScore = e.score
		results = append(results, r)
	}
	// Sort by RRF score descending.
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].HybridScore > results[j-1].HybridScore; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func strPtr(s string) *string { return &s }

// SearchLayered runs FTS search and returns source-notes and synthesized pages
// in separate quota pools. sourceLimit caps source-note results; synthLimit caps
// concept/comparison/decision results. Related docs are attached to each result.
func SearchLayered(db *sql.DB, kbRoot, query string, layer, kind *string, sourceLimit, synthLimit int) ([]SearchResult, []Conflict, error) {
	fetchLimit := (sourceLimit + synthLimit) * 4
	ftsResults, err := multiKindFTS(db, query, layer, kind, fetchLimit)
	if err != nil {
		return nil, nil, err
	}
	results := applyGraphBoost(db, ftsResults)

	// Decay constants: λ controls half-life.
	// source-note: λ=0.00126 → ~1.5yr half-life
	// synthesized (concept/comparison/decision): λ=0.00063 → ~3yr half-life
	const (
		decayLambdaNote  = 0.00126
		decayLambdaSynth = 0.00063
		decayBaseBonus   = 0.012 // max time bonus ≈ top-2 FTS position
	)
	now := time.Now().Unix()

	// Apply synthesizedBoost (multiplicative), authorityBoost, and time decay.
	for i := range results {
		score := 1.0 / (rrfK + float64(i+1)) // base FTS rank score
		if isSynthesizedKind(results[i].Kind) {
			score *= 1.3
		}
		if results[i].Authority > 0 {
			score += float64(results[i].Authority-3) * 0.005
		}
		score += results[i].GraphBoost * 0.01
		// Time decay: newer docs score higher. Skip if no timestamp (doc_timestamp=0).
		if results[i].DocTimestamp > 0 {
			days := float64(now-results[i].DocTimestamp) / 86400.0
			λ := decayLambdaNote
			if isSynthesizedKind(results[i].Kind) {
				λ = decayLambdaSynth
			}
			score += decayBaseBonus * math.Exp(-λ*days)
		}
		results[i].HybridScore = score
	}

	// Sort by HybridScore descending.
	sort.Slice(results, func(i, j int) bool {
		return results[i].HybridScore > results[j].HybridScore
	})

	// Split into source-notes and synthesized, cap each pool.
	var notes, synth []SearchResult
	for _, r := range results {
		if isSynthesizedKind(r.Kind) {
			if len(synth) < synthLimit {
				synth = append(synth, r)
			}
		} else {
			if len(notes) < sourceLimit {
				notes = append(notes, r)
			}
		}
		if len(notes) >= sourceLimit && len(synth) >= synthLimit {
			break
		}
	}

	// Attach related docs to each result (links-based + tag-based multi-hop).
	combined := append(notes, synth...)
	for i := range combined {
		linkRelated := FetchRelated(db, combined[i].ID, 3)
		tagNeighbors := TagExpand(db, []string{combined[i].ID}, 1, 5)
		tagRelated := make([]RelatedDoc, len(tagNeighbors))
		for j, n := range tagNeighbors {
			tagRelated[j] = RelatedDoc{ID: n.ID, Title: n.Title, Kind: n.Kind}
		}
		combined[i].Related = mergeRelated(linkRelated, tagRelated, 8)
	}

	// Collect conflict links.
	ids := make([]string, len(combined))
	for i, r := range combined {
		ids[i] = r.ID
	}
	conflicts := ConflictLinks(db, ids)

	return combined, conflicts, nil
}

// HybridRank performs hybrid FTS search and applies RRF scoring with
// synthesizedBoost (multiplicative 1.3x for concept/comparison/decision)
// and authority adjustments. Returns results sorted by HybridScore descending.
// graph may be nil; embedder may be nil (FTS-only mode).
func HybridRank(fts []SearchResult, graph map[string]float64, conflicts []Conflict, embedder Embedder) []SearchResult {
	results := make([]SearchResult, len(fts))
	copy(results, fts)

	for i := range results {
		rrfScore := 1.0 / (rrfK + float64(i+1))
		if isSynthesizedKind(results[i].Kind) {
			rrfScore *= 1.3
		}
		if graph != nil {
			if boost, ok := graph[results[i].ID]; ok {
				rrfScore += boost * 0.01
			}
		}
		if results[i].Authority > 0 {
			rrfScore += float64(results[i].Authority-3) * 0.005
		}
		results[i].HybridScore = rrfScore
	}

	// Sort by HybridScore descending.
	sort.Slice(results, func(i, j int) bool {
		return results[i].HybridScore > results[j].HybridScore
	})
	return results
}

// Search performs hybrid FTS search over the documents table.
// Always performs GraphExpand and ConflictLinks on the result set.
func Search(db *sql.DB, kbRoot string, query string, layer *string, limit int, embedder Embedder) ([]SearchResult, []GraphNeighbor, []Conflict, error) {
	return SearchFiltered(db, kbRoot, query, layer, nil, limit, embedder)
}

// SearchFiltered is like Search but also accepts an optional kind filter.
func SearchFiltered(db *sql.DB, kbRoot string, query string, layer, kind *string, limit int, embedder Embedder) ([]SearchResult, []GraphNeighbor, []Conflict, error) {
	fetchLimit := limit * 4
	ftsResults, err := multiKindFTS(db, query, layer, kind, fetchLimit)
	if err != nil {
		return nil, nil, nil, err
	}
	results := applyGraphBoost(db, ftsResults)
	if len(results) > limit {
		results = results[:limit]
	}
	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.ID
	}
	neighbors := GraphExpand(db, ids, limit)
	conflicts := ConflictLinks(db, ids)
	return results, neighbors, conflicts, nil
}

// applyGraphBoost applies GraphBoost scores to FTS results and returns them.
func applyGraphBoost(db *sql.DB, ftsResults []SearchResult) []SearchResult {
	if len(ftsResults) == 0 {
		return ftsResults
	}
	ids := make([]string, len(ftsResults))
	for i, r := range ftsResults {
		ids[i] = r.ID
	}
	boostMap := GraphBoost(db, ids)
	for i := range ftsResults {
		ftsResults[i].GraphBoost = boostMap[ftsResults[i].ID]
	}
	return ftsResults
}

// mergeRelated merges two RelatedDoc slices, deduplicating by ID, capped at maxTotal.
// Items from a take priority over items from b.
func mergeRelated(a, b []RelatedDoc, maxTotal int) []RelatedDoc {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]RelatedDoc, 0, maxTotal)
	for _, r := range append(a, b...) {
		if seen[r.ID] || len(out) >= maxTotal {
			continue
		}
		seen[r.ID] = true
		out = append(out, r)
	}
	return out
}

