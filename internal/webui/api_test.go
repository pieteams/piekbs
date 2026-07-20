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

	"github.com/pieteams/piekbs/internal/config"
	"github.com/pieteams/piekbs/internal/kb"
	"github.com/pieteams/piekbs/internal/larkimport"
)

func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "index"), 0o755); err != nil {
		t.Fatal(err)
	}
	return NewServer(dir, 0), dir
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
	// old "zh" is migrated to "zh-CN" by Load()
	if ui["language"] != "zh-CN" {
		t.Errorf("expected zh-CN (migrated from zh), got %v", ui["language"])
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

	server := NewServer(kbRoot, 0)
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
	server := NewServer(t.TempDir(), 0)
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

func TestDistillOutdated_DevVersion_ReturnsZero(t *testing.T) {
	s, dir := newTestServer(t)
	db, err := kb.GlobalDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(kb.CloseGlobalDB)

	// Insert a wiki source-note (would be outdated for non-zero schema versions).
	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp)
		VALUES ('wiki/note.md', 'wiki/note.md', 'wiki', 'source-note', 'Note', '', 'body', 'h1', 1, 3, 0)`)

	// No config.yaml → schema_version defaults to 0 → FindOutdatedNotes returns nil.
	req := httptest.NewRequest(http.MethodGet, "/api/distill/outdated", nil)
	w := httptest.NewRecorder()
	s.handleDistillOutdated(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["count"].(float64) != 0 {
		t.Errorf("expected count 0 for schema_version 0, got %v", resp["count"])
	}
}

func TestDistillOutdated_ReturnsOutdatedCount(t *testing.T) {
	s, dir := newTestServer(t)
	db, err := kb.GlobalDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(kb.CloseGlobalDB)

	// Insert a source-note with schema_version 0 (default, i.e. outdated).
	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
		VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Old', '', 'body', 'h1', 1, 3, 0, 0)`)

	// Write config with schema_version "2" so note (0) is outdated.
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("schema_version: \"2\"\n"), 0o644)

	req := httptest.NewRequest(http.MethodGet, "/api/distill/outdated", nil)
	w := httptest.NewRecorder()
	s.handleDistillOutdated(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["count"].(float64) != 1 {
		t.Errorf("expected count 1, got %v", resp["count"])
	}
}

