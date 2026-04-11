package embedding

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/lee101/gobed"
)

// Engine wraps gobed with a small LRU-ish cache by content hash.
type Engine struct {
	mu     sync.Mutex
	model  *gobed.EmbeddingModel
	cache  map[string][]float32
	maxEnt int
}

// NewEngine loads the embedding model from default paths (may download weights).
func NewEngine() (*Engine, error) {
	m, err := gobed.LoadModel()
	if err != nil {
		return nil, fmt.Errorf("embedding load model: %w", err)
	}
	return &Engine{
		model:  m,
		cache:  make(map[string][]float32),
		maxEnt: 2048,
	}, nil
}

// Embed returns a 1024-dim vector for text.
func (e *Engine) Embed(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	sum := sha256.Sum256([]byte(text))
	key := hex.EncodeToString(sum[:])
	e.mu.Lock()
	if v, ok := e.cache[key]; ok {
		out := append([]float32(nil), v...)
		e.mu.Unlock()
		return out, nil
	}
	e.mu.Unlock()
	vec, err := e.model.Encode(text)
	if err != nil {
		return nil, fmt.Errorf("embedding encode: %w", err)
	}
	e.mu.Lock()
	if len(e.cache) >= e.maxEnt {
		e.cache = make(map[string][]float32)
	}
	e.cache[key] = append([]float32(nil), vec...)
	e.mu.Unlock()
	return vec, nil
}

// Dim returns embedding dimensionality.
func (e *Engine) Dim() int {
	if e.model == nil {
		return 0
	}
	return e.model.EmbedDim
}
