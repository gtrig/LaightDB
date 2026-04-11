package storage

import "time"

// Chunk is a semantic slice of a parent context entry.
type Chunk struct {
	Index     int
	ParentID  string
	Text      string
	Embedding []float32
}

// ContextEntry is the persisted document model (binary codec in codec.go).
type ContextEntry struct {
	ID                string
	Collection        string
	Content           string
	CompactContent    string // token-minimized AI-readable version of Content
	ContentType       string
	Summary           string
	Chunks            []Chunk
	Metadata          map[string]string
	Embedding         []float32
	CreatedAt         time.Time
	UpdatedAt         time.Time
	TokenCount        int
	CompactTokenCount int // token count for CompactContent
}
