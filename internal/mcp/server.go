//go:build fts5

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jasen215/wikiloop/internal/kb"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const serverInstructions = `WikiLoop is a knowledge search engine for this project. It stores distilled
wiki pages (source-notes, concepts, comparisons, decisions) and the raw source
documents they cite. Use it like a search engine: search with multiple keyword
combinations to discover relevant documents, then deep-read the ones that matter.

WikiLoop provides MATERIALS for you to synthesize — it does not generate answers.
You are expected to search iteratively, cross-verify with other sources, and form
your own conclusions.

WHEN TO USE WikiLoop:
  - The user asks "why did we…", "how was this designed", "what was the plan/spec".
  - You need project background, design decisions, or rationale not in the code.
  - You suspect prior art exists ("have we solved this before?").

HOW TO USE:
  1. kb_search(query=…) — Search with a keyword or phrase. Returns up to 5
     source-notes + 3 concept/comparison/decision pages. Each result includes
     related documents for further navigation.
  2. Repeat kb_search with DIFFERENT keywords to cover the topic from multiple
     angles. Do NOT repeat the same query — switch keywords or topic angle.
  3. kb_page(ids=[…]) — Deep-read documents of interest using IDs from
     kb_search results. Pass multiple IDs (up to 5) to scan several at once,
     or use full=true with a single ID to get complete untruncated text.

QUERY EXPANSION (mandatory):
  Before searching, expand the query into aliases, abbreviations, and
  cross-language equivalents. FTS uses exact term matching.
  Examples:
    "召回率" → also search "recall", "Context Recall", "CR"
    "蒸馏"   → also search "distill", "source-note", "知识蒸馏"
    "向量数据库" → also search "vector database", "vector store", "Qdrant", "Milvus"

DO NOT:
  - Repeat the same query — it returns identical results, switch keywords instead
  - Expect WikiLoop to give you the answer — synthesize from what you find
  - Use WikiLoop for questions answerable from current code or git history

