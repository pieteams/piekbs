//go:build fts5

// Package convert handles converting non-markdown files (PDF, HTML, DOCX, etc.)
// to markdown using pure Go parsers.
package convert

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// imgPlaceholderRe matches base64-encoded image placeholders in converted
// documents (e.g. embedded Excel sheets inside docx/pptx).
var imgPlaceholderRe = regexp.MustCompile(`!\[.*?\]\(data:image/[^)]+\)`)

// FindConvertibleFiles walks the raw/ directory under kbRoot and returns
// paths of files that need conversion (have a supported extension and
// no corresponding .md already in raw/converted/).
func FindConvertibleFiles(kbRoot string, registry *Registry) []string {
	rawDir := filepath.Join(kbRoot, "raw")
	convertedDir := filepath.Join(rawDir, "converted")

	var result []string

	_ = filepath.Walk(rawDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if path == convertedDir {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))

		// Skip already-text formats
		if ext == ".md" || ext == ".txt" || ext == ".rst" {
			return nil
		}

		// Only process formats supported by the registry
		if !registry.Supports(ext) {
			return nil
		}

		// Check if a converted version already exists
		rel, _ := filepath.Rel(rawDir, path)
		convertedPath := filepath.Join(convertedDir, strings.TrimSuffix(rel, filepath.Ext(rel))+".md")
		if _, err := os.Stat(convertedPath); err == nil {
			return nil
		}

		result = append(result, path)
		return nil
	})

	return result
}

// ConvertFile converts srcPath to markdown at destPath using the registry.
// Returns true on success, false on failure.
func ConvertFile(registry *Registry, srcPath, destPath string) bool {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "convert: mkdir failed for %s: %v\n", destPath, err)
		return false
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	parser, ok := registry.Get(ext)
	if !ok {
		fmt.Fprintf(os.Stderr, "convert: no parser for %s\n", ext)
		return false
	}

	text, err := parser.Extract(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "convert: failed to convert %s: %v\n", srcPath, err)
		return false
	}

	// Embedded Excel handling: DOCX/PPTX may contain embedded XLSX
	if ext == ".docx" || ext == ".pptx" {
		text = injectEmbeddedXlsx(registry, srcPath, text)
	}

	if err := os.WriteFile(destPath, []byte(text), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "convert: write failed for %s: %v\n", destPath, err)
		return false
	}
	return true
}

// Run finds convertible files under kbRoot and converts them using pure Go parsers.
// Returns the number of files converted. Failures are non-fatal.
func Run(kbRoot string) (int, error) {
	registry := NewRegistry()
	files := FindConvertibleFiles(kbRoot, registry)
	if len(files) == 0 {
		fmt.Println("convert: no files to convert")
		return 0, nil
	}

	rawDir := filepath.Join(kbRoot, "raw")
	convertedDir := filepath.Join(rawDir, "converted")
	converted := 0
	for _, src := range files {
		rel, _ := filepath.Rel(rawDir, src)
		dest := filepath.Join(convertedDir, strings.TrimSuffix(rel, filepath.Ext(rel))+".md")
		fmt.Printf("convert: converting %s → %s\n", src, dest)
		if ConvertFile(registry, src, dest) {
			converted++
			fmt.Printf("convert: done (%d/%d)\n", converted, len(files))
		}
	}

	return converted, nil
}

// injectEmbeddedXlsx extracts embedded .xlsx files from a docx/pptx (which are zip
// archives) and converts each one using the registry. The resulting text
// replaces the first image placeholder in the converted markdown content.
//
// NOTE: With the current pure Go parsers, this function is effectively a no-op
// because DocxParser/PptxParser use extractXMLText which does not generate
// base64 image placeholders. The placeholder pattern only appeared when using
// markitdown. This function is retained for forward compatibility — if a future
// parser produces image placeholders for embedded objects, it will work.
func injectEmbeddedXlsx(registry *Registry, srcPath string, md string) string {
	zr, err := zip.OpenReader(srcPath)
	if err != nil {
		return md
	}
	defer zr.Close()

	xlsxParser, ok := registry.Get(".xlsx")
	if !ok {
		return md
	}

	result := md
	for _, f := range zr.File {
		if !strings.Contains(f.Name, "embeddings/") || !strings.HasSuffix(strings.ToLower(f.Name), ".xlsx") {
			continue
		}
		if !imgPlaceholderRe.MatchString(result) {
			break
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		tmp, err := os.CreateTemp("", "embedded-*.xlsx")
		if err != nil {
			rc.Close()
			continue
		}
		tmpPath := tmp.Name()

		_, copyErr := tmp.ReadFrom(rc)
		rc.Close()
		tmp.Close()
		if copyErr != nil {
			os.Remove(tmpPath)
			continue
		}

		xlText, extractErr := xlsxParser.Extract(tmpPath)
		os.Remove(tmpPath)
		if extractErr != nil {
			continue
		}

		xlText = strings.ReplaceAll(xlText, "| NaN ", "|     ")
		xlText = strings.ReplaceAll(xlText, "\\_", "_")
		xlText = strings.TrimSpace(xlText)
		if xlText == "" {
			continue
		}

		replaced := false
		result = imgPlaceholderRe.ReplaceAllStringFunc(result, func(match string) string {
			if replaced {
				return match
			}
			replaced = true
			return "\n\n" + xlText + "\n\n"
		})
	}
	return result
}
