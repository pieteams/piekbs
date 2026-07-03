//go:build fts5

package convert

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// CSVParser extracts text from .csv files, formatted as a Markdown table.
type CSVParser struct{}

func (p *CSVParser) Extensions() []string { return []string{".csv"} }

func (p *CSVParser) Extract(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("extract csv: %w", err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("extract csv: %w", err)
	}
	if len(records) == 0 {
		return "", nil
	}
	var text strings.Builder
	// Header row
	text.WriteString("| " + strings.Join(records[0], " | ") + " |\n")
	text.WriteString("|" + strings.Repeat("---|", len(records[0])) + "\n")
	// Data rows (max 1000)
	limit := len(records)
	if limit > 1001 { // 1000 data rows + 1 header
		limit = 1001
	}
	for i := 1; i < limit; i++ {
		text.WriteString("| " + strings.Join(records[i], " | ") + " |\n")
	}
	if len(records) > 1001 {
		text.WriteString(fmt.Sprintf("\n... (%d more rows truncated)\n", len(records)-1001))
	}
	return strings.TrimSpace(text.String()), nil
}
