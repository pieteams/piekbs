//go:build fts5

// Package convert handles converting non-markdown files (PDF, HTML, DOCX, etc.)
// to markdown using pure Go parsers.
package convert

import (
	"archive/zip"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// imgPlaceholderRe matches base64-encoded image placeholders in converted
// documents (e.g. embedded Excel sheets inside docx/pptx).
var imgPlaceholderRe = regexp.MustCompile(`!\[.*?\]\(data:image/[^)]+\)`)

const stagingPrefix = ".piekbs-convert-"

// IsStagingPath identifies unpublished output directories under raw/converted.
func IsStagingPath(kbRoot, path string) bool {
	if !strings.HasPrefix(filepath.Base(path), stagingPrefix) {
		return false
	}
	rel, err := filepath.Rel(kbRoot, path)
	return err == nil && strings.HasPrefix(filepath.ToSlash(rel), "raw/converted/")
}

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
		if info, err := os.Stat(convertedPath); err == nil && info.Mode().IsRegular() {
			return nil
		}

		result = append(result, path)
		return nil
	})

	return result
}

// ConvertFile converts srcPath to markdown at destPath using the registry.
func ConvertFile(registry *Registry, srcPath, destPath string) error {
	outputDir := filepath.Dir(destPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	parser, ok := registry.Get(ext)
	if !ok {
		return fmt.Errorf("no parser for %s", ext)
	}

	text, err := parser.Extract(srcPath)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	// Embedded Excel handling: DOCX/PPTX may contain embedded XLSX
	if ext == ".docx" || ext == ".pptx" {
		text = injectEmbeddedXlsx(registry, srcPath, text)
	}

	// Publish completed output atomically so a failed write cannot look converted.
	staging, err := os.MkdirTemp(outputDir, stagingPrefix)
	if err != nil {
		return fmt.Errorf("stage output: %w", err)
	}
	defer os.Remove(staging)
	output := filepath.Join(staging, "content")
	defer os.Remove(output)
	if err := os.WriteFile(output, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if err := os.Rename(output, destPath); err != nil {
		return fmt.Errorf("publish output: %w", err)
	}
	return nil
}

// Run finds convertible files under kbRoot and converts them using pure Go parsers.
// Returns the number converted and any per-file errors after attempting the batch.
func Run(kbRoot string) (int, error) {
	registry := NewRegistry()
	files := FindConvertibleFiles(kbRoot, registry)
	if len(files) == 0 {
		log.Print("convert: no files to convert")
		return 0, nil
	}

	rawDir := filepath.Join(kbRoot, "raw")
	convertedDir := filepath.Join(rawDir, "converted")
	converted := 0
	var failures []error
	for _, src := range files {
		rel, _ := filepath.Rel(rawDir, src)
		dest := filepath.Join(convertedDir, strings.TrimSuffix(rel, filepath.Ext(rel))+".md")
		log.Printf("convert: converting %s → %s", src, dest)
		if err := ConvertFile(registry, src, dest); err != nil {
			failures = append(failures, fmt.Errorf("convert %s: %w", rel, err))
			continue
		}
		converted++
		log.Printf("convert: done (%d/%d)", converted, len(files))
	}

	return converted, errors.Join(failures...)
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
