//go:build fts5

package kb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// KBError is a typed error returned by all KB service functions.
// Code follows HTTP semantics: 400 = client error, 500 = server error.
type KBError struct {
	Code    int
	Message string
}

func (e *KBError) Error() string { return e.Message }

// AppendQueryLog writes a JSONL entry to wiki/query_log_YYYY-MM-DD.jsonl.
// extra is alternating key/value pairs appended to the JSON object.
// Non-fatal: errors are silently ignored.
func AppendQueryLog(kbRoot, tool, query string, extra ...string) {
	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	logPath := filepath.Join(kbRoot, "wiki", "query_log_"+date+".jsonl")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := now.Format("2006-01-02T15:04:05Z")
	if len(extra) >= 2 {
		// Build extra JSON fields: "key": <value>
		var extraJSON string
		for i := 0; i+1 < len(extra); i += 2 {
			extraJSON += fmt.Sprintf(",%q:%s", extra[i], extra[i+1])
		}
		fmt.Fprintf(f, "{\"ts\":%q,\"tool\":%q,\"query\":%q%s}\n", ts, tool, query, extraJSON)
	} else {
		fmt.Fprintf(f, "{\"ts\":%q,\"tool\":%q,\"query\":%q}\n", ts, tool, query)
	}
}

// StatusResult is the unified status response for both MCP and WebUI.
type StatusResult struct {
	Documents    int            `json:"documents"`
	ByLayer      map[string]int `json:"by_layer"`
	ByKind       map[string]int `json:"by_kind"`
	IndexPath    string         `json:"index_path"`
	IndexSize    int64          `json:"index_size"`
	DistillQueue map[string]int `json:"distill_queue"`
}

// SearchResponse wraps SearchLayered results.
type SearchResponse struct {
	Results   []SearchResult `json:"results"`
	Conflicts []Conflict     `json:"conflicts"`
}

// distillQueueStats returns task counts by status from distill_queue.
// Inlined here to avoid an import cycle between kb ↔ distill.
func distillQueueStats(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(`SELECT status, COUNT(*) FROM distill_queue GROUP BY status`)
	if err != nil {
		return map[string]int{"pending": 0, "processing": 0, "done": 0, "failed": 0}, nil
	}
	defer rows.Close()
	counts := map[string]int{"pending": 0, "processing": 0, "done": 0, "failed": 0}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return counts, err
		}
		counts[status] = n
	}
	return counts, rows.Err()
}

// KBStatus returns unified KB status (documents, layers, kinds, queue, index size).
func KBStatus(kbRoot string) (*StatusResult, error) {
	AppendQueryLog(kbRoot, "kb_status", "")
	dbPath := filepath.Join(kbRoot, "index", "kb.sqlite")
	db, err := OpenDB(kbRoot)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	defer db.Close()

	byLayer, total, _ := LayerCounts(db)
	byKind, _ := KindCounts(db)
	queueStats, _ := distillQueueStats(db)

	var indexSize int64
	if fi, err := os.Stat(dbPath); err == nil {
		indexSize = fi.Size()
	}

	return &StatusResult{
		Documents:    total,
		ByLayer:      byLayer,
		ByKind:       byKind,
		IndexPath:    dbPath,
		IndexSize:    indexSize,
		DistillQueue: queueStats,
	}, nil
}

// KBSearch runs layered FTS search and returns results with related docs.
func KBSearch(kbRoot, query string, layer, kind *string, sourceLimit, synthLimit int) (*SearchResponse, error) {
	db, err := OpenDB(kbRoot)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	defer db.Close()

	results, conflicts, err := SearchLayered(db, kbRoot, query, layer, kind, sourceLimit, synthLimit)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}

	// Log with related IDs.
	var relatedIDs []string
	seen := make(map[string]bool)
	for _, r := range results {
		for _, rel := range r.Related {
			if !seen[rel.ID] {
				seen[rel.ID] = true
				relatedIDs = append(relatedIDs, rel.ID)
			}
		}
	}
	if len(relatedIDs) > 0 {
		b, _ := json.Marshal(relatedIDs)
		AppendQueryLog(kbRoot, "kb_search", query, "related", string(b))
	} else {
		AppendQueryLog(kbRoot, "kb_search", query)
	}

	if results == nil {
		results = []SearchResult{}
	}
	if conflicts == nil {
		conflicts = []Conflict{}
	}
	return &SearchResponse{Results: results, Conflicts: conflicts}, nil
}

// KBPage fetches full content for one or more wiki pages by ID (max 5).
func KBPage(kbRoot string, ids []string, full bool) ([]PageResult, error) {
	if len(ids) == 0 {
		return nil, &KBError{Code: 400, Message: "ids is required"}
	}
	if len(ids) > 5 {
		ids = ids[:5]
	}
	AppendQueryLog(kbRoot, "kb_page", strings.Join(ids, ","))
	db, err := OpenDB(kbRoot)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	defer db.Close()

	pages, err := FetchPages(db, kbRoot, ids, full)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	return pages, nil
}

// AddResult is returned by KBAdd.
type AddResult struct {
	Path       string `json:"path"`
	Indexed    bool   `json:"indexed"`
	IndexError string `json:"index_error,omitempty"`
}

// ReindexResult is returned by KBReindex.
type ReindexResult struct {
	Written int    `json:"written"`
	Message string `json:"message"`
}

// LintResult is returned by KBLint.
type LintResult struct {
	Warnings     []LintWarning `json:"warnings"`
	Count        int           `json:"count"`
	RedLinks     []RedLink     `json:"red_links"`
	BrokenLinks  int           `json:"broken_links"`
	Placeholders int           `json:"placeholders"`
}

