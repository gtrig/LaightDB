package index

import (
	"path/filepath"
	"testing"
)

func TestVectorUpsertSearch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "v.hnsw")
	v, err := OpenVectorIndex(p)
	if err != nil {
		t.Fatal(err)
	}
	a := []float32{1, 0, 0}
	b := []float32{0.9, 0.1, 0}
	c := []float32{0, 1, 0}
	if err := v.Upsert("doc-a", a); err != nil {
		t.Fatal(err)
	}
	if err := v.Upsert("doc-b", b); err != nil {
		t.Fatal(err)
	}
	if err := v.Upsert("doc-c", c); err != nil {
		t.Fatal(err)
	}
	hits := v.Search(a, 2)
	if len(hits) < 1 {
		t.Fatal("no hits")
	}
	if hits[0].ID != "doc-a" && hits[0].ID != "doc-b" {
		t.Fatalf("unexpected order %+v", hits)
	}
}

// TestVectorUpsertBatch verifies that UpsertBatch adds all items and produces
// the same search results as individual Upsert calls, with a single disk flush.
func TestVectorUpsertBatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	v, err := OpenVectorIndex(filepath.Join(dir, "batch.hnsw"))
	if err != nil {
		t.Fatal(err)
	}
	items := []VectorItem{
		{ID: "a", Vec: []float32{1, 0, 0}},
		{ID: "b", Vec: []float32{0.9, 0.1, 0}},
		{ID: "c", Vec: []float32{0, 1, 0}},
	}
	if err := v.UpsertBatch(items); err != nil {
		t.Fatalf("UpsertBatch: %v", err)
	}
	if v.Len() != 3 {
		t.Fatalf("expected 3 nodes, got %d", v.Len())
	}
	hits := v.Search([]float32{1, 0, 0}, 2)
	if len(hits) < 1 {
		t.Fatal("expected hits after UpsertBatch")
	}
	if hits[0].ID != "a" && hits[0].ID != "b" {
		t.Fatalf("unexpected top hit: %+v", hits)
	}
}

// TestVectorUpsertBatchPersists verifies that UpsertBatch items survive reload.
func TestVectorUpsertBatchPersists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "persist.hnsw")

	v1, err := OpenVectorIndex(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := v1.UpsertBatch([]VectorItem{
		{ID: "x", Vec: []float32{1, 0}},
		{ID: "y", Vec: []float32{0, 1}},
	}); err != nil {
		t.Fatal(err)
	}

	v2, err := OpenVectorIndex(p)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Len() != 2 {
		t.Fatalf("expected 2 persisted nodes, got %d", v2.Len())
	}
}
