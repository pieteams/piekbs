//go:build fts5

package kb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdownFile_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	content := `---
title: Test Note
type: source-note
sources:
  - raw/article.md
tags: go, testing
description: A test note
---

This is the body content with a [[wikilink]].
`
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte(content), 0644)

	parsed, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Title != "Test Note" {
		t.Errorf("title = %q, want 'Test Note'", parsed.Title)
	}
	if parsed.Kind != "source-note" {
		t.Errorf("kind = %q, want 'source-note'", parsed.Kind)
	}
	if len(parsed.Sources) != 1 || parsed.Sources[0] != "raw/article.md" {
		t.Errorf("sources = %v", parsed.Sources)
	}
	if len(parsed.Wikilinks) != 1 || parsed.Wikilinks[0] != "wikilink" {
		t.Errorf("wikilinks = %v", parsed.Wikilinks)
	}
	if parsed.Description != "A test note" {
		t.Errorf("description = %q", parsed.Description)
	}
	if parsed.Content == "" {
		t.Error("content should not be empty")
	}
}

func TestParseMarkdownFile_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.md")
	os.WriteFile(path, []byte("# Hello\n\nPlain text."), 0644)

	parsed, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Title != "" {
		t.Errorf("title should be empty, got %q", parsed.Title)
	}
	if parsed.Content != "# Hello\n\nPlain text." {
		t.Errorf("content mismatch: %q", parsed.Content)
	}
}

func TestSetFrontmatterSources(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		sources []string
		want    string
	}{
		{
			name:    "empty sources returns unchanged",
			input:   "---\ntitle: foo\n---\nbody",
			sources: []string{},
			want:    "---\ntitle: foo\n---\nbody",
		},
		{
			name:    "no frontmatter returns unchanged",
			input:   "no frontmatter here",
			sources: []string{"raw/a.md"},
			want:    "no frontmatter here",
		},
		{
			name:    "inline sources replaced with block",
			input:   "---\ntitle: foo\nsources: [old.md]\n---\nbody",
			sources: []string{"new.md"},
			want:    "---\ntitle: foo\nsources:\n  - new.md\n---\nbody",
		},
		{
			name:    "block sources replaced",
			input:   "---\ntitle: foo\nsources:\n  - old1.md\n  - old2.md\n---\nbody",
			sources: []string{"new.md"},
			want:    "---\ntitle: foo\nsources:\n  - new.md\n---\nbody",
		},
		{
			name:    "no existing sources field adds it",
			input:   "---\ntitle: foo\n---\nbody",
			sources: []string{"raw/a.md"},
			want:    "---\ntitle: foo\nsources:\n  - raw/a.md\n---\nbody",
		},
		{
			name:    "multiple sources",
			input:   "---\ntitle: foo\n---\nbody",
			sources: []string{"raw/a.md", "raw/b.md"},
			want:    "---\ntitle: foo\nsources:\n  - raw/a.md\n  - raw/b.md\n---\nbody",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SetFrontmatterSources(tt.input, tt.sources)
			if got != tt.want {
				t.Errorf("SetFrontmatterSources(%q, %v)\ngot:\n%s\nwant:\n%s", tt.input, tt.sources, got, tt.want)
			}
		})
	}
}

func TestParseYAMLSimple(t *testing.T) {
	text := `title: My Title
type: concept
sources:
  - raw/a.md
  - raw/b.md
tags: tag1, tag2
`
	fm := parseYAMLSimple(text)
	if fm["title"] != "My Title" {
		t.Errorf("title = %v", fm["title"])
	}
	sources, ok := fm["sources"].([]string)
	if !ok || len(sources) != 2 {
		t.Errorf("sources = %v (type %T)", fm["sources"], fm["sources"])
	}
}
