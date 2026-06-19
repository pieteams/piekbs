//go:build fts5

package distill

import (
	"fmt"
	"strings"

	"github.com/jasen215/wikiloop/internal/kb"
)

type relatedNote struct {
	Path        string
	Title       string
	Description string
}

// findRelatedNotes queries the vector index for source-notes related to content.
// Returns a formatted context string ready for prompt injection.
// Returns "" if embedder is nil, no vec index exists, or on any error.
func findRelatedNotes(kbRoot, content string, embedder kb.Embedder) string {
	if embedder == nil {
		return ""
	}

	vec, err := embedder.Encode(content)
	if err != nil {
		return ""
	}

	layer := "wiki"
	results, err := kb.VecSearch(kbRoot, vec, &layer, 5)
	if err != nil || len(results) == 0 {
		return ""
	}

	var notes []relatedNote
	for _, r := range results {
		// Only include synthesized pages, not source-notes.
		if strings.Contains(r.Path, "/source-notes/") {
			continue
		}
		notes = append(notes, relatedNote{
			Path:        r.Path,
			Title:       r.Title,
			Description: r.Description,
		})
	}
	return buildRelatedContext(notes)
}

// buildRelatedContext formats related notes into a prompt injection string.
func buildRelatedContext(notes []relatedNote) string {
	if len(notes) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n[KB 中已有相关页面]\n")
	for _, n := range notes {
		fmt.Fprintf(&sb, "- %s: %q — %s\n", n.Path, n.Title, n.Description)
	}
	sb.WriteString("\n请在 related_to、contradicts、supports 字段中引用上述路径（如适用）。\n如果内容与某页面矛盾，填入 contradicts；如果支持某论断，填入 supports。\n")
	return sb.String()
}
