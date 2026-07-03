//go:build fts5

package convert

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// CSVParser extracts text from .csv files, formatted as a readable table.
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
	var text strings.Builder
	for i, row := range records {
		if i == 0 {
			text.WriteString("Headers: " + strings.Join(row, " | ") + "\n\n")
			continue
		}
		text.WriteString(strings.Join(row, " | ") + "\n")
		if i >= 1000 {
			text.WriteString(fmt.Sprintf("\n... (%d more rows truncated)\n", len(records)-1000))
			break
		}
	}
	return strings.TrimSpace(text.String()), nil
}
