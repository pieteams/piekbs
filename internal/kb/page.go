//go:build fts5

package kb

import (
	"database/sql"
	"os"
	"path/filepath"
)

const pageMaxChars = 5000

// PageResult holds the full content of a wiki page fetched by ID.
type PageResult struct {
	ID          string                 `json:"id"`
	Path        string                 `json:"path"`
	Title       string                 `json:"title"`
	Kind        string                 `json:"kind"`
	Content     string                 `json:"content"`
	Truncated   bool                   `json:"truncated,omitempty"`
	Unsupported bool                   `json:"unsupported,omitempty"`
	Frontmatter map[string]interface{} `json:"frontmatter,omitempty"`
}

// FetchPages reads one or more wiki pages by document ID and returns their content.
// When len(ids)==1 and full==true, the complete file is returned without truncation.
// In all other cases, content is capped at pageMaxChars characters.
// IDs that don't exist in the DB or whose file can't be read are skipped silently.
// Non-text documents (no .md extension) return Unsupported=true with empty content.
func FetchPages(db *sql.DB, kbRoot string, ids []string, full bool) ([]PageResult, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// full only applies to single-id requests
	applyFull := full && len(ids) == 1

	ph := placeholders(len(ids))
	args := stringsToArgs(ids)
	rows, err := db.Query(
		`SELECT id, path, COALESCE(kind,''), COALESCE(title,'') FROM documents WHERE id IN (`+ph+`)`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type docMeta struct {
		id, path, kind, title string
	}
	var docs []docMeta
	for rows.Next() {
		var d docMeta
		if err := rows.Scan(&d.id, &d.path, &d.kind, &d.title); err == nil {
			docs = append(docs, d)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var results []PageResult
	for _, d := range docs {
		absPath := filepath.Join(kbRoot, filepath.FromSlash(d.path))

		// Non-markdown files are unsupported
		if filepath.Ext(absPath) != ".md" {
			results = append(results, PageResult{
				ID: d.id, Path: d.path, Title: d.title, Kind: d.kind,
				Unsupported: true,
			})
			continue
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			continue // skip unreadable files
		}

		parsed := ParseMarkdown(string(data))
		content := string(data)
		truncated := false

		if !applyFull && len(content) > pageMaxChars {
			content = content[:pageMaxChars]
			truncated = true
		}

		fm := map[string]interface{}{}
		if parsed != nil {
			if parsed.Authority > 0 {
				fm["authority"] = parsed.Authority
			}
			if len(parsed.Sources) > 0 {
				fm["sources"] = parsed.Sources
			}
			if len(parsed.RelatedTo) > 0 {
				fm["related_to"] = parsed.RelatedTo
			}
			if len(parsed.KeyClaims) > 0 {
				fm["key_claims"] = parsed.KeyClaims
			}
		}

		results = append(results, PageResult{
			ID:          d.id,
			Path:        d.path,
			Title:       d.title,
			Kind:        d.kind,
			Content:     content,
			Truncated:   truncated,
			Frontmatter: fm,
		})
	}
	return results, nil
}
