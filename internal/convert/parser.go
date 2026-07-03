//go:build fts5

package convert

import "strings"

// Parser 定义文档解析器接口。
// 所有解析器统一输出 Markdown 格式。
type Parser interface {
	// Extensions 返回此解析器支持的文件扩展名（含点号，如 ".docx"）。
	Extensions() []string
	// Extract 从文件中提取内容，返回 Markdown 格式字符串。
	Extract(path string) (string, error)
}

// Registry 解析器注册表，按扩展名分发到对应解析器。
type Registry struct {
	parsers map[string]Parser
}

// NewRegistry 创建注册表并注册所有内置解析器。
func NewRegistry() *Registry {
	r := &Registry{parsers: make(map[string]Parser)}
	r.Register(&DocxParser{})
	r.Register(&XlsxParser{})
	r.Register(&PptxParser{})
	r.Register(&PDFParser{})
	r.Register(&EpubParser{})
	r.Register(&HTMLParser{})
	r.Register(&CSVParser{})
	r.Register(&EmailParser{})
	return r
}

// Register 注册一个解析器，将其所有扩展名映射到该解析器。
func (r *Registry) Register(p Parser) {
	for _, ext := range p.Extensions() {
		r.parsers[strings.ToLower(ext)] = p
	}
}

// Get 根据文件扩展名获取对应的解析器。
func (r *Registry) Get(ext string) (Parser, bool) {
	p, ok := r.parsers[strings.ToLower(ext)]
	return p, ok
}

// Supports 返回是否有解析器支持该扩展名。
func (r *Registry) Supports(ext string) bool {
	_, ok := r.Get(ext)
	return ok
}

// Extensions 返回所有已注册的扩展名列表。
func (r *Registry) Extensions() []string {
	exts := make([]string, 0, len(r.parsers))
	for ext := range r.parsers {
		exts = append(exts, ext)
	}
	return exts
}

// Stub parsers — Task 2/3 will replace these with real implementations.
// Each stub allows Registry tests to compile and run.
type DocxParser struct{}

func (p *DocxParser) Extensions() []string             { return []string{".docx"} }
func (p *DocxParser) Extract(path string) (string, error) { return "", nil }

type XlsxParser struct{}

func (p *XlsxParser) Extensions() []string             { return []string{".xlsx"} }
func (p *XlsxParser) Extract(path string) (string, error) { return "", nil }

type PptxParser struct{}

func (p *PptxParser) Extensions() []string             { return []string{".pptx"} }
func (p *PptxParser) Extract(path string) (string, error) { return "", nil }

type PDFParser struct{}

func (p *PDFParser) Extensions() []string             { return []string{".pdf"} }
func (p *PDFParser) Extract(path string) (string, error) { return "", nil }

type EpubParser struct{}

func (p *EpubParser) Extensions() []string             { return []string{".epub"} }
func (p *EpubParser) Extract(path string) (string, error) { return "", nil }

type HTMLParser struct{}

func (p *HTMLParser) Extensions() []string             { return []string{".html", ".htm"} }
func (p *HTMLParser) Extract(path string) (string, error) { return "", nil }

type CSVParser struct{}

func (p *CSVParser) Extensions() []string             { return []string{".csv"} }
func (p *CSVParser) Extract(path string) (string, error) { return "", nil }

type EmailParser struct{}

func (p *EmailParser) Extensions() []string             { return []string{".eml", ".msg"} }
func (p *EmailParser) Extract(path string) (string, error) { return "", nil }
