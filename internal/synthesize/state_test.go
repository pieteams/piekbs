//go:build fts5

package synthesize

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSynthState_NewOrChanged_Empty(t *testing.T) {
	state := &SynthState{Processed: map[string]string{}}
	notes := []SourceNote{
		{Path: "wiki/source-notes/a.md", Title: "A"},
		{Path: "wiki/source-notes/b.md", Title: "B"},
	}
	changed := state.NewOrChanged(notes)
	if len(changed) != 2 {
		t.Errorf("empty state: expected 2 changed, got %d", len(changed))
	}
}

func TestSynthState_NewOrChanged_AllSame(t *testing.T) {
	state := &SynthState{Processed: map[string]string{
		"wiki/source-notes/a.md": hashNote(SourceNote{Path: "wiki/source-notes/a.md", Title: "A"}),
	}}
	notes := []SourceNote{{Path: "wiki/source-notes/a.md", Title: "A"}}
	changed := state.NewOrChanged(notes)
	if len(changed) != 0 {
		t.Errorf("same hash: expected 0 changed, got %d", len(changed))
	}
}

func TestSynthState_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "index"), 0o755); err != nil {
		t.Fatal(err)
	}
	state := &SynthState{Processed: map[string]string{"a.md": "abc123"}}
	if err := state.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := LoadSynthState(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Processed["a.md"] != "abc123" {
		t.Errorf("loaded hash mismatch: got %q", loaded.Processed["a.md"])
	}
}

func TestFilterViablePlans(t *testing.T) {
	plans := []PagePlan{
		{Type: "concept", Sources: []string{"a", "b"}},      // < 3 → filtered
		{Type: "concept", Sources: []string{"a", "b", "c"}}, // = 3 → keep
		{Type: "comparison", Sources: []string{"a", "b"}},   // ≥ 2 → keep
		{Type: "decision", Sources: []string{"a"}},           // < 2 → filtered
		{Type: "decision", Sources: []string{"a", "b"}},      // ≥ 2 → keep
	}
	result := filterViablePlans(plans)
	if len(result) != 3 {
		t.Errorf("expected 3 viable plans, got %d", len(result))
	}
}
