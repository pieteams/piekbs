//go:build fts5

package kb

import (
	"database/sql"
	"strings"
)

// RawSource holds a raw source document referenced by wiki pages.
type RawSource struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Title string `json:"title"`
}

// ContextBundle is the result of BuildContext: wiki pages, raw sources,
// graph neighbors, and conflict links for a given question.
type ContextBundle struct {
	Question   string         `json:"question"`
	WikiPages  []SearchResult `json:"wiki_pages"`
	RawSources []RawSource    `json:"raw_sources"`
	GraphPages []GraphNeighbor `json:"graph_pages"`
	Conflicts  []Conflict     `json:"conflicts"`
}

// minHybridScore is the minimum hybrid score for source-notes to be included.
// Synthesized pages (concept/comparison/decision) use a lower threshold because
// multiKindFTS parallel RRF naturally spreads scores across more result lists.
const minHybridScore = 0.025
const minHybridScoreSynthesized = 0.012

// maxRawSources limits the number of cited raw sources returned per bundle
// to avoid flooding the LLM context with loosely-related documents.
const maxRawSources = 3

// BuildContext assembles a ready-to-use context bundle for the given question.
// It searches the wiki layer via hybrid FTS+vector, collects raw sources cited
// by the wiki pages, expands via graph edges, and collects any conflict links.
// Results are compressed: low-score pages and near-duplicate descriptions are
// filtered out before returning. embedder may be nil (falls back to FTS only).
func BuildContext(db *sql.DB, kbRoot string, question string, embedder Embedder, limit int) ContextBundle {
	bundle := ContextBundle{Question: question}

	// 1. Search wiki layer (hybrid when embedder available, FTS-only otherwise).
	// Over-fetch so compression has candidates to work with.
	wikiLayer := "wiki"
	wikiResults, _, _, _ := Search(db, kbRoot, question, &wikiLayer, limit*2, embedder)

	// 2. Compress: remove low-score results and near-duplicate descriptions.
	wikiResults = compressResults(wikiResults, limit)
	bundle.WikiPages = wikiResults

	if len(wikiResults) == 0 {
		return bundle
	}

	// Collect wiki result IDs.
	wikiIDs := make([]string, len(wikiResults))
	for i, r := range wikiResults {
		wikiIDs[i] = r.ID
	}

	// 3. Collect raw sources cited by wiki pages (relation = 'cites').
	// Cap at maxRawSources to avoid flooding LLM context.
	rawSources := fetchCitedSources(db, wikiIDs)
	if len(rawSources) > maxRawSources {
		rawSources = rawSources[:maxRawSources]
	}
	bundle.RawSources = rawSources

	// 4. Graph expand from wiki IDs.
	// Filter to wiki layer only and cap at 3 to reduce noise from unrelated
	// documents that happen to share link relations (e.g. popular hub articles).
	allNeighbors := GraphExpand(db, wikiIDs, limit*2)
	var wikiNeighbors []GraphNeighbor
	for _, n := range allNeighbors {
		if n.Layer == "wiki" {
			wikiNeighbors = append(wikiNeighbors, n)
			if len(wikiNeighbors) >= 3 {
				break
			}
		}
	}
	bundle.GraphPages = wikiNeighbors

	// 5. Conflict links.
	bundle.Conflicts = ConflictLinks(db, wikiIDs)

	return bundle
}

// compressResults filters and deduplicates search results:
//  1. Drop results with HybridScore below minHybridScore (noise).
//  2. Drop near-duplicate descriptions (Jaccard similarity > 0.5),
//     keeping the highest-scoring result per cluster.
//  3. Truncate to limit.
func compressResults(results []SearchResult, limit int) []SearchResult {
	// Step 1: filter low-score results (only when hybrid scores are available).
	hasHybrid := false
	for _, r := range results {
		if r.HybridScore > 0 {
			hasHybrid = true
			break
		}
	}
	var filtered []SearchResult
	for _, r := range results {
		if hasHybrid {
			// Use a lower threshold for synthesized pages (concept/comparison/decision)
			// because multiKindFTS parallel RRF naturally produces lower scores for them.
			threshold := minHybridScore
			if isSynthesizedKind(r.Kind) {
				threshold = minHybridScoreSynthesized
			}
			if r.HybridScore < threshold {
				continue
			}
		}
		// Drop low-quality synthesized pages with very short descriptions.
		if isSynthesizedKind(r.Kind) && len([]rune(r.Description)) < 30 {
			continue
		}
		filtered = append(filtered, r)
	}
	// No fallback: prefer returning fewer high-quality results over flooding
	// the context with low-relevance documents. If nothing passes the score
	// threshold, return the single best result rather than nothing.
	if len(filtered) == 0 && len(results) > 0 {
		filtered = results[:1]
	}

	// Step 2: deduplicate near-identical descriptions.
	var deduped []SearchResult
	for _, r := range filtered {
		duplicate := false
		for _, kept := range deduped {
			if jaccardSimilarity(r.Description, kept.Description) > 0.5 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			deduped = append(deduped, r)
		}
	}

	// Step 3: truncate to limit.
	if len(deduped) > limit {
		deduped = deduped[:limit]
	}
	return deduped
}

// isSynthesizedKind reports whether a page kind is a synthesized wiki page
// (concept, comparison, or decision) as opposed to a source-note or raw doc.
func isSynthesizedKind(kind string) bool {
	switch kind {
	case "concept", "comparison", "decision", "wiki-concept", "wiki-comparison", "wiki-decision":
		return true
	}
	return false
}

// jaccardSimilarity returns the Jaccard similarity between two strings
// based on word-level token sets. Returns 0 for empty inputs.
func jaccardSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	setA := tokenSet(a)
	setB := tokenSet(b)

	intersection := 0
	for t := range setA {
		if setB[t] {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// tokenSet splits text into a set of lowercase words (≥2 chars).
func tokenSet(text string) map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(text)) {
		if len([]rune(w)) >= 2 {
			set[w] = true
		}
	}
	return set
}

// fetchCitedSources returns raw source documents linked from the given doc IDs
// via 'cites' relation.
func fetchCitedSources(db *sql.DB, docIDs []string) []RawSource {
	if len(docIDs) == 0 {
		return nil
	}

	ph := placeholders(len(docIDs))
	args := stringsToArgs(docIDs)

	sqlStr := `
SELECT d.id, d.path, COALESCE(d.title,'')
FROM links l
JOIN documents d ON d.id = l.target_doc_id
WHERE l.source_doc_id IN (` + ph + `)
AND l.relation = 'cites'
AND d.layer = 'raw'`

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var sources []RawSource
	for rows.Next() {
		var s RawSource
		if err := rows.Scan(&s.ID, &s.Path, &s.Title); err == nil && !seen[s.ID] {
			seen[s.ID] = true
			sources = append(sources, s)
		}
	}
	return sources
}
