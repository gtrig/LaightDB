package index

import (
	"testing"
)

func TestGraphIndex_AddOutgoingIncoming(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "A", "B", "child", "user", 1.0)
	g.Add("e2", "A", "C", "related_to", "user", 0.5)

	out := g.Outgoing("A")
	if len(out) != 2 {
		t.Fatalf("Outgoing A: got %d want 2", len(out))
	}
	in := g.Incoming("B")
	if len(in) != 1 {
		t.Fatalf("Incoming B: got %d want 1", len(in))
	}
	if in[0].TargetID != "A" {
		t.Errorf("Incoming B reverse TargetID: got %q want A", in[0].TargetID)
	}
	if in[0].Label != "child" {
		t.Errorf("Incoming B label: got %q want child", in[0].Label)
	}
}

func TestGraphIndex_Remove(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "A", "B", "child", "user", 1.0)
	g.Add("e2", "A", "C", "child", "user", 1.0)
	g.Remove("e1")

	if g.Len() != 1 {
		t.Errorf("Len after Remove: got %d want 1", g.Len())
	}
	out := g.Outgoing("A")
	if len(out) != 1 || out[0].EdgeID != "e2" {
		t.Errorf("Outgoing A after remove: got %+v", out)
	}
	in := g.Incoming("B")
	if len(in) != 0 {
		t.Errorf("Incoming B should be empty after remove, got %+v", in)
	}
}

func TestGraphIndex_RemoveNonExistent(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Remove("does-not-exist") // should not panic
}

func TestGraphIndex_AddReplace(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "A", "B", "child", "user", 1.0)
	g.Add("e1", "A", "C", "related_to", "auto", 0.7) // same edge ID, different target

	if g.Len() != 1 {
		t.Errorf("Len after replace: got %d want 1", g.Len())
	}
	out := g.Outgoing("A")
	if len(out) != 1 || out[0].TargetID != "C" {
		t.Errorf("Outgoing A after replace: got %+v", out)
	}
	// Old reverse index for B should be cleared.
	if len(g.Incoming("B")) != 0 {
		t.Errorf("Incoming B should be empty after replace")
	}
}

func TestGraphIndex_AllNeighborIDs(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "A", "B", "child", "user", 1.0)
	g.Add("e2", "C", "A", "related_to", "user", 1.0)

	n := g.AllNeighborIDs("A")
	if _, ok := n["B"]; !ok {
		t.Error("B should be a neighbour of A (forward)")
	}
	if _, ok := n["C"]; !ok {
		t.Error("C should be a neighbour of A (reverse)")
	}
	if len(n) != 2 {
		t.Errorf("AllNeighborIDs: got %d want 2", len(n))
	}
}

func TestGraphIndex_Neighborhood(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	// A -> B -> C
	g.Add("e1", "A", "B", "child", "user", 1.0)
	g.Add("e2", "B", "C", "child", "user", 1.0)

	hits := g.Neighborhood("A", 2)
	ids := make(map[string]float64)
	for _, h := range hits {
		ids[h.ID] = h.Score
	}
	if ids["B"] == 0 {
		t.Error("B should be in neighborhood of A")
	}
	if ids["C"] == 0 {
		t.Error("C should be in neighborhood of A at depth 2")
	}
	if ids["B"] <= ids["C"] {
		t.Errorf("B (depth 1) should score higher than C (depth 2): B=%v C=%v", ids["B"], ids["C"])
	}
}

func TestGraphIndex_Neighborhood_DepthLimit(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "A", "B", "child", "user", 1.0)
	g.Add("e2", "B", "C", "child", "user", 1.0)

	hits := g.Neighborhood("A", 1)
	ids := make(map[string]struct{})
	for _, h := range hits {
		ids[h.ID] = struct{}{}
	}
	if _, ok := ids["B"]; !ok {
		t.Error("B should be in neighborhood at depth 1")
	}
	if _, ok := ids["C"]; ok {
		t.Error("C should NOT be in neighborhood at maxDepth=1")
	}
}

