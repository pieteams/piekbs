// Package larkimport imports Lark/Feishu Wiki documents and expands embedded
// Base tables into locally searchable text datasets.
package larkimport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const pageSize = 200

var (
	bitableRE = regexp.MustCompile(`<bitable\b[^>]*\btable-id="([^"]+)"[^>]*\btoken="([^"]+)"[^>]*>\s*</bitable>|<bitable\b[^>]*\btoken="([^"]+)"[^>]*\btable-id="([^"]+)"[^>]*>\s*</bitable>`)
	titleRE   = regexp.MustCompile(`<title>(.*?)</title>`)
)

// Runner executes lark-cli and returns stdout.
type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// CLIRunner invokes lark-cli from PATH.
type CLIRunner struct{}

func (CLIRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	path, err := exec.LookPath("lark-cli")
	if err != nil {
		for _, candidate := range []string{"/opt/homebrew/bin/lark-cli", "/usr/local/bin/lark-cli"} {
			if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
				path = candidate
				err = nil
				break
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("lark-cli not found; install it and complete user login first")
	}
	cmd := exec.CommandContext(ctx, path, args...) //nolint:gosec
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("lark-cli %s: %w: %s", strings.Join(args[:2], " "), err, msg)
		}
		return nil, fmt.Errorf("lark-cli %s: %w", strings.Join(args[:2], " "), err)
	}
	return out, nil
}

// Result describes files created by an import.
type Result struct {
	DocumentPath      string
	TablePaths        []string
	TableRows         []int
	DatasetPath       string
	TotalRows         int
	UniqueRows        int
	DuplicatesRemoved int
}

type documentResponse struct {
	OK   bool `json:"ok"`
	Data struct {
		Document struct {
			Content    string `json:"content"`
			DocumentID string `json:"document_id"`
			RevisionID int64  `json:"revision_id"`
		} `json:"document"`
	} `json:"data"`
	Error interface{} `json:"error"`
}

type recordsResponse struct {
	OK   bool `json:"ok"`
	Data struct {
		Data      [][]interface{} `json:"data"`
		Fields    []string        `json:"fields"`
		HasMore   bool            `json:"has_more"`
		RecordIDs []string        `json:"record_id_list"`
	} `json:"data"`
	Error interface{} `json:"error"`
}

type tableRef struct {
	Full      string
	TableID   string
	BaseToken string
}

type importedTable struct {
	Ref    tableRef
	Fields []string
	Rows   [][]interface{}
}

// Import fetches url using the authenticated lark-cli user identity. The
// document is written to raw/lark/<slug>/document.md. Each embedded Base table
// is paginated into a sibling .txt file so it is indexed but not auto-distilled.
func Import(ctx context.Context, kbRoot, url, name string, runner Runner) (*Result, error) {
	if strings.TrimSpace(url) == "" {
		return nil, fmt.Errorf("Lark document URL is required")
	}
	if runner == nil {
		runner = CLIRunner{}
	}

	doc, err := fetchDocument(ctx, runner, url)
	if err != nil {
		return nil, err
	}
	title := extractTitle(doc.Data.Document.Content)
	slug := slugify(name)
	if slug == "" {
		slug = slugify(title)
	}
	if slug == "" {
		slug = strings.ToLower(doc.Data.Document.DocumentID)
	}

	refs := parseTableRefs(doc.Data.Document.Content)
	indexDir := filepath.Join(kbRoot, "index")
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return nil, fmt.Errorf("create index dir: %w", err)
	}
	stage, err := os.MkdirTemp(indexDir, ".lark-import-*")
	if err != nil {
		return nil, fmt.Errorf("create import staging dir: %w", err)
	}
	defer os.RemoveAll(stage)

	content := doc.Data.Document.Content
	result := &Result{}
	var tables []importedTable
	for i, ref := range refs {
		fields, rows, err := fetchAllRecords(ctx, runner, ref.BaseToken, ref.TableID)
		if err != nil {
			return nil, fmt.Errorf("import table %s: %w", ref.TableID, err)
		}
		filename := fmt.Sprintf("table-%02d-%s.snapshot.tsv", i+1, safeID(ref.TableID))
		if err := os.WriteFile(filepath.Join(stage, filename), renderTableDataset(url, ref, fields, rows), 0o644); err != nil {
			return nil, fmt.Errorf("write table snapshot: %w", err)
		}
		replacement := renderTableReference(filename, ref, fields, len(rows))
		content = strings.Replace(content, ref.Full, replacement, 1)
		result.TablePaths = append(result.TablePaths, filepath.ToSlash(filepath.Join("raw", "lark", slug, filename)))
		result.TableRows = append(result.TableRows, len(rows))
		result.TotalRows += len(rows)
		tables = append(tables, importedTable{Ref: ref, Fields: fields, Rows: rows})
	}

	if len(tables) > 0 {
		fields, rows := deduplicateTables(tables)
		datasetName := "records-deduplicated.txt"
		result.UniqueRows = len(rows)
		result.DuplicatesRemoved = result.TotalRows - result.UniqueRows
		result.DatasetPath = filepath.ToSlash(filepath.Join("raw", "lark", slug, datasetName))
		if err := os.WriteFile(filepath.Join(stage, datasetName),
			renderDeduplicatedDataset(url, fields, rows, result.TotalRows, result.DuplicatesRemoved), 0o644); err != nil {
			return nil, fmt.Errorf("write deduplicated dataset: %w", err)
		}
		content += renderDatasetSummary(datasetName, result.TotalRows, result.UniqueRows, result.DuplicatesRemoved)
	}

	document := renderDocument(url, doc.Data.Document.DocumentID, doc.Data.Document.RevisionID, content)
	if err := os.WriteFile(filepath.Join(stage, "document.md"), []byte(document), 0o644); err != nil {
		return nil, fmt.Errorf("write document: %w", err)
	}

	dest := filepath.Join(kbRoot, "raw", "lark", slug)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, fmt.Errorf("create raw lark dir: %w", err)
	}
	if err := os.RemoveAll(dest); err != nil {
		return nil, fmt.Errorf("replace previous import: %w", err)
	}
	if err := os.Rename(stage, dest); err != nil {
		return nil, fmt.Errorf("publish import: %w", err)
	}

	result.DocumentPath = filepath.ToSlash(filepath.Join("raw", "lark", slug, "document.md"))
	return result, nil
}

