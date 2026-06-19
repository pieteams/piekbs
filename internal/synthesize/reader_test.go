//go:build fts5

package synthesize

import (
	"testing"
)

func TestParseFrontmatter_KeyClaims(t *testing.T) {
	input := []byte(`---
type: source-note
title: "Test Note"
description: "A test"
tags: [go, test]
key_claims:
  - "claim one"
  - "claim two"
---

## Summary
content here
`)
	note := parseFrontmatter(input)
	if note.Title != "Test Note" {
		t.Errorf("title: got %q, want %q", note.Title, "Test Note")
	}
	if len(note.KeyClaims) != 2 {
		t.Fatalf("key_claims len: got %d, want 2", len(note.KeyClaims))
	}
	if note.KeyClaims[0] != "claim one" {
		t.Errorf("key_claims[0]: got %q, want %q", note.KeyClaims[0], "claim one")
	}
	if note.KeyClaims[1] != "claim two" {
		t.Errorf("key_claims[1]: got %q, want %q", note.KeyClaims[1], "claim two")
	}
}

func TestParseFrontmatter_KeyClaims_Empty(t *testing.T) {
	input := []byte(`---
type: source-note
title: "No Claims"
---
`)
	note := parseFrontmatter(input)
	if note.KeyClaims != nil {
		t.Errorf("expected nil KeyClaims, got %v", note.KeyClaims)
	}
}
