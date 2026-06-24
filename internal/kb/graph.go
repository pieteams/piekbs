//go:build fts5

package kb

import (
	"database/sql"
	"strings"
)

// GraphNeighbor represents a document reachable via graph edges from a seed set.
type GraphNeighbor struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Layer string `json:"layer"`
	Kind  string `json:"kind"`
	Title string `json:"title"`
}

// Conflict represents a contradicts or supersedes relationship between two documents.
type Conflict struct {
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Relation string `json:"relation"`
}

// GraphBoost returns inbound edge counts per doc_id, normalized to [0, 1] (cap at 10).
func GraphBoost(db *sql.DB, docIDs []string) map[string]float64 {
	if len(docIDs) == 0 {
		return nil
	}

	ph := placeholders(len(docIDs))
	args := stringsToArgs(docIDs)

	sqlStr := `SELECT target_doc_id, COUNT(*) AS n FROM links WHERE target_doc_id IN (` + ph + `) GROUP BY target_doc_id`
	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			continue
		}
		score := float64(n) / 10.0
		if score > 1.0 {
			score = 1.0
		}
		result[id] = score
	}
	return result
}

// GraphExpand returns neighboring documents reachable from docIDs via outbound or inbound edges.
// Excludes the seed set. Ordered by title. Limited to limit results.
func GraphExpand(db *sql.DB, docIDs []string, limit int) []GraphNeighbor {
	if len(docIDs) == 0 {
		return nil
	}

	seedSet := make(map[string]bool, len(docIDs))
	for _, id := range docIDs {
		seedSet[id] = true
	}

	ph := placeholders(len(docIDs))
	args := stringsToArgs(docIDs)

	// Outbound neighbors: docIDs → target via cites/supports/wikilink/related_to
	outSQL := `SELECT DISTINCT target_doc_id FROM links
		WHERE source_doc_id IN (` + ph + `)
		AND relation IN ('cites','supports','wikilink','related_to')`

	// Inbound neighbors: docIDs ← source via supports/wikilink
	inSQL := `SELECT DISTINCT source_doc_id FROM links
		WHERE target_doc_id IN (` + ph + `)
		AND relation IN ('supports','wikilink')`

	neighborIDs := make(map[string]bool)

	for _, q := range []string{outSQL, inSQL} {
		rows, err := db.Query(q, args...)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err == nil && !seedSet[id] {
				neighborIDs[id] = true
			}
		}
		rows.Close()
	}

	if len(neighborIDs) == 0 {
		return nil
	}

	// Fetch document details for neighbor IDs.
	ids := make([]string, 0, len(neighborIDs))
	for id := range neighborIDs {
		ids = append(ids, id)
	}

	ph2 := placeholders(len(ids))
	args2 := stringsToArgs(ids)

	detailSQL := `SELECT id, path, layer, COALESCE(kind,''), COALESCE(title,'')
		FROM documents WHERE id IN (` + ph2 + `) ORDER BY title LIMIT ?`
	args2 = append(args2, limit)

	rows, err := db.Query(detailSQL, args2...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var neighbors []GraphNeighbor
	for rows.Next() {
		var n GraphNeighbor
		if err := rows.Scan(&n.ID, &n.Path, &n.Layer, &n.Kind, &n.Title); err == nil {
			neighbors = append(neighbors, n)
		}
	}
	return neighbors
}

// ConflictLinks returns all contradicts/supersedes relationships between the given documents.
// Checks both directions and deduplicates by (source, target, relation).
func ConflictLinks(db *sql.DB, docIDs []string) []Conflict {
	if len(docIDs) == 0 {
		return nil
	}

	ph := placeholders(len(docIDs))
	args := stringsToArgs(docIDs)

	conflictRelations := `('contradicts','supersedes')`

	// Both outbound and inbound directions.
	sqlStr := `
SELECT source_doc_id, target_doc_id, relation FROM links
WHERE relation IN ` + conflictRelations + `
AND (source_doc_id IN (` + ph + `) OR target_doc_id IN (` + ph + `))`

	combinedArgs := append(args, args...)
	rows, err := db.Query(sqlStr, combinedArgs...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	type key struct{ s, t, r string }
	seen := make(map[key]bool)
	var conflicts []Conflict

	for rows.Next() {
		var c Conflict
		if err := rows.Scan(&c.SourceID, &c.TargetID, &c.Relation); err != nil {
			continue
		}
		k := key{c.SourceID, c.TargetID, c.Relation}
		if !seen[k] {
			seen[k] = true
			conflicts = append(conflicts, c)
		}
	}
	return conflicts
}

// FetchRelated returns related wiki documents for a given document ID.
// It queries both outbound (related_to/supports/cites/wikilink) and
// inbound (supports/wikilink) edges, filters to wiki layer, and returns
// at most limit results.
func FetchRelated(db *sql.DB, docID string, limit int) []RelatedDoc {
	if docID == "" || limit <= 0 {
		return nil
	}
	query := `
SELECT DISTINCT d.id, COALESCE(d.kind,''), COALESCE(d.title,'')
FROM documents d
WHERE d.layer = 'wiki' AND d.id != ? AND (
    d.id IN (
        SELECT target_doc_id FROM links
        WHERE source_doc_id = ?
        AND relation IN ('related_to','supports','cites','wikilink')
    ) OR d.id IN (
        SELECT source_doc_id FROM links
        WHERE target_doc_id = ?
        AND relation IN ('supports','wikilink')
    )
)
ORDER BY d.kind, d.title
LIMIT ?`
	rows, err := db.Query(query, docID, docID, docID, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []RelatedDoc
	for rows.Next() {
		var r RelatedDoc
		if err := rows.Scan(&r.ID, &r.Kind, &r.Title); err == nil {
			result = append(result, r)
		}
	}
	return result
}

// placeholders returns a comma-separated string of n "?" placeholders.
func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	ph := make([]string, n)
	for i := range ph {
		ph[i] = "?"
	}
	return strings.Join(ph, ",")
}

// stringsToArgs converts a []string to []interface{} for sql query args.
func stringsToArgs(ss []string) []interface{} {
	args := make([]interface{}, len(ss))
	for i, s := range ss {
		args[i] = s
	}
	return args
}
