//go:build fts5

package webui

import (
	"context"
	"embed"
	"net/http"

	"github.com/jasen215/wikiloop/internal/larkimport"
)

//go:embed static/*
var staticFiles embed.FS

// Server serves the Web UI and REST API.
type Server struct {
	kbRoot     string
	importLark func(context.Context, string, string, string) (*larkimport.Result, error)
}

// NewServer creates a Server for the given knowledge-base root directory.
func NewServer(kbRoot string) *Server {
	return &Server{
		kbRoot: kbRoot,
		importLark: func(ctx context.Context, kbRoot, url, name string) (*larkimport.Result, error) {
			return larkimport.Import(ctx, kbRoot, url, name, nil)
		},
	}
}

// Handler returns an http.Handler with all routes registered.
//
// Routes:
//
//	GET  /              → index.html
//	GET  /files         → files.html
//	GET  /settings      → settings.html
//	GET  /static/…      → embedded static assets
//	GET  /api/status    → JSON KB stats
//	GET  /api/search    → JSON search results (?q=…&layer=…&limit=…)
//	GET  /api/files     → JSON raw/ file list
//	GET  /api/settings  → JSON config (stub)
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Static assets served from embedded FS under /static/.
	mux.Handle("/static/", http.FileServer(http.FS(staticFiles)))

	// HTML pages — serve files directly from embed.FS.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		serveEmbedded(w, r, "static/index.html")
	})
	mux.HandleFunc("/files", func(w http.ResponseWriter, r *http.Request) {
		serveEmbedded(w, r, "static/files.html")
	})
	mux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		serveEmbedded(w, r, "static/settings.html")
	})

	// REST API.
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/reindex", s.handleReindex)
	mux.HandleFunc("/api/lint", s.handleLint)
	mux.HandleFunc("/api/red-links", s.handleRedLinks)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/import-lark", s.handleImportLark)
	mux.HandleFunc("/api/settings", s.handleSettings)

	return mux
}

// serveEmbedded writes an embedded file to the response with appropriate headers.
func serveEmbedded(w http.ResponseWriter, _ *http.Request, path string) {
	data, err := staticFiles.ReadFile(path)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data) //nolint:errcheck
}