func TestGraphIndex_Neighborhood_StartNotIncluded(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "A", "B", "child", "user", 1.0)
	for _, h := range g.Neighborhood("A", 1) {
		if h.ID == "A" {
			t.Error("start node A should not appear in its own neighborhood results")
		}
	}
}

func TestGraphIndex_SubtreeEdges(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "root", "child1", "child", "user", 1.0)
	g.Add("e2", "root", "child2", "child", "user", 1.0)
	g.Add("e3", "child1", "grandchild", "child", "user", 1.0)

	edges := g.SubtreeEdges("root", 0)
	if len(edges) != 3 {
		t.Errorf("SubtreeEdges: got %d want 3", len(edges))
	}
}

func TestGraphIndex_SubtreeEdges_DepthLimit(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "root", "child1", "child", "user", 1.0)
	g.Add("e2", "child1", "grandchild", "child", "user", 1.0)

	edges := g.SubtreeEdges("root", 1)
	if len(edges) != 1 {
		t.Errorf("SubtreeEdges depth=1: got %d want 1", len(edges))
	}
	if edges[0].EdgeID != "e1" {
		t.Errorf("unexpected edge: %+v", edges[0])
	}
}

func TestGraphIndex_EncodeDecodeSnapshot(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	g.Add("e1", "A", "B", "child", "user", 1.0)
	g.Add("e2", "B", "C", "related_to", "auto", 0.75)
	g.Add("e3", "A", "C", "depends_on", "user", 0.5)

	data := g.EncodeSnapshot()
	g2, err := DecodeGraphSnapshot(data)
	if err != nil {
		t.Fatalf("DecodeGraphSnapshot: %v", err)
	}
	if g2.Len() != 3 {
		t.Errorf("Len after decode: got %d want 3", g2.Len())
	}
	out := g2.Outgoing("A")
	if len(out) != 2 {
		t.Errorf("Outgoing A after decode: got %d want 2", len(out))
	}
	in := g2.Incoming("C")
	if len(in) != 2 {
		t.Errorf("Incoming C after decode: got %d want 2", len(in))
	}
}

func TestGraphIndex_EncodeDecodeSnapshot_Empty(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	data := g.EncodeSnapshot()
	g2, err := DecodeGraphSnapshot(data)
	if err != nil {
		t.Fatalf("DecodeGraphSnapshot empty: %v", err)
	}
	if g2.Len() != 0 {
		t.Errorf("expected empty graph, got %d edges", g2.Len())
	}
}

func TestDecodeGraphSnapshot_Truncated(t *testing.T) {
	t.Parallel()
	_, err := DecodeGraphSnapshot(nil)
	if err == nil {
		t.Error("expected error on nil input")
	}
}

func TestGraphIndex_NeighborhoodScores(t *testing.T) {
	t.Parallel()
	g := NewGraphIndex()
	// Linear chain: A -> B -> C -> D
	g.Add("e1", "A", "B", "child", "user", 1.0)
	g.Add("e2", "B", "C", "child", "user", 1.0)
	g.Add("e3", "C", "D", "child", "user", 1.0)

	hits := g.Neighborhood("A", 0) // unlimited
	scores := make(map[string]float64)
	for _, h := range hits {
		scores[h.ID] = h.Score
	}
	// Depth 1 = 0.5, depth 2 = 0.333, depth 3 = 0.25
	expected := map[string]float64{
		"B": 1.0 / 2.0,
		"C": 1.0 / 3.0,
		"D": 1.0 / 4.0,
	}
	for id, want := range expected {
		got := scores[id]
		if got == 0 {
			t.Errorf("node %s missing from neighborhood", id)
			continue
		}
		diff := got - want
		if diff < -0.0001 || diff > 0.0001 {
			t.Errorf("score[%s]: got %v want %v", id, got, want)
		}
	}
}
