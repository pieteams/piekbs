//go:build fts5

package kb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	chromem "github.com/philippgille/chromem-go"
)

const vecCollectionName = "documents"

// VecStore wraps chromem-go persistent storage for document embeddings.
type VecStore struct {
	db   *chromem.DB
	coll *chromem.Collection
}

// OpenVecStore opens (or creates) the chromem store under kbRoot/index/vectors/.
func OpenVecStore(kbRoot string) (*VecStore, error) {
	dir := filepath.Join(kbRoot, "index", "vectors")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create vec store dir: %w", err)
	}
	db, err := chromem.NewPersistentDB(dir, false)
	if err != nil {
		return nil, fmt.Errorf("open chromem db: %w", err)
	}
	// noopEmbed is never called because we always supply Document.Embedding.
	noopEmbed := func(_ context.Context, _ string) ([]float32, error) {
		return nil, fmt.Errorf("embedding func should not be called directly")
	}
	coll, err := db.GetOrCreateCollection(vecCollectionName, nil, noopEmbed)
	if err != nil {
		return nil, fmt.Errorf("get collection: %w", err)
	}
	return &VecStore{db: db, coll: coll}, nil
}

// VecStoreExists reports whether a non-empty vector store exists for kbRoot.
func VecStoreExists(kbRoot string) bool {
	dir := filepath.Join(kbRoot, "index", "vectors")
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) > 0
}

// Upsert stores a pre-computed embedding for docID.
// meta must contain keys: layer, path, kind, title, description.
func (v *VecStore) Upsert(docID string, embedding []float32, meta map[string]string) error {
	return v.coll.AddDocument(context.Background(), chromem.Document{
		ID:        docID,
		Embedding: embedding,
		Metadata:  meta,
	})
}

// Delete removes the given docIDs from the vector store.
func (v *VecStore) Delete(docIDs []string) error {
	if len(docIDs) == 0 {
		return nil
	}
	return v.coll.Delete(context.Background(), nil, nil, docIDs...)
}

// Count returns the number of stored vectors.
func (v *VecStore) Count() int { return v.coll.Count() }

// Query runs cosine KNN with a pre-computed query vector.
// If layer is non-nil, results are filtered to that layer.
func (v *VecStore) Query(queryVec []float32, layer *string, limit int) ([]SearchResult, error) {
	where := map[string]string{}
	if layer != nil {
		where["layer"] = *layer
	}
	// chromem-go requires nResults <= collection size.
	if n := v.coll.Count(); n > 0 && limit > n {
		limit = n
	}
	res, err := v.coll.QueryEmbedding(
		context.Background(), queryVec, limit, where, nil,
	)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(res))
	for _, r := range res {
		out = append(out, SearchResult{
			ID:          r.ID,
			Path:        r.Metadata["path"],
			Layer:       r.Metadata["layer"],
			Kind:        r.Metadata["kind"],
			Title:       r.Metadata["title"],
			Description: r.Metadata["description"],
			VecScore:    float64(r.Similarity),
		})
	}
	return out, nil
}
