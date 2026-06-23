//go:build fts5

package synthesize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config holds LLM API configuration (shared with distill).
type Config struct {
	BaseURL string
	Token   string
	Model   string
	APIType string // "openai" (default) or "anthropic"
}

func (c Config) IsConfigured() bool {
	return c.BaseURL != "" && c.Token != "" && c.Model != ""
}

func (c Config) isAnthropic() bool {
	return strings.EqualFold(c.APIType, "anthropic")
}

// PagePlan describes one wiki page the LLM proposes to generate.
type PagePlan struct {
	Type        string   `json:"type"`         // "concept", "comparison", or "decision"
	Title       string   `json:"title"`        // proposed page title
	Slug        string   `json:"slug"`         // filename slug, e.g. "llm-memory-systems"
	Description string   `json:"description"`  // one-sentence purpose
	Sources     []string `json:"sources"`      // wiki/source-notes/ paths to use
}

const planSystemPrompt = `You are a knowledge-base curator. Given a list of source-note summaries, identify opportunities to synthesize higher-level wiki pages following the Karpathy LLM Wiki structure.

LANGUAGE RULE (mandatory):
- title: Chinese main title, keep technical terms in their original English form (e.g. "RAG 召回率优化方法论", "GraphRAG vs Agentic Search 对比")
- slug: kebab-case derived from the English terms in title (e.g. "rag-recall-optimization-methodology")
- description: Chinese prose with key technical terms inline in English (e.g. "对比 RAG（Retrieval-Augmented Generation）与长上下文（Long Context）方案的适用场景")
This ensures FTS matches both Chinese queries ("召回率优化") and English queries ("RAG recall optimization").

Output a JSON array. Each element must have:
  type:        "concept" | "comparison" | "decision"
  title:       concise page title (in source language, usually Chinese)
  slug:        kebab-case filename (no extension)
  description: one sentence describing the page's value (in source language)
  sources:     list of source-note paths used

Page type rules:
- concept: a reusable idea, method, or pattern. Can be generated from a single source-note as an initial draft.
- comparison: two or more tools, approaches, or claims worth contrasting. Can start with one source if the source itself contains a comparison.
- decision: a judgment about whether a specific tool/approach suits the current KB's domain.

Do NOT generate:
- Pages that duplicate existing ones (listed below)
- Generic summaries without insight

Output a JSON array only. No other text.`

// Plan asks the LLM to propose concept/comparison/decision pages based on
// the given source-notes. existingTitles lists already-generated pages to skip.
// topic, if non-empty, focuses the plan on a specific subject.
func Plan(cfg Config, notes []SourceNote, existingTitles []string, topic string) ([]PagePlan, error) {
	// Build a compact summary of source-notes for the LLM.
	var sb strings.Builder
	if topic != "" {
		fmt.Fprintf(&sb, "Focus topic: %s\nOnly propose pages relevant to this topic.\n\n", topic)
	}
	sb.WriteString("Source notes available:\n\n")
	for _, n := range notes {
		fmt.Fprintf(&sb, "- path: %s\n  title: %s\n  description: %s\n  tags: [%s]\n",
			n.Path, n.Title, n.Description, strings.Join(n.Tags, ", "))
		if len(n.KeyClaims) > 0 {
			fmt.Fprintf(&sb, "  key_claims: [%s]\n", strings.Join(n.KeyClaims, "; "))
		}
		sb.WriteString("\n")
	}
	if len(existingTitles) > 0 {
		sb.WriteString("Already exists (do not propose):\n")
		for _, t := range existingTitles {
			fmt.Fprintf(&sb, "- %s\n", t)
		}
	}

	respText, err := callLLM(cfg, planSystemPrompt, sb.String())
	if err != nil {
		return nil, err
	}

	// Strip code fences if the LLM wrapped the JSON.
	respText = strings.TrimSpace(respText)
	if strings.HasPrefix(respText, "```") {
		first := strings.Index(respText, "\n")
		last := strings.LastIndex(respText, "```")
		if first >= 0 && last > first {
			respText = strings.TrimSpace(respText[first:last])
		}
	}

	var plans []PagePlan
	if err := json.Unmarshal([]byte(respText), &plans); err != nil {
		return nil, fmt.Errorf("parse plan JSON: %w\nraw response: %s", err, respText)
	}
	return plans, nil
}