func TestDistillRefreshOutdated_MethodNotAllowed(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/distill/refresh-outdated", nil)
	w := httptest.NewRecorder()
	s.handleDistillRefreshOutdated(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestDistillRefreshOutdated_NoOutdated(t *testing.T) {
	s, dir := newTestServer(t)
	db, err := kb.GlobalDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(kb.CloseGlobalDB)
	_ = db

	// No config.yaml → schema_version defaults to 0 → no outdated notes.
	req := httptest.NewRequest(http.MethodPost, "/api/distill/refresh-outdated", nil)
	w := httptest.NewRecorder()
	s.handleDistillRefreshOutdated(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["enqueued"].(float64) != 0 || resp["cleaned"].(float64) != 0 {
		t.Errorf("expected 0 enqueued/cleaned, got %v/%v", resp["enqueued"], resp["cleaned"])
	}
}

func TestDistillRefreshOutdated_EnqueuesAndDeletesWiki(t *testing.T) {
	s, dir := newTestServer(t)
	db, err := kb.GlobalDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(kb.CloseGlobalDB)

	// Create wiki file with source pointing to raw/.
	wikiDir := filepath.Join(dir, "wiki")
	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wikiContent := "---\ntitle: Test\nsources:\n  - raw/foo/bar.md\n---\nbody"
	if err := os.WriteFile(filepath.Join(wikiDir, "old.md"), []byte(wikiContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create the raw source file so it's not treated as orphan.
	rawDir := filepath.Join(dir, "raw", "foo")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rawDir, "bar.md"), []byte("raw content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Insert outdated DB record (schema_version 0, default).
	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
		VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Test', '', 'body', 'h1', 1, 3, 0, 0)`)

	// Write config with schema_version "2" so note (0) is outdated.
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("schema_version: \"2\"\n"), 0o644)

	req := httptest.NewRequest(http.MethodPost, "/api/distill/refresh-outdated", nil)
	w := httptest.NewRecorder()
	s.handleDistillRefreshOutdated(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["enqueued"].(float64) != 1 {
		t.Errorf("expected 1 enqueued, got %v", resp["enqueued"])
	}
	if resp["cleaned"].(float64) != 0 {
		t.Errorf("expected 0 cleaned, got %v", resp["cleaned"])
	}

	// Verify wiki file was deleted.
	if _, err := os.Stat(filepath.Join(wikiDir, "old.md")); !os.IsNotExist(err) {
		t.Error("wiki file should have been deleted")
	}

	// Verify raw file still exists.
	if _, err := os.Stat(filepath.Join(rawDir, "bar.md")); err != nil {
		t.Error("raw file should still exist")
	}

	// Verify distill_queue entry exists with correct path format (relative to raw/).
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM distill_queue WHERE path = 'foo/bar.md'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("expected 1 distill_queue entry for 'foo/bar.md', got %d", count)
	}
}

func TestDistillRefreshOutdated_CleansOrphanNoSource(t *testing.T) {
	s, dir := newTestServer(t)
	db, err := kb.GlobalDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(kb.CloseGlobalDB)

	// Create wiki file with no sources.
	wikiDir := filepath.Join(dir, "wiki")
	if err := os.MkdirAll(wikiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wikiContent := "---\ntitle: Orphan\n---\nbody"
	if err := os.WriteFile(filepath.Join(wikiDir, "orphan.md"), []byte(wikiContent), 0o644); err != nil {
		t.Fatal(err)
	}

	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
		VALUES ('wiki/orphan.md', 'wiki/orphan.md', 'wiki', 'source-note', 'Orphan', '', 'body', 'h1', 1, 3, 0, 0)`)

	// Write config with schema_version "2" so note (0) is outdated.
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("schema_version: \"2\"\n"), 0o644)

	req := httptest.NewRequest(http.MethodPost, "/api/distill/refresh-outdated", nil)
	w := httptest.NewRecorder()
	s.handleDistillRefreshOutdated(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["cleaned"].(float64) != 1 {
		t.Errorf("expected 1 cleaned, got %v", resp["cleaned"])
	}
	if resp["enqueued"].(float64) != 0 {
		t.Errorf("expected 0 enqueued, got %v", resp["enqueued"])
	}

	// Verify orphan wiki file was deleted.
	if _, err := os.Stat(filepath.Join(wikiDir, "orphan.md")); !os.IsNotExist(err) {
		t.Error("orphan wiki file should have been deleted")
	}
}

func TestDistillRefreshOutdated_ConflictWhenMutexHeld(t *testing.T) {
	s, _ := newTestServer(t)

	// Acquire the mutex externally to simulate an in-progress refresh.
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/api/distill/refresh-outdated", nil)
	w := httptest.NewRecorder()
	s.handleDistillRefreshOutdated(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["error"] != "refresh already in progress" {
		t.Errorf("expected conflict error message, got %v", resp["error"])
	}
}

func TestSchemaStatus_NoConfig_ReturnsZeroVersion(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/schema/status", nil)
	w := httptest.NewRecorder()
	s.handleSchemaStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	// No config.yaml → schema_version defaults to 0.
	if resp["schema_version"].(float64) != 0 {
		t.Errorf("expected schema_version 0, got %v", resp["schema_version"])
	}
	// embeddedSchemaVersion is 0 (test default) → not outdated.
	if resp["outdated"] != false {
		t.Errorf("expected outdated false, got %v", resp["outdated"])
	}
}

func TestSchemaStatus_OutdatedWhenConfigBehindEmbedded(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "index"), 0o755)
	// Create server with embedded version 3.
	s := NewServer(dir, 3)

	// Write config with schema_version "1" (behind embedded 3).
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("schema_version: \"1\"\n"), 0o644)

	req := httptest.NewRequest(http.MethodGet, "/api/schema/status", nil)
	w := httptest.NewRecorder()
	s.handleSchemaStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["schema_version"].(float64) != 1 {
		t.Errorf("expected schema_version 1, got %v", resp["schema_version"])
	}
	if resp["outdated"] != true {
		t.Errorf("expected outdated true when config < embedded, got %v", resp["outdated"])
	}
}

func TestSchemaStatus_NotOutdatedWhenConfigMatchesEmbedded(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "index"), 0o755)
	s := NewServer(dir, 2)

	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("schema_version: \"2\"\n"), 0o644)

	req := httptest.NewRequest(http.MethodGet, "/api/schema/status", nil)
	w := httptest.NewRecorder()
	s.handleSchemaStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp["outdated"] != false {
		t.Errorf("expected outdated false when config == embedded, got %v", resp["outdated"])
	}
}
