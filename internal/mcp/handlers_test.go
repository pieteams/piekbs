//go:build fts5

package mcp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jasen215/wikiloop/internal/kb"
)

// makeTempKB creates a temporary KB root with wiki/ and raw/ subdirectories.
func makeTempKB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"wiki", "raw"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	return dir
}

func TestHandleKBStatus_NoDB(t *testing.T) {
	dir := makeTempKB(t)
	result := handleKBStatus(dir)

	docs, ok := result["documents"].(int)
	if !ok {
		t.Fatalf("documents field missing or wrong type: %v", result["documents"])
	}
	if docs != 0 {
		t.Errorf("expected 0 documents, got %d", docs)
	}
}

func TestHandleKBStatus_WithDB(t *testing.T) {
	dir := makeTempKB(t)

	// Write a test file and index it.
	content := "# Test Doc\n\nThis is a test document."
	if err := os.WriteFile(filepath.Join(dir, "wiki", "test.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	db, err := kb.OpenDB(dir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := kb.IndexFiles(db, dir); err != nil {
		t.Fatalf("index files: %v", err)
	}

	result := handleKBStatus(dir)

	docs, ok := result["documents"].(int)
	if !ok {
		t.Fatalf("documents field missing or wrong type: %v", result["documents"])
	}
	if docs < 1 {
		t.Errorf("expected at least 1 document, got %d", docs)
	}
}

func TestHandleKBSearch(t *testing.T) {
	dir := makeTempKB(t)

	content := "# Search Target\n\nContent about searching and finding things."
	if err := os.WriteFile(filepath.Join(dir, "wiki", "search.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	db, err := kb.OpenDB(dir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := kb.IndexFiles(db, dir); err != nil {
		t.Fatalf("index files: %v", err)
	}

	result := handleKBSearch(dir, "search", nil, nil, 10)

	if errMsg, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error: %v", errMsg)
	}

	items, ok := result["results"].([]kb.SearchResult)
	if !ok {
		t.Fatalf("results field wrong type: %T", result["results"])
	}
	if len(items) == 0 {
		t.Error("expected at least one search result")
	}
}

func TestHandleKBReindex(t *testing.T) {
	dir := makeTempKB(t)

	content := "# Reindex Test\n\nContent for reindex test."
	if err := os.WriteFile(filepath.Join(dir, "wiki", "reindex.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result := handleKBReindex(dir, false)

	if errMsg, hasErr := result["error"]; hasErr {
		t.Fatalf("unexpected error: %v", errMsg)
	}

	msg, ok := result["message"].(string)
	if !ok {
		t.Fatalf("message field missing or wrong type: %v", result["message"])
	}
	if msg != "index updated" {
		t.Errorf("expected message 'index updated', got %q", msg)
	}
}

func TestWithAPIKey_NoKey_AllowsAll(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := withAPIKey("", inner)

	for _, hdr := range []string{"", "anything"} {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		if hdr != "" {
			req.Header.Set("x-api-key", hdr)
		}
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("empty key config: header=%q want 200 got %d", hdr, rr.Code)
		}
	}
}

func TestWithAPIKey_WithKey_RejectsWrongKey(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := withAPIKey("secret", inner)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("x-api-key", "wrong")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("wrong key: want 401 got %d", rr.Code)
	}
}

func TestWithAPIKey_WithKey_AllowsCorrectKey(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := withAPIKey("secret", inner)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("x-api-key", "secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("correct key: want 200 got %d", rr.Code)
	}
}

func TestWithAPIKey_WithKey_RejectsMissingKey(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := withAPIKey("secret", inner)

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("missing key: want 401 got %d", rr.Code)
	}
}
