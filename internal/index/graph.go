package index

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

// EdgeRef is a lightweight record of a single adjacency list entry.
type EdgeRef struct {
	EdgeID   string
	TargetID string // the "other" node (neighbour)
	Label    string
	Weight   float64
	Source   string // "user" or "auto"
}

// graphEdgeMeta stores enough data to remove an edge from both directions.
type graphEdgeMeta struct {
	fromID string
	toID   string
}

// GraphIndex is an in-memory bidirectional adjacency index for Edge relationships.
// It mirrors the pattern of FullText and MetadataIndex: in-memory structure,
// EncodeSnapshot/DecodeSnapshot for persistence, rebuild from LSM on cold start.
type GraphIndex struct {
	forward map[string][]EdgeRef       // fromID -> outgoing edges
	reverse map[string][]EdgeRef       // toID   -> incoming edges
	meta    map[string]graphEdgeMeta   // edgeID -> (fromID, toID) for fast Remove
}

// NewGraphIndex creates an empty graph index.
func NewGraphIndex() *GraphIndex {
	return &GraphIndex{
		forward: make(map[string][]EdgeRef),
		reverse: make(map[string][]EdgeRef),
		meta:    make(map[string]graphEdgeMeta),
	}
}

// Add inserts an edge into the adjacency index.
// Calling Add with an existing edgeID replaces it.
func (g *GraphIndex) Add(edgeID, fromID, toID, label, source string, weight float64) {
	g.Remove(edgeID)
	fwd := EdgeRef{EdgeID: edgeID, TargetID: toID, Label: label, Weight: weight, Source: source}
	rev := EdgeRef{EdgeID: edgeID, TargetID: fromID, Label: label, Weight: weight, Source: source}
	g.forward[fromID] = append(g.forward[fromID], fwd)
	g.reverse[toID] = append(g.reverse[toID], rev)
	g.meta[edgeID] = graphEdgeMeta{fromID: fromID, toID: toID}
}

// Remove deletes an edge from both forward and reverse adjacency lists.
func (g *GraphIndex) Remove(edgeID string) {
	m, ok := g.meta[edgeID]
	if !ok {
		return
	}
	delete(g.meta, edgeID)
	g.forward[m.fromID] = removeEdgeRef(g.forward[m.fromID], edgeID)
	if len(g.forward[m.fromID]) == 0 {
		delete(g.forward, m.fromID)
	}
	g.reverse[m.toID] = removeEdgeRef(g.reverse[m.toID], edgeID)
	if len(g.reverse[m.toID]) == 0 {
		delete(g.reverse, m.toID)
	}
}

func removeEdgeRef(refs []EdgeRef, edgeID string) []EdgeRef {
	out := refs[:0]
	for _, r := range refs {
		if r.EdgeID != edgeID {
			out = append(out, r)
		}
	}
	return out
}

// Outgoing returns all outgoing edges from nodeID.
func (g *GraphIndex) Outgoing(nodeID string) []EdgeRef {
	refs := g.forward[nodeID]
	if len(refs) == 0 {
		return nil
	}
	out := make([]EdgeRef, len(refs))
	copy(out, refs)
	return out
}

// Incoming returns all incoming edges to nodeID.
func (g *GraphIndex) Incoming(nodeID string) []EdgeRef {
	refs := g.reverse[nodeID]
	if len(refs) == 0 {
		return nil
	}
	out := make([]EdgeRef, len(refs))
	copy(out, refs)
	return out
}

// AllNeighborIDs returns the set of all node IDs directly connected to nodeID
// via any edge (both directions, depth 1). Used for suggest-links filtering.
func (g *GraphIndex) AllNeighborIDs(nodeID string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, r := range g.forward[nodeID] {
		out[r.TargetID] = struct{}{}
	}
	for _, r := range g.reverse[nodeID] {
		out[r.TargetID] = struct{}{}
	}
	return out
}

// Neighborhood performs a BFS from startID traversing edges in both directions,
// up to maxDepth hops, and returns reachable nodes as RankedIDs scored by
// 1.0 / (1.0 + depth). maxDepth <= 0 means unlimited.
func (g *GraphIndex) Neighborhood(startID string, maxDepth int) []RankedID {
	type item struct {
		id    string
		depth int
	}
	visited := make(map[string]struct{})
	visited[startID] = struct{}{}
	queue := []item{{startID, 0}}
	var out []RankedID
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth > 0 {
			score := 1.0 / (1.0 + float64(cur.depth))
			out = append(out, RankedID{ID: cur.id, Score: score})
		}
		if maxDepth > 0 && cur.depth >= maxDepth {
			continue
		}
		for _, r := range g.forward[cur.id] {
			if _, seen := visited[r.TargetID]; !seen {
				visited[r.TargetID] = struct{}{}
				queue = append(queue, item{r.TargetID, cur.depth + 1})
			}
		}
		for _, r := range g.reverse[cur.id] {
			if _, seen := visited[r.TargetID]; !seen {
				visited[r.TargetID] = struct{}{}
				queue = append(queue, item{r.TargetID, cur.depth + 1})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].ID < out[j].ID
		}
		return out[i].Score > out[j].Score
	})
	return out
}

