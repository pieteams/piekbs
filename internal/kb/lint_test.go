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

// TestCleanBrokenLinks_Placeholder verifies placeholder entries are silently
// deleted with no warning — these are template noise, not actionable gaps.
func TestCleanBrokenLinks_Placeholder(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert a source doc so FK is satisfied.
	db.Exec(`INSERT OR IGNORE INTO documents (id,path,layer,kind,title,description,content,content_hash,updated_at,authority,doc_timestamp) VALUES ('wiki/a.md','wiki/a.md','wiki','source-note','A','','body','h',1,3,0)`)
	// Placeholder target
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/a.md','[]  # related wiki pages','related_to',1.0)`)

	redLinks, warnings, _, placeholders, err := cleanBrokenLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(redLinks) != 0 {
		t.Errorf("want 0 redLinks, got %d", len(redLinks))
	}
	if len(warnings) != 0 {
		t.Errorf("want 0 warnings for placeholder, got %v", warnings)
	}
	if placeholders != 1 {
		t.Errorf("want placeholders=1, got %d", placeholders)
	}

	// Verify row was deleted.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM links WHERE relation='related_to'`).Scan(&n)
	if n != 0 {
		t.Errorf("want 0 links after cleanup, got %d", n)
	}
}

// TestCleanBrokenLinks_PathAndConcept verifies path broken links produce
// broken_related warnings and concept broken links produce missing_concept
// warnings and are collected as RedLinks.
func TestCleanBrokenLinks_PathAndConcept(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec(`INSERT OR IGNORE INTO documents (id,path,layer,kind,title,description,content,content_hash,updated_at,authority,doc_timestamp) VALUES ('wiki/src.md','wiki/src.md','wiki','source-note','S','','body','h',1,3,0)`)
	// Path broken link
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/src.md','wiki/concepts/missing-page.md','related_to',1.0)`)
	// Concept broken link (referenced twice from different sources)
	db.Exec(`INSERT OR IGNORE INTO documents (id,path,layer,kind,title,description,content,content_hash,updated_at,authority,doc_timestamp) VALUES ('wiki/src2.md','wiki/src2.md','wiki','source-note','S2','','body','h2',1,3,0)`)
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/src.md','数字经济','related_to',1.0)`)
	db.Exec(`INSERT INTO links (source_doc_id,target_doc_id,relation,confidence) VALUES ('wiki/src2.md','数字经济','supports',1.0)`)

	redLinks, warnings, brokenPaths, _, err := cleanBrokenLinks(db)
	if err != nil {
		t.Fatal(err)
	}
	if brokenPaths != 1 {
		t.Errorf("want brokenPaths=1, got %d", brokenPaths)
	}
	if len(redLinks) != 1 || redLinks[0].Concept != "数字经济" || redLinks[0].Count != 2 {
		t.Errorf("want 1 redLink '数字经济' count=2, got %+v", redLinks)
	}
	if !findWarning(warnings, "wiki/src.md", "broken_related", "wiki/concepts/missing-page.md") {
		t.Errorf("expected broken_related warning; got %+v", warnings)
	}
	if !findWarning(warnings, "wiki/src.md", "missing_concept", "数字经济") {
		t.Errorf("expected missing_concept warning; got %+v", warnings)
	}
	if !findWarning(warnings, "wiki/src2.md", "missing_concept", "数字经济") {
		t.Errorf("expected missing_concept warning for src2.md; got %+v", warnings)
	}

	// All 3 broken rows deleted.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM links`).Scan(&n)
	if n != 0 {
		t.Errorf("want 0 links after cleanup, got %d", n)
	}
}
