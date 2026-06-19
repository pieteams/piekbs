//go:build fts5

package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateOKFIndex writes an OKF index.md into wikiDir and each of its
// subdirectories that contain markdown pages. Reserved files (index.md, log.md)
// are excluded from listings. The root index.md carries an okf_version marker.
//
// Per-directory index.md lists `* [title](filename) — description` sorted by
// filename. The root additionally lists subdirectories with their page counts.
func GenerateOKFIndex(wikiDir string) error {
	// Collect entries grouped by their containing directory.
	type entry struct{ filename, title, description string }
	dirEntries := map[string][]entry{}

	err := filepath.Walk(wikiDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}
		if okfReserved[info.Name()] {
			return nil
		}
		parsed, perr := ParseMarkdownFile(path)
		if perr != nil {
			return nil
		}
		title := parsed.Title
		if title == "" {
			title = titleFromStem(info.Name())
		}
		dir := filepath.Dir(path)
		dirEntries[dir] = append(dirEntries[dir], entry{info.Name(), title, parsed.Description})
		return nil
	})
	if err != nil {
		return err
	}

	// Write per-directory index.md.
	for dir, entries := range dirEntries {
		sort.Slice(entries, func(i, j int) bool { return entries[i].filename < entries[j].filename })
		var b strings.Builder
		fmt.Fprintf(&b, "# %s\n\n", filepath.Base(dir))
		for _, e := range entries {
			if e.description != "" {
				fmt.Fprintf(&b, "* [%s](%s) — %s\n", e.title, e.filename, e.description)
			} else {
				fmt.Fprintf(&b, "* [%s](%s)\n", e.title, e.filename)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(b.String()), 0o644); err != nil {
			return err
		}
	}

	// Write the root index.md listing subdirectories with page counts.
	subdirs, err := okfSubdirsWithPages(wikiDir)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("---\nokf_version: \"0.1\"\n---\n\n# wiki\n\n")
	for _, sd := range subdirs {
		fmt.Fprintf(&b, "* [%s](%s/) — %d page(s)\n", sd.name, sd.name, sd.count)
	}
	return os.WriteFile(filepath.Join(wikiDir, "index.md"), []byte(b.String()), 0o644)
}

type okfSubdir struct {
	name  string
	count int
}

// okfSubdirsWithPages returns immediate subdirectories of wikiDir that contain
// at least one non-reserved markdown page, with the page count, sorted by name.
func okfSubdirsWithPages(wikiDir string) ([]okfSubdir, error) {
	dirents, err := os.ReadDir(wikiDir)
	if err != nil {
		return nil, err
	}
	var out []okfSubdir
	for _, de := range dirents {
		if !de.IsDir() {
			continue
		}
		count := countMarkdownPages(filepath.Join(wikiDir, de.Name()))
		if count > 0 {
			out = append(out, okfSubdir{de.Name(), count})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

// countMarkdownPages counts non-reserved .md files under dir recursively.
func countMarkdownPages(dir string) int {
	count := 0
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".md" && !okfReserved[info.Name()] {
			count++
		}
		return nil
	})
	return count
}

// PurgeOrphanWikiFiles removes generated files in wiki/source-notes/ and
// raw/converted/ whose filename stem has no corresponding file in raw/.
// Reserved files (index.md, log.md) are kept. Returns the number removed.
func PurgeOrphanWikiFiles(kbRoot string) (int, error) {
	rawDir := filepath.Join(kbRoot, "raw")
	// If raw/ is missing, the survivor set is unknowable — refuse to purge
	// rather than treat every wiki page as an orphan and delete it.
	if _, err := os.Stat(rawDir); err != nil {
		return 0, nil
	}

	rawStems := map[string]bool{}
	convertedDir := filepath.Join(rawDir, "converted")
	walkErr := filepath.Walk(rawDir, func(path string, info os.FileInfo, err error) error {
		// Propagate any walk error: an incomplete rawStems set would cause
		// real source-notes to be misclassified as orphans and deleted.
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// Don't treat generated converted/ output as a raw source.
		if strings.HasPrefix(path, convertedDir+string(filepath.Separator)) {
			return nil
		}
		stem := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		rawStems[stem] = true
		return nil
	})
	if walkErr != nil {
		return 0, fmt.Errorf("scan raw sources: %w", walkErr)
	}

	removed := 0
	for _, sub := range []string{filepath.Join("wiki", "source-notes"), filepath.Join("raw", "converted")} {
		d := filepath.Join(kbRoot, sub)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			continue
		}
		err := filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.ToLower(filepath.Ext(path)) != ".md" || okfReserved[info.Name()] {
				return nil
			}
			stem := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			// converted/ files are named "<rawname>.md" (stem == raw stem);
			// a converted .md whose stem matches a raw file is NOT orphan.
			if !rawStems[stem] {
				if rmErr := os.Remove(path); rmErr == nil {
					removed++
				}
			}
			return nil
		})
		if err != nil {
			return removed, err
		}
	}
	return removed, nil
}

// titleFromStem converts a filename to a display title: drops the extension,
// replaces '-' and '_' with spaces, and title-cases each word.
func titleFromStem(name string) string {
	stem := strings.TrimSuffix(name, filepath.Ext(name))
	stem = strings.NewReplacer("-", " ", "_", " ").Replace(stem)
	words := strings.Fields(stem)
	for i, w := range words {
		r := []rune(w)
		r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}
