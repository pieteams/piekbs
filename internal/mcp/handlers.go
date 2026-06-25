//go:build fts5

package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jasen215/wikiloop/internal/distill"
	"github.com/jasen215/wikiloop/internal/kb"
)

// appendQueryLog appends a JSONL entry to wiki/query_log.jsonl for AI behavior analysis.
// Each line: {"ts":"...","tool":"kb_search|kb_page|kb_status|kb_reindex|kb_lint","query":"..."}
// Non-fatal: errors are silently ignored to avoid disrupting normal operation.
func appendQueryLog(kbRoot, tool, query string) {
	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	logPath := filepath.Join(kbRoot, "wiki", "query_log_"+date+".jsonl")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := now.Format("2006-01-02T15:04:05Z")
	fmt.Fprintf(f, "{\"ts\":%q,\"tool\":%q,\"query\":%q}\n", ts, tool, query)
}

// handleKBStatus returns document and embedding counts plus index file size.
func handleKBStatus(kbRoot string) map[string]interface{} {
	appendQueryLog(kbRoot, "kb_status", "")
	dbPath := filepath.Join(kbRoot, "index", "kb.sqlite")

	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return map[string]interface{}{
			"documents":     0,
			"by_layer":      map[string]int{},
			"embeddings":    0,
			"index_path":    dbPath,
			"index_size_kb": int64(0),
			"distill_queue": map[string]int{},
		}
	}
	defer db.Close()

	byLayer, total, _ := kb.LayerCounts(db)
	queueStats, _ := distill.Stats(db)

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
		"distill_queue": queueStats,
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
	appendQueryLog(kbRoot, "kb_reindex", "")
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
	appendQueryLog(kbRoot, "kb_page", strings.Join(ids, ","))
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

// handleKBAdd writes content to raw/<filename> under kbRoot and triggers an
// incremental index update. Distillation is handled asynchronously by the watcher.
func handleKBAdd(kbRoot, filename, content, sourceURL string, overwrite bool) map[string]interface{} {
	appendQueryLog(kbRoot, "kb_add", filename)

	if strings.TrimSpace(filename) == "" {
		return map[string]interface{}{"error": "filename is required"}
	}
	if strings.Contains(filepath.ToSlash(filename), "..") {
		return map[string]interface{}{"error": "filename must not contain '..'"}
	}

	dst := filepath.Join(kbRoot, "raw", filepath.FromSlash(filename))

	if !overwrite {
		if _, err := os.Stat(dst); err == nil {
			return map[string]interface{}{"error": "file already exists: raw/" + filename + ", use overwrite=true"}
		}
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return map[string]interface{}{"error": "mkdir failed: " + err.Error()}
	}

	body := content
	if strings.TrimSpace(sourceURL) != "" {
		body = "<!-- source: " + sourceURL + " -->\n\n" + content
	}

	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		return map[string]interface{}{"error": "write failed: " + err.Error()}
	}

	// Synchronous incremental index so kb_search finds the file immediately.
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return map[string]interface{}{"path": "raw/" + filename, "indexed": false, "index_error": err.Error()}
	}
	defer db.Close()
	if _, err := kb.IndexFiles(db, kbRoot); err != nil {
		return map[string]interface{}{"path": "raw/" + filename, "indexed": false, "index_error": err.Error()}
	}

	return map[string]interface{}{"path": "raw/" + filename, "indexed": true}
}

// handleKBLint runs deterministic health checks over wiki pages and returns
// the list of warnings (missing required fields, broken source links).
func handleKBLint(kbRoot string) map[string]interface{} {
	appendQueryLog(kbRoot, "kb_lint", "")
	warnings, err := kb.Lint(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"warnings": warnings,
		"count":    len(warnings),
	}
}
