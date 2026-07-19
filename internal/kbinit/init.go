//go:build fts5

package kbinit

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pieteams/piekbs/internal/config"
	"github.com/pieteams/piekbs/internal/kb"
	"github.com/pieteams/piekbs/internal/version"
)

//go:embed schema/*
var schemaFS embed.FS

// Init creates the standard KB directory structure under kbRoot and
// populates schema/ with the bundled authoring rules and templates.
// Existing files are not overwritten unless force is true.
func Init(kbRoot string, force bool) error {
	// Create standard directories.
	for _, dir := range []string{"raw", "wiki/source-notes", "wiki/concepts", "wiki/comparisons", "wiki/decisions", "schema", "index", "models"} {
		if err := os.MkdirAll(filepath.Join(kbRoot, dir), 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}

	if err := writeSchemaFiles(kbRoot, force); err != nil {
		return err
	}

	// Write current PieKBS version as schema_version.
	cfg, err := config.Load(kbRoot)
	if err != nil {
		return fmt.Errorf("load config for schema_version: %w", err)
	}
	if cfg.SchemaVersion == "" {
		cfg.SchemaVersion = version.Version
		if err := config.Save(kbRoot, cfg); err != nil {
			return fmt.Errorf("save schema_version: %w", err)
		}
	}

	return nil
}

// UpgradeSchema overwrites all schema files from the embedded FS and updates
// the schema_version in config.yaml to the current binary version.
// Returns the list of updated file paths relative to schema/ and the previous schema version.
func UpgradeSchema(kbRoot string) ([]string, string, error) {
	// Read previous version before overwriting.
	oldCfg, err := config.Load(kbRoot)
	if err != nil {
		return nil, "", fmt.Errorf("load config: %w", err)
	}
	oldVersion := oldCfg.SchemaVersion

	if err := writeSchemaFiles(kbRoot, true); err != nil {
		return nil, oldVersion, err
	}

	// Collect updated file list.
	schemaDir := filepath.Join(kbRoot, "schema")
	var updated []string
	_ = fs.WalkDir(schemaFS, "schema", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(schemaDir, filepath.Join(kbRoot, path))
		updated = append(updated, rel)
		return nil
	})

	// Update schema_version in config.
	oldCfg.SchemaVersion = version.Version
	if err := config.Save(kbRoot, oldCfg); err != nil {
		return nil, oldVersion, fmt.Errorf("save config: %w", err)
	}

	// Reindex so schema files are searchable via FTS.
	if _, err := kb.KBReindex(kbRoot, false); err != nil {
		return updated, oldVersion, fmt.Errorf("reindex after upgrade: %w", err)
	}

	return updated, oldVersion, nil
}

// writeSchemaFiles copies all embedded schema files into kbRoot/schema/.
// If force is false, existing files are skipped.
func writeSchemaFiles(kbRoot string, force bool) error {
	return fs.WalkDir(schemaFS, "schema", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		dest := filepath.Join(kbRoot, path)
		if !force {
			if _, err := os.Stat(dest); err == nil {
				return nil // already exists, skip
			}
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		data, err := schemaFS.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dest, data, 0o644)
	})
}
