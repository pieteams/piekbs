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
					text.WriteString(docxTableToMarkdown(rows))
					text.WriteString("\n")
				}
				inTable = false
				rows = nil
			}
		}
	}
	return strings.TrimSpace(text.String())
}

// docxTableToMarkdown converts a 2D slice of cell values to a Markdown table.
func docxTableToMarkdown(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	// Find max columns
	maxCol := 0
	for _, r := range rows {
		if len(r) > maxCol {
			maxCol = len(r)
		}
	}
	// Normalize rows to same column count
	grid := make([][]string, len(rows))
	for i, r := range rows {
		grid[i] = make([]string, maxCol)
		copy(grid[i], r)
	}
	// Calculate column widths
	widths := make([]int, maxCol)
	for _, r := range grid {
		for j, v := range r {
			if len(v) > widths[j] {
				widths[j] = len(v)
			}
		}
	}
	// Ensure minimum width
	for j := range widths {
		if widths[j] < 3 {
			widths[j] = 3
		}
	}
	var sb strings.Builder
	// Header row
	sb.WriteString("|")
	for j, v := range grid[0] {
		sb.WriteString(" " + padRight(v, widths[j]) + " |")
	}
	sb.WriteString("\n")
	// Separator
	sb.WriteString("|")
	for j := range grid[0] {
		sb.WriteString(strings.Repeat("-", widths[j]+2) + "|")
	}
	sb.WriteString("\n")
	// Data rows
	for i := 1; i < len(grid); i++ {
		sb.WriteString("|")
		for j, v := range grid[i] {
			sb.WriteString(" " + padRight(v, widths[j]) + " |")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
