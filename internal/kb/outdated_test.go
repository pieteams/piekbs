//go:build fts5

package kb

import (
	"testing"
)

func TestFindOutdatedNotes_ZeroVersion(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	paths, err := FindOutdatedNotes(db, 0)
	if err != nil {
		t.Fatal(err)
	}
	if paths != nil {
		t.Errorf("expected nil for version 0, got %v", paths)
	}
}

func TestFindOutdatedNotes_OlderVersion(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
		VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Old', '', 'body', 'h1', 1, 3, 0, 1)`)

	paths, err := FindOutdatedNotes(db, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "wiki/old.md" {
		t.Errorf("expected [wiki/old.md], got %v", paths)
	}
}

func TestFindOutdatedNotes_SameVersion(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, schema_version)
		VALUES ('wiki/new.md', 'wiki/new.md', 'wiki', 'source-note', 'New', '', 'body', 'h2', 1, 3, 0, 2)`)

	paths, err := FindOutdatedNotes(db, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty, got %v", paths)
	}
}
