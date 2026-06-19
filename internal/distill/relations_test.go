//go:build fts5

package distill

import (
	"strings"
	"testing"
)

// mockEmbedder returns a fixed vector for any text.
type mockEmbedder struct{}

func (m mockEmbedder) Encode(text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}
func (m mockEmbedder) Dimension() int { return 3 }

func TestFindRelatedNotes_NoVecIndex(t *testing.T) {
	// When there is no vec index, findRelatedNotes should return empty string.
	result := findRelatedNotes("/nonexistent-kb", "some content", mockEmbedder{})
	if result != "" {
		t.Errorf("expected empty string for missing KB, got %q", result)
	}
}

func TestBuildRelatedContext(t *testing.T) {
	notes := []relatedNote{
		{Path: "wiki/source-notes/foo.md", Title: "Foo", Description: "About foo"},
		{Path: "wiki/source-notes/bar.md", Title: "Bar", Description: "About bar"},
	}
	result := buildRelatedContext(notes)
	if !strings.Contains(result, "wiki/source-notes/foo.md") {
		t.Errorf("context missing foo path: %s", result)
	}
	if !strings.Contains(result, "Foo") {
		t.Errorf("context missing Foo title: %s", result)
	}
	if !strings.Contains(result, "related_to") {
		t.Errorf("context missing related_to instruction: %s", result)
	}
}
