//go:build fts5

package mcp

import (
	"os"
	"path/filepath"

	"github.com/jasen215/wikiloop/internal/kb"
)

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

// handleKBSearch runs FTS (+ optional vector) search and returns results.
// noVec=true skips vector search even when an embedder is available.
func handleKBSearch(kbRoot, query string, layer *string, limit int, noVec bool, embedder kb.Embedder) map[string]interface{} {
	db, err := kb.OpenDB(kbRoot)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defer db.Close()

	var emb kb.Embedder
	if !noVec {
		emb = embedder
	}
	results, graphPages, conflicts, err := kb.Search(db, kbRoot, query, layer, limit, emb)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"results":     results,
		"graph_pages": graphPages,
		"conflicts":   conflicts,
	}
}

// handleKBContext builds a context bundle for the given question.
func handleKBContext(kbRoot, question string, limit int, noVec bool, embedder kb.Embedder) map[string]interface{} {
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
