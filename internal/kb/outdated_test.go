//go:build fts5

package kb

import (
	"testing"
)

func TestFindOutdatedNotes_DevVersion(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	paths, err := FindOutdatedNotes(db, "dev")
	if err != nil {
		t.Fatal(err)
	}
	if paths != nil {
		t.Errorf("expected nil for dev version, got %v", paths)
	}
}

func TestFindOutdatedNotes_NullVersion(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert a wiki source-note without distill_version.
	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp)
		VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Old', '', 'body', 'h1', 1, 3, 0)`)

	paths, err := FindOutdatedNotes(db, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "wiki/old.md" {
		t.Errorf("expected [wiki/old.md], got %v", paths)
	}
}

func TestFindOutdatedNotes_MatchingVersion(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, distill_version)
		VALUES ('wiki/new.md', 'wiki/new.md', 'wiki', 'source-note', 'New', '', 'body', 'h2', 1, 3, 0, '1.0.0')`)

	paths, err := FindOutdatedNotes(db, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Errorf("expected empty, got %v", paths)
	}
}

func TestFindOutdatedNotes_DifferentVersion(t *testing.T) {
	dir := setupTestKB(t)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.Exec(`INSERT INTO documents (id, path, layer, kind, title, description, content, content_hash, updated_at, authority, doc_timestamp, distill_version)
		VALUES ('wiki/old.md', 'wiki/old.md', 'wiki', 'source-note', 'Old', '', 'body', 'h3', 1, 3, 0, '0.3.0')`)

	paths, err := FindOutdatedNotes(db, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "wiki/old.md" {
		t.Errorf("expected [wiki/old.md], got %v", paths)
	}
}
