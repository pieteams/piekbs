//go:build fts5

package convert

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFParser extracts text from .pdf files using ledongthuc/pdf (pure Go).
type PDFParser struct{}

func (p *PDFParser) Extensions() []string { return []string{".pdf"} }

func (p *PDFParser) Extract(path string) (string, error) {
	// Preprocess: ledongthuc/pdf rejects valid PDFs with a space after the
	// version number (e.g. "%PDF-1.5 \r"). Fix the header if needed.
	actualPath, cleanup, err := fixPDFHeader(path)
	if err != nil {
		return "", fmt.Errorf("extract pdf: %w", err)
	}
	defer cleanup()

	f, r, err := pdf.Open(actualPath)
	if err != nil {
		// Malformed PDF — return a placeholder document so the watcher
		// won't retry this file endlessly.
		return placeholderDoc(path, "malformed or encrypted PDF"), nil
	}
	defer f.Close()
	var text strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text.WriteString(content)
		text.WriteString("\n\n")
	}
	result := strings.TrimSpace(text.String())
	if result == "" {
		// Scanned PDF (images only, no embedded text) — return a placeholder.
		return placeholderDoc(path, "scanned PDF (images only, no embedded text)"), nil
	}
	return result, nil
}

// placeholderDoc returns a Markdown document explaining why conversion failed.
// This prevents the watcher from retrying the file endlessly.
func placeholderDoc(path, reason string) string {
	return fmt.Sprintf("# PDF Conversion Failed\n\n"+
		"**File:** %s\n\n"+
		"**Reason:** %s\n\n"+
		"This file could not be converted to text. "+
		"To index its content, please convert it manually (e.g. using OCR tools) "+
		"and place the result in `raw/converted/`.\n", path, reason)
}

// fixPDFHeader checks if a PDF has a space between the version number and the
// line ending (e.g. "%PDF-1.5 \r") and writes a fixed copy to a temp file.
// Returns the path to use and a cleanup function.
func fixPDFHeader(path string) (string, func(), error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", func() {}, err
	}

	// Check if header needs fixing: %PDF-1.X followed by space then \r or \n
	if len(data) >= 10 && bytes.HasPrefix(data, []byte("%PDF-1.")) && data[8] == ' ' && (data[9] == '\r' || data[9] == '\n') {
		// Remove the extra space
		fixed := make([]byte, len(data)-1)
		copy(fixed, data[:8])
		copy(fixed[8:], data[9:])
		tmp, err := os.CreateTemp("", "pdf-fix-*.pdf")
		if err != nil {
			return "", func() {}, err
		}
		if _, err := tmp.Write(fixed); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return "", func() {}, err
		}
		tmp.Close()
		return tmp.Name(), func() { os.Remove(tmp.Name()) }, nil
	}

	// No fix needed
	return path, func() {}, nil
}
