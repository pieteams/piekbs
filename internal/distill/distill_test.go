//go:build fts5

package distill_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasen215/wikiloop/internal/distill"
)

func TestStripCodeFences(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with markdown fences",
			input: "```markdown\n---\ntitle: Test\n---\n\nBody content\n```",
			want:  "---\ntitle: Test\n---\n\nBody content",
		},
		{
			name:  "without fences",
			input: "---\ntitle: Test\n---\n\nBody content",
			want:  "---\ntitle: Test\n---\n\nBody content",
		},
		{
			name:  "bare code block",
			input: "```\nsome code\n```",
			want:  "some code",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := distill.StripCodeFences(tc.input)
			if got != tc.want {
				t.Errorf("StripCodeFences(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSetSourceField(t *testing.T) {
	const raw = "raw/articles/note.md"
	want := "---\ntype: source-note\ntitle: T\nsources:\n  - " + raw + "\n---\nBody"

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "placeholder honored",
			input: "---\ntype: source-note\ntitle: T\nsources: [\"__RAW_SOURCE__\"]\n---\nBody",
			want:  want,
		},
		{
			name:  "inline list overwritten",
			input: "---\ntype: source-note\ntitle: T\nsources: [https://wrong.example, foo]\n---\nBody",
			want:  want,
		},
		{
			name:  "block list overwritten",
			input: "---\ntype: source-note\ntitle: T\nsources:\n  - https://wrong.example\n  - bar\n---\nBody",
			want:  want,
		},
		{
			name:  "missing sources gets added",
			input: "---\ntype: source-note\ntitle: T\n---\nBody",
			want:  want,
		},
		{
			name:  "no frontmatter unchanged",
			input: "Body only, no frontmatter",
			want:  "Body only, no frontmatter",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := distill.SetSourceField(tc.input, raw)
			if got != tc.want {
				t.Errorf("SetSourceField()\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestDistillFile_WritesLogAndSources(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "raw"), 0755)
	rawPath := filepath.Join(dir, "raw", "topic.md")
	os.WriteFile(rawPath, []byte("raw body about widgets"), 0644)

	// Stub LLM returns a note with a deliberately wrong sources field.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		note := "---\ntype: source-note\ntitle: Widget Overview\nsources: [https://wrong]\n---\n\n## Summary\nstuff"
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"type": "text", "text": note}},
		})
	}))
	defer srv.Close()

	cfg := distill.Config{BaseURL: srv.URL, Token: "k", Model: "m", APIType: "anthropic"}
	if err := distill.DistillFile(cfg, rawPath, dir, nil); err != nil {
		t.Fatal(err)
	}

	// source-note sources must be rewritten to the raw DocID (graph link).
	note, _ := os.ReadFile(filepath.Join(dir, "wiki", "source-notes", "topic.md"))
	if !strings.Contains(string(note), "sources:\n  - raw/topic.md") {
		t.Errorf("note sources not rewritten to raw path:\n%s", note)
	}
	if strings.Contains(string(note), "wrong") {
		t.Errorf("LLM's bogus sources value leaked into note:\n%s", note)
	}

	// log.md must contain a greppable ingest entry using the note's title.
	logBytes, err := os.ReadFile(filepath.Join(dir, "wiki", "log.md"))
	if err != nil {
		t.Fatalf("log.md not written: %v", err)
	}
	logStr := string(logBytes)
	if !strings.Contains(logStr, "] ingest | Widget Overview") {
		t.Errorf("log.md missing ingest entry with title:\n%s", logStr)
	}
}

func TestFindNewFiles(t *testing.T) {
	// Create temp KB root
	kbRoot := t.TempDir()

	rawDir := filepath.Join(kbRoot, "raw")
	notesDir := filepath.Join(kbRoot, "wiki", "source-notes")

	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create raw/a.md and raw/b.md
	if err := os.WriteFile(filepath.Join(rawDir, "a.md"), []byte("# A"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "b.md"), []byte("# B"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create wiki/source-notes/a.md (already distilled)
	if err := os.WriteFile(filepath.Join(notesDir, "a.md"), []byte("# Note A"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := distill.FindNewFiles(kbRoot)
	if len(files) != 1 {
		t.Fatalf("FindNewFiles returned %d files, want 1; got: %v", len(files), files)
	}

	// The returned path should contain b.md
	if filepath.Base(files[0]) != "b.md" {
		t.Errorf("expected b.md, got %q", files[0])
	}
}

func TestCallLLM(t *testing.T) {
	// Build a fake Anthropic-format response
	type contentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type response struct {
		Content []contentBlock `json:"content"`
	}

	expectedText := "---\ntitle: Distilled\n---\n\nBody"
	resp := response{
		Content: []contentBlock{
			{Type: "text", Text: expectedText},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected /v1/messages, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") == "" {
			t.Error("missing x-api-key header")
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	cfg := distill.Config{
		BaseURL: srv.URL,
		Token:   "test-token",
		Model:   "claude-test",
		APIType: "anthropic",
	}

	got, err := distill.CallLLM(cfg, "", "raw content here", "")
	if err != nil {
		t.Fatalf("CallLLM error: %v", err)
	}
	if got != expectedText {
		t.Errorf("CallLLM = %q, want %q", got, expectedText)
	}
}
