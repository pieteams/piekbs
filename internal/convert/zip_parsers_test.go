//go:build fts5

package convert

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func createTestDocx(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "test.docx")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("word/document.xml")
	w.Write([]byte(`<?xml version="1.0"?><w:document><w:body><w:p><w:r><w:t>Hello World</w:t></w:r></w:p></w:body></w:document>`))
	zw.Close()
	f.Close()
	return path
}

func createTestXlsx(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "test.xlsx")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w1, _ := zw.Create("xl/sharedStrings.xml")
	w1.Write([]byte(`<?xml version="1.0"?><sst><si><t>Foo</t></si><si><t>Bar</t></si></sst>`))
	w2, _ := zw.Create("xl/worksheets/sheet1.xml")
	w2.Write([]byte(`<?xml version="1.0"?><worksheet><sheetData><row><c><v>1</v></c></row></sheetData></worksheet>`))
	zw.Close()
	f.Close()
	return path
}

func createTestPptx(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "test.pptx")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("ppt/slides/slide1.xml")
	w.Write([]byte(`<?xml version="1.0"?><p:sld><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>Slide One</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`))
	zw.Close()
	f.Close()
	return path
}

func createTestEpub(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "test.epub")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, _ := zw.Create("chapter1.xhtml")
	w.Write([]byte(`<?xml version="1.0"?><html><body><p><t>Chapter Content</t></p></body></html>`))
	zw.Close()
	f.Close()
	return path
}

func TestDocxParser_Extract(t *testing.T) {
	dir := t.TempDir()
	path := createTestDocx(t, dir)
	p := &DocxParser{}
	result, err := p.Extract(path)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	t.Logf("DOCX extracted: %q", result)
}

func TestXlsxParser_Extract(t *testing.T) {
	dir := t.TempDir()
	path := createTestXlsx(t, dir)
	p := &XlsxParser{}
	result, err := p.Extract(path)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	t.Logf("XLSX extracted: %q", result)
}

func TestPptxParser_Extract(t *testing.T) {
	dir := t.TempDir()
	path := createTestPptx(t, dir)
	p := &PptxParser{}
	result, err := p.Extract(path)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	t.Logf("PPTX extracted: %q", result)
}

func TestEpubParser_Extract(t *testing.T) {
	dir := t.TempDir()
	path := createTestEpub(t, dir)
	p := &EpubParser{}
	result, err := p.Extract(path)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	t.Logf("EPUB extracted: %q", result)
}

func TestDocxParser_Extensions(t *testing.T) {
	p := &DocxParser{}
	exts := p.Extensions()
	if len(exts) != 1 || exts[0] != ".docx" {
		t.Errorf("expected [.docx], got %v", exts)
	}
}
