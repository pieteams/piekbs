//go:build fts5

package embed

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// FindOrtLib returns the path to libonnxruntime dynamic library.
// Search order:
//  1. configPath — explicit path from config.yaml runtime.ort_lib
//  2. WIKILOOP_ORT_LIB env var
//  3. Bundled next to the binary (self-contained distribution):
//     macOS .app: Contents/Frameworks/
//     Linux:      same directory as the binary
//  4. Homebrew (/opt/homebrew/lib, /usr/local/lib)
//  5. System paths (/usr/lib)
//  6. kbRoot/ort/lib/ (manual download)
//
// Returns an error with install instructions if not found.
func FindOrtLib(configPath, kbRoot string) (string, error) {
	// 1. config.yaml explicit path.
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		return "", fmt.Errorf("runtime.ort_lib=%q: file not found", configPath)
	}

	// 2. Env var.
	if v := os.Getenv("WIKILOOP_ORT_LIB"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v, nil
		}
		return "", fmt.Errorf("WIKILOOP_ORT_LIB=%q: file not found", v)
	}

	libName := ortLibName()

	var candidates []string

	// 3. Bundled with the binary (self-contained distribution).
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			// macOS .app: Contents/MacOS/../Frameworks/
			filepath.Join(exeDir, "..", "Frameworks", libName),
			// Linux / standalone binary: same dir
			filepath.Join(exeDir, libName),
		)
	}

	candidates = append(candidates,
		// 4. Homebrew arm64
		filepath.Join("/opt/homebrew/lib", libName),
		"/opt/homebrew/lib/libonnxruntime.dylib",
		// Homebrew x86_64
		filepath.Join("/usr/local/lib", libName),
		"/usr/local/lib/libonnxruntime.dylib",
		// 5. System
		"/usr/lib/"+libName,
		// 6. Manual download next to KB
		filepath.Join(kbRoot, "ort", "lib", libName),
		filepath.Join(kbRoot, "ort", "lib", "libonnxruntime.dylib"),
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf(
		"libonnxruntime not found\n" +
			"The library should be bundled with the release package.\n" +
			"If running from source: brew install onnxruntime\n" +
			"Or set in config.yaml:\n" +
			"  runtime:\n" +
			"    ort_lib: /path/to/libonnxruntime.dylib",
	)
}

func ortLibName() string {
	switch runtime.GOOS {
	case "darwin":
		return "libonnxruntime.dylib"
	case "linux":
		return "libonnxruntime.so"
	default:
		return "onnxruntime.dll"
	}
}

