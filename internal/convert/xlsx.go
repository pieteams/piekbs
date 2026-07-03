//go:build fts5

package convert

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
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
	// 1. Shared strings (most cell values are stored here)
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
	// 2. Sheet data (extracts both <t> and <v> tags)
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content := extractXlsxSheetText(rc)
			rc.Close()
			if content != "" {
				text.WriteString("\n--- Sheet: " + f.Name + " ---\n")
				text.WriteString(content)
			}
		}
	}
	return strings.TrimSpace(text.String()), nil
}

// extractXlsxSheetText extracts text from worksheet XML, collecting both
// <t> (shared string content) and <v> (numeric cell values).
// <row> elements are converted to newlines for readability.
func extractXlsxSheetText(r io.Reader) string {
	decoder := xml.NewDecoder(r)
	var text strings.Builder
	var inT, inV bool
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inT = true
			case "v":
				inV = true
			case "row":
				text.WriteString("\n")
			}
		case xml.CharData:
			if inT || inV {
				text.Write(t)
				text.WriteString(" ")
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inT = false
			case "v":
				inV = false
			}
		}
	}
	return strings.TrimSpace(text.String())
}
