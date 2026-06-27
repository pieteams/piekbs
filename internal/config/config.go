package config

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SettingsRequest is the JSON shape accepted by PUT /api/settings.
// Pointer fields: nil = do not modify; empty string = clear the value.
type SettingsRequest struct {
	Distill struct {
		BaseURL *string `json:"base_url"`
		Token   *string `json:"token"`
		Model   *string `json:"model"`
		APIType *string `json:"api_type"`
	} `json:"distill"`
	Embedding struct {
		IdleTimeout *string `json:"idle_timeout"`
	} `json:"embedding"`
	UI struct {
		Language *string `json:"language"`
	} `json:"ui"`
}

type Config struct {
	Server    ServerConfig
	Distill   DistillConfig
	Embedding EmbeddingConfig
	Runtime   RuntimeConfig
	UI        UIConfig
}

type UIConfig struct {
	Language string `yaml:"language"` // one of: en, zh-CN, zh-TW, ru, de, fr, es, ko
}

type RuntimeConfig struct {
	OrtLib    string // path to libonnxruntime dylib (optional, auto-detected if empty)
	SqliteVec string // Deprecated: sqlite-vec replaced by chromem-go; field ignored
}

type ServerConfig struct {
	Host   string
	Port   int
	APIKey string
}

type DistillConfig struct {
	BaseURL string
	Token   string
	Model   string
	APIType string // "openai" (default) or "anthropic"
	Workers int    // concurrent distill workers (default 3)
}

// IsConfigured returns true when all three LLM fields are non-empty.
func (c DistillConfig) IsConfigured() bool {
	return c.BaseURL != "" && c.Token != "" && c.Model != ""
}

type EmbeddingConfig struct {
	IdleTimeout time.Duration
}

func detectSystemLanguage() string {
	for _, env := range []string{"LANG", "LC_ALL", "LANGUAGE"} {
		v := os.Getenv(env)
		if strings.HasPrefix(v, "zh_TW") || strings.HasPrefix(v, "zh_HK") {
			return "zh-TW"
		}
		if strings.HasPrefix(v, "zh") {
			return "zh-CN"
		}
		if strings.HasPrefix(v, "ru") {
			return "ru"
		}
		if strings.HasPrefix(v, "de") {
			return "de"
		}
		if strings.HasPrefix(v, "fr") {
			return "fr"
		}
		if strings.HasPrefix(v, "es") {
			return "es"
		}
		if strings.HasPrefix(v, "ko") {
			return "ko"
		}
	}
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("defaults", "read", "NSGlobalDomain", "AppleLocale").Output()
		if err == nil {
			locale := strings.TrimSpace(string(out))
			if strings.HasPrefix(locale, "zh_TW") || strings.HasPrefix(locale, "zh_HK") {
				return "zh-TW"
			}
			if strings.HasPrefix(locale, "zh") {
				return "zh-CN"
			}
			if strings.HasPrefix(locale, "ru") {
				return "ru"
			}
			if strings.HasPrefix(locale, "de") {
				return "de"
			}
			if strings.HasPrefix(locale, "fr") {
				return "fr"
			}
			if strings.HasPrefix(locale, "es") {
				return "es"
			}
			if strings.HasPrefix(locale, "ko") {
				return "ko"
			}
		}
	}
	return "en"
}

func Load(kbRoot string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8766,
		},
		Distill: DistillConfig{
			Workers: 3,
		},
		Embedding: EmbeddingConfig{
			IdleTimeout: 5 * time.Minute,
		},
	}

	configPath := filepath.Join(kbRoot, "config.yaml")
	var hasLegacyZh bool
	if rawData, err := os.ReadFile(configPath); err == nil {
		hasLegacyZh = bytes.Contains(rawData, []byte(`language: "zh"`)) ||
			bytes.Contains(rawData, []byte("language: zh"))
		if err := parseYAML(configPath, cfg); err != nil {
			return nil, fmt.Errorf("parse config.yaml: %w", err)
		}
	}

	if v := os.Getenv("WIKILOOP_API_KEY"); v != "" {
		cfg.Server.APIKey = v
	}
	if v := os.Getenv("WIKILOOP_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("WIKILOOP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("WIKILOOP_DISTILL_BASE_URL"); v != "" {
		cfg.Distill.BaseURL = v
	}
	if v := os.Getenv("WIKILOOP_DISTILL_TOKEN"); v != "" {
		cfg.Distill.Token = v
	}
	if v := os.Getenv("WIKILOOP_DISTILL_MODEL"); v != "" {
		cfg.Distill.Model = v
	}
	if v := os.Getenv("WIKILOOP_DISTILL_API_TYPE"); v != "" {
		cfg.Distill.APIType = v
	}
	if v := os.Getenv("WIKILOOP_EMBEDDING_IDLE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Embedding.IdleTimeout = d
		}
	}
	if v := os.Getenv("WIKILOOP_ORT_LIB"); v != "" {
		cfg.Runtime.OrtLib = v
	}
	if v := os.Getenv("WIKILOOP_SQLITE_VEC"); v != "" {
		cfg.Runtime.SqliteVec = v
	}

	if cfg.UI.Language == "" {
		if hasLegacyZh {
			// Migrate legacy "zh" (rejected by setUIField) to "zh-CN"
			cfg.UI.Language = "zh-CN"
		} else {
			cfg.UI.Language = detectSystemLanguage()
		}
		_ = Save(kbRoot, cfg) // best-effort write; ignore error
	}

	return cfg, nil
}

