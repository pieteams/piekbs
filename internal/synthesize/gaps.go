//go:build fts5

package synthesize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

const gapsSystemPrompt = `You are a knowledge-base curator analyzing a knowledge base for gaps.

Given a list of source-note summaries and an optional research topic, identify:
1. What subjects are well-covered by existing notes
2. What important subjects are missing or underrepresented
3. Specific resources or topics the user should add to improve coverage

Format your response as Markdown with these sections:
## Covered
(bullet list of well-covered areas)

## Gaps
(bullet list of missing or underrepresented areas)

## Suggestions
(bullet list of specific resources, papers, or topics to add)

Be specific and actionable. Reference actual note titles where relevant.`

// RunGaps analyzes knowledge coverage for a topic and writes a gap report.
// Output: index/gaps/<slug>.md
func RunGaps(kbRoot string, cfg Config, topic string) error {
	if !cfg.IsConfigured() {
		return fmt.Errorf("LLM config incomplete: BaseURL, Token, and Model are all required")
	}

	allNotes, err := LoadSourceNotes(kbRoot)
	if err != nil {
		return fmt.Errorf("load source notes: %w", err)
	}

	notes := filterByTopic(allNotes, topic)
	if len(notes) == 0 {
		return fmt.Errorf("no source-notes found for topic %q", topic)
	}

	prompt := buildGapsPrompt(notes, topic)
	report, err := callLLM(cfg, gapsSystemPrompt, prompt)
	if err != nil {
		return fmt.Errorf("call LLM: %w", err)
	}

	slug := TopicSlug(topic)
	if slug == "" {
		slug = fmt.Sprintf("gaps-%d", time.Now().Unix())
	}
	outPath := filepath.Join(kbRoot, "index", "gaps", slug+".md")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create gaps dir: %w", err)
	}

	header := fmt.Sprintf("# Knowledge Gap Analysis: %s\n\n> Generated: %s\n> Notes analyzed: %d\n\n",
		topic, time.Now().Format("2006-01-02"), len(notes))
	return os.WriteFile(outPath, []byte(header+report), 0o644)
}

// buildGapsPrompt constructs the user message for gap analysis.
func buildGapsPrompt(notes []SourceNote, topic string) string {
	var sb strings.Builder
	if topic != "" {
		fmt.Fprintf(&sb, "Research topic: %s\n\n", topic)
	}
	sb.WriteString("Source notes in this knowledge base:\n\n")
	for _, n := range notes {
		fmt.Fprintf(&sb, "- %s\n  title: %s\n  description: %s\n  tags: [%s]\n",
			n.Path, n.Title, n.Description, strings.Join(n.Tags, ", "))
		if len(n.KeyClaims) > 0 {
			fmt.Fprintf(&sb, "  key_claims: [%s]\n", strings.Join(n.KeyClaims, "; "))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// TopicSlug converts a topic string to a kebab-case filename slug.
// Preserves CJK and other Unicode letters/digits; separators become hyphens.
func TopicSlug(topic string) string {
	var sb strings.Builder
	prevWasSep := true
	for _, r := range strings.ToLower(topic) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			prevWasSep = false
		} else if !prevWasSep {
			sb.WriteRune('-')
			prevWasSep = true
		}
	}
	s := strings.TrimRight(sb.String(), "-")
	if len([]rune(s)) > 80 {
		s = string([]rune(s)[:80])
	}
	return s
}
