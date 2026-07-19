//go:build fts5

package kbinit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pieteams/piekbs/internal/config"
	"github.com/pieteams/piekbs/internal/version"
)

func TestInitWritesSchemaVersion(t *testing.T) {
	dir := t.TempDir()

	// Save original version and restore after test.
	origVer := version.Version
	version.Version = "1.2.3"
	t.Cleanup(func() { version.Version = origVer })

	if err := Init(dir, false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if cfg.SchemaVersion != "1.2.3" {
		t.Errorf("SchemaVersion = %q, want %q", cfg.SchemaVersion, "1.2.3")
	}
}

func TestInitDoesNotOverwriteExistingSchemaVersion(t *testing.T) {
	dir := t.TempDir()

	// Pre-create config with an existing schema_version.
	cfg := &config.Config{SchemaVersion: "old"}
	if err := config.Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	version.Version = "9.9.9"
	if err := Init(dir, false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	got, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if got.SchemaVersion != "old" {
		t.Errorf("SchemaVersion = %q, want %q (should not overwrite)", got.SchemaVersion, "old")
	}
}

func TestUpgradeSchema(t *testing.T) {
	dir := t.TempDir()

	origVer := version.Version
	version.Version = "2.0.0"
	t.Cleanup(func() { version.Version = origVer })

	// First init to set up the KB.
	if err := Init(dir, false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Change version and upgrade.
	version.Version = "3.0.0"
	updated, oldVer, err := UpgradeSchema(dir)
	if err != nil {
		t.Fatalf("UpgradeSchema: %v", err)
	}
	if oldVer != "2.0.0" {
		t.Errorf("oldVersion = %q, want %q", oldVer, "2.0.0")
	}

	if len(updated) == 0 {
		t.Fatal("UpgradeSchema returned no updated files")
	}

	// Verify config was updated.
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if cfg.SchemaVersion != "3.0.0" {
		t.Errorf("SchemaVersion = %q, want %q", cfg.SchemaVersion, "3.0.0")
	}

	// Verify at least one schema file exists on disk.
	for _, rel := range updated {
		full := filepath.Join(dir, "schema", rel)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("updated file %s not found on disk: %v", rel, err)
		}
	}
}
