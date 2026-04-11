---
name: embedding-vector-search
description: Generate text embeddings with gobed and implement HNSW vector search. Use when working on internal/embedding/, internal/index/hnsw.go, or implementing vector similarity search.
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

## HNSW Vector Index (`internal/index/hnsw.go`)

Custom implementation. Key algorithm details:

### Data Structures

```go
type HNSWIndex struct {
    nodes      map[uint64]*node
    entryPoint uint64
    maxLayer   int
    M          int     // max connections per layer (default 16)
    M0         int     // max connections at layer 0 (default 2*M = 32)
    efConstr   int     // construction beam width (default 200)
    efSearch   int     // search beam width (default 64)
    mL         float64 // level generation factor: 1/ln(M)
    dims       int     // vector dimensions (1024 for gobed)
    mu         sync.RWMutex
}

type node struct {
    id      uint64
    vector  []float32
    layers  [][]uint64  // neighbors at each layer
    maxLay  int
}
```

### Insert Algorithm

```
func (h *HNSWIndex) Insert(id uint64, vec []float32):
    1. l = floor(-ln(rand()) * h.mL)    // random max layer
    2. ep = h.entryPoint
    3. For layer = h.maxLayer down to l+1:
         ep = greedyClosest(vec, ep, layer)   // ef=1, single best
    4. For layer = l down to 0:
         neighbors = searchLayer(vec, ep, h.efConstr, layer)
         select top-M (or top-M0 for layer 0) by distance
         bidirectionally connect new node <-> selected neighbors
         prune any neighbor exceeding max connections
         ep = closest from neighbors
    5. If l > h.maxLayer: update entry point and maxLayer
```

### Search Algorithm

```
func (h *HNSWIndex) Search(query []float32, k int) []SearchResult:
    1. ep = h.entryPoint
    2. For layer = h.maxLayer down to 1:
         ep = greedyClosest(query, ep, layer)  // ef=1
    3. candidates = searchLayer(query, ep, h.efSearch, 0)
    4. Return top-k from candidates sorted by distance
```

### searchLayer (beam search)

```
func searchLayer(q []float32, ep uint64, ef int, layer int) []uint64:
    visited = {ep}
    candidates = min-heap{ep}     // closest first
    results = max-heap{ep}        // farthest first, capped at ef

    while candidates not empty:
        c = candidates.pop()      // closest unexamined
        f = results.peek()        // farthest in results
        if dist(c, q) > dist(f, q): break  // no improvement possible

        for each neighbor n of c at layer:
            if n in visited: continue
            visited.add(n)
            f = results.peek()
            if dist(n, q) < dist(f, q) or len(results) < ef:
                candidates.push(n)
                results.push(n)
                if len(results) > ef: results.pop()  // remove farthest

    return results as sorted slice
```

### Distance Function

Cosine similarity via normalized dot product:

```go
func cosineSimilarity(a, b []float32) float32 {
    var dot, normA, normB float32
    for i := range a {
        dot += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    if normA == 0 || normB == 0 {
        return 0
    }
    return dot / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}
```

### Persistence

Serialize HNSW graph to disk for recovery:
- Save: write all nodes (id, vector, layer connections) in binary format
- Load: read binary file, reconstruct in-memory graph
- Trigger: on graceful shutdown, after batch insertions

### Tuning Guide

| Parameter | Default | Increase for | Decrease for |
|---|---|---|---|
| M | 16 | Better recall, larger datasets | Memory savings |
| efConstruction | 200 | Higher index quality | Faster builds |
| efSearch | 64 | Better recall at query time | Lower latency |
