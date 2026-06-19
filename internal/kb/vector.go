//go:build fts5

package kb

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// chunkDoc splits a document's content into sections for fine-grained embedding.
// Each chunk is prefixed with document context so the vector captures "this
// excerpt is about X in document Y" rather than a naked paragraph.
//
// Splitting strategy: split on "## " headings (level-2). If the document has no
// level-2 headings, fall back to a single chunk of the whole content.
// Each returned chunk is ready to pass to Embedder.Encode.
func chunkDoc(title, content string) []string {
	// Split on lines starting with "## "
	var sections []struct{ heading, body string }
	var cur struct{ heading, body string }
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## ") {
			if cur.body != "" || cur.heading != "" {
				sections = append(sections, cur)
			}
			cur = struct{ heading, body string }{heading: strings.TrimPrefix(line, "## "), body: ""}
		} else {
			if cur.body == "" {
				cur.body = line
			} else {
				cur.body += "\n" + line
			}
		}
	}
	if cur.body != "" || cur.heading != "" {
		sections = append(sections, cur)
	}

	if len(sections) <= 1 {
		// No meaningful sections — embed the whole content as one chunk.
		return []string{content}
	}

	chunks := make([]string, 0, len(sections))
	for _, s := range sections {
		body := strings.TrimSpace(s.body)
		if body == "" {
			continue
		}
		var prefix string
		if s.heading != "" {
			prefix = fmt.Sprintf("此段来自《%s》，讨论\"%s\"：\n", title, s.heading)
		} else {
			prefix = fmt.Sprintf("此段来自《%s》：\n", title)
		}
		chunks = append(chunks, prefix+body)
	}
	if len(chunks) == 0 {
		return []string{content}
	}
	return chunks
}

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

		// Delete any previously stored chunks for this document before re-embedding.
		if full {
			if err := vs.DeleteDoc(d.id); err != nil {
				log.Printf("embed: delete old chunks %s: %v", d.id, err)
			}
		}

		chunks := chunkDoc(d.title, d.content)
		anyWritten := false
		for i, chunk := range chunks {
			vec, err := embedder.Encode(chunk)
			if err != nil {
				log.Printf("embed: skip chunk %s#%d: %v", d.id, i, err)
				continue
			}
			chunkID := fmt.Sprintf("%s#%d", d.id, i)
			meta := map[string]string{
				"doc_id":      d.id,
				"layer":       d.layer,
				"path":        d.path,
				"kind":        d.kind,
				"title":       d.title,
				"description": d.description,
			}
			if err := vs.Upsert(chunkID, vec, meta); err != nil {
				return written, skipped, fmt.Errorf("upsert vec %s: %w", chunkID, err)
			}
			anyWritten = true
		}

		if !anyWritten {
			skipped++
			continue
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
// Chunk-level results are deduplicated to one result per document (highest score wins).
// Returns nil (not error) if no vec store exists — callers degrade to FTS-only.
func VecSearch(kbRoot string, queryVec []float32, layer *string, limit int) ([]SearchResult, error) {
	if !VecStoreExists(kbRoot) {
		return nil, nil
	}
	vs, err := OpenVecStore(kbRoot)
	if err != nil {
		return nil, nil
	}
	// Over-fetch to account for multiple chunks per doc collapsing into one.
	raw, err := vs.Query(queryVec, layer, limit*4)
	if err != nil {
		return nil, err
	}
	// Deduplicate: keep highest-scoring chunk per doc_id.
	seen := make(map[string]int) // doc_id → index in deduped
	var deduped []SearchResult
	for _, r := range raw {
		docID := r.DocID
		if docID == "" {
			docID = r.ID // fallback for legacy single-chunk entries
		}
		if idx, ok := seen[docID]; ok {
			if r.VecScore > deduped[idx].VecScore {
				deduped[idx] = r
				deduped[idx].ID = docID
			}
		} else {
			seen[docID] = len(deduped)
			r.ID = docID
			deduped = append(deduped, r)
		}
		if len(deduped) == limit {
			break
		}
	}
	return deduped, nil
}
