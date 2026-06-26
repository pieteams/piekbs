//go:build fts5

package kb

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// OpenDB opens (or creates) kb.sqlite under kbRoot/index/.
func OpenDB(kbRoot string) (*sql.DB, error) {
	indexDir := filepath.Join(kbRoot, "index")
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return nil, fmt.Errorf("create index dir: %w", err)
	}

	dbPath := filepath.Join(indexDir, "kb.sqlite")

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	// Clean up legacy sqlite-vec virtual table if present from a previous install.
	_, _ = db.Exec("DROP TABLE IF EXISTS vec_documents")

	if err := migrateDescription(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func migrateDescription(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(documents)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasDescription := false
	hasAuthority := false
	hasDocTimestamp := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "description" {
			hasDescription = true
		}
		if name == "authority" {
			hasAuthority = true
		}
		if name == "doc_timestamp" {
			hasDocTimestamp = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasDescription {
		if _, err := db.Exec("ALTER TABLE documents ADD COLUMN description TEXT"); err != nil {
			return err
		}
	}
	if !hasAuthority {
		if _, err := db.Exec("ALTER TABLE documents ADD COLUMN authority INTEGER NOT NULL DEFAULT 3"); err != nil {
			return err
		}
	}
	if !hasDocTimestamp {
		if _, err := db.Exec("ALTER TABLE documents ADD COLUMN doc_timestamp INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS distill_queue (
    path        TEXT PRIMARY KEY,
    status      TEXT NOT NULL DEFAULT 'pending',
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error  TEXT,
    queued_at   INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
)`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS document_tags (
    doc_id  TEXT NOT NULL,
    tag     TEXT NOT NULL,
    source  TEXT NOT NULL DEFAULT 'tag',
    PRIMARY KEY (doc_id, tag),
    FOREIGN KEY (doc_id) REFERENCES documents(id)
)`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_document_tags_tag ON document_tags(tag)`); err != nil {
		return err
	}

	return nil
}

// ContentHash returns the first 16 hex chars of SHA-256 of text.
func ContentHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h)[:16]
}

// LayerCounts queries the number of documents per layer and the total.
// Returns (byLayer map, total, error). Used by both MCP and Web UI status.
func LayerCounts(db *sql.DB) (map[string]int, int, error) {
	rows, err := db.Query("SELECT layer, COUNT(*) FROM documents GROUP BY layer")
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	byLayer := map[string]int{}
	total := 0
	for rows.Next() {
		var layer string
		var n int
		if err := rows.Scan(&layer, &n); err != nil {
			return nil, 0, err
		}
		byLayer[layer] = n
		total += n
	}
	return byLayer, total, rows.Err()
}

// KindCounts queries document counts broken down by kind within the wiki layer.
// Returns map of kind→count for source-note, concept, comparison, decision.
func KindCounts(db *sql.DB) (map[string]int, error) {
	rows, err := db.Query(
		"SELECT COALESCE(kind,'source-note'), COUNT(*) FROM documents WHERE layer='wiki' GROUP BY kind")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{
		"source-note": 0,
		"concept":     0,
		"comparison":  0,
		"decision":    0,
	}
	for rows.Next() {
		var kind string
		var n int
		if err := rows.Scan(&kind, &n); err != nil {
			return nil, err
		}
		counts[kind] = n
	}
	return counts, rows.Err()
}
