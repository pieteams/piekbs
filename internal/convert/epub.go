//go:build fts5

package convert

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"
)

// EpubParser extracts text from .epub files (ZIP containing XHTML chapters).
type EpubParser struct{}

func (p *EpubParser) Extensions() []string { return []string{".epub"} }

func (p *EpubParser) Extract(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("extract epub: %w", err)
	}
	defer r.Close()
	var text strings.Builder
	chapterNum := 0
	for _, f := range r.File {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext != ".xhtml" && ext != ".html" && ext != ".htm" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		content := extractXMLText(rc)
		rc.Close()
		if content != "" {
			chapterNum++
			text.WriteString(fmt.Sprintf("\n--- Chapter %d ---\n", chapterNum))
			text.WriteString(content)
		}
	}
	return strings.TrimSpace(text.String()), nil
}
