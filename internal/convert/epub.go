//go:build fts5

package convert

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// EpubParser extracts text from .epub files (ZIP containing XHTML chapters).
type EpubParser struct{}

func (p *EpubParser) Extensions() []string { return []string{".epub"} }

func (p *EpubParser) Extract(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("extract epub: %w", err)
	}
	defer r.Close()

	// Two-phase parsing: try spine order first, fallback to ZIP storage order
	ordered := epubSpineOrder(r)
	if ordered == nil {
		ordered = epubFallbackOrder(r)
	}

	var text strings.Builder
	chapterNum := 0
	for _, name := range ordered {
		var rc *zip.File
		for _, f := range r.File {
			if f.Name == name {
				rc = f
				break
			}
		}
		if rc == nil {
			continue
		}
		frc, err := rc.Open()
		if err != nil {
			continue
		}
		content := extractHTMLText(frc)
		frc.Close()
		if content != "" {
			chapterNum++
			text.WriteString(fmt.Sprintf("\n--- Chapter %d ---\n", chapterNum))
			text.WriteString(content)
		}
	}
	return strings.TrimSpace(text.String()), nil
}

// epubSpineOrder parses EPUB container.xml + OPF and returns the spine-defined
// chapter order. Returns nil on failure; caller should fallback to ZIP order.
func epubSpineOrder(r *zip.ReadCloser) []string {
	// 1. Find OPF path from META-INF/container.xml
	opfPath := findOPFPath(r)
	if opfPath == "" {
		return nil
	}

	// 2. 解析 OPF 文件
	opfFile := findFile(r, opfPath)
	if opfFile == nil {
		return nil
	}
	opfrc, err := opfFile.Open()
	if err != nil {
		return nil
	}
	defer opfrc.Close()

	opfDir := filepath.Dir(opfPath)
	manifest, spineIDs := parseOPF(xml.NewDecoder(opfrc))
	if len(spineIDs) == 0 {
		return nil
	}

	// 3. Map spine idref to file paths
	var result []string
	for _, id := range spineIDs {
		href, ok := manifest[id]
		if !ok {
			continue
		}
		// href is relative to the OPF directory
		fullPath := filepath.Join(opfDir, href)
		result = append(result, fullPath)
	}
	return result
}

// findOPFPath extracts the OPF file path from container.xml.
func findOPFPath(r *zip.ReadCloser) string {
	for _, f := range r.File {
		if f.Name == "META-INF/container.xml" {
			rc, err := f.Open()
			if err != nil {
				return ""
			}
			defer rc.Close()
			var container struct {
				Rootfiles []struct {
					FullPath string `xml:"full-path,attr"`
				} `xml:"rootfiles>rootfile"`
			}
			if err := xml.NewDecoder(rc).Decode(&container); err != nil {
				return ""
			}
			if len(container.Rootfiles) > 0 {
				return container.Rootfiles[0].FullPath
			}
		}
	}
	return ""
}

// parseOPF parses an OPF file and returns the manifest (id→href map) and spine order (idref list).
func parseOPF(r *xml.Decoder) (manifest map[string]string, spineIDs []string) {
	manifest = make(map[string]string)
	for {
		tok, err := r.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok {
			switch se.Name.Local {
			case "item":
				var id, href string
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "id":
						id = attr.Value
					case "href":
						href = attr.Value
					}
				}
				if id != "" && href != "" {
					manifest[id] = href
				}
			case "itemref":
				for _, attr := range se.Attr {
					if attr.Name.Local == "idref" {
						spineIDs = append(spineIDs, attr.Value)
					}
				}
			}
		}
	}
	return
}

// findFile looks up a file by exact name in the ZIP archive.
func findFile(r *zip.ReadCloser, name string) *zip.File {
	for _, f := range r.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// epubFallbackOrder returns XHTML/HTML files in ZIP storage order (fallback).
func epubFallbackOrder(r *zip.ReadCloser) []string {
	var result []string
	for _, f := range r.File {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if ext == ".xhtml" || ext == ".html" || ext == ".htm" {
			result = append(result, f.Name)
		}
	}
	return result
}

// extractHTMLText reads an HTML document from r and returns plain text content.
// Script and style elements are skipped. Block-level elements produce newlines.
func extractHTMLText(r io.Reader) string {
	data, err := io.ReadAll(r)
	if err != nil {
		return ""
	}
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	var text strings.Builder
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
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
	return strings.TrimSpace(text.String())
}
