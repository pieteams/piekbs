//go:build fts5

package convert

import (
	"archive/zip"
	"fmt"
	"strings"
)

// XlsxParser extracts text from .xlsx files (ZIP containing shared strings and sheets).
type XlsxParser struct{}

func (p *XlsxParser) Extensions() []string { return []string{".xlsx"} }

func (p *XlsxParser) Extract(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("extract xlsx: %w", err)
	}
	defer r.Close()
	var text strings.Builder
	// 1. 共享字符串（大部分 cell 值在这里）
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			text.WriteString(extractXMLText(rc))
			rc.Close()
		}
	}
	// 2. 各 sheet 数据
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content := extractXMLText(rc)
			rc.Close()
			if content != "" {
				text.WriteString("\n--- Sheet: " + f.Name + " ---\n")
				text.WriteString(content)
			}
		}
	}
	return strings.TrimSpace(text.String()), nil
}
