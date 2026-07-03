//go:build fts5

package convert

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"golang.org/x/net/html"
)

// HTMLParser extracts text from .html/.htm files by traversing the DOM tree.
type HTMLParser struct{}

func (p *HTMLParser) Extensions() []string { return []string{".html", ".htm"} }

func (p *HTMLParser) Extract(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("extract html: %w", err)
	}
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("extract html: parse: %w", err)
	}
	var text strings.Builder
	extractNode(doc, &text)
	return strings.TrimSpace(text.String()), nil
}

// extractNode traverses the DOM tree and extracts text with Markdown formatting.
// Tables are converted to Markdown tables. Headings use # prefix.
func extractNode(n *html.Node, text *strings.Builder) {
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		return
	}

	// Handle headings
	if n.Type == html.ElementNode {
		switch n.Data {
		case "h1":
			text.WriteString("# ")
		case "h2":
			text.WriteString("## ")
		case "h3":
			text.WriteString("### ")
		case "h4":
			text.WriteString("#### ")
		case "h5":
			text.WriteString("##### ")
		case "h6":
			text.WriteString("###### ")
		}
	}

	// Handle tables specially
	if n.Type == html.ElementNode && n.Data == "table" {
		rows := parseHTMLTable(n)
		if len(rows) > 0 {
			text.WriteString("\n")
			text.WriteString(gridToMarkdown(rows))
			text.WriteString("\n")
		}
		return // don't traverse children again
	}

	// Handle text nodes
	if n.Type == html.TextNode {
		text.WriteString(n.Data)
	}

	// Recurse children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractNode(c, text)
	}

	// Block elements produce newlines
	if n.Type == html.ElementNode {
		switch n.Data {
		case "p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6", "li":
			text.WriteString("\n")
		}
	}
}

// parseHTMLTable extracts a 2D grid from a <table> element.
// Handles <tr> directly under <table> or wrapped in <tbody>/<thead>.
func parseHTMLTable(tableNode *html.Node) [][]string {
	var rows [][]string
	for child := tableNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		// Direct <tr> children
		if child.Data == "tr" {
			row := extractHTMLRow(child)
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
		// <tbody>/<thead> wrapping <tr> children
		if child.Data == "tbody" || child.Data == "thead" {
			for tr := child.FirstChild; tr != nil; tr = tr.NextSibling {
				if tr.Type == html.ElementNode && tr.Data == "tr" {
					row := extractHTMLRow(tr)
					if len(row) > 0 {
						rows = append(rows, row)
					}
				}
			}
		}
	}
	return rows
}

func extractHTMLRow(tr *html.Node) []string {
	var row []string
	for td := tr.FirstChild; td != nil; td = td.NextSibling {
		if td.Type != html.ElementNode || (td.Data != "td" && td.Data != "th") {
			continue
		}
		cell := extractCellText(td)
		row = append(row, cell)
	}
	return row
}

// extractCellText collects all text content from a table cell element.
func extractCellText(td *html.Node) string {
	var sb strings.Builder
	for c := td.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		} else if c.Type == html.ElementNode {
			extractCellTextInto(c, &sb)
		}
	}
	return strings.TrimSpace(sb.String())
}

func extractCellTextInto(n *html.Node, sb *strings.Builder) {
	if n.Type == html.TextNode {
		sb.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractCellTextInto(c, sb)
	}
}
