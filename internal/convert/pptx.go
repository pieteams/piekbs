//go:build fts5

package convert

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PptxParser extracts text from .pptx files (ZIP containing slide XMLs).
type PptxParser struct{}

func (p *PptxParser) Extensions() []string { return []string{".pptx"} }

func (p *PptxParser) Extract(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("extract pptx: %w", err)
	}
	defer r.Close()

	// 1. Get slide order from presentation.xml
	slideOrder := pptxSlideOrder(r)

	// 2. Build rId → embedded file path map from slide relationships
	relsMap := parsePptxRels(r)

	// 3. Pre-extract embedded Excel files
	embedCache := extractPptxEmbeds(r, relsMap)

	// 4. Extract slide text in presentation order
	var text strings.Builder
	slideNum := 0
	for _, slidePath := range slideOrder {
		slideFile := findZipFile(r, slidePath)
		if slideFile == nil {
			continue
		}
		rc, err := slideFile.Open()
		if err != nil {
			continue
		}
		content := extractPptxSlide(rc, relsMap, embedCache)
		rc.Close()
		if content != "" {
			slideNum++
			text.WriteString(fmt.Sprintf("\n--- Slide %d ---\n", slideNum))
			text.WriteString(content)
		}
	}
	return strings.TrimSpace(text.String()), nil
}

// pptxSlideOrder reads presentation.xml to get the slide order defined in <p:sldIdLst>.
func pptxSlideOrder(r *zip.ReadCloser) []string {
	// 1. Parse presentation.xml.rels to map r:id → slide path
	presRelsMap := make(map[string]string)
	for _, f := range r.File {
		if f.Name != "ppt/_rels/presentation.xml.rels" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
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
				if id != "" && target != "" && strings.Contains(relType, "slide") && !strings.Contains(relType, "slideLayout") && !strings.Contains(relType, "slideMaster") && !strings.Contains(relType, "notesSlide") {
					presRelsMap[id] = "ppt/" + target
				}
			}
		}
		rc.Close()
		break
	}

	// 2. Parse presentation.xml to get slide order from <p:sldIdLst>
	for _, f := range r.File {
		if f.Name != "ppt/presentation.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		var result []string
		decoder := xml.NewDecoder(rc)
		inSldIdLst := false
		for {
			tok, err := decoder.Token()
			if err != nil {
				break
			}
			switch t := tok.(type) {
			case xml.StartElement:
				if t.Name.Local == "sldIdLst" {
					inSldIdLst = true
				}
				if inSldIdLst && t.Name.Local == "sldId" {
					for _, attr := range t.Attr {
						if attr.Name.Local == "id" && attr.Name.Space == "http://schemas.openxmlformats.org/officeDocument/2006/relationships" {
							if slidePath, ok := presRelsMap[attr.Value]; ok {
								result = append(result, slidePath)
							}
						}
					}
				}
			case xml.EndElement:
				if t.Name.Local == "sldIdLst" {
					inSldIdLst = false
				}
			}
		}
		rc.Close()
		if len(result) > 0 {
			return result
		}
	}
	return nil
}

// findZipFile looks up a file by exact name in the ZIP archive.
func findZipFile(r *zip.ReadCloser, name string) *zip.File {
	for _, f := range r.File {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// parsePptxRels parses all ppt/slides/_rels/slide*.xml.rels and returns rId → target path map.
func parsePptxRels(r *zip.ReadCloser) map[string]string {
	result := make(map[string]string)
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "ppt/slides/_rels/slide") || !strings.HasSuffix(f.Name, ".xml.rels") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
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
				if id != "" && target != "" && strings.Contains(relType, "package") {
					// Resolve relative path: base is the slide directory, not _rels directory
					// ppt/slides/_rels/slide165.xml.rels → ppt/slides/ (owner directory)
					relDir := filepath.Dir(filepath.Dir(f.Name)) // go up from _rels to slide dir
					resolved := filepath.Join(relDir, target)
					resolved = filepath.Clean(resolved) // normalize ../
					result[id] = resolved
				}
			}
		}
		rc.Close()
	}
	return result
}

// extractPptxEmbeds pre-extracts embedded Excel files and returns a cache of path → text.
func extractPptxEmbeds(r *zip.ReadCloser, relsMap map[string]string) map[string]string {
	cache := make(map[string]string)
	seen := make(map[string]bool)
	for _, target := range relsMap {
		if seen[target] {
			continue
		}
		seen[target] = true
		if !strings.HasSuffix(strings.ToLower(target), ".xlsx") {
			continue
		}
		for _, f := range r.File {
			if f.Name != target {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				continue
			}
			tmp, err := os.CreateTemp("", "pptx-embed-*.xlsx")
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
				cache[target] = xlText
			}
			// Note: OLE2 Excel files (Excel 97-2003) are not supported
			// Only modern XLSX (ZIP-based) can be extracted
		}
	}
	return cache
}

// extractPptxSlide parses a slide XML with table awareness and embedded content support.
// Uses the same table-aware approach as DOCX: <a:tbl> → <a:tr> → <a:tc> → <a:t>.
func extractPptxSlide(r interface{ Read([]byte) (int, error) }, relsMap map[string]string, embedCache map[string]string) string {
	decoder := xml.NewDecoder(r)
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
			case "oleObj":
				for _, attr := range t.Attr {
					if attr.Name.Local == "id" {
						if targetPath, ok := relsMap[attr.Value]; ok {
							if embedText, ok := embedCache[targetPath]; ok {
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
	return strings.TrimSpace(text.String())
}
