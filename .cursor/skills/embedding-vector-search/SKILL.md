---
name: embedding-vector-search
description: Generate text embeddings with gobed and use coder/hnsw for vector search. Use when working on internal/embedding/, internal/index/vector.go, or implementing vector similarity search.
---

# Embedding & Vector Search

## gobed Embeddings (`internal/embedding/engine.go`)

Pure Go static embeddings using `github.com/lee101/gobed`.

Model: `sentence-transformers/static-retrieval-mrl-en-v1` -- 1024 dimensions, int8 quantized.

```go
import "github.com/lee101/gobed"

type EmbeddingEngine struct {
    model *gobed.Model
}

func NewEmbeddingEngine() (*EmbeddingEngine, error) {
    model, err := gobed.LoadModel()
    if err != nil {
        return nil, fmt.Errorf("load embedding model: %w", err)
    }
    return &EmbeddingEngine{model: model}, nil
}

// Single text embedding
func (e *EmbeddingEngine) Embed(text string) ([]float32, error) {
    embedding, err := e.model.Encode(text)
    // returns 1024-dim float32 vector
    return embedding, err
}

// Similarity between two texts
func (e *EmbeddingEngine) Similarity(a, b string) (float32, error) {
    return e.model.Similarity(a, b)
}
```

Requirements: 119 MB model weights auto-downloaded on first `LoadModel()`.

## HNSW Vector Index (`internal/index/vector.go`)

Uses `github.com/coder/hnsw` (v0.6+). Do NOT implement HNSW from scratch.

### Types

- `hnsw.Node[K]` has **`Key K`** and **`Value hnsw.Vector`** (`[]float32`).
- `hnsw.MakeNode(key, vec)` builds a node.
- `(*Graph[K]).Search(query Vector, k int) []Node[K]` -- iterate with `n.Key`, `n.Value`.

### Basic usage

```go
import "github.com/coder/hnsw"

g := hnsw.NewGraph[uint64]()
g.Add(hnsw.MakeNode(1, []float32{ /* 1024 dims */ }))

neighbors := g.Search(queryVec, 10)
for _, n := range neighbors {
    _ = n.Key    // doc ID
    _ = n.Value  // embedding vector
}
```

### Persistence (`SavedGraph`)

Use `LoadSavedGraph` + `Save` for file-backed graphs (see package docs):

```go
sg, err := hnsw.LoadSavedGraph[uint64]("/data/vector.hnsw")
if err != nil { /* ... */ }

sg.Add(hnsw.MakeNode(id, vec))
if err := sg.Save(); err != nil { /* ... */ }
```

`LoadSavedGraph` returns an empty graph if the file does not exist yet.

### Wrapper sketch

```go
type VectorIndex struct {
    saved *hnsw.SavedGraph[uint64]
    mu    sync.RWMutex
}

func (v *VectorIndex) Insert(id uint64, vec []float32) {
    v.mu.Lock()
    defer v.mu.Unlock()
    v.saved.Add(hnsw.MakeNode(id, vec))
}

func (v *VectorIndex) Delete(id uint64) bool {
    v.mu.Lock()
    defer v.mu.Unlock()
    return v.saved.Delete(id)
}

func (v *VectorIndex) Search(query []float32, k int) []uint64 {
    v.mu.RLock()
    defer v.mu.RUnlock()
    nodes := v.saved.Search(query, k)
    ids := make([]uint64, len(nodes))
    for i, n := range nodes {
        ids[i] = n.Key
    }
    return ids
}
```

### Additional Reference

- https://pkg.go.dev/github.com/coder/hnsw
- https://github.com/coder/hnsw
