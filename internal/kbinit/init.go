//go:build fts5

package kbinit

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

	// Copy bundled schema files into kbRoot/schema/.
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
