//go:build fts5

package kb

import (
	"os"
	"regexp"
	"strings"
)

var (
	fmRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n`)
	wlRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
)

// ParsedDocument holds the parsed frontmatter and body of a markdown file.
type ParsedDocument struct {
	Title       string
	Kind        string
	Description string
	Tags        []string
	Sources     []string
	Content     string
	Wikilinks   []string
	Contradicts []string
	Supersedes  []string
	Supports    []string
	RelatedTo   []string
	RawFM       map[string]interface{}
}

// ParseMarkdownFile reads a file from disk and parses it.
func ParseMarkdownFile(path string) (*ParsedDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseMarkdown(string(data)), nil
}

// ParseMarkdown parses frontmatter and body from a markdown string.
func ParseMarkdown(text string) *ParsedDocument {
	pd := &ParsedDocument{RawFM: make(map[string]interface{})}

	m := fmRe.FindStringSubmatchIndex(text)
	if m != nil {
		fmText := text[m[2]:m[3]]
		pd.RawFM = parseYAMLSimple(fmText)
		pd.Content = strings.TrimSpace(text[m[1]:])
	} else {
		pd.Content = strings.TrimSpace(text)
	}

	fm := pd.RawFM
	if v, ok := fm["title"].(string); ok {
		pd.Title = v
	}
	if v, ok := fm["kind"].(string); ok {
		pd.Kind = v
	}
	if v, ok := fm["type"].(string); ok && pd.Kind == "" {
		pd.Kind = v
	}
	if v, ok := fm["description"].(string); ok {
		pd.Description = v
	}

	pd.Tags = asStringList(fm["tags"])
	pd.Sources = asStringList(fm["sources"])
	pd.Contradicts = asStringList(fm["contradicts"])
	pd.Supersedes = asStringList(fm["supersedes"])
	pd.Supports = asStringList(fm["supports"])
	pd.RelatedTo = asStringList(fm["related_to"])

	pd.Wikilinks = wlRe.FindAllString(pd.Content, -1)
	for i, wl := range pd.Wikilinks {
		pd.Wikilinks[i] = wl[2 : len(wl)-2]
	}

	return pd
}

// parseYAMLSimple is a minimal YAML parser for frontmatter.
// It handles: scalar key:value, list items with `- `, nested lists with `  - `.
func parseYAMLSimple(text string) map[string]interface{} {
	result := make(map[string]interface{})
	var currentKey string
	var currentList []string

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimRight(line, " ")
		if strings.HasPrefix(line, "  - ") || strings.HasPrefix(line, "- ") {
			item := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "  - "), "- "))
			if currentList != nil {
				currentList = append(currentList, item)
			}
			continue
		}
		if strings.Contains(trimmed, ":") {
			if currentList != nil && currentKey != "" {
				result[currentKey] = currentList
				currentList = nil
			}
			idx := strings.Index(trimmed, ":")
			key := strings.TrimSpace(trimmed[:idx])
			val := strings.TrimSpace(trimmed[idx+1:])
			val = strings.Trim(val, `"'`)

			if val == "" || val == "[]" {
				currentKey = key
				currentList = []string{}
				result[key] = currentList
			} else {
				currentKey = key
				currentList = nil
				result[key] = val
			}
		}
	}
	if currentList != nil && currentKey != "" {
		result[currentKey] = currentList
	}
	return result
}

// asStringList coerces a frontmatter value to []string.
// Handles nil, []string, and comma-separated string.
func asStringList(v interface{}) []string {
	if v == nil {
		return nil
	}
	if list, ok := v.([]string); ok {
		return list
	}
	if s, ok := v.(string); ok {
		if s == "" {
			return nil
		}
		return strings.Split(s, ",")
	}
	return nil
}
