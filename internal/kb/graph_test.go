//go:build fts5

package kb

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func setupGraphTest(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := setupTestKB(t)

	db, err := OpenDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	docs := []struct {
		path    string
		content string
	}{
		{
			filepath.Join(dir, "wiki", "concepts", "a.md"),
			"---\ntitle: A\nkind: concept\nsources:\n  - raw/x.md\n---\nBody [[B]]",
		},
		{
			filepath.Join(dir, "wiki", "concepts", "b.md"),
			"---\ntitle: B\nkind: concept\ncontradicts:\n  - wiki/concepts/a.md\n---\nContradicts A",
		},
		{
			filepath.Join(dir, "wiki", "concepts", "c.md"),
			"---\ntitle: C\nkind: concept\nsupports:\n  - wiki/concepts/a.md\n---\nSupports A",
		},
		{
			filepath.Join(dir, "raw", "x.md"),
			"---\ntitle: Raw X\nkind: source-note\n---\nRaw content",
		},
	}

	for _, d := range docs {
		if err := os.MkdirAll(filepath.Dir(d.path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(d.path, []byte(d.content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := IndexFiles(db, dir); err != nil {
		t.Fatal(err)
	}

	return dir, db
}

func TestGraphBoost(t *testing.T) {
	_, db := setupGraphTest(t)

	// a.md is cited by c.md (supports) and wikilinked by b (via [[B]] in a → no, but c supports a)
	// a.md is target of: c.md's supports link → inbound edge exists
	aID := "wiki/concepts/a.md"
	boosts := GraphBoost(db, []string{aID})

	if len(boosts) == 0 {
		t.Fatal("expected boost map to have entry for a.md")
	}
	v, ok := boosts[aID]
	if !ok {
		t.Fatalf("no boost for %q, got map: %v", aID, boosts)
	}
	if v <= 0 {
		t.Errorf("boost for %q = %v, want > 0", aID, v)
	}
	if v > 1.0 {
		t.Errorf("boost for %q = %v, want <= 1.0", aID, v)
	}
}

func TestGraphExpand(t *testing.T) {
	_, db := setupGraphTest(t)

	aID := "wiki/concepts/a.md"
	neighbors := GraphExpand(db, []string{aID}, 20)

	if len(neighbors) == 0 {
		t.Errorf("expected neighbors for %q, got none", aID)
	}

	// Seed should not appear in neighbors
	for _, n := range neighbors {
		if n.ID == aID {
			t.Errorf("seed %q should not appear in neighbors", aID)
		}
	}
}

func TestConflictLinks(t *testing.T) {
	_, db := setupGraphTest(t)

	// b.md contradicts a.md
	ids := []string{"wiki/concepts/a.md", "wiki/concepts/b.md"}
	conflicts := ConflictLinks(db, ids)

	if len(conflicts) == 0 {
		t.Errorf("expected at least one conflict between a and b, got none")
	}

	found := false
	for _, c := range conflicts {
		if c.Relation == "contradicts" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a 'contradicts' conflict, got: %v", conflicts)
	}
}

func TestFetchRelated(t *testing.T) {
	db := newTestDB(t)
	// Insert two docs and a link between them (content/content_hash/updated_at required)
	now := int64(1700000000)
	db.Exec(`INSERT INTO documents(id,path,layer,kind,title,content,content_hash,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		"wiki/source-notes/a.md", "wiki/source-notes/a.md", "wiki", "source-note", "Doc A", "body", "hash1", now)
	db.Exec(`INSERT INTO documents(id,path,layer,kind,title,content,content_hash,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		"wiki/concepts/b.md", "wiki/concepts/b.md", "wiki", "concept", "Doc B", "body", "hash2", now)
	db.Exec(`INSERT INTO links(source_doc_id,target_doc_id,relation,confidence) VALUES(?,?,?,?)`,
		"wiki/source-notes/a.md", "wiki/concepts/b.md", "related_to", 1.0)

	related := FetchRelated(db, "wiki/source-notes/a.md", 3)
	if len(related) != 1 {
		t.Fatalf("expected 1 related doc, got %d", len(related))
	}
	if related[0].ID != "wiki/concepts/b.md" {
		t.Fatalf("expected 'wiki/concepts/b.md', got %q", related[0].ID)
	}
	if related[0].Kind != "concept" {
		t.Fatalf("expected kind 'concept', got %q", related[0].Kind)
	}
}
