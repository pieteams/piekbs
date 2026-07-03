//go:build fts5

package convert

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"strings"
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
			return extractDocxText(rc), nil
		}
	}
	return "", fmt.Errorf("extract docx: no document.xml in %s", path)
}

// extractDocxText parses DOCX XML with awareness of table structure.
// Tables (<w:tbl>) are converted to Markdown table format.
// Paragraphs (<w:p>) produce newlines. Text is collected from <w:t> tags.
func extractDocxText(r interface{ Read([]byte) (int, error) }) string {
	decoder := xml.NewDecoder(r)
	var text strings.Builder
	var inT bool              // inside <w:t>
	var inTable bool          // inside <w:tbl>
	var currentRow []string   // cells in current row
	var currentCell strings.Builder // text in current cell
	var rows [][]string       // all rows in current table

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tbl":
				inTable = true
				rows = nil
			case "tr":
				currentRow = nil
			case "tc":
				currentCell.Reset()
			case "t":
				inT = true
			case "p":
				if !inTable {
					text.WriteString("\n")
				}
			case "br":
				if !inTable {
					text.WriteString("\n")
				}
			}
		case xml.CharData:
			if inT {
				if inTable {
					currentCell.Write(t)
				} else {
					text.Write(t)
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inT = false
			case "tc":
				if inTable {
					currentRow = append(currentRow, strings.TrimSpace(currentCell.String()))
				}
			case "tr":
				if inTable && len(currentRow) > 0 {
					rows = append(rows, currentRow)
				}
			case "tbl":
				if inTable && len(rows) > 0 {
					text.WriteString("\n")
					text.WriteString(gridToMarkdown(rows))
					text.WriteString("\n")
				}
				inTable = false
				rows = nil
			}
		}
	}
	return strings.TrimSpace(text.String())
}

