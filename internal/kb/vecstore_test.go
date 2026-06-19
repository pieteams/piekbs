//go:build fts5

package kb

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenVecStore_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	vs, err := OpenVecStore(dir)
	if err != nil {
		t.Fatalf("OpenVecStore: %v", err)
	}
	if vs == nil {
		t.Fatal("expected non-nil VecStore")
	}
	if _, err := os.Stat(filepath.Join(dir, "index", "vectors")); err != nil {
		t.Errorf("index/vectors dir not created: %v", err)
	}
}

func TestVecStore_UpsertAndQuery(t *testing.T) {
	dir := t.TempDir()
	vs, err := OpenVecStore(dir)
	if err != nil {
		t.Fatalf("OpenVecStore: %v", err)
	}

	// Insert two 4-dim vectors with different layers.
	vec1 := []float32{1, 0, 0, 0}
	vec2 := []float32{0, 1, 0, 0}
	if err := vs.Upsert("doc1", vec1, map[string]string{
		"layer": "wiki", "path": "wiki/a.md", "kind": "", "title": "A", "description": "",
	}); err != nil {
		t.Fatalf("Upsert doc1: %v", err)
	}
	if err := vs.Upsert("doc2", vec2, map[string]string{
		"layer": "raw", "path": "raw/b.md", "kind": "", "title": "B", "description": "",
	}); err != nil {
		t.Fatalf("Upsert doc2: %v", err)
	}

	if vs.Count() != 2 {
		t.Errorf("Count() = %d, want 2", vs.Count())
	}

	// Query with vec1 — doc1 should be top result.
	results, err := vs.Query(vec1, nil, 2)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	if results[0].ID != "doc1" {
		t.Errorf("top result ID = %q, want doc1", results[0].ID)
	}
}

func TestVecStore_LayerFilter(t *testing.T) {
	dir := t.TempDir()
	vs, _ := OpenVecStore(dir)

	vec := []float32{1, 0, 0, 0}
	_ = vs.Upsert("doc1", vec, map[string]string{"layer": "wiki", "path": "wiki/a.md", "kind": "", "title": "A", "description": ""})
	_ = vs.Upsert("doc2", vec, map[string]string{"layer": "raw", "path": "raw/b.md", "kind": "", "title": "B", "description": ""})

	layer := "wiki"
	results, _ := vs.Query(vec, &layer, 5)
	for _, r := range results {
		if r.Layer != "wiki" {
			t.Errorf("got layer %q, want wiki", r.Layer)
		}
	}
}

func TestVecStoreExists(t *testing.T) {
	dir := t.TempDir()
	if VecStoreExists(dir) {
		t.Error("should not exist before OpenVecStore")
	}
	vs, _ := OpenVecStore(dir)
	_ = vs.Upsert("x", []float32{1, 0}, map[string]string{"layer": "wiki", "path": "", "kind": "", "title": "", "description": ""})
	if !VecStoreExists(dir) {
		t.Error("should exist after upsert")
	}
}

func makeTempKBWithDocs(t *testing.T) (string, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"wiki", "raw"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(dir, "wiki", "a.md"), []byte("# Hello\nworld"), 0o644)
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	if _, err := IndexFiles(db, dir); err != nil {
		t.Fatalf("IndexFiles: %v", err)
	}
	return dir, db
}

type fixedEmbedder struct{ dim int }

func (f fixedEmbedder) Encode(text string) ([]float32, error) {
	v := make([]float32, f.dim)
	if len(text) > 0 {
		v[0] = float32(text[0]) / 255.0
	}
	return v, nil
}
func (f fixedEmbedder) Dimension() int { return f.dim }

func TestEmbedDocuments_WritesVecStore(t *testing.T) {
	dir, db := makeTempKBWithDocs(t)
	defer db.Close()

	emb := fixedEmbedder{dim: 4}
	written, skipped, err := EmbedDocuments(db, dir, emb, "test-model", false)
	if err != nil {
		t.Fatalf("EmbedDocuments: %v", err)
	}
	if written == 0 {
		t.Errorf("written=0, want >0 (skipped=%d)", skipped)
	}
	if !VecStoreExists(dir) {
		t.Error("VecStore should exist after EmbedDocuments")
	}
}

func TestVecSearch_ReturnsResults(t *testing.T) {
	dir, db := makeTempKBWithDocs(t)
	defer db.Close()

	emb := fixedEmbedder{dim: 4}
	if _, _, err := EmbedDocuments(db, dir, emb, "test-model", false); err != nil {
		t.Fatalf("EmbedDocuments: %v", err)
	}

	queryVec, _ := emb.Encode("hello")
	results, err := VecSearch(dir, queryVec, nil, 5)
	if err != nil {
		t.Fatalf("VecSearch: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least one VecSearch result")
	}
}

func TestEmbedDocuments_FullRebuild(t *testing.T) {
	dir, db := makeTempKBWithDocs(t)
	defer db.Close()

	emb := fixedEmbedder{dim: 4}
	// First pass
	if _, _, err := EmbedDocuments(db, dir, emb, "test-model", false); err != nil {
		t.Fatal(err)
	}
	// Full rebuild should not error
	written, _, err := EmbedDocuments(db, dir, emb, "test-model", true)
	if err != nil {
		t.Fatalf("full rebuild: %v", err)
	}
	if written == 0 {
		t.Error("full rebuild should re-write all docs")
	}
}

func TestOpenDB_StandardDriver(t *testing.T) {
	dir := t.TempDir()
	db, err := OpenDB(dir)
	if err != nil {
		t.Fatalf("OpenDB with standard driver: %v", err)
	}
	defer db.Close()

	// FTS5 must work — this is what the fts5 build tag enables.
	if _, err := db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS _test_fts USING fts5(content)"); err != nil {
		t.Errorf("FTS5 not available: %v", err)
	}
	_, _ = db.Exec("DROP TABLE IF EXISTS _test_fts")
}
