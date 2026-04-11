package storage

import (
	"testing"
	"time"
)

func TestCodecBackwardCompat(t *testing.T) {
	t.Parallel()
	now := time.Unix(1700000000, 123456789)
	old := ContextEntry{
		ID:         "u2",
		Collection: "c",
		Content:    "old data",
		TokenCount: 10,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	data := Encode(old)
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.CompactContent != "" {
		t.Fatalf("expected empty compact, got %q", got.CompactContent)
	}
	if got.CompactTokenCount != 0 {
		t.Fatalf("expected 0 compact tokens, got %d", got.CompactTokenCount)
	}
}

func TestCodecRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Unix(1700000000, 123456789)
	e := ContextEntry{
		ID:                "u1",
		Collection:        "c",
		Content:           "hello world",
		CompactContent:    "hello world",
		ContentType:       "doc",
		Summary:           "sum",
		Chunks: []Chunk{
			{Index: 0, ParentID: "u1", Text: "a", Embedding: []float32{0.1, 0.2}},
		},
		Metadata:          map[string]string{"k": "v", "a": "b"},
		Embedding:         []float32{1, 2, 3},
		CreatedAt:         now,
		UpdatedAt:         now,
		TokenCount:        42,
		CompactTokenCount: 30,
	}
	data := Encode(e)
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != e.ID || got.Collection != e.Collection || got.Content != e.Content {
		t.Fatalf("%+v", got)
	}
	if got.CompactContent != e.CompactContent {
		t.Fatalf("compact content: got %q want %q", got.CompactContent, e.CompactContent)
	}
	if got.CompactTokenCount != 30 {
		t.Fatalf("compact token count: got %d want 30", got.CompactTokenCount)
	}
	if len(got.Metadata) != 2 || got.Metadata["k"] != "v" {
		t.Fatalf("meta %+v", got.Metadata)
	}
	if len(got.Chunks) != 1 || got.Chunks[0].Text != "a" {
		t.Fatal(got.Chunks)
	}
	if len(got.Embedding) != 3 {
		t.Fatal(got.Embedding)
	}
	if got.TokenCount != 42 {
		t.Fatal(got.TokenCount)
	}
}
