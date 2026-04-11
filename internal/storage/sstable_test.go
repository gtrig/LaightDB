package storage

import (
	"path/filepath"
	"testing"
)

func TestSSTWriteRead(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "t.sst")
	w, err := NewSSTWriter(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Append("a", []byte("1")); err != nil {
		t.Fatal(err)
	}
	if err := w.Append("b", []byte("2")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := OpenSSTReader(p)
	if err != nil {
		t.Fatal(err)
	}
	v, ok := r.Get("a")
	if !ok || string(v) != "1" {
		t.Fatalf("a: %v %q", ok, v)
	}
	v, ok = r.Get("b")
	if !ok || string(v) != "2" {
		t.Fatalf("b: %v %q", ok, v)
	}
}