// KBAdd writes text content to raw/<filename> and triggers incremental indexing.
func KBAdd(kbRoot, filename, content, sourceURL string, overwrite bool) (*AddResult, error) {
	AppendQueryLog(kbRoot, "kb_add", filename)
	if strings.TrimSpace(filename) == "" {
		return nil, &KBError{Code: 400, Message: "filename is required"}
	}
	if strings.Contains(filepath.ToSlash(filename), "..") {
		return nil, &KBError{Code: 400, Message: "filename must not contain '..'"}
	}

	dst := filepath.Join(kbRoot, "raw", filepath.FromSlash(filename))
	if !overwrite {
		if _, err := os.Stat(dst); err == nil {
			return nil, &KBError{Code: 400, Message: "file already exists: raw/" + filename + ", use overwrite=true"}
		}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return nil, &KBError{Code: 500, Message: "mkdir failed: " + err.Error()}
	}

	body := content
	if strings.TrimSpace(sourceURL) != "" {
		body = "<!-- source: " + sourceURL + " -->\n\n" + content
	}
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		return nil, &KBError{Code: 500, Message: "write failed: " + err.Error()}
	}

	db, err := OpenDB(kbRoot)
	if err != nil {
		return &AddResult{Path: "raw/" + filename, Indexed: false, IndexError: err.Error()}, nil
	}
	defer db.Close()
	if _, err := IndexFiles(db, kbRoot); err != nil {
		return &AddResult{Path: "raw/" + filename, Indexed: false, IndexError: err.Error()}, nil
	}
	return &AddResult{Path: "raw/" + filename, Indexed: true}, nil
}

// KBUpload writes binary content from r to raw/<filename> (max 100MB).
func KBUpload(kbRoot, filename string, r io.Reader) error {
	AppendQueryLog(kbRoot, "kb_upload", filename)
	if filename == "" || filename == "." {
		return &KBError{Code: 400, Message: "invalid filename"}
	}
	dst := filepath.Join(kbRoot, "raw", filepath.Base(filename))
	const maxUploadBytes = 100 << 20

	out, err := os.Create(dst)
	if err != nil {
		return &KBError{Code: 500, Message: "create file: " + err.Error()}
	}
	defer out.Close()

	if _, err := io.Copy(out, io.LimitReader(r, maxUploadBytes+1)); err != nil {
		os.Remove(dst)
		return &KBError{Code: 500, Message: "write file: " + err.Error()}
	}
	if fi, _ := out.Stat(); fi != nil && fi.Size() > maxUploadBytes {
		out.Close()
		os.Remove(dst)
		return &KBError{Code: 400, Message: "file too large (max 100 MB)"}
	}
	return nil
}

// KBReindex walks kbRoot and re-indexes documents.
func KBReindex(kbRoot string, full bool) (*ReindexResult, error) {
	AppendQueryLog(kbRoot, "kb_reindex", "")
	db, err := OpenDB(kbRoot)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	defer db.Close()

	indexFn := IndexFiles
	if full {
		indexFn = IndexFilesFull
	}
	written, err := indexFn(db, kbRoot)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	return &ReindexResult{Written: written, Message: "index updated"}, nil
}

// KBLint runs deterministic health checks over wiki pages and cleans broken
// links from the links table. Side effect: deletes broken related_to/supports/
// contradicts rows and writes wiki/index/red_links.json.
func KBLint(kbRoot string) (*LintResult, error) {
	AppendQueryLog(kbRoot, "kb_lint", "")

	// File-level lint (missing fields, broken sources).
	warnings, err := Lint(kbRoot)
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	if warnings == nil {
		warnings = []LintWarning{}
	}

	// Broken-link cleanup (requires DB).
	db, err := OpenDB(kbRoot)
	if err != nil {
		// Non-fatal: return file warnings even if DB unavailable.
		return &LintResult{Warnings: warnings, Count: len(warnings)}, nil
	}
	defer db.Close()

	redLinks, blWarnings, brokenPaths, placeholders, err := cleanBrokenLinks(db)
	if err != nil {
		return nil, &KBError{Code: 500, Message: "clean broken links: " + err.Error()}
	}
	warnings = append(warnings, blWarnings...)
	if redLinks == nil {
		redLinks = []RedLink{}
	}

	// Write red_links.json (overwrite each run).
	if err := writeRedLinks(kbRoot, redLinks); err != nil {
		return nil, &KBError{Code: 500, Message: "write red_links.json: " + err.Error()}
	}

	return &LintResult{
		Warnings:     warnings,
		Count:        len(warnings),
		RedLinks:     redLinks,
		BrokenLinks:  brokenPaths,
		Placeholders: placeholders,
	}, nil
}

// writeRedLinks writes redLinks as JSON to wiki/index/red_links.json,
// sorted by count descending. Creates the directory if needed.
func writeRedLinks(kbRoot string, redLinks []RedLink) error {
	dir := filepath.Join(kbRoot, "wiki", "index")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(redLinks)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "red_links.json"), data, 0o644)
}

// FileInfo describes a single file in the raw/ directory.
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // relative path from raw/
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"` // Unix timestamp
}

// KBListFiles returns FileInfo for each regular file under kbRoot/raw/,
// recursively, skipping hidden files and the converted/ directory.
func KBListFiles(kbRoot string) ([]FileInfo, error) {
	rawDir := filepath.Join(kbRoot, "raw")
	if _, err := os.Stat(rawDir); os.IsNotExist(err) {
		return []FileInfo{}, nil
	}
	var files []FileInfo
	err := filepath.Walk(rawDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() && name == "converted" && filepath.Dir(path) == rawDir {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(rawDir, path)
		files = append(files, FileInfo{
			Name:    name,
			Path:    rel,
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
		return nil
	})
	if err != nil {
		return nil, &KBError{Code: 500, Message: err.Error()}
	}
	return files, nil
}
