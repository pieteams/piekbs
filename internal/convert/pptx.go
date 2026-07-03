//go:build fts5

package convert

import (
	"archive/zip"
	"fmt"
	"strings"
)

// PptxParser extracts text from .pptx files (ZIP containing slide XMLs).
type PptxParser struct{}

func (p *PptxParser) Extensions() []string { return []string{".pptx"} }

func (p *PptxParser) Extract(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("extract pptx: %w", err)
	}
	defer r.Close()
	var text strings.Builder
	slideNum := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content := extractXMLText(rc)
			rc.Close()
			if content != "" {
				slideNum++
				text.WriteString(fmt.Sprintf("\n--- Slide %d ---\n", slideNum))
				text.WriteString(content)
			}
		}
	}
	return strings.TrimSpace(text.String()), nil
}
