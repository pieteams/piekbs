// Package convert handles converting non-markdown files (PDF, HTML, DOCX, etc.)
// to markdown using external tools (markitdown or pandoc).
package convert

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// convertibleExtensions lists file extensions that can be converted to markdown.
var convertibleExtensions = map[string]bool{
	".pdf":  true,
	".html": true,
	".docx": true,
	".doc":  true,
	".epub": true,
	".odt":  true,
}

// FindConverter looks for markitdown then pandoc in PATH.
// Returns the full path of the first tool found, or empty string if neither is available.
func FindConverter() string {
	for _, tool := range []string{"markitdown", "pandoc"} {
		if path, err := exec.LookPath(tool); err == nil {
			return path
		}
	}
	return ""
}

// FindConvertibleFiles walks the raw/ directory under kbRoot and returns
// paths of files that need conversion (have a convertible extension and
// no corresponding .md already in raw/converted/).
func FindConvertibleFiles(kbRoot string) []string {
	rawDir := filepath.Join(kbRoot, "raw")
	convertedDir := filepath.Join(rawDir, "converted")

	var result []string

	_ = filepath.Walk(rawDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip the converted/ subdirectory entirely
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

		// Only process known convertible extensions
		if !convertibleExtensions[ext] {
			return nil
		}

		// Check if a converted version already exists
		rel, _ := filepath.Rel(rawDir, path)
		convertedPath := filepath.Join(convertedDir, strings.TrimSuffix(rel, filepath.Ext(rel))+".md")
		if _, err := os.Stat(convertedPath); err == nil {
			// Already converted — skip
			return nil
		}

		result = append(result, path)
		return nil
	})

	return result
}

// ConvertFile converts srcPath to markdown at destPath using the given converter binary.
// Returns true on success, false on failure.
func ConvertFile(converter, srcPath, destPath string) bool {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "convert: mkdir failed for %s: %v\n", destPath, err)
		return false
	}

	type result struct {
		err error
	}
	ch := make(chan result, 1)

	go func() {
		var err error
		converterBase := filepath.Base(converter)

		switch {
		case strings.HasSuffix(converterBase, "markitdown"):
			var out bytes.Buffer
			cmd := exec.Command(converter, srcPath) //nolint:gosec
			cmd.Stdout = &out
			cmd.Stderr = os.Stderr
			if err = cmd.Run(); err == nil {
				content := out.Bytes()
				// Post-process docx/pptx: extract embedded Excel tables and
				// replace image placeholders with the converted table markdown.
				ext := strings.ToLower(filepath.Ext(srcPath))
				if ext == ".docx" || ext == ".pptx" {
					content = injectEmbeddedXlsx(converter, srcPath, content)
				}
				err = os.WriteFile(destPath, content, 0o644)
			}

		case strings.HasSuffix(converterBase, "pandoc"):
			cmd := exec.Command(converter, srcPath, "-t", "markdown", "-o", destPath) //nolint:gosec
			cmd.Stderr = os.Stderr
			err = cmd.Run()

		default:
			err = fmt.Errorf("unknown converter: %s", converter)
		}

		ch <- result{err: err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "convert: failed to convert %s: %v\n", srcPath, r.err)
			return false
		}
		return true
	case <-time.After(120 * time.Second):
		fmt.Fprintf(os.Stderr, "convert: timeout converting %s\n", srcPath)
		return false
	}
}

// imgPlaceholderRe matches base64-encoded image placeholders produced by markitdown
// for embedded OLE objects (e.g. embedded Excel sheets inside docx/pptx).
var imgPlaceholderRe = regexp.MustCompile(`!\[.*?\]\(data:image/[^)]+\)`)

// injectEmbeddedXlsx extracts embedded .xlsx files from a docx/pptx (which are zip
// archives) and converts each one to markdown using markitdown. The resulting table
// markdown replaces the first image placeholder in the converted markdown content.
// If extraction or conversion fails for any sheet, that placeholder is left unchanged.
func injectEmbeddedXlsx(converter, srcPath string, md []byte) []byte {
	zr, err := zip.OpenReader(srcPath)
	if err != nil {
		return md
	}
	defer zr.Close()

	result := md
	for _, f := range zr.File {
		// Embedded Excel files live under word/embeddings/ or ppt/embeddings/
		if !strings.Contains(f.Name, "embeddings/") || !strings.HasSuffix(strings.ToLower(f.Name), ".xlsx") {
			continue
		}
		// Only replace if there is still a placeholder to fill.
		if !imgPlaceholderRe.Match(result) {
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

		var out bytes.Buffer
		cmd := exec.Command(converter, tmpPath) //nolint:gosec
		cmd.Stdout = &out
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Remove(tmpPath)
			continue
		}
		os.Remove(tmpPath)

		// Clean up common markitdown artifacts from Excel conversion.
		xlMD := strings.ReplaceAll(out.String(), "| NaN ", "|     ")
		xlMD = strings.ReplaceAll(xlMD, "\\_", "_")
		xlMD = strings.TrimSpace(xlMD)
		if xlMD == "" {
			continue
		}

		// Replace the first image placeholder with the Excel table markdown.
		replaced := false
		result = imgPlaceholderRe.ReplaceAllFunc(result, func(match []byte) []byte {
			if replaced {
				return match
			}
			replaced = true
			return []byte("\n\n" + xlMD + "\n\n")
		})
	}
	return result
}

// Run finds a converter, discovers convertible files under kbRoot, and converts them.
// Progress messages are printed to stdout. Always returns (0, nil) — failures are non-fatal.
func Run(kbRoot string) (int, error) {
	converter := FindConverter()
	if converter == "" {
		fmt.Println("convert: no converter found (markitdown or pandoc required); skipping")
		return 0, nil
	}

	files := FindConvertibleFiles(kbRoot)
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
		if ConvertFile(converter, src, dest) {
			converted++
			fmt.Printf("convert: done (%d/%d)\n", converted, len(files))
		}
	}

	return 0, nil
}
