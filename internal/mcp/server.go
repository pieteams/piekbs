//go:build fts5

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jasen215/wikiloop/internal/embed"
	"github.com/jasen215/wikiloop/internal/kb"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const serverInstructions = `WikiLoop is a persistent, cross-session knowledge base for this project. It stores
distilled wiki pages, the raw source documents they cite, a concept graph linking
related pages, and recorded conflicts between sources. It is the project's long-term
memory: design decisions, specs, roadmaps/phase plans, past approaches, incident
notes, and the "why" behind the code — knowledge that is NOT recoverable by reading
the current code or git history.

WHEN TO USE WikiLoop (prefer it over guessing or relying on the code alone):
  - The user asks "why did we…", "how was this designed", "what was the plan/spec".
  - You are about to change architecture, a protocol, an index/KB, or other
    long-lived design, and need the original rationale or constraints.
  - You need project background, requirements provenance, or roadmap/phase context
    that the code does not explain on its own.
  - Your answer would shape a long-term decision and you are not certain of the
    background.
  - You suspect prior art exists ("have we solved this before?").

HOW TO USE:
  - Start with kb_context(question=…) for a ready-to-use bundle: relevant wiki
    pages + the raw sources they cite + graph neighbors + known conflicts. This is
    the primary entry point for "give me the background on X".
  - Use kb_search(query=…) when you want to browse ranked hits or scope to a layer
    (wiki / raw / schema).
  - kb_status reports index health; kb_reindex refreshes the FTS index after the
    KB files change.

DO NOT use WikiLoop for questions answerable from the current code, file structure,
or git history — read those directly instead. WikiLoop is read-only knowledge; it
does not reflect uncommitted local edits.`

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

	// Initialize embedder: find model dir then create embedder.
	var embedder kb.Embedder
	if modelDir := embed.FindModelDir(filepath.Join(kbRoot, "models")); modelDir != "" {
		embedder = embed.NewONNXEmbedder(modelDir, 384, 5*time.Minute)
	}

	registerTools(s, kbRoot, embedder)

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
		mcp.WithDescription("Search the WikiLoop knowledge base (hybrid FTS + vector) and return ranked results."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
		mcp.WithString("layer", mcp.Description("Filter layer: wiki, raw, or schema")),
		mcp.WithNumber("limit", mcp.Description("Maximum results (default 10)")),
		mcp.WithBoolean("no_vec", mcp.Description("Disable vector search (default false)")),
	)
	s.AddTool(searchTool, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		limit := req.GetInt("limit", 10)
		noVec := req.GetBool("no_vec", false)

		var layer *string
		if l := req.GetString("layer", ""); l != "" {
			layer = &l
		}

		data := handleKBSearch(kbRoot, query, layer, limit, noVec, embedder)
		return toolResultJSON(data)
	})

	// kb_context ─────────────────────────────────────────────────────────────
	contextTool := mcp.NewTool("kb_context",
		mcp.WithDescription("Build a ready-to-use context bundle for a question: wiki pages, raw sources, graph neighbors, conflicts."),
		mcp.WithString("question", mcp.Required(), mcp.Description("Question to build context for")),
		mcp.WithNumber("limit", mcp.Description("Maximum wiki pages (default 5)")),
		mcp.WithBoolean("no_vec", mcp.Description("Disable vector search (default false)")),
	)
	s.AddTool(contextTool, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		question := req.GetString("question", "")
		limit := req.GetInt("limit", 5)
		noVec := req.GetBool("no_vec", false)

		data := handleKBContext(kbRoot, question, limit, noVec, embedder)
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
