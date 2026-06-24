//go:build fts5

package webui

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jasen215/wikiloop/internal/config"
	"github.com/jasen215/wikiloop/internal/kb"
)

// handleStatus returns document/embedding counts, by-layer breakdown, and index file size.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	dbPath := filepath.Join(s.kbRoot, "index", "kb.sqlite")

	db, err := kb.OpenDB(s.kbRoot)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"documents":     0,
			"by_layer":      map[string]int{},
			"embeddings":    0,
			"index_path":    dbPath,
			"index_size_kb": int64(0),
		})
		return
	}
	defer db.Close()

	byLayer, total, _ := kb.LayerCounts(db)
	byKind, _ := kb.KindCounts(db)

	var indexSize int64
	if fi, err := os.Stat(dbPath); err == nil {
		indexSize = fi.Size()
	}

	writeJSON(w, map[string]interface{}{
		"documents":  total,
		"by_layer":   byLayer,
		"by_kind":    byKind,
		"index_path": dbPath,
		"index_size": indexSize,
	})
}

// handleSearch runs FTS search and returns results.
// Query params: q (required), layer (optional), limit (optional, default 10).
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, map[string]interface{}{"results": []kb.SearchResult{}})
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	var layer *string
	if l := r.URL.Query().Get("layer"); l != "" {
		layer = &l
	}

	db, err := kb.OpenDB(s.kbRoot)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	defer db.Close()

	results, graphPages, conflicts, err := kb.Search(db, s.kbRoot, q, layer, limit, nil)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}

	writeJSON(w, map[string]interface{}{
		"results":     results,
		"graph_pages": graphPages,
		"conflicts":   conflicts,
	})
}

// FileInfo describes a single file in the raw/ directory.
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"` // relative path from raw/ (e.g. "ebooks/Chip/xxx.md")
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"` // Unix timestamp
}

// handleFiles lists files in the raw/ subdirectory of kbRoot.
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	files, err := listRawFiles(s.kbRoot)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"files": files})
}

// handleUpload saves an uploaded file to raw/ under kbRoot.
// Accepts multipart/form-data with a "file" field.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, map[string]interface{}{"error": "parse form: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": "read file field: " + err.Error()})
		return
	}
	defer file.Close()

	filename := filepath.Base(header.Filename)
	if filename == "" || filename == "." {
		writeJSON(w, map[string]interface{}{"error": "invalid filename"})
		return
	}

	dst := filepath.Join(s.kbRoot, "raw", filename)
	const maxUploadBytes = 100 << 20 // 100 MB

	out, err := os.Create(dst)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": "create file: " + err.Error()})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, io.LimitReader(file, maxUploadBytes+1)); err != nil {
		os.Remove(dst)
		writeJSON(w, map[string]interface{}{"error": "write file: " + err.Error()})
		return
	}
	if fi, _ := out.Stat(); fi != nil && fi.Size() > maxUploadBytes {
		out.Close()
		os.Remove(dst)
		http.Error(w, "file too large (max 100 MB)", http.StatusRequestEntityTooLarge)
		return
	}

	writeJSON(w, map[string]interface{}{"ok": true, "filename": filename})
}

// handleImportLark imports a Lark/Feishu Wiki URL and expands embedded Base
// tables into searchable local text datasets.
func (s *Server) handleImportLark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]interface{}{"error": "invalid JSON: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		writeJSON(w, map[string]interface{}{"error": "Lark Wiki URL is required"})
		return
	}
	result, err := s.importLark(r.Context(), s.kbRoot, req.URL, req.Name)
	if err != nil {
		writeJSON(w, map[string]interface{}{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{
		"ok":                 true,
		"document_path":      result.DocumentPath,
		"table_paths":        result.TablePaths,
		"table_rows":         result.TableRows,
		"dataset_path":       result.DatasetPath,
		"total_rows":         result.TotalRows,
		"unique_rows":        result.UniqueRows,
		"duplicates_removed": result.DuplicatesRemoved,
	})
}

// handleSettings reads (GET) or writes (PUT) config.yaml.
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := config.Load(s.kbRoot)
		if err != nil {
			writeJSON(w, map[string]interface{}{"error": err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{
			"server": map[string]interface{}{
				"host": cfg.Server.Host,
				"port": cfg.Server.Port,
			},
			"distill": map[string]interface{}{
				"base_url":         cfg.Distill.BaseURL,
				"model":            cfg.Distill.Model,
				"api_type":         cfg.Distill.APIType,
				"token_configured": cfg.Distill.Token != "",
			},
			"embedding": map[string]interface{}{
				"idle_timeout": cfg.Embedding.IdleTimeout.String(),
			},
			"ui": map[string]interface{}{
				"language": cfg.UI.Language,
			},
		})

	case http.MethodPut:
		var req config.SettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]interface{}{"error": "invalid JSON: " + err.Error()})
			return
		}
		cfg, err := config.Load(s.kbRoot)
		if err != nil {
			writeJSON(w, map[string]interface{}{"error": err.Error()})
			return
		}
		if req.Distill.BaseURL != nil {
			cfg.Distill.BaseURL = *req.Distill.BaseURL
		}
		if req.Distill.Token != nil {
			cfg.Distill.Token = *req.Distill.Token
		}
		if req.Distill.Model != nil {
			cfg.Distill.Model = *req.Distill.Model
		}
		if req.Distill.APIType != nil {
			cfg.Distill.APIType = *req.Distill.APIType
		}
		if req.Embedding.IdleTimeout != nil {
			if d, err := time.ParseDuration(*req.Embedding.IdleTimeout); err == nil {
				cfg.Embedding.IdleTimeout = d
			}
		}
		if req.UI.Language != nil {
			if *req.UI.Language == "zh" || *req.UI.Language == "en" {
				cfg.UI.Language = *req.UI.Language
			}
		}
		if err := config.Save(s.kbRoot, cfg); err != nil {
			writeJSON(w, map[string]interface{}{"error": "save config: " + err.Error()})
			return
		}
		writeJSON(w, map[string]interface{}{"ok": true})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// writeJSON encodes v as JSON and writes it to w with Content-Type application/json.
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// listRawFiles returns FileInfo for each regular file under kbRoot/raw/,
// recursively, skipping hidden files and the converted/ directory.
// Returns an empty slice (not an error) when the directory doesn't exist.
func listRawFiles(kbRoot string) ([]FileInfo, error) {
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

		// Skip hidden files and directories.
		if strings.HasPrefix(name, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip the converted/ directory (generated output).
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
		return nil, err
	}
	return files, nil
}
