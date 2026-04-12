package context

import (
	"context"
	"testing"
)

// TestSuggestLinks_NoEmbedder verifies that SuggestLinks returns nil when no
// embedder is configured (the normal test-store case), without error.
func TestSuggestLinks_NoEmbedder(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "some content"})
	links, err := s.SuggestLinks(ctx, id, 0.7, 5)
	if err != nil {
		t.Fatalf("SuggestLinks: %v", err)
	}
	// Without an embedder, should return nil, not error.
	if links != nil {
		t.Errorf("expected nil links without embedder, got %v", links)
	}
}

// TestSuggestLinks_NodeNotFound verifies error handling for unknown node IDs.
func TestSuggestLinks_NodeNotFound(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	_, err := s.SuggestLinks(context.Background(), "does-not-exist", 0.7, 5)
	if err == nil {
		t.Error("expected error for non-existent node")
	}
}

// TestSuggestLinks_ExcludesLinkedNodes verifies that existing direct neighbors
// are filtered from suggestions. We can't test this deeply without an embedder
// but we verify the filter logic via GraphIndex indirectly (allNeighborIDs).
func TestSuggestLinks_ExcludesLinkedNodes_ViaGraphIndex(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	idA, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "node A"})
	idB, _ := s.Put(ctx, PutRequest{Collection: "test", Content: "node B"})
	s.PutEdge(ctx, PutEdgeRequest{FromID: idA, ToID: idB, Label: "child"}) //nolint:errcheck

	neighbors := s.graph.AllNeighborIDs(idA)
	if _, ok := neighbors[idB]; !ok {
		t.Errorf("expected idB in AllNeighborIDs of idA")
	}
}
