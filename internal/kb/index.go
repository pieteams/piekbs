//go:build fts5

package kb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var textExtensions = map[string]bool{".md": true, ".txt": true, ".rst": true, ".tsv": true, ".csv": true}
var okfReserved = map[string]bool{"index.md": true, "log.md": true}

// DocID returns the relative slash-separated path used as document primary key.
func DocID(kbRoot string, path string) string {
	rel, _ := filepath.Rel(kbRoot, path)
	return filepath.ToSlash(rel)
}

// Layer returns the KB layer (raw/wiki/schema) for a given file path.
func Layer(kbRoot string, path string) string {
	rel, _ := filepath.Rel(kbRoot, path)
	parts := strings.SplitN(rel, string(filepath.Separator), 2)
	if len(parts) == 0 {
		return "raw"
	}
	switch parts[0] {
	case "raw", "wiki", "schema":
		return parts[0]
	default:
		return "raw"
	}
}

// IndexFiles walks kbRoot and upserts changed documents into db (incremental:
// files whose content_hash is unchanged are skipped).
// Returns the number of files written (new or updated).
func IndexFiles(db *sql.DB, kbRoot string) (int, error) {
	return indexFiles(db, kbRoot, false)
}

// IndexFilesFull re-indexes every document regardless of content_hash, forcing
// re-parse and FTS rebuild. Stale documents are still purged.
// Returns the number of files written.
func IndexFilesFull(db *sql.DB, kbRoot string) (int, error) {
	return indexFiles(db, kbRoot, true)
}

func indexFiles(db *sql.DB, kbRoot string, force bool) (int, error) {
	currentIDs := make(map[string]bool)
	written := 0

	err := filepath.Walk(kbRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, _ := filepath.Rel(kbRoot, path)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		if len(parts) == 0 || (parts[0] != "raw" && parts[0] != "wiki" && parts[0] != "schema") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !textExtensions[ext] {
			return nil
		}
		if okfReserved[info.Name()] {
			return nil
		}
		// skip synthesized draft pages (not enough source coverage yet)
		if strings.Contains(rel, string(filepath.Separator)+"_draft"+string(filepath.Separator)) ||
			strings.HasSuffix(filepath.Dir(rel), "_draft") {
			return nil
		}

		did := DocID(kbRoot, path)
		currentIDs[did] = true

		changed, err := upsertDocument(db, kbRoot, path, did, force)
		if err != nil {
			return fmt.Errorf("index %s: %w", rel, err)
		}
		if changed {
			written++
		}
		return nil
	})
	if err != nil {
		return written, err
	}

	purged, err := purgeStale(db, kbRoot, currentIDs)
	if err != nil {
		return written, fmt.Errorf("purge: %w", err)
	}
	_ = purged

	// OKF maintenance: regenerate wiki index.md files and remove orphaned
	// generated files. Best-effort — failures here don't fail indexing.
	wikiDir := filepath.Join(kbRoot, "wiki")
	if _, statErr := os.Stat(wikiDir); statErr == nil {
		if genErr := GenerateOKFIndex(wikiDir); genErr != nil {
			return written, fmt.Errorf("okf index: %w", genErr)
		}
	}
	if _, purgeErr := PurgeOrphanWikiFiles(kbRoot); purgeErr != nil {
		return written, fmt.Errorf("purge orphans: %w", purgeErr)
	}

	return written, nil
}

func upsertDocument(db *sql.DB, kbRoot, path, did string, force bool) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	text := string(data)
	h := ContentHash(text)

	if !force {
		var existingHash sql.NullString
		err = db.QueryRow("SELECT content_hash FROM documents WHERE id = ?", did).Scan(&existingHash)
		if err == nil && existingHash.String == h {
			return false, nil
		}
	}

	parsed := ParseMarkdown(text)

	var distillVersion sql.NullString
	if v, ok := parsed.RawFM["distill_version"]; ok {
		if s, ok := v.(string); ok && s != "" {
			distillVersion = sql.NullString{String: s, Valid: true}
		}
	}

	layer := Layer(kbRoot, path)
	title := parsed.Title
	if title == "" {
		stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		title = strings.ReplaceAll(stem, "-", " ")
	}

	now := time.Now().Unix()
	rel, _ := filepath.Rel(kbRoot, path)

	// Fallback to file mtime when frontmatter has no timestamp.
	docTs := parsed.DocTimestamp
	if docTs == 0 {
		if fi, err := os.Stat(path); err == nil {
			docTs = fi.ModTime().Unix()
		}
	}

	_, err = db.Exec(`
		INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, source_uri, updated_at, authority, doc_timestamp, distill_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			path=excluded.path, layer=excluded.layer, kind=excluded.kind,
			title=excluded.title, description=excluded.description, content=excluded.content,
			content_hash=excluded.content_hash, source_uri=excluded.source_uri,
			updated_at=excluded.updated_at, authority=excluded.authority,
			doc_timestamp=excluded.doc_timestamp, distill_version=excluded.distill_version
	`, did, filepath.ToSlash(rel), layer, parsed.Kind, title, parsed.Description,
		text, h, nil, now, parsed.Authority, docTs, distillVersion)
	if err != nil {
		return false, err
	}

	upsertLinks(db, did, parsed)
	upsertDocumentTags(db, did, parsed.KeyClaims)
	return true, nil
}

