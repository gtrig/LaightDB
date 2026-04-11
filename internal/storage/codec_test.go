package storage

import (
	"testing"
	"time"
)

func TestCodecRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Unix(1700000000, 123456789)
	e := ContextEntry{
		ID:          "u1",
		Collection:  "c",
		Content:     "hello",
		ContentType: "doc",
		Summary:     "sum",
		Chunks: []Chunk{
			{Index: 0, ParentID: "u1", Text: "a", Embedding: []float32{0.1, 0.2}},
		},
		Metadata:   map[string]string{"k": "v", "a": "b"},
		Embedding:  []float32{1, 2, 3},
		CreatedAt:  now,
		UpdatedAt:  now,
		TokenCount: 42,
	}
	data := Encode(e)
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != e.ID || got.Collection != e.Collection || got.Content != e.Content {
		t.Fatalf("%+v", got)
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
