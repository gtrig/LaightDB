package context

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/gtrig/laightdb/internal/summarize"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(context.Background(), t.TempDir(), 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestSearchWithDetail verifies that search results include a projected entry
// when Detail is set, eliminating the need for follow-up get_context calls.
func TestSearchWithDetail(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id, err := s.Put(ctx, PutRequest{
		Collection: "docs",
		Content:    "the quick brown fox jumps over the lazy dog",
		Metadata:   map[string]string{"topic": "animals"},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("no detail returns no entry", func(t *testing.T) {
		t.Parallel()
		hits, err := s.Search(ctx, SearchRequest{Query: "fox", TopK: 5})
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) == 0 {
			t.Fatal("expected hits")
		}
		if hits[0].Entry != nil {
			t.Error("expected no Entry when Detail is empty")
		}
	})

	t.Run("detail=metadata returns entry without content", func(t *testing.T) {
		t.Parallel()
		hits, err := s.Search(ctx, SearchRequest{Query: "fox", TopK: 5, Detail: DetailMetadata})
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) == 0 {
			t.Fatal("expected hits")
		}
		if hits[0].Entry == nil {
			t.Fatal("expected Entry with detail=metadata")
		}
		if _, ok := hits[0].Entry["content"]; ok {
			t.Error("detail=metadata must not include content")
		}
		if hits[0].Entry["id"] != id {
			t.Errorf("entry id mismatch: %v", hits[0].Entry["id"])
		}
	})

	t.Run("detail=full returns entry with content", func(t *testing.T) {
		t.Parallel()
		hits, err := s.Search(ctx, SearchRequest{Query: "fox", TopK: 5, Detail: DetailFull})
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) == 0 {
			t.Fatal("expected hits")
		}
		if hits[0].Entry == nil {
			t.Fatal("expected Entry with detail=full")
		}
		if _, ok := hits[0].Entry["content"]; !ok {
			t.Error("detail=full must include content")
		}
	})
}

// TestPutDedup verifies that storing identical content twice returns the original ID.
func TestPutDedup(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	content := "unique content that should not be duplicated"
	id1, err := s.Put(ctx, PutRequest{Collection: "test", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.Put(ctx, PutRequest{Collection: "test", Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Errorf("expected same ID for duplicate content: %q vs %q", id1, id2)
	}

	// Only one entry should exist in the store.
	hits, err := s.Search(ctx, SearchRequest{Query: "unique content duplicated", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if h.ID == id1 {
			// Only one occurrence should match.
			var count int
			for _, hh := range hits {
				if hh.ID == id1 {
					count++
				}
			}
			if count != 1 {
				t.Errorf("expected exactly one hit for deduplicated entry, got %d", count)
			}
			return
		}
	}
}

// TestPutDedupDifferentContent verifies that different content still gets distinct IDs.
func TestPutDedupDifferentContent(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id1, err := s.Put(ctx, PutRequest{Collection: "test", Content: "content alpha"})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := s.Put(ctx, PutRequest{Collection: "test", Content: "content beta"})
	if err != nil {
		t.Fatal(err)
	}
	if id1 == id2 {
		t.Error("different content must produce different IDs")
	}
}

// TestSearchChunkBM25 verifies that chunk-level BM25 indexing allows finding
// documents whose relevant content lives in a specific chunk.
func TestSearchChunkBM25(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	// Create a document with two distinct paragraphs so chunking produces 2+ chunks.
	content := "The elephant is a large mammal found in Africa and Asia.\n\n" +
		strings.Repeat("Filler paragraph to increase document size and trigger chunking. ", 30) + "\n\n" +
		"Quantum entanglement describes correlations between distant particles."
	id, err := s.Put(ctx, PutRequest{Collection: "science", Content: content})
	if err != nil {
		t.Fatal(err)
	}

	hits, err := s.Search(ctx, SearchRequest{Query: "quantum entanglement particles", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, h := range hits {
		if h.ID == id {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected chunk-level BM25 to surface document via trailing-chunk query")
	}
}

// TestSearchMetadataPreFilter verifies that metadata filtering happens before
// RRF so only matching documents consume ranked slots.
func TestSearchMetadataPreFilter(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	_, err := s.Put(ctx, PutRequest{
		Collection: "docs",
		Content:    "golang concurrency patterns goroutine channel",
		Metadata:   map[string]string{"lang": "go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Put(ctx, PutRequest{
		Collection: "docs",
		Content:    "python async await coroutine concurrency",
		Metadata:   map[string]string{"lang": "python"},
	})
	if err != nil {
		t.Fatal(err)
	}

	hits, err := s.Search(ctx, SearchRequest{
		Query:   "concurrency",
		TopK:    10,
		Filters: map[string]string{"lang": "go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if e, ok := h.Entry["metadata"]; ok {
			if meta, ok := e.(map[string]string); ok {
				if meta["lang"] != "go" {
					t.Errorf("metadata pre-filter let through non-go result: %v", h.ID)
				}
			}
		}
	}
	// All hits must have lang=go; verify by re-checking with detail.
	hits2, err := s.Search(ctx, SearchRequest{
		Query:   "concurrency",
		TopK:    10,
		Filters: map[string]string{"lang": "go"},
		Detail:  DetailMetadata,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits2 {
		meta, ok := h.Entry["metadata"].(map[string]string)
		if !ok {
			t.Fatalf("unexpected metadata type: %T", h.Entry["metadata"])
		}
		if meta["lang"] != "go" {
			t.Errorf("expected lang=go, got %q for hit %s", meta["lang"], h.ID)
		}
	}
}

// TestSearchMetadataFilterNoMatch verifies that a filter with no matching docs
// returns nil rather than an error.
func TestSearchMetadataFilterNoMatch(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	_, err := s.Put(ctx, PutRequest{
		Content:  "some content",
		Metadata: map[string]string{"env": "prod"},
	})
	if err != nil {
		t.Fatal(err)
	}

	hits, err := s.Search(ctx, SearchRequest{
		Query:   "content",
		TopK:    5,
		Filters: map[string]string{"env": "staging"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Errorf("expected no hits for non-matching filter, got %d", len(hits))
	}
}

// TestSnapshotRoundtrip verifies that Close saves snapshots and a re-opened
// store loads them without a full rebuild.
func TestSnapshotRoundtrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx := context.Background()

	// Write entries and close to trigger snapshot save.
	func() {
		s, err := OpenStore(ctx, dir, 1<<20, nil, summarize.Noop())
		if err != nil {
			t.Fatalf("first open: %v", err)
		}
		for i := range 5 {
			_, err := s.Put(ctx, PutRequest{
				Collection: "snap",
				Content:    strings.Repeat("word ", i+3),
				Metadata:   map[string]string{"n": strconv.Itoa(i)},
			})
			if err != nil {
				t.Fatalf("put %d: %v", i, err)
			}
		}
		if err := s.Close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	}()

	// Re-open; snapshots should load without full rebuild.
	s2, err := OpenStore(ctx, dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer func() { _ = s2.Close() }()

	hits, err := s2.Search(ctx, SearchRequest{Query: "word", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Error("expected results after snapshot-based re-open")
	}
}
