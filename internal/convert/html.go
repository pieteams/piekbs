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
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		// h1-h6: 进入时写 Markdown 前缀
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
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6", "li", "tr":
				text.WriteString("\n")
			}
		}
	}
	traverse(doc)
	return strings.TrimSpace(text.String()), nil
}
