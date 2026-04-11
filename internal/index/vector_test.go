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
