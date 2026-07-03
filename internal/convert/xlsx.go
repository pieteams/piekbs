//go:build fts5

package convert

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"strings"
)

// XlsxParser extracts text from .xlsx files as Markdown tables.
type XlsxParser struct{}

func (p *XlsxParser) Extensions() []string { return []string{".xlsx"} }

func (p *XlsxParser) Extract(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("extract xlsx: %w", err)
	}
	defer r.Close()

	// 1. Parse shared strings table
	sharedStrings := parseSharedStrings(r)

	// 2. Parse each sheet and build Markdown tables
	var text strings.Builder
	sheetNum := 0
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "xl/worksheets/sheet") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		rows := parseSheet(rc, sharedStrings)
		rc.Close()
		if len(rows) == 0 {
			continue
		}
		sheetNum++
		text.WriteString(fmt.Sprintf("\n--- Sheet: %s ---\n", f.Name))
		text.WriteString(sheetToMarkdown(rows))
	}
	return strings.TrimSpace(text.String()), nil
}

// parseSharedStrings reads xl/sharedStrings.xml and returns the string table.
func parseSharedStrings(r *zip.ReadCloser) []string {
	for _, f := range r.File {
		if f.Name != "xl/sharedStrings.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		defer rc.Close()
		var result []string
		decoder := xml.NewDecoder(rc)
		var inT bool
		for {
			tok, err := decoder.Token()
			if err != nil {
				break
			}
			switch t := tok.(type) {
			case xml.StartElement:
				if t.Name.Local == "t" {
					inT = true
				}
			case xml.CharData:
				if inT {
					result = append(result, string(t))
				}
			case xml.EndElement:
				if t.Name.Local == "t" {
					inT = false
				}
			}
		}
		return result
	}
	return nil
}

// cell represents one cell in the sheet grid.
type cell struct {
	col int // 0-based column index
	val string
}

// row represents one row of cells.
type row struct {
	cells []cell
}

// parseSheet reads a worksheet XML and returns rows of cells.
// sharedStrings is the string table from xl/sharedStrings.xml.
func parseSheet(r interface{ Read([]byte) (int, error) }, sharedStrings []string) []row {
	decoder := xml.NewDecoder(r)
	var rows []row
	var currentRow *row
	var inV, inT bool            // inV = <v>, inT = <t> (inline string in cell)
	var cellType string          // cell type attribute (t="s" means shared string)
	var cellRef string           // cell reference like "A1", "B2"
	var cellCol int              // parsed column index from cellRef
	var cellVal strings.Builder  // accumulated cell value
	var rowIdx int               // current row index (from <row r="...">)

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				rowIdx = 0
				for _, attr := range t.Attr {
					if attr.Name.Local == "r" {
						fmt.Sscanf(attr.Value, "%d", &rowIdx)
					}
				}
				currentRow = &row{}
			case "c":
				cellType = ""
				cellRef = ""
				cellCol = 0
				for _, attr := range t.Attr {
					switch attr.Name.Local {
					case "t":
						cellType = attr.Value
					case "r":
						cellRef = attr.Value
					}
				}
				cellCol = colIndex(cellRef)
				cellVal.Reset()
			case "v":
				inV = true
			case "t":
				inT = true
			case "is":
				// inline string container (not used for value extraction)
			}
		case xml.CharData:
			if inV || inT {
				cellVal.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				inV = false
			case "t":
				inT = false
			case "is":
				// end of inline string container
			case "c":
				// Cell complete — resolve value
				val := cellVal.String()
				if cellType == "s" {
					// Shared string reference
					idx := 0
					fmt.Sscanf(val, "%d", &idx)
					if idx >= 0 && idx < len(sharedStrings) {
						val = sharedStrings[idx]
					}
				}
				if currentRow != nil {
					currentRow.cells = append(currentRow.cells, cell{col: cellCol, val: val})
				}
			case "row":
				if currentRow != nil && len(currentRow.cells) > 0 {
					rows = append(rows, *currentRow)
				}
				currentRow = nil
			}
		}
	}
	return rows
}

// colIndex converts a cell reference like "A1" to a 0-based column index.
// "A" → 0, "B" → 1, ..., "Z" → 25, "AA" → 26, etc.
func colIndex(ref string) int {
	col := 0
	hasLetter := false
	for _, c := range ref {
		if c >= 'A' && c <= 'Z' {
			col = col*26 + int(c-'A') + 1
			hasLetter = true
		} else if c >= 'a' && c <= 'z' {
			col = col*26 + int(c-'a') + 1
			hasLetter = true
		} else {
			break // stop at digit
		}
	}
	if !hasLetter {
		return 0
	}
	return col - 1 // 0-based
}

// sheetToMarkdown converts parsed XLSX rows to a Markdown table.
func sheetToMarkdown(rows []row) string {
	if len(rows) == 0 {
		return ""
	}
	// Find max columns
	maxCol := 0
	for _, r := range rows {
		for _, c := range r.cells {
			if c.col+1 > maxCol {
				maxCol = c.col + 1
			}
		}
	}
	// Build grid from sparse cell representation
	grid := make([][]string, len(rows))
	for i := range grid {
		grid[i] = make([]string, maxCol)
	}
	for i, r := range rows {
		for _, c := range r.cells {
			if c.col < maxCol {
				grid[i][c.col] = strings.TrimSpace(c.val)
			}
		}
	}
	return gridToMarkdown(grid)
}
