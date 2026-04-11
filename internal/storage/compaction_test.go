package storage

import (
	"path/filepath"
	"testing"
)

func TestCompactMerge(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := filepath.Join(dir, "a.sst")
	b := filepath.Join(dir, "b.sst")
	out := filepath.Join(dir, "out.sst")

	w1, err := NewSSTWriter(a)
	if err != nil {
		t.Fatal(err)
	}
	_ = w1.Append("k1", []byte("a"))
	_ = w1.Close()
	w2, err := NewSSTWriter(b)
	if err != nil {
		t.Fatal(err)
	}
	_ = w2.Append("k1", []byte("b"))
	_ = w2.Append("k2", []byte("2"))
	_ = w2.Close()

	if err := CompactMerge(a, b, out); err != nil {
		t.Fatal(err)
	}
	r, err := OpenSSTReader(out)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := r.Get("k1")
	if !ok || string(v) != "b" {
		t.Fatalf("newer wins: %q %v", v, ok)
	}
}
