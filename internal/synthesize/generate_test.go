//go:build fts5

package synthesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractSourceCount(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
	}{
		{"present", []byte("---\nsource_count: 3\ntitle: foo\n---\n"), 3},
		{"absent", []byte("---\ntitle: foo\n---\n"), 0},
		{"zero", []byte("---\nsource_count: 0\n---\n"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSourceCount(tt.input)
			if got != tt.want {
				t.Errorf("extractSourceCount(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestIncrementSourceCount(t *testing.T) {
	input := []byte("---\nsource_count: 2\ntitle: foo\n---\n\ncontent")
	got := incrementSourceCount(input, 1)
	if !strings.Contains(string(got), "source_count: 3") {
		t.Errorf("expected source_count: 3, got: %s", got)
	}
}

func TestIncrementSourceCountAbsent(t *testing.T) {
	input := []byte("---\ntitle: foo\n---\n\ncontent")
	got := incrementSourceCount(input, 1)
	if !strings.Contains(string(got), "source_count: 1") {
		t.Errorf("expected source_count: 1 to be inserted, got: %s", got)
	}
}

func TestAppendOrCreateWritesToDraftWhenSingleSource(t *testing.T) {
	dir := t.TempDir()
	// Create required wiki subdirs
	os.MkdirAll(filepath.Join(dir, "wiki", "concepts"), 0o755)
	os.MkdirAll(filepath.Join(dir, "wiki", "concepts", "_draft"), 0o755)

	cfg := Config{BaseURL: "http://localhost", Token: "test", Model: "test"}
	p := PagePlan{
		Type:        "concept",
		Slug:        "test-concept",
		Title:       "Test Concept",
		Description: "A test",
		Sources:     []string{"wiki/source-notes/a.md"},
	}

	// We can't call LLM in tests, so just verify the destination path logic
	dest, isDraft := resolveDestPath(dir, p)
	if !isDraft {
		t.Error("expected isDraft=true for single source")
	}
	expectedDraft := filepath.Join(dir, "wiki", "concepts", "_draft", "test-concept.md")
	if dest != expectedDraft {
		t.Errorf("expected %q, got %q", expectedDraft, dest)
	}
	_ = cfg
}

func TestResolveDestPathMultipleSources(t *testing.T) {
	dir := t.TempDir()
	p := PagePlan{
		Type:    "concept",
		Slug:    "test-concept",
		Sources: []string{"a.md", "b.md"},
	}
	dest, isDraft := resolveDestPath(dir, p)
	if isDraft {
		t.Error("expected isDraft=false for 2 sources")
	}
	expected := filepath.Join(dir, "wiki", "concepts", "test-concept.md")
	if dest != expected {
		t.Errorf("expected %q, got %q", expected, dest)
	}
}

func TestGraduateFromDraft(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "wiki", "concepts", "_draft"), 0o755)
	os.MkdirAll(filepath.Join(dir, "wiki", "concepts"), 0o755)

	draftPath := filepath.Join(dir, "wiki", "concepts", "_draft", "test.md")
	// Write a page with source_count=2 (at threshold)
	content := "---\ntype: concept\nsource_count: 2\n---\n\nContent"
	os.WriteFile(draftPath, []byte(content), 0o644)

	p := PagePlan{Type: "concept", Slug: "test", Sources: []string{"a.md"}}
	err := graduateFromDraft(dir, p, draftPath)
	if err != nil {
		t.Fatal(err)
	}

	formalPath := filepath.Join(dir, "wiki", "concepts", "test.md")
	if _, err := os.Stat(formalPath); os.IsNotExist(err) {
		t.Error("expected page to be graduated to formal directory")
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Error("expected draft page to be removed after graduation")
	}
}

func TestAppendOrCreate_CreatesNewPage(t *testing.T) {
	dir := t.TempDir()
	// Create minimal KB structure
	os.MkdirAll(filepath.Join(dir, "wiki", "concepts"), 0o755)
	os.MkdirAll(filepath.Join(dir, "schema", "templates"), 0o755)

	p := PagePlan{
		Type:        "concept",
		Title:       "Test Concept",
		Slug:        "test-concept",
		Description: "A test",
		Sources:     []string{},
	}
	// cfg not configured → should skip silently (no error)
	cfg := Config{}
	err := AppendOrCreate(cfg, dir, p)
	if err != nil {
		t.Errorf("unconfigured LLM should skip silently, got error: %v", err)
	}
}
