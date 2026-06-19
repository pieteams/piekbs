//go:build fts5

// Package distill handles calling an external LLM API to generate wiki
// source-notes from raw documents.
package distill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jasen215/wikiloop/internal/kb"
)

// Config holds the LLM API configuration.
type Config struct {
	BaseURL string
	Token   string
	Model   string
	APIType string // "openai" (default) or "anthropic"
}

func (c Config) isAnthropic() bool {
	return strings.EqualFold(c.APIType, "anthropic")
}

// IsConfigured returns true when all three fields are non-empty.
func (c Config) IsConfigured() bool {
	return c.BaseURL != "" && c.Token != "" && c.Model != ""
}

// stripConvertedPrefix removes a leading "converted/" path segment so that
// raw/converted/foo/bar.md maps to wiki/source-notes/foo/bar.md.
func stripConvertedPrefix(rel string) string {
	const prefix = "converted" + string(filepath.Separator)
	if strings.HasPrefix(rel, prefix) {
		return rel[len(prefix):]
	}
	return rel
}

// StripCodeFences removes a leading ```[language] ... ``` wrapper if present.
func StripCodeFences(text string) string {
	text = strings.TrimSpace(text)

	// Find opening fence line
	firstNL := strings.IndexByte(text, '\n')
	if firstNL < 0 {
		return text
	}
	firstLine := text[:firstNL]
	if !strings.HasPrefix(firstLine, "```") {
		return text
	}

	// Must end with closing fence
	if !strings.HasSuffix(text, "```") {
		return text
	}

	// Strip first line and last fence line
	inner := text[firstNL+1 : len(text)-3]
	return strings.TrimRight(inner, "\n")
}

// titleRe extracts the title value from a frontmatter `title:` line,
// tolerating optional surrounding quotes.
var titleRe = regexp.MustCompile(`(?m)^title:\s*"?([^"\n]*)"?\s*$`)

// titleFromNote returns the frontmatter title, or fallback if none is found.
func titleFromNote(note, fallback string) string {
	if m := titleRe.FindStringSubmatch(note); m != nil {
		if t := strings.TrimSpace(m[1]); t != "" {
			return t
		}
	}
	return fallback
}

// appendLog appends a greppable, chronological entry to wiki/log.md following
// the OKF/Karpathy convention: `## [YYYY-MM-DD] ingest | <title>`. The file is
// created if absent. Failures are non-fatal and returned for logging.
func appendLog(kbRoot, title string) error {
	logPath := filepath.Join(kbRoot, "wiki", "log.md")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return err
	}
	entry := fmt.Sprintf("## [%s] ingest | %s\n", time.Now().Format("2006-01-02"), title)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

// SetSourceField rewrites the `sources` field inside the leading YAML
// frontmatter to point at rawRel (a KB-root-relative path like "raw/foo.md").
// It removes any sources entry the LLM produced — inline (`sources: [...]`) or
// block list (`sources:\n  - ...`) — and inserts a canonical block form so the
// wiki↔raw graph link (parsed as a `cites` relation) is always correct.
// If no frontmatter is found, the text is returned unchanged.
func SetSourceField(text, rawRel string) string {
	if !strings.HasPrefix(text, "---\n") {
		return text
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return text
	}
	fmStart := 4
	fmEnd := 4 + end // index of the '\n' before closing '---'
	fm := text[fmStart:fmEnd]
	rest := text[fmEnd:]

	var out []string
	skippingList := false
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		// Drop list items belonging to a sources: block we're removing.
		if skippingList {
			if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
				continue
			}
			skippingList = false
		}
		if val, ok := strings.CutPrefix(trimmed, "sources:"); ok {
			// Inline form (sources: [..]) → just drop the line.
			// Block form (sources:) with no value → drop following list items too.
			if strings.TrimSpace(val) == "" {
				skippingList = true
			}
			continue
		}
		out = append(out, line)
	}

	newFM := strings.Join(out, "\n")
	newFM = strings.TrimRight(newFM, "\n")
	return "---\n" + newFM + "\nsources:\n  - " + rawRel + rest
}

// FindNewFiles walks kbRoot/raw/ for .md files that do not yet have a
// corresponding source-note under kbRoot/wiki/source-notes/.
// index.md and log.md are skipped.
func FindNewFiles(kbRoot string) []string {
	rawDir := filepath.Join(kbRoot, "raw")
	notesDir := filepath.Join(kbRoot, "wiki", "source-notes")

	var result []string
	_ = filepath.Walk(rawDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		base := filepath.Base(path)
		if base == "index.md" || base == "log.md" {
			return nil
		}

		// Build relative path from rawDir, stripping converted/ prefix so that
		// raw/converted/foo/bar.md maps to wiki/source-notes/foo/bar.md.
		rel, err := filepath.Rel(rawDir, path)
		if err != nil {
			return nil
		}
		rel = stripConvertedPrefix(rel)

		notePath := filepath.Join(notesDir, rel)
		if _, err := os.Stat(notePath); os.IsNotExist(err) {
			result = append(result, path)
		}
		return nil
	})
	return result
}

