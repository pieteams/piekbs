//go:build fts5

package synthesize

import (
	"os"
	"path/filepath"
	"testing"
)

func makeTempKBWithNotes(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	notesDir := filepath.Join(dir, "wiki", "source-notes")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
type: source-note
title: "Vector Databases Overview"
description: "Survey of vector database options."
tags: [vector-db, databases]
key_claims:
  - "chromem-go is zero-dependency"
---
## Summary
Vector databases store embeddings.
`
	if err := os.WriteFile(filepath.Join(notesDir, "vector-db.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "index", "gaps"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestBuildGapsPrompt(t *testing.T) {
	notes := []SourceNote{
		{Path: "wiki/source-notes/a.md", Title: "A", Description: "About A", Tags: []string{"ai"}},
	}
	prompt := buildGapsPrompt(notes, "AI tooling")
	if len(prompt) == 0 {
		t.Error("expected non-empty prompt")
	}
	if prompt == buildGapsPrompt(notes, "") {
		t.Error("topic should affect prompt")
	}
}