func fetchDocument(ctx context.Context, runner Runner, url string) (*documentResponse, error) {
	out, err := runner.Run(ctx,
		"docs", "+fetch",
		"--api-version", "v2",
		"--as", "user",
		"--doc", url,
		"--doc-format", "markdown",
		"--detail", "simple",
		"--format", "json",
	)
	if err != nil {
		return nil, err
	}
	var resp documentResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("decode Lark document response: %w", err)
	}
	if !resp.OK || resp.Data.Document.Content == "" {
		return nil, fmt.Errorf("Lark document fetch failed: %v", resp.Error)
	}
	return &resp, nil
}

func fetchAllRecords(ctx context.Context, runner Runner, baseToken, tableID string) ([]string, [][]interface{}, error) {
	var fields []string
	var rows [][]interface{}
	for offset := 0; ; {
		out, err := runner.Run(ctx,
			"base", "+record-list",
			"--base-token", baseToken,
			"--table-id", tableID,
			"--limit", strconv.Itoa(pageSize),
			"--offset", strconv.Itoa(offset),
			"--as", "user",
			"--format", "json",
		)
		if err != nil {
			return nil, nil, err
		}
		var resp recordsResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, nil, fmt.Errorf("decode Base records response: %w", err)
		}
		if !resp.OK {
			return nil, nil, fmt.Errorf("Base records fetch failed: %v", resp.Error)
		}
		if len(fields) == 0 {
			fields = resp.Data.Fields
		}
		rows = append(rows, resp.Data.Data...)
		if !resp.Data.HasMore || len(resp.Data.Data) == 0 {
			break
		}
		offset += len(resp.Data.Data)
	}
	return fields, rows, nil
}

func parseTableRefs(content string) []tableRef {
	matches := bitableRE.FindAllStringSubmatch(content, -1)
	refs := make([]tableRef, 0, len(matches))
	for _, match := range matches {
		tableID, token := match[1], match[2]
		if tableID == "" {
			token, tableID = match[3], match[4]
		}
		refs = append(refs, tableRef{Full: match[0], TableID: tableID, BaseToken: token})
	}
	return refs
}

