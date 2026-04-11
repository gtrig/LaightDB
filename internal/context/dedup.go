package context

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
)

// ContentHash returns hex SHA-256 of raw content.
func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// HashKey is the engine key for a content hash pointer.
func HashKey(hash string) string { return "h:" + hash }

// NearDuplicate checks cosine similarity against a probe vector (e.g. main embedding).
func NearDuplicate(a, b []float32, threshold float32) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	var dot float32
	var na, nb float32
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return false
	}
	sim := dot / (float32(math.Sqrt(float64(na))) * float32(math.Sqrt(float64(nb))))
	return sim >= threshold
}

// CosineSimilarity returns similarity in [0,1] for equal-length vectors.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot float32
	var na, nb float32
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (float32(math.Sqrt(float64(na))) * float32(math.Sqrt(float64(nb))))
}

// DedupResult is returned when checking for duplicates.
type DedupResult struct {
	ExistingID string
	Hash       string
}

// FindHashDuplicate looks up hash key in engine.
func FindHashDuplicate(get func(string) ([]byte, bool), content string) (DedupResult, bool) {
	h := ContentHash(content)
	key := HashKey(h)
	v, ok := get(key)
	if !ok {
		return DedupResult{Hash: h}, false
	}
	return DedupResult{ExistingID: string(v), Hash: h}, true
}

// RecordHash maps content hash to document id.
func RecordHash(put func(string, []byte) error, content, docID string) error {
	h := ContentHash(content)
	return put(HashKey(h), []byte(docID))
}
