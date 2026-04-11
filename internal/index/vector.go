package index

import (
	"fmt"

	"github.com/coder/hnsw"
)

// VectorIndex wraps coder/hnsw with persistence.
type VectorIndex struct {
	sg *hnsw.SavedGraph[string]
}

// OpenVectorIndex loads or creates an on-disk HNSW graph.
func OpenVectorIndex(path string) (*VectorIndex, error) {
	sg, err := hnsw.LoadSavedGraph[string](path)
	if err != nil {
		return nil, fmt.Errorf("vector index: %w", err)
	}
	return &VectorIndex{sg: sg}, nil
}

// Upsert adds or replaces a vector for docID.
func (v *VectorIndex) Upsert(docID string, vec []float32) error {
	v.sg.Add(hnsw.MakeNode(docID, vec))
	return v.sg.Save()
}

// Delete removes a document vector.
func (v *VectorIndex) Delete(docID string) bool {
	ok := v.sg.Delete(docID)
	if ok {
		_ = v.sg.Save()
	}
	return ok
}

// Search returns nearest neighbors by cosine distance (lower distance = better).
func (v *VectorIndex) Search(query []float32, k int) []RankedID {
	if k <= 0 {
		k = 10
	}
	nodes := v.sg.Search(query, k)
	out := make([]RankedID, 0, len(nodes))
	for _, n := range nodes {
		d := hnsw.CosineDistance(query, n.Value)
		sim := 1.0 - float64(d)
		if sim < 0 {
			sim = 0
		}
		out = append(out, RankedID{ID: n.Key, Score: sim})
	}
	return out
}

// Len returns node count.
func (v *VectorIndex) Len() int { return v.sg.Len() }
