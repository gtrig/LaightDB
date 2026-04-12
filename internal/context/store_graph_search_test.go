package context

import (
	"context"
	"testing"
)

// TestGraphBoostedSearch verifies that graph proximity influences result ranking.
// We store three entries, create edges A->B and A->C, then search with A as FocusNodeID.
// B and C should appear in results (boosted by graph proximity) even without a query.
func TestGraphBoostedSearch_GraphHitsWithNoQuery(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	idA, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "alpha node content"})
	idB, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "beta node content"})
	idC, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "gamma node content"})

	// Link A->B and A->C
	s.PutEdge(ctx, PutEdgeRequest{FromID: idA, ToID: idB, Label: "child"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: idA, ToID: idC, Label: "child"}) //nolint:errcheck

	hits, err := s.Search(ctx, SearchRequest{
		FocusNodeID: idA,
		MaxDepth:    2,
		TopK:        10,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	foundB, foundC := false, false
	for _, h := range hits {
		if h.ID == idB {
			foundB = true
		}
		if h.ID == idC {
			foundC = true
		}
	}
	if !foundB {
		t.Errorf("expected idB in graph-boosted search results")
	}
	if !foundC {
		t.Errorf("expected idC in graph-boosted search results")
	}
}

// TestGraphBoostedSearch_DepthFilter ensures that nodes beyond maxDepth are excluded.
func TestGraphBoostedSearch_DepthFilter(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	idA, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "root node"})
	idB, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "depth one node"})
	idC, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "depth two node"})

	s.PutEdge(ctx, PutEdgeRequest{FromID: idA, ToID: idB, Label: "child"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: idB, ToID: idC, Label: "child"}) //nolint:errcheck

	hits, err := s.Search(ctx, SearchRequest{
		FocusNodeID: idA,
		MaxDepth:    1,
		TopK:        10,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	foundB, foundC := false, false
	for _, h := range hits {
		if h.ID == idB {
			foundB = true
		}
		if h.ID == idC {
			foundC = true
		}
	}
	if !foundB {
		t.Errorf("expected idB (depth 1) in results at maxDepth=1")
	}
	if foundC {
		t.Errorf("expected idC (depth 2) to be excluded at maxDepth=1")
	}
}

// TestGraphBoostedSearch_NoFocusNode verifies standard search still works without FocusNodeID.
func TestGraphBoostedSearch_NoFocusNode(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "the quick brown fox"})
	hits, err := s.Search(ctx, SearchRequest{Query: "quick fox", TopK: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	found := false
	for _, h := range hits {
		if h.ID == id {
			found = true
		}
	}
	if !found {
		t.Error("standard search without focus node should still find documents")
	}
}
