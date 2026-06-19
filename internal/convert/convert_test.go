package convert

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindConvertibleFiles(t *testing.T) {
	dir := t.TempDir()
	rawDir := filepath.Join(dir, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create convertible files
	for _, name := range []string{"doc.pdf", "page.html"} {
		if err := os.WriteFile(filepath.Join(rawDir, name), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Create a markdown file (should be skipped)
	if err := os.WriteFile(filepath.Join(rawDir, "note.md"), []byte("# note"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := FindConvertibleFiles(dir)
	if len(files) != 2 {
		t.Errorf("expected 2 convertible files, got %d: %v", len(files), files)
	}
}

func TestFindConvertibleFiles_SkipConverted(t *testing.T) {
	dir := t.TempDir()
	rawDir := filepath.Join(dir, "raw")
	convertedDir := filepath.Join(rawDir, "converted")
	if err := os.MkdirAll(convertedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a PDF in raw/
	if err := os.WriteFile(filepath.Join(rawDir, "doc.pdf"), []byte("pdf content"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create the corresponding converted markdown
	if err := os.WriteFile(filepath.Join(convertedDir, "doc.md"), []byte("# doc"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := FindConvertibleFiles(dir)
	if len(files) != 0 {
		t.Errorf("expected 0 convertible files (already converted), got %d: %v", len(files), files)
	}
}

func TestFindConverter(t *testing.T) {
	// Just ensure the function runs without panicking.
	// The result depends on the environment; either a path or empty string is valid.
	result := FindConverter()
	t.Logf("FindConverter() = %q", result)
}
