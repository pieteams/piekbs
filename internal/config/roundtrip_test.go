//go:build fts5

package config

import "testing"

// TestConfigValueRoundTrip guards the Save/parse symmetry. Save writes values
// with %q (Go-quoted), so values containing backslashes or double-quotes must
// survive a save→load cycle. Before the fix, splitKV only stripped the outer
// quotes without unescaping, corrupting tokens/api keys with special chars.
func TestConfigValueRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Distill.Token = `abc\def"ghi`
	cfg.Distill.BaseURL = "https://api.example.com/v1"
	cfg.Distill.Model = "gpt-4o"
	cfg.Server.APIKey = `key"with\slash`

	if err := Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if got.Distill.Token != cfg.Distill.Token {
		t.Errorf("token round-trip: got %q want %q", got.Distill.Token, cfg.Distill.Token)
	}
	if got.Server.APIKey != cfg.Server.APIKey {
		t.Errorf("api_key round-trip: got %q want %q", got.Server.APIKey, cfg.Server.APIKey)
	}
	if got.Distill.BaseURL != cfg.Distill.BaseURL {
		t.Errorf("base_url round-trip: got %q want %q", got.Distill.BaseURL, cfg.Distill.BaseURL)
	}
}