func extractTitle(content string) string {
	match := titleRE.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return html.UnescapeString(strings.TrimSpace(match[1]))
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	dash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case r >= '\u4e00' && r <= '\u9fff':
			b.WriteRune(r)
			dash = false
		default:
			if b.Len() > 0 && !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func safeID(value string) string {
	value = regexp.MustCompile(`[^A-Za-z0-9_-]+`).ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func renderDocument(url, documentID string, revisionID int64, content string) string {
	return fmt.Sprintf(`---
source_url: %q
lark_document_id: %q
lark_revision_id: %d
importer: "wikiloop-lark"
---

%s
`, url, documentID, revisionID, content)
}

func renderTableReference(filename string, ref tableRef, fields []string, rowCount int) string {
	return fmt.Sprintf(`> **Embedded Lark Base table imported**
>
> - Rows: %d
> - Fields: %s
> - Original snapshot: [%s](%s)
> - Availability: Preserved for audit; WikiLoop indexes the deduplicated dataset below.
> - Lark table ID: %s
`, rowCount, strings.Join(fields, ", "), filename, filename, ref.TableID)
}

func renderDatasetSummary(filename string, total, unique, removed int) string {
	return fmt.Sprintf(`

# Deduplicated imported records

- Source rows: %d
- Unique records: %d
- Duplicate submissions removed: %d
- Searchable dataset: [%s](%s)
`, total, unique, removed, filename, filename)
}

func renderTableDataset(url string, ref tableRef, fields []string, rows [][]interface{}) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "Source URL: %s\nLark table ID: %s\nRows: %d\n\n",
		url, ref.TableID, len(rows))
	b.WriteString(strings.Join(fields, "\t"))
	b.WriteByte('\n')
	for _, row := range rows {
		cells := make([]string, len(row))
		for i, cell := range row {
			cells[i] = cleanCell(cell)
		}
		b.WriteString(strings.Join(cells, "\t"))
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func deduplicateTables(tables []importedTable) ([]string, [][]string) {
	var fields []string
	fieldSeen := map[string]bool{}
	for _, table := range tables {
		for _, field := range table.Fields {
			if !fieldSeen[field] {
				fieldSeen[field] = true
				fields = append(fields, field)
			}
		}
	}
	fields = append(fields, "来源表格")

	type canonicalRow struct {
		values    map[string]string
		urlKey    string
		submitKey string
		topicID   int64
	}
	var records []canonicalRow
	byURL := map[string]int{}
	bySubmission := map[string]int{}

	for _, table := range tables {
		for _, rawRow := range table.Rows {
			values := map[string]string{"来源表格": table.Ref.TableID}
			for i, field := range table.Fields {
				if i < len(rawRow) {
					values[field] = cleanCell(rawRow[i])
				}
			}
			url := valueByField(values, "链接", "url", "link")
			title := valueByField(values, "标题", "title")
			nickname := valueByField(values, "昵称", "nickname", "作者", "author", "用户", "user", "name")
			urlKey := normalizeURL(url)
			submitKey := normalizeText(nickname) + "\x00" + normalizeText(title)
			if title == "" || nickname == "" {
				submitKey = ""
			}
			candidate := canonicalRow{
				values:    values,
				urlKey:    urlKey,
				submitKey: submitKey,
				topicID:   topicID(url),
			}

			existing := -1
			if urlKey != "" {
				if idx, ok := byURL[urlKey]; ok {
					existing = idx
				}
			}
			if existing < 0 && submitKey != "" {
				if idx, ok := bySubmission[submitKey]; ok {
					existing = idx
				}
			}
			if existing >= 0 {
				if candidate.topicID > records[existing].topicID {
					old := records[existing]
					if old.urlKey != "" {
						delete(byURL, old.urlKey)
					}
					records[existing] = candidate
					if urlKey != "" {
						byURL[urlKey] = existing
					}
					if submitKey != "" {
						bySubmission[submitKey] = existing
					}
				}
				continue
			}

			idx := len(records)
			records = append(records, candidate)
			if urlKey != "" {
				byURL[urlKey] = idx
			}
			if submitKey != "" {
				bySubmission[submitKey] = idx
			}
		}
	}

	rows := make([][]string, 0, len(records))
	for _, record := range records {
		row := make([]string, len(fields))
		for i, field := range fields {
			row[i] = record.values[field]
		}
		rows = append(rows, row)
	}
	return fields, rows
}

func valueByField(values map[string]string, candidates ...string) string {
	for field, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(field))
		for _, candidate := range candidates {
			if strings.Contains(normalized, candidate) {
				return value
			}
		}
	}
	return ""
}

func normalizeURL(value string) string {
	value = strings.TrimSpace(value)
	if match := regexp.MustCompile(`\((https?://[^)]+)\)`).FindStringSubmatch(value); len(match) == 2 {
		value = match[1]
	}
	return strings.ToLower(strings.TrimRight(value, "/"))
}

func normalizeText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func topicID(value string) int64 {
	match := regexp.MustCompile(`/topic/(\d+)`).FindStringSubmatch(value)
	if len(match) != 2 {
		return 0
	}
	n, _ := strconv.ParseInt(match[1], 10, 64)
	return n
}

func renderDeduplicatedDataset(url string, fields []string, rows [][]string, total, removed int) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "Source URL: %s\nSource rows: %d\nUnique records: %d\nDuplicates removed: %d\n\n",
		url, total, len(rows), removed)
	b.WriteString(strings.Join(fields, "\t"))
	b.WriteByte('\n')
	for _, row := range rows {
		b.WriteString(strings.Join(row, "\t"))
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func cleanCell(value interface{}) string {
	var text string
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		text = v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			text = fmt.Sprint(v)
		} else {
			text = string(data)
		}
	}
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.TrimSpace(text)
}
