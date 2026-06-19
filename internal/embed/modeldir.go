//go:build fts5

package embed

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FindModelDir returns the directory containing model.onnx and tokenizer.json.
// Search order:
//  1. configModelDir — explicit path from config (e.g. kbRoot/models),
//     also scans one level of subdirectories (e.g. kbRoot/models/bge-small-zh/)
//  2. Source tree: internal/embed/models/ (dev mode — place model here locally,
//     the directory is git-ignored so model files are not committed)
//
// Models are NOT bundled inside the binary or .app. Download the model archive
// from GitHub releases and extract it into the KB directory:
//   <WIKILOOP_KB>/models/bge-small-zh/
//
// Returns empty string if not found.
func FindModelDir(configModelDir string) string {
	// 1. Explicit config path.
	//    a) The path itself is a model dir (e.g. kbRoot/models/bge-small-zh).
	//    b) The path is a parent directory — scan one level of subdirs.
	if configModelDir != "" {
		if isModelDir(configModelDir) {
			return configModelDir
		}
		// Scan immediate subdirectories (e.g. kbRoot/models/bge-small-zh/).
		if found := findModelInSubdirs(configModelDir); found != "" {
			return found
		}
	}

	// 2. Source tree (dev mode — models checked into the repo for development).
	devPath := filepath.Join("internal", "embed", "models", "bge-small-zh")
	if isModelDir(devPath) {
		return devPath
	}

	return ""
}

// findModelInSubdirs scans one level of subdirectories under dir for a model dir.
func findModelInSubdirs(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if isModelDir(p) {
			return p
		}
	}
	return ""
}

func isModelDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "model.onnx"))
	return err == nil
}

// ModelName reads the model name from meta.json in modelDir.
// Returns "unknown" if the file is absent or malformed.
func ModelName(modelDir string) string {
	b, err := os.ReadFile(filepath.Join(modelDir, "meta.json"))
	if err != nil {
		return "unknown"
	}
	var m struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(b, &m); err != nil || m.Model == "" {
		return "unknown"
	}
	return m.Model
}