// buildGeneratePrompt returns the system prompt for generating a specific page type.
func buildGeneratePrompt(kbRoot, pageType string) string {
	templatePath := filepath.Join(kbRoot, "schema", "templates", pageType+".md")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return defaultGeneratePrompt(pageType)
	}
	return fmt.Sprintf(`You are a knowledge-base curator. Generate a wiki %s page in Markdown with YAML frontmatter.

LANGUAGE RULE (mandatory):
- title: Chinese main title, keep technical terms in their original English form
  GOOD: "RAG 召回率优化全链路方法论"  BAD: "Full-Chain RAG Recall Optimization Methodology"
- description: Chinese prose with key technical terms inline in English
  GOOD: "对比 RAG（Retrieval-Augmented Generation）与长上下文（Long Context）方案的适用场景与权衡"
- body: Chinese prose, technical terms inline bilingual (e.g. "混合检索（Hybrid Search，BM25+向量）")
This dual-language approach ensures FTS matches both Chinese queries ("召回率") and English queries ("recall rate").

Output MUST be valid Markdown. Do NOT wrap output in code fences.
Begin directly with the YAML frontmatter (---).

Use this template as the exact structure:

%s`, pageType, string(data))
}

func defaultGeneratePrompt(pageType string) string {
	return fmt.Sprintf(`You are a knowledge-base curator. Generate a wiki %s page in Markdown with YAML frontmatter.

LANGUAGE RULE (mandatory):
- title: Chinese main title, keep technical terms in their original English form
- description: Chinese prose with key technical terms inline in English
- body: Chinese prose, technical terms inline bilingual
This ensures FTS matches both Chinese and English queries.

Output MUST be valid Markdown with YAML frontmatter. Do NOT wrap output in code fences.
The frontmatter must include: type, title, description, tags, sources, timestamp.
Begin directly with the YAML frontmatter (---).`, pageType)
}

// callLLM dispatches to Anthropic or OpenAI wire format based on cfg.APIType.
func callLLM(cfg Config, system, userContent string) (string, error) {
	if cfg.isAnthropic() {
		return callAnthropicAPI(cfg, system, userContent)
	}
	return callOpenAIAPI(cfg, system, userContent)
}

func callAnthropicAPI(cfg Config, system, userContent string) (string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		System    string    `json:"system"`
		Messages  []message `json:"messages"`
	}
	type response struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	reqBody, _ := json.Marshal(request{
		Model: cfg.Model, MaxTokens: 4096, System: system,
		Messages: []message{{Role: "user", Content: userContent}},
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.Token)
	req.Header.Set("anthropic-version", "2023-06-01")

	body, err := doRequest(req)
	if err != nil {
		return "", err
	}
	var res response
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(res.Content) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}
	return res.Content[0].Text, nil
}

func callOpenAIAPI(cfg Config, system, userContent string) (string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		Messages  []message `json:"messages"`
	}
	type response struct {
		Choices []struct {
			Message message `json:"message"`
		} `json:"choices"`
	}

	reqBody, _ := json.Marshal(request{
		Model: cfg.Model, MaxTokens: 4096,
		Messages: []message{{Role: "system", Content: system}, {Role: "user", Content: userContent}},
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+"/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	body, err := doRequest(req)
	if err != nil {
		return "", err
	}
	var res response
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(res.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}
	return res.Choices[0].Message.Content, nil
}

func doRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API %d: %s", resp.StatusCode, body)
	}
	return body, nil
}
