//go:build fts5

package synthesize

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Generate reads the source-note content for each plan entry, calls the LLM,
// and writes the result to wiki/<type>/<slug>.md under kbRoot.
// Returns the number of pages written.
func Generate(cfg Config, kbRoot string, plans []PagePlan) (int, error) {
	written := 0
	for _, p := range plans {
		if err := AppendOrCreate(cfg, kbRoot, p); err != nil {
			return written, fmt.Errorf("append-or-create %s/%s: %w", p.Type, p.Slug, err)
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

// extractSourceCount reads the source_count field from YAML frontmatter.
// Returns 0 if absent or unparseable.
func extractSourceCount(data []byte) int {
	re := regexp.MustCompile(`(?m)^source_count:\s*(\d+)`)
	m := re.FindSubmatch(data)
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

// incrementSourceCount updates source_count in YAML frontmatter by delta.
// If source_count is absent, inserts it after the opening "---" line.
func incrementSourceCount(data []byte, delta int) []byte {
	re := regexp.MustCompile(`(?m)^source_count:\s*\d+`)
	current := extractSourceCount(data)
	newVal := fmt.Sprintf("source_count: %d", current+delta)
	if re.Match(data) {
		return re.ReplaceAll(data, []byte(newVal))
	}
	// Insert after opening "---\n"
	return []byte(strings.Replace(string(data), "---\n", "---\n"+newVal+"\n", 1))
}

// appendToPage reads an existing wiki page and asks the LLM to append
// new information from p.Sources. Preserves existing structure.
func appendToPage(cfg Config, kbRoot, destPath string, p PagePlan) error {
	existing, err := os.ReadFile(destPath)
	if err != nil {
		return err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "You have an existing wiki %s page. Append new information from the source notes below.\n", p.Type)
	fmt.Fprintf(&sb, "IMPORTANT: preserve the existing page structure. Only ADD new facts, observations, or comparisons at the end of relevant sections. Do not rewrite existing content.\n\n")
	fmt.Fprintf(&sb, "=== EXISTING PAGE ===\n%s\n\n", string(existing))
	fmt.Fprintf(&sb, "=== NEW SOURCE NOTES TO INCORPORATE ===\n\n")

	for _, src := range p.Sources {
		srcPath := filepath.Join(kbRoot, filepath.FromSlash(src))
		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}
		fmt.Fprintf(&sb, "=== %s ===\n%s\n\n", src, string(data))
	}
	fmt.Fprintf(&sb, "Output the complete updated page (frontmatter + all sections). Do NOT wrap in code fences.\n")

	system := buildGeneratePrompt(kbRoot, p.Type)
	updated, err := callLLM(cfg, system, sb.String())
	if err != nil {
		return err
	}

	// Update source_count in the result.
	updated = string(incrementSourceCount([]byte(updated), len(p.Sources)))

	return os.WriteFile(destPath, []byte(updated), 0o644)
}

// rewritePage reorganises an existing wiki page when enough sources have accumulated.
// It passes the existing page content (which already contains distilled summaries
// of all previous sources) plus the new source notes to the LLM, asking it to
// restructure the page for clarity and completeness. No need to re-read original
// raw articles — the existing page is the accumulated knowledge base.
func rewritePage(cfg Config, kbRoot, destPath string, p PagePlan) error {
	existing, err := os.ReadFile(destPath)
	if err != nil {
		return err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "You have an existing wiki %s page that has grown with multiple appended sections and needs reorganisation.\n", p.Type)
	fmt.Fprintf(&sb, "The existing page already contains distilled knowledge from previous source notes.\n\n")
	fmt.Fprintf(&sb, "Your task: REORGANISE the page into a coherent, well-structured document.\n")
	fmt.Fprintf(&sb, "- Merge overlapping or redundant sections\n")
	fmt.Fprintf(&sb, "- Improve flow and logical structure\n")
	fmt.Fprintf(&sb, "- Incorporate the new source notes below\n")
	fmt.Fprintf(&sb, "- Preserve ALL factual content — do not discard knowledge\n\n")
	fmt.Fprintf(&sb, "=== EXISTING PAGE (accumulated knowledge) ===\n%s\n\n", string(existing))

	if len(p.Sources) > 0 {
		fmt.Fprintf(&sb, "=== NEW SOURCE NOTES TO INCORPORATE ===\n\n")
		for _, src := range p.Sources {
			srcPath := filepath.Join(kbRoot, filepath.FromSlash(src))
			data, err := os.ReadFile(srcPath)
			if err != nil {
				continue
			}
			fmt.Fprintf(&sb, "=== %s ===\n%s\n\n", src, string(data))
		}
	}
	fmt.Fprintf(&sb, "Output the complete reorganised page (frontmatter + all sections). Do NOT wrap in code fences.\n")

	system := buildGeneratePrompt(kbRoot, p.Type)
	updated, err := callLLM(cfg, system, sb.String())
	if err != nil {
		return err
	}

	updated = string(incrementSourceCount([]byte(updated), len(p.Sources)))
	return os.WriteFile(destPath, []byte(updated), 0o644)
}

const draftThreshold = 2 // source_count < draftThreshold → _draft/

// resolveDestPath returns the destination file path and whether it goes to _draft/.
// New pages with fewer sources than draftThreshold are written to _draft/.
// Existing pages (already on disk) keep their current location.
func resolveDestPath(kbRoot string, p PagePlan) (dest string, isDraft bool) {
	formalPath := filepath.Join(kbRoot, "wiki", p.Type+"s", p.Slug+".md")
	draftPath := filepath.Join(kbRoot, "wiki", p.Type+"s", "_draft", p.Slug+".md")

	// If page already exists (either location), keep its current location.
	if _, err := os.Stat(formalPath); err == nil {
		return formalPath, false
	}
	if _, err := os.Stat(draftPath); err == nil {
		return draftPath, true
	}

	// New page: check source count.
	if len(p.Sources) < draftThreshold {
		return draftPath, true
	}
	return formalPath, false
}

// graduateFromDraft moves a page from _draft/ to the formal directory
// when its source_count reaches draftThreshold.
func graduateFromDraft(kbRoot string, p PagePlan, draftPath string) error {
	data, err := os.ReadFile(draftPath)
	if err != nil {
		return nil
	}
	if extractSourceCount(data) < draftThreshold {
		return nil // not ready yet
	}
	formalPath := filepath.Join(kbRoot, "wiki", p.Type+"s", p.Slug+".md")
	if err := os.MkdirAll(filepath.Dir(formalPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(draftPath, formalPath); err != nil {
		return err
	}
	return nil
}

// AppendOrCreate either appends new source-note content to an existing wiki page
// (matched by slug) or creates a new one. Silently skips if cfg is not configured.
func AppendOrCreate(cfg Config, kbRoot string, p PagePlan) error {
	if !cfg.IsConfigured() {
		return nil
	}
	dest, isDraft := resolveDestPath(kbRoot, p)
	_ = isDraft // used only for path resolution

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(dest); err == nil {
		existing, readErr := os.ReadFile(dest)
		if readErr == nil && extractSourceCount(existing)+len(p.Sources) >= 5 {
			return rewritePage(cfg, kbRoot, dest, p)
		}
		err := appendToPage(cfg, kbRoot, dest, p)
		if err != nil {
			return err
		}
		// After append, check if draft page should graduate to formal.
		if _, stillDraft := resolveDestPath(kbRoot, p); stillDraft {
			_ = graduateFromDraft(kbRoot, p, dest)
		}
		return nil
	}

	// New page.
	return generateAndWrite(cfg, kbRoot, dest, p)
}

// generateAndWrite generates a page from scratch and writes it to dest.
func generateAndWrite(cfg Config, kbRoot, dest string, p PagePlan) error {
	content, err := generatePage(cfg, kbRoot, p)
	if err != nil {
		return err
	}
	// Set initial source_count.
	content = string(incrementSourceCount([]byte(content), len(p.Sources)))
	return os.WriteFile(dest, []byte(content), 0o644)
}

// RunIncremental triggers synthesize for a single newly-distilled source-note.
// It loads the note, finds related source-notes (same tags), and calls
// AppendOrCreate for each proposed page. Silently skips if cfg not configured.
func RunIncremental(cfg Config, kbRoot, notePath string) error {
	if !cfg.IsConfigured() {
		return nil
	}
	allNotes, err := LoadSourceNotes(kbRoot)
	if err != nil {
		return err
	}
	return runIncrementalWithNotes(cfg, kbRoot, notePath, allNotes)
}

// RunIncrementalWithNotes is like RunIncremental but accepts a pre-loaded
// allNotes slice, avoiding repeated disk scans when processing many notes.
func RunIncrementalWithNotes(cfg Config, kbRoot, notePath string, allNotes []SourceNote) error {
	if !cfg.IsConfigured() {
		return nil
	}
	return runIncrementalWithNotes(cfg, kbRoot, notePath, allNotes)
}

func runIncrementalWithNotes(cfg Config, kbRoot, notePath string, allNotes []SourceNote) error {
	if len(allNotes) == 0 {
		return nil
	}

	// Find the new note in the loaded list.
	var newNote *SourceNote
	for i, n := range allNotes {
		if filepath.ToSlash(n.Path) == filepath.ToSlash(notePath) {
			newNote = &allNotes[i]
			break
		}
	}
	if newNote == nil {
		// Note not indexed yet, skip.
		return nil
	}

	// Find related notes sharing at least one tag with the new note.
	tagSet := make(map[string]bool)
	for _, t := range newNote.Tags {
		tagSet[strings.ToLower(t)] = true
	}
	var related []SourceNote
	related = append(related, *newNote)
	for _, n := range allNotes {
		if n.Path == newNote.Path {
			continue
		}
		for _, t := range n.Tags {
			if tagSet[strings.ToLower(t)] {
				related = append(related, n)
				break
			}
		}
		if len(related) >= 10 { // cap to avoid huge LLM prompts
			break
		}
	}

	existing := collectExistingTitles(kbRoot)
	plans, err := Plan(cfg, related, existing, "")
	if err != nil {
		return err
	}
	plans = filterViablePlans(plans)

	for _, p := range plans {
		if err := AppendOrCreate(cfg, kbRoot, p); err != nil {
			// Log but don't abort — incremental synthesize is best-effort.
			fmt.Printf("synthesize incremental: %s/%s: %v\n", p.Type, p.Slug, err)
		}
	}
	return nil
}
