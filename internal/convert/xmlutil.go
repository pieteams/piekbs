//go:build fts5

package convert

import (
	"encoding/xml"
	"io"
	"strings"
)

// extractXMLText 遍历 XML 流，收集 <t> 元素的文本内容。
// <p> 和 <br> 元素转换为换行。用于 Office 格式和 EPUB 解析。
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