CITATION RULES (mandatory):
  - Always cite source paths in your answer using the id/path field.
    Example: "According to [wiki/source-notes/xxx.md], ..."
  - If context is insufficient, say so explicitly rather than hallucinating.
  - If a conflict appears in results, acknowledge both sides.`

// Start creates an MCP HTTP server, registers KB tools, and listens on addr.
// apiKey is reserved for future auth middleware; currently unused.
func Start(addr, kbRoot, apiKey string) error {
	mux := http.NewServeMux()
	RegisterRoutes(mux, kbRoot)
	return http.ListenAndServe(addr, mux)
}

// RegisterRoutes adds MCP endpoints to the given mux.
// Use this when sharing a mux with other handlers (e.g., Web UI).
func RegisterRoutes(mux *http.ServeMux, kbRoot string, apiKey ...string) {
	key := ""
	if len(apiKey) > 0 {
		key = apiKey[0]
	}
	s := mcpserver.NewMCPServer(
		"wikiloop",
		"1.0.0",
		mcpserver.WithToolCapabilities(false),
		mcpserver.WithInstructions(serverInstructions),
	)

	registerTools(s, kbRoot, nil) // embedder always nil after devectorization

	httpSrv := mcpserver.NewStreamableHTTPServer(s)

	handler := withCORS(withAPIKey(key, httpSrv))
	mux.Handle("/mcp", handler)
	mux.Handle("/mcp/", handler)
}

// registerTools adds the four KB tools to s.
func registerTools(s *mcpserver.MCPServer, kbRoot string, embedder kb.Embedder) {
	// kb_status ─────────────────────────────────────────────────────────────
	statusTool := mcp.NewTool("kb_status",
		mcp.WithDescription("Report WikiLoop KB index stats (document counts, embedding coverage, index path/size)."),
	)
	s.AddTool(statusTool, func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data := handleKBStatus(kbRoot)
		return toolResultJSON(data)
	})

	// kb_search ──────────────────────────────────────────────────────────────
	searchTool := mcp.NewTool("kb_search",
		mcp.WithDescription("Search the WikiLoop knowledge base (FTS) and return ranked results."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithString("layer", mcp.Description("Filter layer: wiki, raw, or schema")),
		mcp.WithString("kind", mcp.Description("Filter page kind: source-note, concept, comparison, decision")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 10)")),
	)
	s.AddTool(searchTool, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		limit := req.GetInt("limit", 10)

		var layer *string
		if l := req.GetString("layer", ""); l != "" {
			layer = &l
		}
		var kind *string
		if k := req.GetString("kind", ""); k != "" {
			kind = &k
		}

		data := handleKBSearch(kbRoot, query, layer, kind, limit)
		return toolResultJSON(data)
	})

	// kb_context is superseded by kb_search + kb_page pattern.
	// Kept commented for reference. Remove after kb_page has been stable for 30 days.
	// contextTool := mcp.NewTool("kb_context",
	// 	mcp.WithDescription("Build a ready-to-use context bundle for a question: wiki pages, raw sources, graph neighbors, conflicts."),
	// 	mcp.WithString("question", mcp.Required(), mcp.Description("Question to build context for")),
	// 	mcp.WithNumber("limit", mcp.Description("Maximum wiki pages (default 10)")),
	// 	mcp.WithBoolean("no_vec", mcp.Description("Disable vector search (default false)")),
	// )
	// s.AddTool(contextTool, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 	question := req.GetString("question", "")
	// 	limit := req.GetInt("limit", 10)
	// 	noVec := req.GetBool("no_vec", false)
	//
	// 	data := handleKBContext(kbRoot, question, limit, noVec, embedder)
	// 	return toolResultJSON(data)
	// })

	// kb_page ─────────────────────────────────────────────────────────────────
	pageTool := mcp.NewTool("kb_page",
		mcp.WithDescription("Fetch full content of one or more wiki pages by ID. Use after kb_search to deep-read documents of interest. IDs come from kb_search result id fields. Pass full=true with a single ID to get complete untruncated text."),
		mcp.WithArray("ids", mcp.Required(), mcp.Description("Document IDs to fetch (1-5, from kb_search result id fields)")),
		mcp.WithBoolean("full", mcp.Description("Return complete untruncated text. Only applies when a single ID is passed (default false)")),
	)
	s.AddTool(pageTool, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ids := req.GetStringSlice("ids", nil)
		full := req.GetBool("full", false)
		data := handleKBPage(kbRoot, ids, full)
		return toolResultJSON(data)
	})

	// kb_reindex ─────────────────────────────────────────────────────────────
	reindexTool := mcp.NewTool("kb_reindex",
		mcp.WithDescription("Rebuild the WikiLoop FTS index incrementally after KB files change."),
		mcp.WithBoolean("full", mcp.Description("Force complete rebuild (default false)")),
	)
	s.AddTool(reindexTool, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		full := req.GetBool("full", false)
		data := handleKBReindex(kbRoot, full)
		return toolResultJSON(data)
	})

	// kb_lint ──────────────────────────────────────────────────────────────────
	lintTool := mcp.NewTool("kb_lint",
		mcp.WithDescription("Health-check wiki pages: report missing required frontmatter fields and broken source links (deterministic, no LLM)."),
	)
	s.AddTool(lintTool, func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		data := handleKBLint(kbRoot)
		return toolResultJSON(data)
	})
}

// withAPIKey rejects requests missing a valid x-api-key header.
// If key is empty, all requests are allowed (auth disabled).
func withAPIKey(key string, h http.Handler) http.Handler {
	if key == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != key {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// withCORS adds permissive CORS headers for local MCP clients.
func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// toolResultJSON marshals data to JSON and returns a text CallToolResult.
func toolResultJSON(data map[string]interface{}) (*mcp.CallToolResult, error) {
	if errMsg, hasErr := data["error"]; hasErr {
		return mcp.NewToolResultError(fmt.Sprintf("%v", errMsg)), nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(b)), nil
}
