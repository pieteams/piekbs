package larkimport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls int
}

func (f *fakeRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	f.calls++
	if len(args) >= 2 && args[0] == "docs" && args[1] == "+fetch" {
		return []byte(`{"ok":true,"data":{"document":{"content":"<title>Test &amp; Import</title>\n# Results\n<bitable table-id=\"tblA\" token=\"baseA\"></bitable>","document_id":"docA","revision_id":7}}}`), nil
	}
	if len(args) >= 2 && args[0] == "base" && args[1] == "+record-list" {
		offset := argValue(args, "--offset")
		switch offset {
		case "0":
			return []byte(`{"ok":true,"data":{"data":[["Title 1","Alice"],["Title 2","Bob"]],"fields":["Title","Name"],"has_more":true}}`), nil
		case "2":
			return []byte(`{"ok":true,"data":{"data":[["Title 3","Carol"]],"fields":["Title","Name"],"has_more":false}}`), nil
		}
	}
	return nil, fmt.Errorf("unexpected args: %v", args)
}

func TestImportExpandsBitableIntoSearchableDataset(t *testing.T) {
	kbRoot := t.TempDir()
	runner := &fakeRunner{}

	result, err := Import(context.Background(), kbRoot, "https://example.larkoffice.com/wiki/abc", "", runner)
	if err != nil {
		t.Fatal(err)
	}
	if result.DocumentPath != "raw/lark/test-import/document.md" {
		t.Fatalf("document path = %q", result.DocumentPath)
	}
	if len(result.TableRows) != 1 || result.TableRows[0] != 3 {
		t.Fatalf("table rows = %v, want [3]", result.TableRows)
	}
	if result.UniqueRows != 3 || result.DuplicatesRemoved != 0 {
		t.Fatalf("unique=%d removed=%d, want 3/0", result.UniqueRows, result.DuplicatesRemoved)
	}

	document, err := os.ReadFile(filepath.Join(kbRoot, filepath.FromSlash(result.DocumentPath)))
	if err != nil {
		t.Fatal(err)
	}
	docText := string(document)
	for _, want := range []string{
		`source_url: "https://example.larkoffice.com/wiki/abc"`,
		"Rows: 3",
		"table-01-tblA.snapshot.tsv",
		"records-deduplicated.txt",
	} {
		if !strings.Contains(docText, want) {
			t.Errorf("document missing %q:\n%s", want, docText)
		}
	}
	if strings.Contains(docText, "<bitable") {
		t.Fatal("document still contains an unexpanded bitable tag")
	}

	dataset, err := os.ReadFile(filepath.Join(kbRoot, filepath.FromSlash(result.DatasetPath)))
	if err != nil {
		t.Fatal(err)
	}
	dataText := string(dataset)
	if !strings.Contains(dataText, "Title\tName\t来源表格\n") || !strings.Contains(dataText, "Title 3\tCarol\ttblA") {
		t.Fatalf("dataset content is incomplete:\n%s", dataText)
	}
	if runner.calls != 3 {
		t.Fatalf("runner calls = %d, want 3", runner.calls)
	}
}

func TestDeduplicateTablesKeepsNewestRepeatedSubmission(t *testing.T) {
	tables := []importedTable{
		{
			Ref:    tableRef{TableID: "newer"},
			Fields: []string{"社区昵称", "创意帖链接", "创意帖标题"},
			Rows: [][]interface{}{
				{"alice", "https://forum.trae.cn/t/topic/100", "Same idea"},
				{"bob", "https://forum.trae.cn/t/topic/101", "Same idea"},
			},
		},
		{
			Ref:    tableRef{TableID: "older"},
			Fields: []string{"社区昵称", "创意帖链接", "创意帖标题"},
			Rows: [][]interface{}{
				{"alice", "https://forum.trae.cn/t/topic/90", "Same idea"},
				{"carol", "https://forum.trae.cn/t/topic/102", "Unique idea"},
			},
		},
	}

	fields, rows := deduplicateTables(tables)
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	joined := make([]string, len(rows))
	for i, row := range rows {
		joined[i] = strings.Join(row, "\t")
	}
	all := strings.Join(joined, "\n")
	if strings.Contains(all, "/topic/90") {
		t.Fatalf("older duplicate was retained:\n%s", all)
	}
	if !strings.Contains(all, "/topic/100") {
		t.Fatalf("newer duplicate was not retained:\n%s", all)
	}
	if !strings.Contains(all, "bob") || !strings.Contains(all, "carol") {
		t.Fatalf("same title from another user or unique row was removed:\n%s", all)
	}
	if fields[len(fields)-1] != "来源表格" {
		t.Fatalf("last field = %q, want 来源表格", fields[len(fields)-1])
	}
}

func TestParseTableRefsSupportsAttributeOrder(t *testing.T) {
	content := `<bitable table-id="one" token="base1"></bitable>
<bitable token="base2" table-id="two"></bitable>`
	refs := parseTableRefs(content)
	if len(refs) != 2 {
		t.Fatalf("refs = %d, want 2", len(refs))
	}
	if refs[0].TableID != "one" || refs[0].BaseToken != "base1" {
		t.Fatalf("first ref = %+v", refs[0])
	}
	if refs[1].TableID != "two" || refs[1].BaseToken != "base2" {
		t.Fatalf("second ref = %+v", refs[1])
	}
}

func argValue(args []string, name string) string {
	for i := range args {
		if args[i] == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return strconv.Itoa(-1)
}
