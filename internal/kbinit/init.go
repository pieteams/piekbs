//go:build fts5

package kbinit

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pieteams/piekbs/internal/config"
	"github.com/pieteams/piekbs/internal/kb"
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

	schemaVer, err := ReadSchemaVersion()
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	cfg, err := config.Load(kbRoot)
	if err != nil {
		return fmt.Errorf("load config for schema_version: %w", err)
	}
	if cfg.SchemaVersion == "0" || cfg.SchemaVersion == "" || force {
		cfg.SchemaVersion = strconv.Itoa(schemaVer)
		if err := config.Save(kbRoot, cfg); err != nil {
			return fmt.Errorf("save schema_version: %w", err)
		}
	}

	return nil
}

// UpgradeSchema overwrites all schema files from the embedded FS and updates
// the schema_version in config.yaml to the current schema VERSION.
// Returns the list of updated file paths relative to schema/ and the previous schema version.
func UpgradeSchema(kbRoot string) ([]string, int, error) {
	// Read previous version before overwriting.
	oldCfg, err := config.Load(kbRoot)
	if err != nil {
		return nil, 0, fmt.Errorf("load config: %w", err)
	}
	oldVersion := oldCfg.SchemaVersionInt()

	if err := writeSchemaFiles(kbRoot, true); err != nil {
		return nil, oldVersion, err
	}

	newVersion, err := ReadSchemaVersion()
	if err != nil {
		return nil, oldVersion, fmt.Errorf("read schema version: %w", err)
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

	// Update config.
	oldCfg.SchemaVersion = strconv.Itoa(newVersion)
	if err := config.Save(kbRoot, oldCfg); err != nil {
		return nil, oldVersion, fmt.Errorf("save config: %w", err)
	}

	// Reindex.
	if _, err := kb.KBReindex(kbRoot, false); err != nil {
		return updated, oldVersion, fmt.Errorf("reindex: %w", err)
	}

	return updated, oldVersion, nil
}

// ReadSchemaVersion reads the schema VERSION from the embedded FS.
func ReadSchemaVersion() (int, error) {
	data, err := schemaFS.ReadFile("schema/VERSION")
	if err != nil {
		return 0, err
	}
	var v int
	_, err = fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &v)
	return v, err
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
