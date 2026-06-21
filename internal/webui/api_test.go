//go:build fts5

package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasen215/wikiloop/internal/config"
	"github.com/jasen215/wikiloop/internal/larkimport"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "index"), 0o755); err != nil {
		t.Fatal(err)
	}
	return NewServer(dir), dir
}

func TestSettingsGetIncludesLanguage(t *testing.T) {
	s, dir := newTestServer(t)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("ui:\n  language: \"zh\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	w := httptest.NewRecorder()
	s.handleSettings(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	ui, ok := resp["ui"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'ui' key")
	}
	if ui["language"] != "zh" {
		t.Errorf("expected zh, got %v", ui["language"])
	}
}

func TestSettingsPutLanguage(t *testing.T) {
	s, dir := newTestServer(t)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("ui:\n  language: \"zh\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/settings", strings.NewReader(`{"ui":{"language":"en"}}`))
	w := httptest.NewRecorder()
	s.handleSettings(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true {
		t.Fatalf("PUT failed: %v", resp)
	}

	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"en"`) {
		t.Errorf("config.yaml should contain en, got: %s", data)
	}
}

func TestSettingsDoesNotExposeOrClearSavedToken(t *testing.T) {
	kbRoot := t.TempDir()
	cfg := &config.Config{}
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 8766
	cfg.Distill.BaseURL = "https://api.deepseek.com"
	cfg.Distill.Token = "secret-token"
	cfg.Distill.Model = "deepseek-chat"
	cfg.Distill.APIType = "openai"
	if err := config.Save(kbRoot, cfg); err != nil {
		t.Fatal(err)
	}

	server := NewServer(kbRoot)
	getReq := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	getRec := httptest.NewRecorder()
	server.handleSettings(getRec, getReq)

	if strings.Contains(getRec.Body.String(), "secret-token") {
		t.Fatal("GET /api/settings exposed the saved token")
	}
	var getBody struct {
		Distill struct {
			TokenConfigured bool `json:"token_configured"`
		} `json:"distill"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &getBody); err != nil {
		t.Fatal(err)
	}
	if !getBody.Distill.TokenConfigured {
		t.Fatal("GET /api/settings did not report that a token is configured")
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/settings",
		strings.NewReader(`{"distill":{"base_url":"https://api.deepseek.com/v1","model":"deepseek-chat","api_type":"openai"}}`))
	putRec := httptest.NewRecorder()
	server.handleSettings(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("PUT /api/settings status = %d, body = %s", putRec.Code, putRec.Body.String())
	}

	saved, err := config.Load(kbRoot)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Distill.Token != "secret-token" {
		t.Fatalf("saved token = %q, want preserved token", saved.Distill.Token)
	}
}

func TestImportLarkAPI(t *testing.T) {
	server := NewServer(t.TempDir())
	server.importLark = func(_ context.Context, _, url, name string) (*larkimport.Result, error) {
		if url != "https://example.larkoffice.com/wiki/abc" || name != "" {
			t.Fatalf("unexpected import request: url=%q name=%q", url, name)
		}
		return &larkimport.Result{
			DocumentPath:      "raw/lark/test/document.md",
			TablePaths:        []string{"raw/lark/test/table.snapshot.tsv"},
			TableRows:         []int{123},
			DatasetPath:       "raw/lark/test/records-deduplicated.txt",
			TotalRows:         123,
			UniqueRows:        120,
			DuplicatesRemoved: 3,
		}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/import-lark",
		strings.NewReader(`{"url":"https://example.larkoffice.com/wiki/abc"}`))
	rec := httptest.NewRecorder()
	server.handleImportLark(rec, req)

	var body struct {
		OK                bool  `json:"ok"`
		TableRows         []int `json:"table_rows"`
		UniqueRows        int   `json:"unique_rows"`
		DuplicatesRemoved int   `json:"duplicates_removed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK || len(body.TableRows) != 1 || body.TableRows[0] != 123 ||
		body.UniqueRows != 120 || body.DuplicatesRemoved != 3 {
		t.Fatalf("unexpected response: %s", rec.Body.String())
	}
}
