//go:build fts5

package convert

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFParser extracts text from .pdf files using ledongthuc/pdf (pure Go).
type PDFParser struct{}

func (p *PDFParser) Extensions() []string { return []string{".pdf"} }

func (p *PDFParser) Extract(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("extract pdf: %w", err)
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
		return "", fmt.Errorf("extract pdf: no text content in %s", path)
	}
	return result, nil
}
