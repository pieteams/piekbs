package watcher

import (
	"path/filepath"
	"testing"
)

// TestIsConvertedPath guards against the rel[:2] slice-bounds panic that fired
// when rel was "." (the converted dir itself) or a single-char name — which
// happens the moment ConvertFile creates raw/converted/ under a live watcher.
func TestIsConvertedPath(t *testing.T) {
	rawDir := filepath.FromSlash("/kb/raw")
	convertedDir := filepath.Join(rawDir, "converted")

	inside := []string{
		convertedDir,                          // rel "."  (formerly panicked)
		filepath.Join(convertedDir, "x"),      // rel "x"  (formerly panicked)
		filepath.Join(convertedDir, "a.md"),   // rel "a.md"
		filepath.Join(convertedDir, "s", "b"), // nested
	}
	for _, p := range inside {
		if !isConvertedPath(p, rawDir) {
			t.Errorf("isConvertedPath(%q) = false, want true", p)
		}
	}

	outside := []string{
		filepath.Join(rawDir, "note.md"),             // rel "../note.md"
		filepath.Join(rawDir, "convertedX", "c.md"),  // sibling dir, not converted/
	}
	for _, p := range outside {
		if isConvertedPath(p, rawDir) {
			t.Errorf("isConvertedPath(%q) = true, want false", p)
		}
	}
}
