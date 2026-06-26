//go:build fts5

package kb

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// requiredFMFields are frontmatter fields every wiki page must declare.
var requiredFMFields = []string{"title", "type", "sources", "timestamp"}

// LintWarning is a single issue found by Lint, tied to a KB-root-relative path.
type LintWarning struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`   // "missing_field" | "broken_source" | "broken_related" | "missing_concept"
	Detail string `json:"detail"` // field name or source path
}

// Lint checks every wiki page (excluding reserved index.md/log.md) for:
//   - missing required frontmatter fields (title/type/sources/timestamp)
//   - broken source links (a sources entry pointing at a nonexistent file)
//
// A page with no sources is already reported via the missing 'sources' field,
// so there is no separate orphan check. It performs only deterministic checks —
// no LLM, no semantic staleness or contradiction analysis. Returns warnings in
// stable path order.
func Lint(kbRoot string) ([]LintWarning, error) {
	wikiDir := filepath.Join(kbRoot, "wiki")
	if _, err := os.Stat(wikiDir); os.IsNotExist(err) {
		return nil, nil
	}

	var warnings []LintWarning

	err := filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" || okfReserved[info.Name()] {
			return nil
		}
		// Skip _draft/ pages — insufficient sources is expected, not a lint error.
		rel0, _ := filepath.Rel(kbRoot, path)
		if strings.Contains(filepath.ToSlash(rel0), "/_draft/") {
			return nil
		}

		parsed, perr := ParseMarkdownFile(path)
		if perr != nil {
			return nil
		}
		rel, _ := filepath.Rel(kbRoot, path)
		rel = filepath.ToSlash(rel)

		// Missing required frontmatter fields.
		for _, field := range requiredFMFields {
			if isFMFieldAbsent(parsed.RawFM[field]) {
				warnings = append(warnings, LintWarning{
					Path: rel, Kind: "missing_field", Detail: field,
				})
			}
		}

		// Broken source links.
		for _, src := range parsed.Sources {
			src = strings.TrimSpace(src)
			if src == "" {
				continue
			}
			if _, statErr := os.Stat(filepath.Join(kbRoot, filepath.FromSlash(src))); os.IsNotExist(statErr) {
				warnings = append(warnings, LintWarning{
					Path: rel, Kind: "broken_source", Detail: src,
				})
			}
		}
		return nil
	})
	if err != nil {
		return warnings, err
	}
	return warnings, nil
}

// isFMFieldAbsent reports whether a frontmatter value is missing or empty.
// Treats nil and empty string as absent. An explicit empty list (sources: [])
// is considered present — it means "no sources" rather than "field missing".
func isFMFieldAbsent(v interface{}) bool {
	switch val := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(val) == ""
	case []string:
		_ = val
		return false // explicit empty list is present
	default:
		return false
	}
}

// RedLink represents a concept name referenced in related_to/supports/contradicts
// that has no corresponding wiki page. Used to surface knowledge gaps.
type RedLink struct {
	Concept      string   `json:"concept"`
	Count        int      `json:"count"`
	ReferencedBy []string `json:"referenced_by"`
}

// cleanBrokenLinks scans links table for broken related_to/supports/contradicts
// entries and deletes them all. Returns:
//   - redLinks: concept-name broken links (no "/" in target) for red_links.json
//   - warnings: LintWarnings for path broken links (broken_related) and concept
//     broken links (missing_concept)
//   - brokenPaths: count of path-format broken links deleted
//   - placeholders: count of placeholder entries deleted (silent)
//
// Side effect: deletes broken rows from the links table.
func cleanBrokenLinks(db *sql.DB) ([]RedLink, []LintWarning, int, int, error) {
	const selectSQL = `
SELECT l.id, l.relation, l.source_doc_id, l.target_doc_id,
  CASE
    WHEN trim(l.target_doc_id) = ''
      OR l.target_doc_id LIKE '#%'
      OR l.target_doc_id LIKE '[%'       THEN 'placeholder'
    WHEN instr(l.target_doc_id, '/') > 0 THEN 'path'
    ELSE                                      'concept'
  END AS link_type
FROM links l
LEFT JOIN documents d ON d.id = l.target_doc_id
WHERE l.relation IN ('related_to', 'supports', 'contradicts')
  AND d.id IS NULL`

	rows, err := db.Query(selectSQL)
	if err != nil {
		return nil, nil, 0, 0, err
	}
	defer rows.Close()

	conceptMap := make(map[string]*RedLink)
	var warnings []LintWarning
	brokenPaths, placeholders := 0, 0

	for rows.Next() {
		var id int64
		var relation, sourceDID, targetDID, linkType string
		if err := rows.Scan(&id, &relation, &sourceDID, &targetDID, &linkType); err != nil {
			return nil, nil, 0, 0, err
		}
		switch linkType {
		case "placeholder":
			placeholders++
		case "path":
			brokenPaths++
			warnings = append(warnings, LintWarning{
				Path:   sourceDID,
				Kind:   "broken_related",
				Detail: targetDID,
			})
		case "concept":
			rl, ok := conceptMap[targetDID]
			if !ok {
				rl = &RedLink{Concept: targetDID}
				conceptMap[targetDID] = rl
			}
			rl.Count++
			rl.ReferencedBy = append(rl.ReferencedBy, sourceDID)
			warnings = append(warnings, LintWarning{
				Path:   sourceDID,
				Kind:   "missing_concept",
				Detail: targetDID,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, 0, 0, err
	}

	// Delete all broken rows in one statement.
	const deleteSQL = `
DELETE FROM links
WHERE relation IN ('related_to', 'supports', 'contradicts')
  AND id NOT IN (
    SELECT l.id FROM links l
    JOIN documents d ON d.id = l.target_doc_id
    WHERE l.relation IN ('related_to', 'supports', 'contradicts')
  )`
	if _, err := db.Exec(deleteSQL); err != nil {
		return nil, nil, 0, 0, err
	}

	// Convert concept map to sorted slice (by count desc).
	redLinks := make([]RedLink, 0, len(conceptMap))
	for _, rl := range conceptMap {
		redLinks = append(redLinks, *rl)
	}
	sort.Slice(redLinks, func(i, j int) bool {
		return redLinks[i].Count > redLinks[j].Count
	})
	return redLinks, warnings, brokenPaths, placeholders, nil
}
