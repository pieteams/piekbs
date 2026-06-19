//go:build fts5

package kb

import (
	"database/sql"
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

// BuildContext assembles a ready-to-use context bundle for the given question.
// It searches the wiki layer via hybrid FTS+vector, collects raw sources cited
// by the wiki pages, expands via graph edges, and collects any conflict links.
// embedder may be nil (falls back to FTS only).
func BuildContext(db *sql.DB, kbRoot string, question string, embedder Embedder, limit int) ContextBundle {
	bundle := ContextBundle{Question: question}

	// 1. Search wiki layer (hybrid when embedder available, FTS-only otherwise).
	wikiLayer := "wiki"
	wikiResults, _, _, _ := Search(db, kbRoot, question, &wikiLayer, limit, embedder)
	bundle.WikiPages = wikiResults

	if len(wikiResults) == 0 {
		return bundle
	}

	// Collect wiki result IDs.
	wikiIDs := make([]string, len(wikiResults))
	for i, r := range wikiResults {
		wikiIDs[i] = r.ID
	}

	// 2. Collect raw sources cited by wiki pages (relation = 'cites').
	bundle.RawSources = fetchCitedSources(db, wikiIDs)

	// 3. Graph expand from wiki IDs.
	bundle.GraphPages = GraphExpand(db, wikiIDs, limit)

	// 4. Conflict links.
	bundle.Conflicts = ConflictLinks(db, wikiIDs)

	return bundle
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
