//go:build fts5

package kb

import (
	"os"
	"path/filepath"
	"testing"
)

// findWarning reports whether warnings contains one matching kind+detail for path.
func findWarning(ws []LintWarning, path, kind, detail string) bool {
	for _, w := range ws {
		if w.Path == path && w.Kind == kind && w.Detail == detail {
			return true
		}
	}
	return false
}

// TestLint_MissingFieldsAndBrokenSource verifies the deterministic health
// checks: a page missing required frontmatter fields is flagged per field, a
// page citing a nonexistent source is flagged broken_source, and a complete
// page backed by a real raw file produces no warnings. These guard KB integrity
// (every wiki page must be sourced and well-formed) without any LLM.
func TestLint_MissingFieldsAndBrokenSource(t *testing.T) {
	dir := setupTestKB(t)
	notesDir := filepath.Join(dir, "wiki", "source-notes")

	// A real raw file so a valid page has something to cite.
	os.WriteFile(filepath.Join(dir, "raw", "real.md"), []byte("raw"), 0644)

	// Complete, valid page — no warnings expected.
	os.WriteFile(filepath.Join(notesDir, "good.md"),
		[]byte("---\ntitle: Good\ntype: source-note\nsources:\n  - raw/real.md\ntimestamp: 2026-06-17\n---\nbody"), 0644)

	// Missing title and timestamp; sources points at a nonexistent file.
	os.WriteFile(filepath.Join(notesDir, "bad.md"),
		[]byte("---\ntype: source-note\nsources:\n  - raw/ghost.md\n---\nbody"), 0644)

	// Reserved file must be ignored entirely.
	os.WriteFile(filepath.Join(dir, "wiki", "index.md"), []byte("# idx"), 0644)

	warnings, err := Lint(dir)
	if err != nil {
		t.Fatal(err)
	}

	// good.md must produce no warnings.
	for _, w := range warnings {
		if w.Path == "wiki/source-notes/good.md" {
			t.Errorf("good.md should be clean, got: %+v", w)
		}
	}

	// bad.md: missing title + timestamp, and a broken source.
	bad := "wiki/source-notes/bad.md"
	if !findWarning(warnings, bad, "missing_field", "title") {
		t.Errorf("expected missing title warning for bad.md; got %+v", warnings)
	}
	if !findWarning(warnings, bad, "missing_field", "timestamp") {
		t.Errorf("expected missing timestamp warning for bad.md; got %+v", warnings)
	}
	if !findWarning(warnings, bad, "broken_source", "raw/ghost.md") {
		t.Errorf("expected broken_source warning for bad.md; got %+v", warnings)
	}

	// index.md is reserved — must never appear.
	for _, w := range warnings {
		if w.Path == "wiki/index.md" {
			t.Errorf("reserved index.md should not be linted, got: %+v", w)
		}
	}
}
