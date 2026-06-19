//go:build fts5

package kb

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Embedder abstracts the embedding model so kb doesn't depend on embed package.
type Embedder interface {
	Encode(text string) ([]float32, error)
	Dimension() int
}

// EmbedDocuments encodes all documents and stores vectors in the VecStore.
// If full=true, deletes index/vectors/ and clears the embeddings table first.
// Returns (written, skipped, error).
func EmbedDocuments(db *sql.DB, kbRoot string, embedder Embedder, modelName string, full bool) (int, int, error) {
	// Detect divergence: vec store missing but embeddings table has rows.
	// This happens when index/vectors/ is deleted while the DB survives.
	// Without this guard, incremental embed would skip all docs permanently.
	if !full && !VecStoreExists(kbRoot) {
		var embCount int
		_ = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&embCount)
		if embCount > 0 {
			log.Printf("vec store missing but embeddings table has %d rows — forcing full rebuild", embCount)
			full = true
			if _, err := db.Exec("DELETE FROM embeddings"); err != nil {
				return 0, 0, fmt.Errorf("clear embeddings: %w", err)
			}
		}
	}

	if full {
		// Delete the chromem directory to force full rebuild.
		vecDir := filepath.Join(kbRoot, "index", "vectors")
		if err := os.RemoveAll(vecDir); err != nil {
			return 0, 0, fmt.Errorf("remove vec store: %w", err)
		}
		if _, err := db.Exec("DELETE FROM embeddings"); err != nil {
			return 0, 0, fmt.Errorf("clear embeddings: %w", err)
		}
	}

	vs, err := OpenVecStore(kbRoot)
	if err != nil {
		return 0, 0, fmt.Errorf("open vec store: %w", err)
	}

	rows, err := db.Query(
		"SELECT d.id, d.path, d.layer, COALESCE(d.kind,''), COALESCE(d.title,''), COALESCE(d.description,''), d.content FROM documents d",
	)
	if err != nil {
		return 0, 0, fmt.Errorf("query documents: %w", err)
	}
	defer rows.Close()

	type doc struct {
		id, path, layer, kind, title, description, content string
	}
	var docs []doc
	for rows.Next() {
		var d doc
		if err := rows.Scan(&d.id, &d.path, &d.layer, &d.kind, &d.title, &d.description, &d.content); err != nil {
			return 0, 0, fmt.Errorf("scan: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	written, skipped := 0, 0
	for _, d := range docs {
		if !full {
			var exists int
			_ = db.QueryRow("SELECT 1 FROM embeddings WHERE doc_id = ?", d.id).Scan(&exists)
			if exists == 1 {
				skipped++
				continue
			}
		}

		vec, err := embedder.Encode(d.content)
		if err != nil {
			log.Printf("embed: skip %s: %v", d.id, err)
			skipped++
			continue
		}

		meta := map[string]string{
			"layer":       d.layer,
			"path":        d.path,
			"kind":        d.kind,
			"title":       d.title,
			"description": d.description,
		}
		if err := vs.Upsert(d.id, vec, meta); err != nil {
			return written, skipped, fmt.Errorf("upsert vec %s: %w", d.id, err)
		}

		if _, err := db.Exec(`
			INSERT INTO embeddings(doc_id, model, dim, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(doc_id) DO UPDATE SET
				model=excluded.model, dim=excluded.dim, updated_at=excluded.updated_at`,
			d.id, modelName, embedder.Dimension(), time.Now().Unix(),
		); err != nil {
			return written, skipped, fmt.Errorf("insert embeddings: %w", err)
		}
		written++
	}
	return written, skipped, nil
}

// VecSearch runs cosine KNN using the VecStore.
// Returns nil (not error) if no vec store exists — callers degrade to FTS-only.
func VecSearch(kbRoot string, queryVec []float32, layer *string, limit int) ([]SearchResult, error) {
	if !VecStoreExists(kbRoot) {
		return nil, nil
	}
	vs, err := OpenVecStore(kbRoot)
	if err != nil {
		// Degrade gracefully.
		return nil, nil
	}
	return vs.Query(queryVec, layer, limit)
}
