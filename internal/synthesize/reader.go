//go:build fts5

package synthesize

import (
	"os"
	"path/filepath"
	"strings"
)

// SourceNote holds the frontmatter fields extracted from a source-note page.
type SourceNote struct {
	Path        string   // wiki-root-relative path, e.g. wiki/source-notes/foo.md
	Title       string
	Description string
	Tags        []string
	KeyClaims   []string // core claims extracted during distillation
}

// LoadSourceNotes reads all source-note pages under wiki/source-notes/ in kbRoot
// and returns their frontmatter. index.md and log.md are skipped.
func LoadSourceNotes(kbRoot string) ([]SourceNote, error) {
	notesDir := filepath.Join(kbRoot, "wiki", "source-notes")
	var notes []SourceNote

	err := filepath.WalkDir(notesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		base := filepath.Base(path)
		if base == "index.md" || base == "log.md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		note := parseFrontmatter(data)
		rel, _ := filepath.Rel(kbRoot, path)
		note.Path = filepath.ToSlash(rel)
		notes = append(notes, note)
		return nil
	})
	return notes, err
}

// parseFrontmatter extracts title, description, and tags from YAML frontmatter.
// It uses simple line-by-line parsing — no external YAML library needed.
func parseFrontmatter(data []byte) SourceNote {
	var note SourceNote
	text := string(data)

	// Find frontmatter block between --- markers.
	if !strings.HasPrefix(text, "---") {
		return note
	}
	end := strings.Index(text[3:], "\n---")
	if end < 0 {
		return note
	}
	fm := text[3 : end+3]

	inTags := false
	inKeyClaims := false
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimRight(line, " \r")

		if strings.HasPrefix(line, "title:") {
			note.Title = strings.TrimSpace(stripQuotes(strings.TrimPrefix(line, "title:")))
			inTags, inKeyClaims = false, false
		} else if strings.HasPrefix(line, "description:") {
			note.Description = strings.TrimSpace(stripQuotes(strings.TrimPrefix(line, "description:")))
			inTags, inKeyClaims = false, false
		} else if strings.HasPrefix(line, "tags:") {
			inTags = true
			inKeyClaims = false
			// Inline list: tags: [a, b, c]
			val := strings.TrimSpace(strings.TrimPrefix(line, "tags:"))
			if strings.HasPrefix(val, "[") {
				note.Tags = parseInlineTags(val)
				inTags = false
			}
		} else if strings.HasPrefix(line, "key_claims:") {
			inKeyClaims = true
			inTags = false
			val := strings.TrimSpace(strings.TrimPrefix(line, "key_claims:"))
			if strings.HasPrefix(val, "[") {
				note.KeyClaims = parseInlineTags(val)
				inKeyClaims = false
			}
		} else if inTags && strings.HasPrefix(strings.TrimSpace(line), "-") {
			tag := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
			note.Tags = append(note.Tags, stripQuotes(tag))
		} else if inKeyClaims && strings.HasPrefix(strings.TrimSpace(line), "-") {
			claim := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
			note.KeyClaims = append(note.KeyClaims, stripQuotes(claim))
		} else if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inTags, inKeyClaims = false, false
		}
	}
	return note
}

func parseInlineTags(s string) []string {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	var tags []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(stripQuotes(t))
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func stripQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == s[len(s)-1] && (s[0] == '"' || s[0] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}
