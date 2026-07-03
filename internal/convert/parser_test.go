//go:build fts5

package convert

import (
	"strings"
	"testing"
)

func TestNewRegistry_HasAllParsers(t *testing.T) {
	r := NewRegistry()
	expectedExts := []string{".docx", ".xlsx", ".pptx", ".pdf", ".epub", ".html", ".htm", ".csv", ".eml", ".msg"}
	for _, ext := range expectedExts {
		if !r.Supports(ext) {
			t.Errorf("registry should support %s", ext)
		}
	}
}

func TestRegistry_Supports_Unregistered(t *testing.T) {
	r := NewRegistry()
	if r.Supports(".xyz") {
		t.Error("registry should not support .xyz")
	}
}

func TestRegistry_Get_ReturnsParser(t *testing.T) {
	r := NewRegistry()
	p, ok := r.Get(".docx")
	if !ok {
		t.Fatal("expected parser for .docx")
	}
	if len(p.Extensions()) == 0 {
		t.Error("parser should return at least one extension")
	}
}

func TestExtractXMLText_Basic(t *testing.T) {
	xml := `<root><p><t>Hello</t></p><p><t>World</t></p></root>`
	result := extractXMLText(strings.NewReader(xml))
	if !strings.Contains(result, "Hello") {
		t.Errorf("expected 'Hello' in result, got: %s", result)
	}
	if !strings.Contains(result, "World") {
		t.Errorf("expected 'World' in result, got: %s", result)
	}
}

func TestExtractXMLText_Empty(t *testing.T) {
	xml := `<root></root>`
	result := extractXMLText(strings.NewReader(xml))
	if result != "" {
		t.Errorf("expected empty string, got: %q", result)
	}
}
