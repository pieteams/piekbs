//go:build fts5

package kb

import (
	"os"
	"path/filepath"
	"testing"
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
