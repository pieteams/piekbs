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

	// 1. Build rId → embedded file path map from relationships
	relsMap := parseDocxRels(r)

	// 2. Parse document.xml with OLE object awareness
	text, err := extractDocxWithEmbeds(r, relsMap)
	if err != nil {
		return "", err
	}
	return text, nil
}

// parseDocxRels parses word/_rels/document.xml.rels and returns rId → target path map.
func parseDocxRels(r *zip.ReadCloser) map[string]string {
	result := make(map[string]string)
	for _, f := range r.File {
		if f.Name != "word/_rels/document.xml.rels" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return result
		}
		defer rc.Close()
		decoder := xml.NewDecoder(rc)
		for {
			tok, err := decoder.Token()
			if err != nil {
				break
			}
			if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "Relationship" {
				var id, target, relType string
				for _, attr := range se.Attr {
					switch attr.Name.Local {
					case "Id":
						id = attr.Value
					case "Target":
						target = attr.Value
					case "Type":
						relType = attr.Value
					}
				}
				// Only map embedded packages (OLE objects like Excel)
				if id != "" && target != "" && strings.Contains(relType, "package") {
					result[id] = "word/" + target
				}
			}
		}
	}
	return result
}

// extractDocxWithEmbeds parses document.xml and inserts embedded content at OLE positions.
func extractDocxWithEmbeds(r *zip.ReadCloser, relsMap map[string]string) (string, error) {
	var docXML *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docXML = f
			break
		}
	}
	if docXML == nil {
		return "", fmt.Errorf("extract docx: no document.xml")
	}

	// Pre-extract all embedded files to temp paths
	embedCache := make(map[string]string) // rels path → extracted text
	for _, f := range r.File {
		relsPath := f.Name
		// Check if this file is an embedded package
		isEmbed := false
		for _, target := range relsMap {
			if target == relsPath {
				isEmbed = true
				break
			}
		}
		if !isEmbed {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(f.Name), ".xlsx") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
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
		if err == nil && xlText != "" {
			embedCache[relsPath] = xlText
		}
	}

	// Parse document.xml, inserting embedded content at OLE positions
	rc, err := docXML.Open()
	if err != nil {
		return "", fmt.Errorf("extract docx: open document.xml: %w", err)
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var text strings.Builder
	var inT bool
	var inTable bool
	var currentRow []string
	var currentCell strings.Builder
	var rows [][]string

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
			case "OLEObject":
				// Detect embedded OLE object and insert its content
				for _, attr := range t.Attr {
					if attr.Name.Local == "id" && attr.Name.Space == "http://schemas.openxmlformats.org/officeDocument/2006/relationships" {
						if embedPath, ok := relsMap[attr.Value]; ok {
							if embedText, ok := embedCache[embedPath]; ok {
								text.WriteString("\n\n" + embedText + "\n\n")
							}
						}
					}
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
	result := strings.TrimSpace(text.String())

	// Remove placeholder text for embedded objects (content is already inserted)
	for _, placeholder := range []string{
		"点击图片可查看完整电子表格",
		"点击图片查看完整电子表格",
		"点击查看完整电子表格",
	} {
		result = strings.ReplaceAll(result, placeholder, "")
	}

	return strings.TrimSpace(result), nil
}
