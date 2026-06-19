//go:build fts5

package kb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerateOKFIndex_SubdirAndRoot verifies that OKF index generation creates
// a per-directory index.md listing each page, and a root index.md carrying the
// okf_version marker and subdirectory page counts. This is the OKF spec
// contract; if listings or the version marker regress, navigation breaks.
func TestGenerateOKFIndex_SubdirAndRoot(t *testing.T) {
	dir := setupTestKB(t)
	notesDir := filepath.Join(dir, "wiki", "source-notes")

	os.WriteFile(filepath.Join(notesDir, "alpha.md"),
		[]byte("---\ntitle: Alpha Note\ndescription: first\n---\nbody"), 0644)
	os.WriteFile(filepath.Join(notesDir, "beta.md"),
		[]byte("---\ntitle: Beta Note\n---\nbody"), 0644)

	if err := GenerateOKFIndex(filepath.Join(dir, "wiki")); err != nil {
		t.Fatal(err)
	}

	// Per-directory index lists both pages, with description when present.
	sub, err := os.ReadFile(filepath.Join(notesDir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	subStr := string(sub)
	if !strings.Contains(subStr, "* [Alpha Note](alpha.md) — first") {
		t.Errorf("subdir index missing alpha entry with description:\n%s", subStr)
	}
	if !strings.Contains(subStr, "* [Beta Note](beta.md)") {
		t.Errorf("subdir index missing beta entry:\n%s", subStr)
	}

	// Root index carries okf_version and the subdir with its page count.
	root, err := os.ReadFile(filepath.Join(dir, "wiki", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	rootStr := string(root)
	if !strings.Contains(rootStr, `okf_version: "0.1"`) {
		t.Errorf("root index missing okf_version marker:\n%s", rootStr)
	}
	if !strings.Contains(rootStr, "* [source-notes](source-notes/) — 2 page(s)") {
		t.Errorf("root index missing source-notes count:\n%s", rootStr)
	}
}

// TestPurgeOrphanWikiFiles verifies that a source-note whose raw/ origin was
// deleted gets removed, while a source-note backed by an existing raw file and
// reserved files are kept. This protects against stale wiki pages lingering
// after their source disappears.
func TestPurgeOrphanWikiFiles(t *testing.T) {
	dir := setupTestKB(t)
	notesDir := filepath.Join(dir, "wiki", "source-notes")

	// raw/kept.md exists → source-note kept.md is backed.
	os.WriteFile(filepath.Join(dir, "raw", "kept.md"), []byte("raw"), 0644)
	os.WriteFile(filepath.Join(notesDir, "kept.md"), []byte("note"), 0644)
	// no raw/orphan.md → source-note orphan.md is an orphan.
	os.WriteFile(filepath.Join(notesDir, "orphan.md"), []byte("note"), 0644)
	// reserved file must survive.
	os.WriteFile(filepath.Join(notesDir, "index.md"), []byte("# idx"), 0644)

	removed, err := PurgeOrphanWikiFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Errorf("removed %d, want 1", removed)
	}
	if _, err := os.Stat(filepath.Join(notesDir, "orphan.md")); !os.IsNotExist(err) {
		t.Error("orphan.md should have been removed")
	}
	if _, err := os.Stat(filepath.Join(notesDir, "kept.md")); err != nil {
		t.Error("kept.md should survive (backed by raw/kept.md)")
	}
	if _, err := os.Stat(filepath.Join(notesDir, "index.md")); err != nil {
		t.Error("index.md is reserved and must survive")
	}
}

// TestPurgeOrphanWikiFiles_MissingRawIsSafe guards against catastrophic data
// loss: when raw/ is absent (e.g. an unmounted drive or a path typo), the
// survivor set is unknowable, so purge must delete nothing rather than treat
// every source-note as an orphan. A regression here silently wipes the wiki.
func TestPurgeOrphanWikiFiles_MissingRawIsSafe(t *testing.T) {
	dir := t.TempDir()
	notesDir := filepath.Join(dir, "wiki", "source-notes")
	os.MkdirAll(notesDir, 0755)
	// A real source-note exists, but there is no raw/ directory at all.
	os.WriteFile(filepath.Join(notesDir, "note.md"), []byte("note"), 0644)

	removed, err := PurgeOrphanWikiFiles(dir)
	if err != nil {
		t.Fatalf("missing raw/ should be a safe no-op, got error: %v", err)
	}
	if removed != 0 {
		t.Errorf("removed %d with no raw/ dir, want 0 (must not wipe notes)", removed)
	}
	if _, err := os.Stat(filepath.Join(notesDir, "note.md")); err != nil {
		t.Error("note.md must survive when raw/ is missing")
	}
}
