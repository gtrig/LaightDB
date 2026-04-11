package storage

import (
	"testing"
)

func TestEnginePutGetFlush(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	e, err := OpenEngine(dir, 512)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	val := Encode(ContextEntry{ID: "x", Content: "hello"})
	if err := e.Put("d:x", val); err != nil {
		t.Fatal(err)
	}
	if e.MemLen() == 0 {
		t.Fatal("expected mem")
	}
	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	got, ok := e.Get("d:x")
	if !ok {
		t.Fatal("get after flush")
	}
	ent, err := Decode(got)
	if err != nil || ent.Content != "hello" {
		t.Fatalf("%+v %v", ent, err)
	}
}

func TestEngineDeleteShadowsSST(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	e, err := OpenEngine(dir, 100)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	_ = e.Put("k1", []byte{1, 2, 3})
	_ = e.Flush()
	if err := e.Delete("k1"); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.Get("k1"); ok {
		t.Fatal("expected miss after delete")
	}
}
