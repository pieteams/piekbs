//go:build fts5

package convert

import (
	"encoding/xml"
	"io"
	"strings"
)

// extractXMLText traverses an XML stream and collects text from <t> elements.
// <p> and <br> elements are converted to newlines. Used by PPTX and legacy paths.
func extractXMLText(r io.Reader) string {
	decoder := xml.NewDecoder(r)
	var text strings.Builder
	var inText bool
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inText = true
			}
			if t.Name.Local == "p" || t.Name.Local == "br" {
				text.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				text.Write(t)
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
				text.WriteString(" ")
			}
		}
	}
	return strings.TrimSpace(text.String())
}

// gridToMarkdown converts a 2D grid of strings to a Markdown table.
// First row is treated as the header. Shared by DOCX and XLSX parsers.
func gridToMarkdown(grid [][]string) string {
	if len(grid) == 0 {
		return ""
	}
	maxCol := 0
	for _, r := range grid {
		if len(r) > maxCol {
			maxCol = len(r)
		}
	}
	// Normalize rows to same column count
	for i := range grid {
		if len(grid[i]) < maxCol {
			padded := make([]string, maxCol)
			copy(padded, grid[i])
			grid[i] = padded
		}
	}
	// Trim trailing empty rows
	for len(grid) > 0 {
		last := grid[len(grid)-1]
		empty := true
		for _, v := range last {
			if v != "" {
				empty = false
				break
			}
		}
		if !empty {
			break
		}
		grid = grid[:len(grid)-1]
	}
	if len(grid) == 0 {
		return ""
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

// padRight pads a string with spaces to reach the given width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
