//go:build fts5

package convert

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
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
	w1.Write([]byte(`<?xml version="1.0"?><sst><si><t>Name</t></si><si><t>Value</t></si><si><t>Foo</t></si></sst>`))
	// Header row uses shared strings (t="s"), data row has shared string + numeric cell
	w2, _ := zw.Create("xl/worksheets/sheet1.xml")
	w2.Write([]byte(`<?xml version="1.0"?><worksheet><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row><row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2"><v>42</v></c></row></sheetData></worksheet>`))
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
	// presentation.xml with slide order (r namespace declared)
	pres, _ := zw.Create("ppt/presentation.xml")
	pres.Write([]byte(`<?xml version="1.0"?><p:presentation xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><p:sldIdLst><p:sldId r:id="rId1"></p:sldId></p:sldIdLst></p:presentation>`))
	// presentation.xml.rels
	presRels, _ := zw.Create("ppt/_rels/presentation.xml.rels")
	presRels.Write([]byte(`<?xml version="1.0"?><Relationships><Relationship Id="rId1" Target="slides/slide1.xml" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide"/></Relationships>`))
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

	// container.xml points to OPF file
	cw, _ := zw.Create("META-INF/container.xml")
	cw.Write([]byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`))

	// content.opf defines manifest and spine (note: spine order is ch2→ch1, intentionally reversed to test sorting)
	ow, _ := zw.Create("OEBPS/content.opf")
	ow.Write([]byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch2"/>
    <itemref idref="ch1"/>
  </spine>
</package>`))

	// 两个章节内容
	w1, _ := zw.Create("OEBPS/chapter1.xhtml")
	w1.Write([]byte(`<?xml version="1.0"?><html><body><p><t>First Chapter</t></p></body></html>`))
	w2, _ := zw.Create("OEBPS/chapter2.xhtml")
	w2.Write([]byte(`<?xml version="1.0"?><html><body><p><t>Second Chapter</t></p></body></html>`))

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
	// 验证共享字符串（<t>）被提取
	if !strings.Contains(result, "Foo") {
		t.Errorf("expected 'Foo' in result, got: %q", result)
	}
	// Verify numeric cell values (<v>) are also extracted
	if !strings.Contains(result, "42") {
		t.Errorf("expected '42' (numeric cell) in result, got: %q", result)
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
	// Verify spine order: ch2 comes before ch1
	idx2 := strings.Index(result, "Second Chapter")
	idx1 := strings.Index(result, "First Chapter")
	if idx2 == -1 || idx1 == -1 {
		t.Fatalf("expected both chapters in output, got: %q", result)
	}
	if idx2 > idx1 {
		t.Errorf("spine order violated: 'Second Chapter' (idx=%d) should come before 'First Chapter' (idx=%d)\nresult: %q", idx2, idx1, result)
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
