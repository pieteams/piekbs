//go:build fts5

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var validLangSet = map[string]bool{
	"en": true, "zh-CN": true, "zh-TW": true,
	"ru": true, "de": true, "fr": true, "es": true, "ko": true,
}

func TestLoadSetsDefaultLanguage(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UI.Language == "" {
		t.Error("UI.Language should be set after Load, got empty string")
	}
	if !validLangSet[cfg.UI.Language] {
		t.Errorf("UI.Language must be one of 8 valid codes, got %q", cfg.UI.Language)
	}
	// Should have been written to config.yaml
	data, err := os.ReadFile(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not written: %v", err)
	}
	if !contains(string(data), "language:") {
		t.Error("config.yaml should contain language: field")
	}
}

func TestLoadReadsExistingLanguage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("ui:\n  language: \"en\"\n"), 0o644)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.UI.Language != "en" {
		t.Errorf("expected en, got %q", cfg.UI.Language)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestLoadConfig_Defaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("default host = %q, want 127.0.0.1", cfg.Server.Host)
	}
	if cfg.Server.Port != 8766 {
		t.Errorf("default port = %d, want 8766", cfg.Server.Port)
	}
	if cfg.Embedding.IdleTimeout != 5*time.Minute {
		t.Errorf("default idle_timeout = %v, want 5m0s", cfg.Embedding.IdleTimeout)
	}
}

func TestLoadConfig_FromYAML(t *testing.T) {
	dir := t.TempDir()
	yaml := `server:
  host: "0.0.0.0"
  port: 9999
  api_key: "secret"
distill:
  base_url: "http://llm.local"
  token: "tok123"
  model: "claude-3"
  api_type: "anthropic"
embedding:
  idle_timeout: "15m"
`
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0644)
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("host = %q, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("port = %d, want 9999", cfg.Server.Port)
	}
	if cfg.Server.APIKey != "secret" {
		t.Errorf("api_key = %q, want secret", cfg.Server.APIKey)
	}
	if cfg.Distill.BaseURL != "http://llm.local" {
		t.Errorf("base_url = %q", cfg.Distill.BaseURL)
	}
	if cfg.Distill.APIType != "anthropic" {
		t.Errorf("api_type = %q, want anthropic", cfg.Distill.APIType)
	}
	if cfg.Embedding.IdleTimeout != 15*time.Minute {
		t.Errorf("idle_timeout = %v, want 15m0s", cfg.Embedding.IdleTimeout)
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WIKILOOP_API_KEY", "envkey")
	t.Setenv("WIKILOOP_PORT", "7777")
	t.Setenv("WIKILOOP_DISTILL_TOKEN", "deepseek-secret")
	t.Setenv("WIKILOOP_DISTILL_API_TYPE", "openai")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.APIKey != "envkey" {
		t.Errorf("api_key = %q, want envkey", cfg.Server.APIKey)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("port = %d, want 7777", cfg.Server.Port)
	}
	if cfg.Distill.Token != "deepseek-secret" {
		t.Errorf("distill token env override was not applied")
	}
	if cfg.Distill.APIType != "openai" {
		t.Errorf("distill api_type = %q, want openai", cfg.Distill.APIType)
	}
}

func TestSaveConfigUsesPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{}
	cfg.Distill.Token = "secret"

	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config.yaml permissions = %o, want 600", got)
	}
}

func TestSetUIField_ValidLanguages(t *testing.T) {
	validLangs := []string{"en", "zh-CN", "zh-TW", "ru", "de", "fr", "es", "ko"}
	for _, lang := range validLangs {
		cfg := &Config{}
		setUIField(cfg, "language", lang)
		if cfg.UI.Language != lang {
			t.Errorf("setUIField language=%q: got %q, want %q", lang, cfg.UI.Language, lang)
		}
	}
}

func TestSetUIField_InvalidLanguage(t *testing.T) {
	cfg := &Config{UI: UIConfig{Language: "en"}}
	setUIField(cfg, "language", "ja") // 不支持的语言
	if cfg.UI.Language != "en" {
		t.Errorf("invalid language should be rejected, got %q", cfg.UI.Language)
	}
}

func TestSetUIField_RejectsLegacyZh(t *testing.T) {
	cfg := &Config{UI: UIConfig{Language: "en"}}
	setUIField(cfg, "language", "zh")
	if cfg.UI.Language != "en" {
		t.Errorf("legacy 'zh' should be rejected by setUIField, got %q", cfg.UI.Language)
	}
}

func TestLoad_MigratesOldZh(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("ui:\n  language: \"zh\"\n"), 0o644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UI.Language != "zh-CN" {
		t.Errorf("old 'zh' should migrate to 'zh-CN', got %q", cfg.UI.Language)
	}
	data, _ := os.ReadFile(cfgPath)
	if !contains(string(data), "zh-CN") {
		t.Errorf("config.yaml should be updated to zh-CN, got:\n%s", data)
	}
}
