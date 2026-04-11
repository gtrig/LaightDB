package index

import "testing"

func TestTokenize(t *testing.T) {
	t.Parallel()
	got := Tokenize("Hello, world! 123")
	if len(got) != 3 || got[0] != "hello" || got[1] != "world" || got[2] != "123" {
		t.Fatalf("%v", got)
	}
}

func TestBM25Search(t *testing.T) {
	t.Parallel()
	f := NewFullText()
	f.AddDocument("d1", "the quick brown fox")
	f.AddDocument("d2", "the lazy dog")
	hits := f.Search("the fox", 5)
	if len(hits) < 1 {
		t.Fatal("expected hits")
	}
	if hits[0].ID != "d1" {
		t.Fatalf("want d1 first got %+v", hits)
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	t.Parallel()
	f := NewFullText()
	f.AddDocument("a", "hello world")
	f.AddDocument("b", "world peace")
	data := f.EncodeSnapshot()
	g, err := DecodeSnapshot(data)
	if err != nil {
		t.Fatal(err)
	}
	if g.N != f.N || len(g.terms) != len(f.terms) {
		t.Fatalf("mismatch")
	}
}
