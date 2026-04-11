package context

import (
	"encoding/json"

	"github.com/gtrig/laightdb/internal/storage"
)

// DetailLevel controls how much data is returned.
type DetailLevel string

const (
	DetailMetadata DetailLevel = "metadata"
	DetailSummary  DetailLevel = "summary"
	DetailFull     DetailLevel = "full"
)

// Project returns a JSON-serializable view of the entry.
func Project(e storage.ContextEntry, d DetailLevel) map[string]any {
	m := map[string]any{
		"id":           e.ID,
		"collection":   e.Collection,
		"content_type": e.ContentType,
		"metadata":     e.Metadata,
		"token_count":  e.TokenCount,
		"created_at":   e.CreatedAt,
		"updated_at":   e.UpdatedAt,
	}
	switch d {
	case DetailFull:
		m["content"] = e.Content
		m["summary"] = e.Summary
		m["chunks"] = e.Chunks
	case DetailSummary:
		m["summary"] = e.Summary
	case DetailMetadata:
	}
	return m
}

// ProjectJSON encodes projection as JSON bytes.
func ProjectJSON(e storage.ContextEntry, d DetailLevel) ([]byte, error) {
	return json.Marshal(Project(e, d))
}
