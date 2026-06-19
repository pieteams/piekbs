//go:build fts5

package synthesize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Generate reads the source-note content for each plan entry, calls the LLM,
// and writes the result to wiki/<type>/<slug>.md under kbRoot.
// Returns the number of pages written.
func Generate(cfg Config, kbRoot string, plans []PagePlan) (int, error) {
	written := 0
	for _, p := range plans {
		dest := filepath.Join(kbRoot, "wiki", p.Type+"s", p.Slug+".md")

		// Skip if already exists.
		if _, err := os.Stat(dest); err == nil {
			continue
		}

		content, err := generatePage(cfg, kbRoot, p)
		if err != nil {
			return written, fmt.Errorf("generate %s/%s: %w", p.Type, p.Slug, err)
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// generatePage builds the user prompt from source-note content and calls the LLM.
func generatePage(cfg Config, kbRoot string, p PagePlan) (string, error) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Generate a %s page with:\n  title: %s\n  description: %s\n\n",
		p.Type, p.Title, p.Description)
	fmt.Fprintf(&sb, "Use these source notes as your knowledge base:\n\n")

	for _, src := range p.Sources {
		srcPath := filepath.Join(kbRoot, filepath.FromSlash(src))
		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}
		fmt.Fprintf(&sb, "=== %s ===\n%s\n\n", src, string(data))
	}

	// Inject sources list into prompt so LLM sets the frontmatter correctly.
	sourcesJSON := `["` + strings.Join(p.Sources, `", "`) + `"]`
	fmt.Fprintf(&sb, "Set the frontmatter sources field to: %s\n", sourcesJSON)
	fmt.Fprintf(&sb, "Set the frontmatter timestamp to: %s\n", time.Now().UTC().Format("2006-01-02"))

	system := buildGeneratePrompt(kbRoot, p.Type)
	return callLLM(cfg, system, sb.String())
}

// Run is the top-level entry point: load source-notes, plan, generate.
// topic filters notes by tags/title match when non-empty.
// full bypasses SynthState and processes all notes.
func Run(kbRoot string, cfg Config, topic string, full bool) (int, error) {
	if !cfg.IsConfigured() {
		return 0, fmt.Errorf("LLM config incomplete: BaseURL, Token, and Model are all required")
	}

	allNotes, err := LoadSourceNotes(kbRoot)
	if err != nil {
		return 0, fmt.Errorf("load source notes: %w", err)
	}
	if len(allNotes) == 0 {
		return 0, nil
	}

	// Filter by topic (tags/title match).
	notes := filterByTopic(allNotes, topic)
	if len(notes) == 0 {
		return 0, nil
	}

	// Incremental: skip unchanged notes unless --full.
	var toProcess []SourceNote
	var state *SynthState
	if !full {
		state, err = LoadSynthState(kbRoot)
		if err != nil {
			return 0, fmt.Errorf("load synth state: %w", err)
		}
		toProcess = state.NewOrChanged(notes)
		if len(toProcess) == 0 {
			return 0, nil // nothing changed
		}
	} else {
		toProcess = notes
		state = &SynthState{Processed: map[string]string{}}
	}

	existing := collectExistingTitles(kbRoot)

	plans, err := Plan(cfg, notes, existing, "")
	if err != nil {
		return 0, fmt.Errorf("plan: %w", err)
	}

	plans = filterViablePlans(plans)
	if len(plans) == 0 {
		// Record hashes even if no pages generated (avoids re-processing next run).
		state.Record(toProcess)
		_ = state.Save(kbRoot)
		return 0, nil
	}

	written, err := Generate(cfg, kbRoot, plans)
	if err != nil {
		return written, err
	}

	state.Record(toProcess)
	if err := state.Save(kbRoot); err != nil {
		return written, fmt.Errorf("save synth state: %w", err)
	}
	return written, nil
}

// filterByTopic returns notes whose title or tags contain topic (case-insensitive).
// Returns all notes if topic is empty.
func filterByTopic(notes []SourceNote, topic string) []SourceNote {
	if topic == "" {
		return notes
	}
	topic = strings.ToLower(topic)
	var out []SourceNote
	for _, n := range notes {
		if strings.Contains(strings.ToLower(n.Title), topic) {
			out = append(out, n)
			continue
		}
		for _, tag := range n.Tags {
			if strings.Contains(strings.ToLower(tag), topic) {
				out = append(out, n)
				break
			}
		}
	}
	return out
}

// collectExistingTitles scans wiki/concepts/, wiki/comparisons/, wiki/decisions/
// and returns the title frontmatter values found.
func collectExistingTitles(kbRoot string) []string {
	var titles []string
	for _, dir := range []string{"concepts", "comparisons", "decisions"} {
		dirPath := filepath.Join(kbRoot, "wiki", dir)
		entries, _ := os.ReadDir(dirPath)
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dirPath, e.Name()))
			if err != nil {
				continue
			}
			note := parseFrontmatter(data)
			if note.Title != "" {
				titles = append(titles, note.Title)
			}
		}
	}
	return titles
}
