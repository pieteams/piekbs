package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
	if cfg.Embedding.IdleTimeout != 15*time.Minute {
		t.Errorf("idle_timeout = %v, want 15m0s", cfg.Embedding.IdleTimeout)
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WIKILOOP_API_KEY", "envkey")
	t.Setenv("WIKILOOP_PORT", "7777")

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
}