// anthropicRequest is the JSON body sent to the Anthropic messages endpoint.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the subset of the Anthropic messages response we care about.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// CallLLM sends rawContent to the configured LLM and returns the generated text.
// kbRoot is used to load a custom system prompt from schema/templates/source-note.md
// if present; otherwise the built-in default prompt is used.
// relatedContext, if non-empty, is appended to the system prompt to inject
// related-page context discovered via vector search.
// api_type in config selects the wire format: "anthropic" or "openai" (default).
func CallLLM(config Config, kbRoot, rawContent, relatedContext string) (string, error) {
	system := buildSystemPrompt(kbRoot)
	if relatedContext != "" {
		system = system + relatedContext
	}
	if config.isAnthropic() {
		return callAnthropic(config, system, rawContent)
	}
	return callOpenAI(config, system, rawContent)
}

// callAnthropic uses the Anthropic Messages API format.
func callAnthropic(config Config, system, userContent string) (string, error) {
	reqBody := anthropicRequest{
		Model:     config.Model,
		MaxTokens: 4096,
		System:    system,
		Messages:  []anthropicMessage{{Role: "user", Content: userContent}},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(config.BaseURL, "/") + "/v1/messages"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.Token)
	req.Header.Set("anthropic-version", "2023-06-01")

	respBody, err := doHTTP(req)
	if err != nil {
		return "", err
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	var parts []string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("LLM returned no text content")
	}
	return strings.Join(parts, ""), nil
}

// callOpenAI uses the OpenAI Chat Completions API format (compatible with most providers).
func callOpenAI(config Config, system, userContent string) (string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		Messages  []message `json:"messages"`
	}
	type choice struct {
		Message message `json:"message"`
	}
	type response struct {
		Choices []choice `json:"choices"`
	}

	reqBody, err := json.Marshal(request{
		Model:     config.Model,
		MaxTokens: 4096,
		Messages: []message{
			{Role: "system", Content: system},
			{Role: "user", Content: userContent},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(config.BaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.Token)

	respBody, err := doHTTP(req)
	if err != nil {
		return "", err
	}

	var apiResp response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}
	return apiResp.Choices[0].Message.Content, nil
}

// doHTTP executes the request with a 120s timeout and returns the response body.
func doHTTP(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API returned %d: %s", resp.StatusCode, body)
	}
	return body, nil
}

// DistillFile reads the raw file at rawPath, calls the LLM with optional
// related-page context from vector search, and writes the source-note.
// embedder may be nil — when nil, vector lookup is skipped.
func DistillFile(config Config, rawPath, kbRoot string, embedder kb.Embedder) error {
	rawContent, err := os.ReadFile(rawPath)
	if err != nil {
		return fmt.Errorf("read raw file: %w", err)
	}

	relatedContext := findRelatedNotes(kbRoot, string(rawContent), embedder)

	generated, err := CallLLM(config, kbRoot, string(rawContent), relatedContext)
	if err != nil {
		return fmt.Errorf("call LLM: %w", err)
	}

	generated = StripCodeFences(generated)

	rawDir := filepath.Join(kbRoot, "raw")
	rel, err := filepath.Rel(rawDir, rawPath)
	if err != nil {
		return fmt.Errorf("relative path: %w", err)
	}

	// Force the sources field to the raw source's DocID (KB-root-relative,
	// slash-separated) so index.go links the note to its raw source via `cites`.
	rawDocID := "raw/" + filepath.ToSlash(rel)
	generated = SetSourceField(generated, rawDocID)

	// Strip converted/ prefix: raw/converted/foo/bar.md → source-notes/foo/bar.md
	noteRel := stripConvertedPrefix(rel)
	notePath := filepath.Join(kbRoot, "wiki", "source-notes", noteRel)
	if err := os.MkdirAll(filepath.Dir(notePath), 0o755); err != nil {
		return fmt.Errorf("create notes dir: %w", err)
	}

	if err := os.WriteFile(notePath, []byte(generated), 0o644); err != nil {
		return fmt.Errorf("write note: %w", err)
	}

	// Record a chronological ingest entry in wiki/log.md.
	stem := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	if err := appendLog(kbRoot, titleFromNote(generated, stem)); err != nil {
		return fmt.Errorf("append log: %w", err)
	}
	return nil
}

// Run finds all undistilled raw files and distills each one.
// Returns (count, nil) on success or (0, err) on the first failure.
func Run(kbRoot string, config Config) (int, error) {
	if !config.IsConfigured() {
		return 0, fmt.Errorf("LLM config incomplete: BaseURL, Token, and Model are all required")
	}

	files := FindNewFiles(kbRoot)
	if len(files) == 0 {
		return 0, nil
	}

	for _, rawPath := range files {
		rel, _ := filepath.Rel(filepath.Join(kbRoot, "raw"), rawPath)
		fmt.Printf("distilling: raw/%s\n", rel)
		if err := DistillFile(config, rawPath, kbRoot, nil); err != nil {
			return 0, fmt.Errorf("distill %s: %w", rawPath, err)
		}
		fmt.Printf("distilled:  raw/%s\n", rel)
	}
	return len(files), nil
}
