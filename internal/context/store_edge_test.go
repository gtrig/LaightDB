package context

import (
	"context"
	"testing"
)

func TestPutEdge_Basic(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id, err := s.PutEdge(ctx, PutEdgeRequest{
		FromID: "node-a",
		ToID:   "node-b",
		Label:  "child",
		Weight: 1.0,
	})
	if err != nil {
		t.Fatalf("PutEdge: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty edge ID")
	}
}

func TestPutEdge_MissingFromOrTo(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	_, err := s.PutEdge(ctx, PutEdgeRequest{ToID: "b"})
	if err == nil {
		t.Error("expected error for missing FromID")
	}
	_, err = s.PutEdge(ctx, PutEdgeRequest{FromID: "a"})
	if err == nil {
		t.Error("expected error for missing ToID")
	}
}

func TestGetEdge(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id, _ := s.PutEdge(ctx, PutEdgeRequest{
		FromID:   "a",
		ToID:     "b",
		Label:    "related_to",
		Weight:   0.75,
		Source:   "auto",
		Metadata: map[string]string{"k": "v"},
	})
	e, err := s.GetEdge(ctx, id)
	if err != nil {
		t.Fatalf("GetEdge: %v", err)
	}
	if e.FromID != "a" || e.ToID != "b" {
		t.Errorf("FromID/ToID mismatch: %+v", e)
	}
	if e.Label != "related_to" {
		t.Errorf("Label: got %q want related_to", e.Label)
	}
	if e.Weight != 0.75 {
		t.Errorf("Weight: got %v want 0.75", e.Weight)
	}
	if e.Source != "auto" {
		t.Errorf("Source: got %q want auto", e.Source)
	}
	if e.Metadata["k"] != "v" {
		t.Errorf("Metadata[k]: got %q want v", e.Metadata["k"])
	}
}

func TestGetEdge_NotFound(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	_, err := s.GetEdge(context.Background(), "does-not-exist")
	if err == nil {
		t.Error("expected error for missing edge")
	}
}

func TestDeleteEdge(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id, _ := s.PutEdge(ctx, PutEdgeRequest{FromID: "a", ToID: "b", Label: "child"})
	if err := s.DeleteEdge(ctx, id); err != nil {
		t.Fatalf("DeleteEdge: %v", err)
	}
	// Verify gone from LSM.
	_, err := s.GetEdge(ctx, id)
	if err == nil {
		t.Error("expected error after delete")
	}
	// Verify gone from graph index.
	if len(s.GraphOutgoing("a")) != 0 {
		t.Error("GraphOutgoing should be empty after delete")
	}
	if len(s.GraphIncoming("b")) != 0 {
		t.Error("GraphIncoming should be empty after delete")
	}
}

func TestDeleteEdge_NotFound(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	err := s.DeleteEdge(context.Background(), "missing")
	if err == nil {
		t.Error("expected error deleting non-existent edge")
	}
}

func TestListEdgesFrom(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	s.PutEdge(ctx, PutEdgeRequest{FromID: "root", ToID: "child1", Label: "child"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: "root", ToID: "child2", Label: "child"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: "other", ToID: "root", Label: "child"})  //nolint:errcheck

	edges, err := s.ListEdgesFrom(ctx, "root")
	if err != nil {
		t.Fatalf("ListEdgesFrom: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("ListEdgesFrom: got %d want 2", len(edges))
	}
	for _, e := range edges {
		if e.FromID != "root" {
			t.Errorf("unexpected FromID: %q", e.FromID)
		}
	}
}

func TestListEdgesTo(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	s.PutEdge(ctx, PutEdgeRequest{FromID: "a", ToID: "target", Label: "child"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: "b", ToID: "target", Label: "child"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: "c", ToID: "other", Label: "child"})  //nolint:errcheck

	edges, err := s.ListEdgesTo(ctx, "target")
	if err != nil {
		t.Fatalf("ListEdgesTo: %v", err)
	}
	if len(edges) != 2 {
		t.Errorf("ListEdgesTo: got %d want 2", len(edges))
	}
}

func TestGraphIndexSync(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id, _ := s.PutEdge(ctx, PutEdgeRequest{FromID: "x", ToID: "y", Label: "child"})

	out := s.GraphOutgoing("x")
	if len(out) != 1 || out[0].TargetID != "y" {
		t.Errorf("GraphOutgoing after put: %+v", out)
	}
	in := s.GraphIncoming("y")
	if len(in) != 1 || in[0].TargetID != "x" {
		t.Errorf("GraphIncoming after put: %+v", in)
	}

	_ = s.DeleteEdge(ctx, id)
	if len(s.GraphOutgoing("x")) != 0 {
		t.Error("GraphOutgoing should be empty after delete")
	}
}

func TestSubtreeEdges(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	s.PutEdge(ctx, PutEdgeRequest{FromID: "root", ToID: "A", Label: "child"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: "root", ToID: "B", Label: "child"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: "A", ToID: "A1", Label: "child"})   //nolint:errcheck

	edges := s.SubtreeEdges("root", 0)
	if len(edges) != 3 {
		t.Errorf("SubtreeEdges: got %d want 3", len(edges))
	}
}

func TestStoreStats_IncludesEdges(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	s.PutEdge(ctx, PutEdgeRequest{FromID: "a", ToID: "b"}) //nolint:errcheck
	s.PutEdge(ctx, PutEdgeRequest{FromID: "b", ToID: "c"}) //nolint:errcheck

	st, err := s.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st["edges"] != 2 {
		t.Errorf("stats edges: got %v want 2", st["edges"])
	}
}

func TestPutEdge_DefaultSource(t *testing.T) {
	t.Parallel()
	s := openTestStore(t)
	ctx := context.Background()

	id, _ := s.PutEdge(ctx, PutEdgeRequest{FromID: "a", ToID: "b", Label: "child"})
	e, err := s.GetEdge(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if e.Source != "user" {
		t.Errorf("default Source: got %q want user", e.Source)
	}
}
