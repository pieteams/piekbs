//go:build fts5

package convert

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
	"unicode/utf8"

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
	if isGarbled(result) {
		// Font encoding issue — try Ghostscript as fallback (handles CJK fonts).
		if gsText, err := extractPDFWithGS(path); err == nil && gsText != "" && !isGarbled(gsText) {
			return gsText, nil
		}
		return placeholderDoc(path, "font encoding issue (garbled text, likely CJK PDF with custom fonts)"), nil
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

// isGarbled checks if text appears to be garbled due to font encoding issues.
// Heuristic: if too many bytes are non-printable or invalid UTF-8, the text
// is likely not properly decoded. This commonly happens with CJK PDFs that
// use custom font encodings without proper ToUnicode CMap tables.
func isGarbled(s string) bool {
	if len(s) < 50 {
		return false // too short to judge
	}
	// Sample first 2000 runes for performance
	sample := s
	if len(sample) > 2000 {
		sample = sample[:2000]
	}
	total := 0
	garbled := 0
	for _, r := range sample {
		total++
		if r == '\n' || r == '\r' || r == '\t' || r == ' ' {
			continue
		}
		if r >= 0x20 && r <= 0x7E {
			continue // printable ASCII
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsPunct(r) {
			continue // valid Unicode
		}
		// Check for invalid UTF-8 sequences
		if !utf8.ValidString(string(r)) {
			garbled++
			continue
		}
		// High bytes that are not valid CJK or other known ranges
		if r > 0xFF {
			continue // likely valid Unicode (CJK, etc.)
		}
		// Bytes 0x80-0xFF that are not part of valid UTF-8
		garbled++
	}
	if total == 0 {
		return false
	}
	// If more than 20% of characters are garbled, consider the text unreadable
	return float64(garbled)/float64(total) > 0.20
}

// extractPDFWithGS uses Ghostscript (gs) to extract text from a PDF.
// This handles CJK fonts and complex encodings that the pure Go parser cannot.
// Returns empty string if gs is not available or extraction fails.
func extractPDFWithGS(path string) (string, error) {
	gsPath, err := exec.LookPath("gs")
	if err != nil {
		return "", fmt.Errorf("gs not found in PATH")
	}
	tmp, err := os.CreateTemp("", "gs-text-*.txt")
	if err != nil {
		return "", err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	cmd := exec.Command(gsPath, "-dNOPAUSE", "-dBATCH", "-sDEVICE=txtwrite", "-sOutputFile="+tmp.Name(), path)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gs failed: %w", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
