//go:build fts5

package kb

import (
	"os"
	"path/filepath"
	"strings"
)

// requiredFMFields are frontmatter fields every wiki page must declare.
var requiredFMFields = []string{"title", "type", "sources", "timestamp"}

// LintWarning is a single issue found by Lint, tied to a KB-root-relative path.
type LintWarning struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`   // "missing_field" | "broken_source"
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
// Treats nil, empty string, and empty list as absent.
func isFMFieldAbsent(v interface{}) bool {
	switch val := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(val) == ""
	case []string:
		return len(val) == 0
	default:
		return false
	}
}