func upsertLinks(db *sql.DB, docID string, parsed *ParsedDocument) {
	db.Exec("DELETE FROM links WHERE source_doc_id = ?", docID)

	for _, src := range parsed.Sources {
		db.Exec("INSERT OR IGNORE INTO links (source_doc_id, target_doc_id, relation, confidence) VALUES (?, ?, ?, ?)",
			docID, src, "cites", 1.0)
	}
	for _, wl := range parsed.Wikilinks {
		slug := strings.ToLower(strings.ReplaceAll(wl, " ", "-"))
		db.Exec("INSERT OR IGNORE INTO links (source_doc_id, target_doc_id, relation, confidence) VALUES (?, ?, ?, ?)",
			docID, slug, "wikilink", 0.9)
	}
	for _, rel := range []struct {
		key    string
		values []string
	}{
		{"contradicts", parsed.Contradicts},
		{"supersedes", parsed.Supersedes},
		{"supports", parsed.Supports},
		{"related_to", parsed.RelatedTo},
	} {
		for _, target := range rel.values {
			db.Exec("INSERT OR IGNORE INTO links (source_doc_id, target_doc_id, relation, confidence) VALUES (?, ?, ?, ?)",
				docID, target, rel.key, 1.0)
		}
	}
}

// knownTypes is the set of entity type words used in 【name|type】 notation.
// Entities whose name matches a type word are noise (e.g. 【技术|库】) and skipped.
var knownTypes = map[string]bool{
	"技术": true, "概念": true, "组织": true, "人物": true,
	"产品": true, "项目": true, "地点": true,
}

// entityRe matches 【name|type】 inline entity annotations.
var entityRe = regexp.MustCompile(`【([^|】]+)\|([^】]+)】`)

// tagNoiseRe matches tags that are noise: URLs, WeChat metadata, HTML entities, lone punctuation.
var tagNoiseRe = regexp.MustCompile(`(?i)https?://|mmbiz|appmsg|wx_fmt|chksm|bizuin|^http|^nbsp$|^sources$|^pattern$|^from$|^wiki\]$|^\[`)
var asciiLowerOnlyRe = regexp.MustCompile(`^[a-z]+$`)

// isNoisyTag reports whether a tag should be filtered out.
func isNoisyTag(t string) bool {
	if utf8.RuneCountInString(t) > 20 {
		return true
	}
	// Pure ASCII lowercase short words that are HTML/Markdown artifacts
	if len(t) <= 8 && t == strings.ToLower(t) && asciiLowerOnlyRe.MatchString(t) {
		switch t {
		case "nbsp", "sources", "pattern", "from", "wiki", "tag", "type", "true", "false", "null":
			return true
		}
	}
	return tagNoiseRe.MatchString(t)
}

var asciiStartRe = regexp.MustCompile(`^[a-zA-Z]`)

// normalizeTag normalizes a tag for consistent matching:
// - trims whitespace
// - title-cases the first rune for all-lowercase ASCII tags (e.g. "karpathy" → "Karpathy")
func normalizeTag(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return t
	}
	if t == strings.ToLower(t) && asciiStartRe.MatchString(t) {
		r := []rune(t)
		r[0] = unicode.ToUpper(r[0])
		return string(r)
	}
	return t
}

// upsertDocumentTags replaces all document_tags for docID using only entities
// extracted from key_claims (source='claim'). key_claims entities are authored
// by the LLM to represent the core named entities in each claim, so they have
// much higher precision than full-text entity annotations or frontmatter tags.
func upsertDocumentTags(db *sql.DB, docID string, claims []string) {
	db.Exec("DELETE FROM document_tags WHERE doc_id = ?", docID) //nolint:errcheck

	for _, claim := range claims {
		for _, m := range entityRe.FindAllStringSubmatch(claim, -1) {
			name := normalizeTag(m[1])
			if name == "" || utf8.RuneCountInString(name) < 2 || knownTypes[name] || isNoisyTag(name) {
				continue
			}
			db.Exec("INSERT OR IGNORE INTO document_tags (doc_id, tag, source) VALUES (?, ?, 'claim')", docID, name) //nolint:errcheck
		}
	}
}

func purgeStale(db *sql.DB, kbRoot string, currentIDs map[string]bool) (int, error) {
	rows, err := db.Query("SELECT id FROM documents")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var stale []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		if !currentIDs[id] {
			stale = append(stale, id)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(stale) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(stale))
	args := make([]interface{}, len(stale))
	for i, id := range stale {
		placeholders[i] = "?"
		args[i] = id
	}
	ph := strings.Join(placeholders, ",")

	db.Exec(fmt.Sprintf("DELETE FROM embeddings WHERE doc_id IN (%s)", ph), args...)
	db.Exec(fmt.Sprintf("DELETE FROM document_tags WHERE doc_id IN (%s)", ph), args...) //nolint:errcheck
	db.Exec(fmt.Sprintf("DELETE FROM links WHERE source_doc_id IN (%s) OR target_doc_id IN (%s)", ph, ph), append(args, args...)...)
	db.Exec(fmt.Sprintf("DELETE FROM documents WHERE id IN (%s)", ph), args...)

	return len(stale), nil
}
