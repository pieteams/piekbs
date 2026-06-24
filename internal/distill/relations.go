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

// findRelatedNotes previously queried the vector index for related content.
// Vector search has been removed; this always returns "" until Task 2 adds FTS-based lookup.
func findRelatedNotes(kbRoot, content string, embedder kb.Embedder) string {
	return ""
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
