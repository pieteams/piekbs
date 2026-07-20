//go:build fts5

package kbinit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pieteams/piekbs/internal/config"
)

// schemaVersionInt is the value embedded in schema/VERSION.
// We read it once to keep tests in sync with the embedded file.
var schemaVersionInt int

func init() {
	v, err := ReadSchemaVersion()
	if err != nil {
		panic(fmt.Sprintf("ReadSchemaVersion: %v", err))
	}
	schemaVersionInt = v
}

func TestInitWritesSchemaVersion(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir, false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	want := fmt.Sprintf("%d", schemaVersionInt)
	if cfg.SchemaVersion != want {
		t.Errorf("SchemaVersion = %q, want %q", cfg.SchemaVersion, want)
	}
}

func TestInitDoesNotOverwriteExistingSchemaVersion(t *testing.T) {
	dir := t.TempDir()

	// Pre-create config with an existing non-zero schema_version.
	cfg := &config.Config{SchemaVersion: "42"}
	if err := config.Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Init(dir, false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	got, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	if got.SchemaVersion != "42" {
		t.Errorf("SchemaVersion = %q, want %q (should not overwrite)", got.SchemaVersion, "42")
	}
}

func TestInitOverwritesZeroVersion(t *testing.T) {
	dir := t.TempDir()

	// Pre-create config with schema_version "0" (treated as unset).
	cfg := &config.Config{SchemaVersion: "0"}
	if err := config.Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Init(dir, false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	got, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	want := fmt.Sprintf("%d", schemaVersionInt)
	if got.SchemaVersion != want {
		t.Errorf("SchemaVersion = %q, want %q (should overwrite zero)", got.SchemaVersion, want)
	}
}

func TestUpgradeSchema(t *testing.T) {
	dir := t.TempDir()

	// First init to set up the KB.
	if err := Init(dir, false); err != nil {
		t.Fatalf("Init: %v", err)
	}

	updated, oldVer, err := UpgradeSchema(dir)
	if err != nil {
		t.Fatalf("UpgradeSchema: %v", err)
	}
	if oldVer != schemaVersionInt {
		t.Errorf("oldVersion = %d, want %d", oldVer, schemaVersionInt)
	}

	if len(updated) == 0 {
		t.Fatal("UpgradeSchema returned no updated files")
	}

	// Verify config was updated.
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load config: %v", err)
	}
	want := fmt.Sprintf("%d", schemaVersionInt)
	if cfg.SchemaVersion != want {
		t.Errorf("SchemaVersion = %q, want %q", cfg.SchemaVersion, want)
	}

	// Verify at least one schema file exists on disk.
	for _, rel := range updated {
		full := filepath.Join(dir, "schema", rel)
		if _, err := os.Stat(full); err != nil {
			t.Errorf("updated file %s not found on disk: %v", rel, err)
		}
	}
}
