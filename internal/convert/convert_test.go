//go:build fts5

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
	registry := NewRegistry()

	for _, name := range []string{"doc.pdf", "page.html"} {
		if err := os.WriteFile(filepath.Join(rawDir, name), []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(rawDir, "note.md"), []byte("# note"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := FindConvertibleFiles(dir, registry)
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
	registry := NewRegistry()

	if err := os.WriteFile(filepath.Join(rawDir, "doc.pdf"), []byte("pdf content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(convertedDir, "doc.md"), []byte("# doc"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := FindConvertibleFiles(dir, registry)
	if len(files) != 0 {
		t.Errorf("expected 0 convertible files, got %d: %v", len(files), files)
	}
}

// TestInjectEmbeddedXlsx_NoPlaceholder verifies that content without image
// placeholders is returned unchanged — embedded Excel extraction must not
// corrupt documents that have no OLE objects.
func TestInjectEmbeddedXlsx_NoPlaceholder(t *testing.T) {
	registry := NewRegistry()
	md := "# Title\n\nSome text without any image placeholder."
	result := injectEmbeddedXlsx(registry, "dummy.docx", md)
	if result != md {
		t.Errorf("expected unchanged content, got: %s", result)
	}
}

// TestInjectEmbeddedXlsx_NoEmbeddings verifies that a docx zip with no
// embeddings/ entries leaves the markdown unchanged.
func TestInjectEmbeddedXlsx_NoEmbeddings(t *testing.T) {
	tmp, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	zw := zip.NewWriter(tmp)
	w, _ := zw.Create("word/document.xml")
	w.Write([]byte("<doc/>"))
	zw.Close()
	tmp.Close()

	registry := NewRegistry()
	md := "# Doc\n\n![img](data:image/png;base64,abc)"
	result := injectEmbeddedXlsx(registry, tmp.Name(), md)
	if !strings.Contains(result, "data:image/png") {
		t.Errorf("expected placeholder to remain, got: %s", result)
	}
}

func TestRun_Integration(t *testing.T) {
	dir := t.TempDir()
	rawDir := filepath.Join(dir, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a CSV file
	if err := os.WriteFile(filepath.Join(rawDir, "data.csv"), []byte("name,age\nAlice,30\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	n, err := Run(dir)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 file converted, got %d", n)
	}

	// Verify output exists
	convertedPath := filepath.Join(rawDir, "converted", "data.md")
	data, err := os.ReadFile(convertedPath)
	if err != nil {
		t.Fatalf("converted file not found: %v", err)
	}
	if !strings.Contains(string(data), "Alice") {
		t.Errorf("expected 'Alice' in converted output, got: %s", data)
	}
}
