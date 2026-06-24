//go:build fts5

package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jasen215/wikiloop/internal/kb"
)

// appendQueryLog appends a JSONL entry to wiki/query_log.jsonl for AI behavior analysis.
// Each line: {"ts":"...","tool":"kb_context|kb_search","query":"..."}
// Non-fatal: errors are silently ignored to avoid disrupting normal operation.
func appendQueryLog(kbRoot, tool, query string) {
	logPath := filepath.Join(kbRoot, "wiki", "query_log.jsonl")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	// Escape double quotes in query
	escaped := ""
	for _, c := range query {
		if c == '"' {
			escaped += `\"`
		} else if c == '\n' {
			escaped += `\n`
		} else {
			escaped += string(c)
		}
	}
	fmt.Fprintf(f, "{\"ts\":%q,\"tool\":%q,\"query\":%q}\n", ts, tool, query)
}

// handleKBStatus returns document and embedding counts plus index file size.
func handleKBStatus(kbRoot string) map[string]interface{} {
	dbPath := filepath.Join(kbRoot, "index", "kb.sqlite")

	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return map[string]interface{}{
			"documents":    0,
			"by_layer":     map[string]int{},
			"embeddings":   0,
			"index_path":   dbPath,
			"index_size_kb": int64(0),
		}
	}
	defer db.Close()

	byLayer, total, _ := kb.LayerCounts(db)

	var embCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&embCount)

	var indexSizeKB int64
	if fi, err := os.Stat(dbPath); err == nil {
		indexSizeKB = fi.Size() / 1024
	}

	return map[string]interface{}{
		"documents":     total,
		"by_layer":      byLayer,
		"embeddings":    embCount,
		"index_path":    dbPath,
		"index_size_kb": indexSizeKB,
	}
}

// handleKBSearch runs layered FTS search and returns results with related docs.
func handleKBSearch(kbRoot, query string, layer, kind *string, limit int) map[string]interface{} {
	appendQueryLog(kbRoot, "kb_search", query)
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defer db.Close()

	sourceLimit := limit
	if sourceLimit <= 0 {
		sourceLimit = 5
	}
	synthLimit := min(3, sourceLimit/2)
	if synthLimit < 1 {
		synthLimit = 1
	}

	results, conflicts, err := kb.SearchLayered(db, kbRoot, query, layer, kind, sourceLimit, synthLimit)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"results":   results,
		"conflicts": conflicts,
	}
}

// handleKBContext builds a context bundle for the given question.
func handleKBContext(kbRoot, question string, limit int, noVec bool, embedder kb.Embedder) map[string]interface{} {
	appendQueryLog(kbRoot, "kb_context", question)
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defer db.Close()

	var emb kb.Embedder
	if !noVec {
		emb = embedder
	}
	bundle := kb.BuildContext(db, kbRoot, question, emb, limit)

	return map[string]interface{}{
		"question":    bundle.Question,
		"wiki_pages":  bundle.WikiPages,
		"raw_sources": bundle.RawSources,
		"graph_pages": bundle.GraphPages,
		"conflicts":   bundle.Conflicts,
	}
}

// handleKBReindex walks kbRoot and re-indexes documents.
// full=true forces re-index of every document; full=false is incremental.
func handleKBReindex(kbRoot string, full bool) map[string]interface{} {
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defer db.Close()

	indexFn := kb.IndexFiles
	if full {
		indexFn = kb.IndexFilesFull
	}
	written, err := indexFn(db, kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"message": "index updated",
		"written": written,
	}
}

// handleKBPage fetches full content for one or more wiki pages by ID.
func handleKBPage(kbRoot string, ids []string, full bool) map[string]interface{} {
	if len(ids) == 0 {
		return map[string]interface{}{"error": "ids is required"}
	}
	if len(ids) > 5 {
		ids = ids[:5] // cap at 5
	}
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defer db.Close()

	pages, err := kb.FetchPages(db, kbRoot, ids, full)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"pages": pages}
}

// handleKBLint runs deterministic health checks over wiki pages and returns
// the list of warnings (missing required fields, broken source links).
func handleKBLint(kbRoot string) map[string]interface{} {
	warnings, err := kb.Lint(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"warnings": warnings,
		"count":    len(warnings),
	}
}
