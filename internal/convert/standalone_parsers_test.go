//go:build fts5

package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPDFParser_Extensions(t *testing.T) {
	p := &PDFParser{}
	exts := p.Extensions()
	if len(exts) != 1 || exts[0] != ".pdf" {
		t.Errorf("expected [.pdf], got %v", exts)
	}
}

func TestHTMLParser_Extract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.html")
	os.WriteFile(path, []byte(`<html><body><h1>Title</h1><p>Hello <b>World</b></p></body></html>`), 0o644)
	p := &HTMLParser{}
	result, err := p.Extract(path)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	// 验证 h1 转换为 Markdown 标题
	if !strings.Contains(result, "# Title") {
		t.Errorf("expected '# Title' in output, got: %q", result)
	}
	// 验证文本内容
	if !strings.Contains(result, "Hello") {
		t.Errorf("expected 'Hello' in output, got: %q", result)
	}
	t.Logf("HTML extracted: %q", result)
}

func TestCSVParser_Extract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	os.WriteFile(path, []byte("name,age\nAlice,30\nBob,25\n"), 0o644)
	p := &CSVParser{}
	result, err := p.Extract(path)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	// 验证 Markdown 表格格式
	if !strings.Contains(result, "| name | age |") {
		t.Errorf("expected '| name | age |' in output, got: %q", result)
	}
	if !strings.Contains(result, "|---|") {
		t.Errorf("expected '|---|' in output, got: %q", result)
	}
	if !strings.Contains(result, "Alice") {
		t.Errorf("expected 'Alice' in output, got: %q", result)
	}
	t.Logf("CSV extracted: %q", result)
}

func TestEmailParser_Extract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.eml")
	os.WriteFile(path, []byte("From: alice@example.com\nTo: bob@example.com\nSubject: Test\nDate: 2026-07-02\n\nHello Bob!\n"), 0o644)
	p := &EmailParser{}
	result, err := p.Extract(path)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	t.Logf("Email extracted: %q", result)
}

func TestHTMLParser_Extensions(t *testing.T) {
	p := &HTMLParser{}
	exts := p.Extensions()
	if len(exts) != 2 {
		t.Errorf("expected 2 extensions, got %v", exts)
	}
}

func TestCSVParser_Extensions(t *testing.T) {
	p := &CSVParser{}
	exts := p.Extensions()
	if len(exts) != 1 || exts[0] != ".csv" {
		t.Errorf("expected [.csv], got %v", exts)
	}
}

func TestEmailParser_Extensions(t *testing.T) {
	p := &EmailParser{}
	exts := p.Extensions()
	if len(exts) != 2 {
		t.Errorf("expected 2 extensions, got %v", exts)
	}
}