func parseYAML(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " ")
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		indented := strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")
		trimmed := strings.TrimSpace(line)

		if indented && section != "" {
			key, val := splitKV(trimmed)
			switch section {
			case "server":
				setServerField(cfg, key, val)
			case "distill":
				setDistillField(cfg, key, val)
			case "embedding":
				setEmbeddingField(cfg, key, val)
			case "runtime":
				setRuntimeField(cfg, key, val)
			case "ui":
				setUIField(cfg, key, val)
			}
		} else {
			key, val := splitKV(trimmed)
			if val == "" {
				section = key
			} else {
				section = ""
			}
		}
	}
	return scanner.Err()
}

func splitKV(s string) (string, string) {
	before, after, found := strings.Cut(s, ":")
	if !found {
		return s, ""
	}
	key := strings.TrimSpace(before)
	val := strings.TrimSpace(after)
	// Strip outer quotes and return the interior verbatim. This handles
	// quoted values like `"https://api.example.com/v1"` which contain ':'
	// — the key is split at the first ':' but the value is the full quoted
	// string, so we only need to remove the surrounding quote characters.
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
	}
	return key, val
}

func setServerField(cfg *Config, key, val string) {
	switch key {
	case "host":
		cfg.Server.Host = val
	case "port":
		if p, err := strconv.Atoi(val); err == nil {
			cfg.Server.Port = p
		}
	case "api_key":
		cfg.Server.APIKey = val
	}
}

func setDistillField(cfg *Config, key, val string) {
	switch key {
	case "base_url":
		cfg.Distill.BaseURL = val
	case "token":
		cfg.Distill.Token = val
	case "model":
		cfg.Distill.Model = val
	case "api_type":
		cfg.Distill.APIType = val
	case "workers":
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			cfg.Distill.Workers = n
		}
	}
}

func setEmbeddingField(cfg *Config, key, val string) {
	switch key {
	case "idle_timeout":
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Embedding.IdleTimeout = d
		}
	}
}

func setRuntimeField(cfg *Config, key, val string) {
	switch key {
	case "ort_lib":
		cfg.Runtime.OrtLib = val
	case "sqlite_vec":
		// Deprecated: sqlite-vec replaced by chromem-go; value accepted but ignored.
	}
}

func setUIField(cfg *Config, key, val string) {
	switch key {
	case "language":
		valid := map[string]bool{
			"en": true, "zh-CN": true, "zh-TW": true,
			"ru": true, "de": true, "fr": true, "es": true, "ko": true,
		}
		if valid[val] {
			cfg.UI.Language = val
		}
	}
}

// Save writes cfg back to <kbRoot>/config.yaml.
// Only user-facing fields are written (runtime fields are env-var controlled).
// Values are written unquoted so parseYAML/splitKV can read them back correctly.
func Save(kbRoot string, cfg *Config) error {
	var b strings.Builder
	b.WriteString("server:\n")
	fmt.Fprintf(&b, "  host: %q\n", cfg.Server.Host)
	fmt.Fprintf(&b, "  port: %d\n", cfg.Server.Port)
	fmt.Fprintf(&b, "  api_key: %q\n", cfg.Server.APIKey)
	b.WriteString("\ndistill:\n")
	fmt.Fprintf(&b, "  base_url: %q\n", cfg.Distill.BaseURL)
	fmt.Fprintf(&b, "  token: %q\n", cfg.Distill.Token)
	fmt.Fprintf(&b, "  model: %q\n", cfg.Distill.Model)
	fmt.Fprintf(&b, "  api_type: %q\n", cfg.Distill.APIType)
	fmt.Fprintf(&b, "  workers: %d\n", cfg.Distill.Workers)
	b.WriteString("\nembedding:\n")
	fmt.Fprintf(&b, "  idle_timeout: %q\n", cfg.Embedding.IdleTimeout.String())
	b.WriteString("\nui:\n")
	fmt.Fprintf(&b, "  language: %q\n", cfg.UI.Language)

	path := filepath.Join(kbRoot, "config.yaml")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
