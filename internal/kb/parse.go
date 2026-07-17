//go:build fts5

package kb

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	fmRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?`)
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
	KeyClaims   []string
	Authority    int   // 1-5, default 3
	DocTimestamp int64 // Unix timestamp from frontmatter `timestamp` field; 0 if absent
	RawFM        map[string]interface{}
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

	pd.Authority = 3 // default
	if v, ok := fm["authority"].(string); ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 5 {
			pd.Authority = n
		}
	}

	// Parse doc_timestamp from frontmatter `timestamp` field (ISO 8601).
	if v, ok := fm["timestamp"].(string); ok && v != "" {
		for _, layout := range []string{
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05-07:00",
			"2006-01-02",
		} {
			if t, err := time.Parse(layout, v); err == nil {
				pd.DocTimestamp = t.Unix()
				break
			}
		}
	}

	pd.Tags = asStringList(fm["tags"])
	pd.Sources = asStringList(fm["sources"])
	pd.Contradicts = asStringList(fm["contradicts"])
	pd.Supersedes = asStringList(fm["supersedes"])
	pd.Supports = asStringList(fm["supports"])
	pd.RelatedTo = asStringList(fm["related_to"])
	pd.KeyClaims = asStringList(fm["key_claims"])

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
			item := strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "  - "), "- ")), `"'`)
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
			rawVal := strings.TrimSpace(trimmed[idx+1:])
			val := strings.Trim(rawVal, `"'`)

			// Quoted empty string ("" or '') is a scalar empty value, not a list.
			isQuotedEmpty := (rawVal == `""` || rawVal == `''`)

			if val == "[]" {
				// Explicit empty list.
				currentKey = key
				currentList = []string{}
				result[key] = currentList
			} else if val == "" && !isQuotedEmpty {
				// No inline value — next lines may be list items.
				currentKey = key
				currentList = []string{}
				result[key] = currentList
			} else if strings.HasPrefix(rawVal, "[") {
				// Inline JSON-style list: ["a", "b", ...] — store as string,
				// do NOT enter list mode so subsequent lines are parsed normally.
				currentKey = key
				currentList = nil
				result[key] = rawVal
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
// Handles nil, []string, inline JSON array string (["a","b"]), and comma-separated string.
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
		// Inline JSON array: ["a", "b", ...] or [a, b, ...]
		if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
			inner := s[1 : len(s)-1]
			parts := strings.Split(inner, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.Trim(p, `"'`)
				if p != "" {
					out = append(out, p)
				}
			}
			return out
		}
		return strings.Split(s, ",")
	}
	return nil
}

// SetFrontmatterSources rewrites the sources: field in YAML frontmatter
// to the given list. Existing sources (inline or block form) are removed.
// If frontmatter is absent or sources is empty, text is returned unchanged.
func SetFrontmatterSources(text string, sources []string) string {
	if len(sources) == 0 || !strings.HasPrefix(text, "---\n") {
		return text
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return text
	}
	fmEnd := 4 + end
	// fmEnd points at the '\n' before closing '---'; rest skips '\n---\n'
	fm, rest := text[4:fmEnd], text[fmEnd+4:]

	var out []string
	skipping := false
	for _, line := range strings.Split(fm, "\n") {
		trimmed := strings.TrimSpace(line)
		if skipping {
			if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
				continue
			}
			skipping = false
		}
		if val, ok := strings.CutPrefix(trimmed, "sources:"); ok {
			if strings.TrimSpace(val) == "" {
				skipping = true
			}
			continue
		}
		out = append(out, line)
	}

	var srcLines []string
	for _, s := range sources {
		srcLines = append(srcLines, "  - "+s)
	}
	newFM := strings.TrimRight(strings.Join(out, "\n"), "\n")
	return "---\n" + newFM + "\nsources:\n" + strings.Join(srcLines, "\n") + "\n---" + rest
}
