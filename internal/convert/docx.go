//go:build fts5

package convert

import (
	"archive/zip"
	"fmt"
)

// DocxParser extracts text from .docx files (ZIP containing word/document.xml).
type DocxParser struct{}

func (p *DocxParser) Extensions() []string { return []string{".docx"} }

func (p *DocxParser) Extract(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("extract docx: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("extract docx: open document.xml: %w", err)
			}
			defer rc.Close()
			return extractXMLText(rc), nil
		}
	}
	return "", fmt.Errorf("extract docx: no document.xml in %s", path)
}