// SubtreeEdges performs a directed BFS from startID following only outgoing edges,
// and returns the edge refs encountered (the edge pointing away from the root).
// maxDepth <= 0 means unlimited.
func (g *GraphIndex) SubtreeEdges(startID string, maxDepth int) []EdgeRef {
	type item struct {
		id    string
		depth int
	}
	visited := make(map[string]struct{})
	visited[startID] = struct{}{}
	queue := []item{{startID, 0}}
	var out []EdgeRef
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if maxDepth > 0 && cur.depth >= maxDepth {
			continue
		}
		for _, r := range g.forward[cur.id] {
			out = append(out, r)
			if _, seen := visited[r.TargetID]; !seen {
				visited[r.TargetID] = struct{}{}
				queue = append(queue, item{r.TargetID, cur.depth + 1})
			}
		}
	}
	return out
}

// Len returns the total number of edges in the index.
func (g *GraphIndex) Len() int { return len(g.meta) }

// EncodeSnapshot serializes the graph index for persistence.
func (g *GraphIndex) EncodeSnapshot() []byte {
	type full struct {
		edgeID string
		fromID string
		toID   string
		label  string
		source string
		weight float64
	}
	all := make([]full, 0, len(g.meta))
	for edgeID, m := range g.meta {
		label, source, weight := g.findLabelSourceWeight(edgeID, m.fromID)
		all = append(all, full{edgeID, m.fromID, m.toID, label, source, weight})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].edgeID < all[j].edgeID })

	var buf []byte
	buf = graphAppendUvarint(buf, uint64(len(all)))
	for _, e := range all {
		buf = graphAppendString(buf, e.edgeID)
		buf = graphAppendString(buf, e.fromID)
		buf = graphAppendString(buf, e.toID)
		buf = graphAppendString(buf, e.label)
		buf = graphAppendString(buf, e.source)
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], math.Float64bits(e.weight))
		buf = append(buf, b[:]...)
	}
	return buf
}

func (g *GraphIndex) findLabelSourceWeight(edgeID, fromID string) (label, source string, weight float64) {
	for _, r := range g.forward[fromID] {
		if r.EdgeID == edgeID {
			return r.Label, r.Source, r.Weight
		}
	}
	return "", "", 0
}

// DecodeGraphSnapshot restores a GraphIndex from EncodeSnapshot output.
func DecodeGraphSnapshot(data []byte) (*GraphIndex, error) {
	g := NewGraphIndex()
	p := 0
	n, np, err := graphReadUvarint(data, p)
	if err != nil {
		return nil, fmt.Errorf("graph snapshot count: %w", err)
	}
	p = np
	for i := uint64(0); i < n; i++ {
		edgeID, np2, err := graphReadString(data, p)
		if err != nil {
			return nil, fmt.Errorf("graph snapshot edgeID: %w", err)
		}
		fromID, np3, err := graphReadString(data, np2)
		if err != nil {
			return nil, fmt.Errorf("graph snapshot fromID: %w", err)
		}
		toID, np4, err := graphReadString(data, np3)
		if err != nil {
			return nil, fmt.Errorf("graph snapshot toID: %w", err)
		}
		label, np5, err := graphReadString(data, np4)
		if err != nil {
			return nil, fmt.Errorf("graph snapshot label: %w", err)
		}
		source, np6, err := graphReadString(data, np5)
		if err != nil {
			return nil, fmt.Errorf("graph snapshot source: %w", err)
		}
		if np6+8 > len(data) {
			return nil, fmt.Errorf("graph snapshot: truncated weight")
		}
		weight := math.Float64frombits(binary.LittleEndian.Uint64(data[np6 : np6+8]))
		p = np6 + 8
		g.Add(edgeID, fromID, toID, label, source, weight)
	}
	return g, nil
}

func graphAppendUvarint(buf []byte, x uint64) []byte {
	var scratch [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(scratch[:], x)
	return append(buf, scratch[:n]...)
}

func graphAppendString(buf []byte, s string) []byte {
	buf = graphAppendUvarint(buf, uint64(len(s)))
	return append(buf, s...)
}

func graphReadUvarint(data []byte, p int) (uint64, int, error) {
	if p >= len(data) {
		return 0, p, fmt.Errorf("truncated uvarint")
	}
	v, n := binary.Uvarint(data[p:])
	if n <= 0 {
		return 0, p, fmt.Errorf("bad uvarint")
	}
	return v, p + n, nil
}

func graphReadString(data []byte, p int) (string, int, error) {
	l, np, err := graphReadUvarint(data, p)
	if err != nil {
		return "", p, err
	}
	if np+int(l) > len(data) {
		return "", p, fmt.Errorf("truncated string")
	}
	return string(data[np : np+int(l)]), np + int(l), nil
}
