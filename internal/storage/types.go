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

// Edge is a directed relationship between two ContextEntry nodes.
// Serialized via binary codec in edge_codec.go.
type Edge struct {
	ID        string            `json:"id"`
	FromID    string            `json:"from_id"`
	ToID      string            `json:"to_id"`
	Label     string            `json:"label"`    // e.g. "child", "related_to", "depends_on", "auto_similar"
	Weight    float64           `json:"weight"`   // user importance or cosine similarity for auto edges
	Source    string            `json:"source"`   // "user" or "auto"
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
}
