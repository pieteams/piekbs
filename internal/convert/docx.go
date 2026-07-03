//go:build fts5

package convert

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"os"
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

	var text string
	var embeddedXlsx []string

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("extract docx: open document.xml: %w", err)
			}
			text = extractDocxText(rc)
			rc.Close()
		}
		// Collect embedded Excel files
		if strings.HasPrefix(f.Name, "word/embeddings/") && strings.HasSuffix(strings.ToLower(f.Name), ".xlsx") {
			embeddedXlsx = append(embeddedXlsx, f.Name)
		}
	}

	if text == "" {
		return "", fmt.Errorf("extract docx: no document.xml in %s", path)
	}

	// Extract embedded Excel files and append to output
	for _, name := range embeddedXlsx {
		for _, f := range r.File {
			if f.Name != name {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			// Write embedded xlsx to temp file for XlsxParser
			tmp, err := os.CreateTemp("", "docx-embed-*.xlsx")
			if err != nil {
				rc.Close()
				continue
			}
			if _, err := tmp.ReadFrom(rc); err != nil {
				rc.Close()
				tmp.Close()
				os.Remove(tmp.Name())
				continue
			}
			rc.Close()
			tmp.Close()

			xlsxParser := &XlsxParser{}
			xlText, err := xlsxParser.Extract(tmp.Name())
			os.Remove(tmp.Name())
			if err != nil || xlText == "" {
				continue
			}
			text += "\n\n--- Embedded Spreadsheet: " + name + " ---\n\n" + xlText
		}
	}

	return text, nil
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

