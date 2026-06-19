//go:build fts5

package embed

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/daulet/tokenizers"
	ort "github.com/yalue/onnxruntime_go"
)

// modelMeta holds optional metadata read from meta.json in the model directory.
type modelMeta struct {
	Dim     int    `json:"dim"`
	Pooling string `json:"pooling"` // "mean" (default) or "cls"
}

// ONNXEmbedder encodes text via ONNX runtime with lazy load and idle eviction.
// modelDir must contain model.onnx and tokenizer.json.
// Optionally, meta.json may specify dim and pooling strategy.
type ONNXEmbedder struct {
	modelDir    string
	dim         int
	pooling     string // "mean" or "cls"
	idleTimeout time.Duration

	mu        sync.Mutex
	session   *ort.DynamicAdvancedSession
	tokenizer *tokenizers.Tokenizer
	lastUsed  time.Time
	timer     *time.Timer
}

// NewONNXEmbedder creates an ONNXEmbedder without loading the model.
// dim and pooling are read from meta.json if present; the passed dim is used
// as fallback when meta.json is absent or has no dim field.
func NewONNXEmbedder(modelDir string, dim int, idleTimeout time.Duration) *ONNXEmbedder {
	e := &ONNXEmbedder{
		modelDir:    modelDir,
		dim:         dim,
		pooling:     "mean",
		idleTimeout: idleTimeout,
	}
	// Load meta.json if present.
	if b, err := os.ReadFile(filepath.Join(modelDir, "meta.json")); err == nil {
		var m modelMeta
		if json.Unmarshal(b, &m) == nil {
			if m.Dim > 0 {
				e.dim = m.Dim
			}
			if m.Pooling != "" {
				e.pooling = m.Pooling
			}
		}
	}
	return e
}

// Dimension returns the vector dimensionality.
func (e *ONNXEmbedder) Dimension() int {
	return e.dim
}

// Encode tokenizes text, runs ONNX inference, and returns a normalized mean-pooled vector.
func (e *ONNXEmbedder) Encode(text string) ([]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.ensureLoaded(); err != nil {
		return nil, err
	}
	e.touch()

	return e.encode(text)
}

// Unload releases the model and tokenizer from memory.
func (e *ONNXEmbedder) Unload() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.unload()
}

// IsLoaded reports whether the model is currently loaded.
func (e *ONNXEmbedder) IsLoaded() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.session != nil
}

// ensureLoaded loads model and tokenizer if not already loaded. Caller must hold mu.
func (e *ONNXEmbedder) ensureLoaded() error {
	if e.session != nil {
		return nil
	}

	if !ort.IsInitialized() {
		if err := ort.InitializeEnvironment(); err != nil {
			return fmt.Errorf("ort init: %w", err)
		}
	}

	tokData, err := os.ReadFile(filepath.Join(e.modelDir, "tokenizer.json"))
	if err != nil {
		return fmt.Errorf("read tokenizer: %w", err)
	}
	tok, err := tokenizers.FromBytesWithTruncation(tokData, 512, tokenizers.TruncationDirectionRight)
	if err != nil {
		return fmt.Errorf("load tokenizer: %w", err)
	}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		tok.Close()
		return fmt.Errorf("ort session opts: %w", err)
	}
	defer opts.Destroy()

	sess, err := ort.NewDynamicAdvancedSession(
		filepath.Join(e.modelDir, "model.onnx"),
		[]string{"input_ids", "attention_mask", "token_type_ids"},
		[]string{"last_hidden_state"},
		opts,
	)
	if err != nil {
		tok.Close()
		return fmt.Errorf("ort new session: %w", err)
	}

	e.tokenizer = tok
	e.session = sess
	e.resetTimer()
	return nil
}

// encode runs inference. Caller must hold mu and must have called ensureLoaded.
func (e *ONNXEmbedder) encode(text string) ([]float32, error) {
	enc := e.tokenizer.EncodeWithOptions(text, true,
		tokenizers.WithReturnTypeIDs(),
		tokenizers.WithReturnAttentionMask(),
	)

	ids := enc.IDs
	mask := enc.AttentionMask
	typeIDs := enc.TypeIDs
	seqLen := len(ids)

	if seqLen == 0 {
		return make([]float32, e.dim), nil
	}

	idsI64 := make([]int64, seqLen)
	maskI64 := make([]int64, seqLen)
	typeI64 := make([]int64, seqLen)
	for i := 0; i < seqLen; i++ {
		idsI64[i] = int64(ids[i])
		if i < len(mask) {
			maskI64[i] = int64(mask[i])
		}
		if i < len(typeIDs) {
			typeI64[i] = int64(typeIDs[i])
		}
	}

	shape := ort.NewShape(1, int64(seqLen))
	tIDs, err := ort.NewTensor(shape, idsI64)
	if err != nil {
		return nil, fmt.Errorf("tensor ids: %w", err)
	}
	defer tIDs.Destroy()

	tMask, err := ort.NewTensor(shape, maskI64)
	if err != nil {
		return nil, fmt.Errorf("tensor mask: %w", err)
	}
	defer tMask.Destroy()

	tType, err := ort.NewTensor(shape, typeI64)
	if err != nil {
		return nil, fmt.Errorf("tensor type: %w", err)
	}
	defer tType.Destroy()

	outShape := ort.NewShape(1, int64(seqLen), int64(e.dim))
	tOut, err := ort.NewEmptyTensor[float32](outShape)
	if err != nil {
		return nil, fmt.Errorf("tensor out: %w", err)
	}
	defer tOut.Destroy()

	if err := e.session.Run(
		[]ort.Value{tIDs, tMask, tType},
		[]ort.Value{tOut},
	); err != nil {
		return nil, fmt.Errorf("ort run: %w", err)
	}

	hidden := tOut.GetData()
	vec := make([]float32, e.dim)

	if e.pooling == "cls" {
		// CLS token pooling: take the first token embedding.
		copy(vec, hidden[:e.dim])
	} else {
		// Mean pooling: average token embeddings weighted by attention mask.
		var maskSum float32
		for t := 0; t < seqLen; t++ {
			m := float32(maskI64[t])
			maskSum += m
			for d := 0; d < e.dim; d++ {
				vec[d] += hidden[t*e.dim+d] * m
			}
		}
		if maskSum > 0 {
			for d := range vec {
				vec[d] /= maskSum
			}
		}
	}

	// L2 normalize.
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for d := range vec {
			vec[d] = float32(float64(vec[d]) / norm)
		}
	}

	return vec, nil
}

// touch resets the idle eviction timer. Caller must hold mu.
func (e *ONNXEmbedder) touch() {
	e.lastUsed = time.Now()
	e.resetTimer()
}

// resetTimer restarts the idle eviction timer. Caller must hold mu.
func (e *ONNXEmbedder) resetTimer() {
	if e.timer != nil {
		e.timer.Stop()
	}
	e.timer = time.AfterFunc(e.idleTimeout, func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if time.Since(e.lastUsed) >= e.idleTimeout {
			e.unload()
		}
	})
}

// unload releases resources. Caller must hold mu.
func (e *ONNXEmbedder) unload() {
	if e.timer != nil {
		e.timer.Stop()
		e.timer = nil
	}
	if e.session != nil {
		e.session.Destroy()
		e.session = nil
	}
	if e.tokenizer != nil {
		e.tokenizer.Close()
		e.tokenizer = nil
	}
}
