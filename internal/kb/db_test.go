//go:build fts5

package kb

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestOpenDB_CreatesSchema(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	tables := []string{"documents", "links", "document_fts", "embeddings"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE name = ?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestOpenDB_CreatesIndexDir(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	indexDir := filepath.Join(dir, "index")
	if _, err := os.Stat(indexDir); os.IsNotExist(err) {
		t.Error("index/ directory not created")
	}
}

func TestOpenDB_MigratesDescriptionColumn(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	db2, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	rows, _ := db2.Query("PRAGMA table_info(documents)")
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt interface{}
		var pk int
		rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		if name == "description" {
			found = true
		}
	}
	db2.Close()
	if !found {
		t.Error("description column not found after migration")
	}
}

func TestDistillQueueTableExists(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var name string
	err = db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='distill_queue'",
	).Scan(&name)
	if err != nil {
		t.Fatalf("distill_queue table not found: %v", err)
	}
	if name != "distill_queue" {
		t.Errorf("expected distill_queue, got %q", name)
	}
}

func TestDistillQueueMigration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "index", "kb.sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}

	// 建一个没有 distill_queue 的旧库
	oldDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = oldDB.Exec(`CREATE TABLE IF NOT EXISTS documents (
        id TEXT PRIMARY KEY, path TEXT NOT NULL, layer TEXT NOT NULL,
        kind TEXT, title TEXT, description TEXT, content TEXT NOT NULL,
        content_hash TEXT NOT NULL, source_uri TEXT,
        updated_at INTEGER NOT NULL, authority INTEGER NOT NULL DEFAULT 3,
        doc_timestamp INTEGER NOT NULL DEFAULT 0)`)
	if err != nil {
		t.Fatal(err)
	}
	oldDB.Close()

	// 用 OpenDB 打开，触发迁移
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var name string
	err = db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='table' AND name='distill_queue'",
	).Scan(&name)
	if err != nil || name != "distill_queue" {
		t.Fatalf("migration failed: distill_queue not found: %v", err)
	}
}

func TestContentHash(t *testing.T) {
	h1 := ContentHash("hello world")
	h2 := ContentHash("hello world")
	h3 := ContentHash("different")
	if h1 != h2 {
		t.Errorf("same input should produce same hash: %q != %q", h1, h2)
	}
	if h1 == h3 {
		t.Errorf("different input should produce different hash")
	}
	if len(h1) != 16 {
		t.Errorf("hash length = %d, want 16", len(h1))
	}
}
