package convert

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
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

// TestInjectEmbeddedXlsx_NoPlaceholder verifies that content without image
// placeholders is returned unchanged — embedded Excel extraction must not
// corrupt documents that have no OLE objects.
func TestInjectEmbeddedXlsx_NoPlaceholder(t *testing.T) {
	md := []byte("# Title\n\nSome text without any image placeholder.")
	result := injectEmbeddedXlsx("markitdown", "dummy.docx", md)
	if string(result) != string(md) {
		t.Errorf("expected unchanged content, got: %s", result)
	}
}

// TestInjectEmbeddedXlsx_NoEmbeddings verifies that a docx zip with no
// embeddings/ entries leaves the markdown unchanged.
func TestInjectEmbeddedXlsx_NoEmbeddings(t *testing.T) {
	// Build a minimal zip file with no embeddings.
	tmp, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	zw := zip.NewWriter(tmp)
	w, _ := zw.Create("word/document.xml")
	w.Write([]byte("<doc/>")) //nolint:errcheck
	zw.Close()
	tmp.Close()

	md := []byte("# Doc\n\n![img](data:image/png;base64,abc)")
	result := injectEmbeddedXlsx("markitdown", tmp.Name(), md)
	// No embeddings → placeholder must survive unchanged.
	if !strings.Contains(string(result), "data:image/png") {
		t.Errorf("expected placeholder to remain, got: %s", result)
	}
}
