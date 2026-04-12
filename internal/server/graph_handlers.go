package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/index"
)

// handlePostEdge creates a new directed edge between two context entries.
//
// POST /v1/edges
// Body: {"from_id":"...","to_id":"...","label":"child","weight":1.0,"source":"user","metadata":{}}
func (s *HTTPServer) handlePostEdge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromID   string            `json:"from_id"`
		ToID     string            `json:"to_id"`
		Label    string            `json:"label"`
		Weight   float64           `json:"weight"`
		Source   string            `json:"source"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := s.Store.PutEdge(r.Context(), lctx.PutEdgeRequest{
		FromID:   req.FromID,
		ToID:     req.ToID,
		Label:    req.Label,
		Weight:   req.Weight,
		Source:   req.Source,
		Metadata: req.Metadata,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// handleGetEdge retrieves a single edge by ID.
//
// GET /v1/edges/{id}
func (s *HTTPServer) handleGetEdge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	e, err := s.Store.GetEdge(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(e)
}

// handleListEdges lists edges by from or to node ID.
//
// GET /v1/edges?from=<nodeID>  or  ?to=<nodeID>
func (s *HTTPServer) handleListEdges(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" && to == "" {
		http.Error(w, "from or to query parameter required", http.StatusBadRequest)
		return
	}
	var edges []interface{}
	if from != "" {
		got, err := s.Store.ListEdgesFrom(r.Context(), from)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, e := range got {
			edges = append(edges, e)
		}
	} else {
		got, err := s.Store.ListEdgesTo(r.Context(), to)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, e := range got {
			edges = append(edges, e)
		}
	}
	if edges == nil {
		edges = []interface{}{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"edges": edges})
}

// handleDeleteEdge removes an edge by ID.
//
// DELETE /v1/edges/{id}
func (s *HTTPServer) handleDeleteEdge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.Store.DeleteEdge(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGraphNeighbors returns the immediate (or depth-limited) graph neighbours of a node.
//
// GET /v1/graph/{id}/neighbors?depth=1
func (s *HTTPServer) handleGraphNeighbors(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	if depth <= 0 {
		depth = 1
	}
	hits := s.Store.GraphNeighborhood(nodeID, depth)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"neighbors": hits})
}

// handleGraphSubtree returns a directed BFS subtree rooted at a node.
//
// GET /v1/graph/{id}/subtree?depth=3
func (s *HTTPServer) handleGraphSubtree(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	edges := s.Store.SubtreeEdges(nodeID, depth)
	// Convert EdgeRef slice to a JSON-friendly form.
	type edgeJSON struct {
		EdgeID   string  `json:"edge_id"`
		TargetID string  `json:"target_id"`
		Label    string  `json:"label"`
		Weight   float64 `json:"weight"`
		Source   string  `json:"source"`
	}
	out := make([]edgeJSON, 0, len(edges))
	for _, e := range edges {
		out = append(out, edgeJSON{
			EdgeID:   e.EdgeID,
			TargetID: e.TargetID,
			Label:    e.Label,
			Weight:   e.Weight,
			Source:   e.Source,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"root": nodeID, "edges": out})
}

// handleGraphSearch performs a 3-signal graph-boosted search.
//
// POST /v1/graph/search
// Body: {"query":"...","focus_node_id":"...","max_depth":2,"top_k":10,"collection":"...","detail":"summary"}
func (s *HTTPServer) handleGraphSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query       string            `json:"query"`
		FocusNodeID string            `json:"focus_node_id"`
		MaxDepth    int               `json:"max_depth"`
		Collection  string            `json:"collection"`
		Filters     map[string]string `json:"filters"`
		TopK        int               `json:"top_k"`
		Detail      string            `json:"detail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hits, err := s.Store.Search(r.Context(), lctx.SearchRequest{
		Query:       req.Query,
		Collection:  req.Collection,
		Filters:     req.Filters,
		TopK:        req.TopK,
		Detail:      lctx.DetailLevel(req.Detail),
		FocusNodeID: req.FocusNodeID,
		MaxDepth:    req.MaxDepth,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalTokens, totalCompact := 0, 0
	for _, h := range hits {
		totalTokens += h.TokenCount
		totalCompact += h.CompactTokenCount
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"hits":                hits,
		"total_token_count":   totalTokens,
		"total_compact_count": totalCompact,
		"total_tokens_saved":  totalTokens - totalCompact,
	})
}

// handleGraphSuggestLinks returns vector-discovered link suggestions for a node.
//
// GET /v1/graph/{id}/suggest-links?threshold=0.7&top_k=10
func (s *HTTPServer) handleGraphSuggestLinks(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")
	threshold, err := strconv.ParseFloat(r.URL.Query().Get("threshold"), 64)
	if err != nil || threshold <= 0 {
		threshold = 0.7
	}
	topK, _ := strconv.Atoi(r.URL.Query().Get("top_k"))
	if topK <= 0 {
		topK = 10
	}
	suggestions, err := s.Store.SuggestLinks(r.Context(), nodeID, threshold, topK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if suggestions == nil {
		suggestions = []lctx.SuggestedLink{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"suggestions": suggestions})
}

// handleGraphOverview returns a bulk snapshot of all nodes and edges for the 3D UI.
//
// GET /v1/graph/overview?collection=&limit=500
func (s *HTTPServer) handleGraphOverview(w http.ResponseWriter, r *http.Request) {
	collection := r.URL.Query().Get("collection")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	overview, err := s.Store.GraphOverview(r.Context(), collection, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(overview)
}

// edgeRefsToJSON is a helper used by the subtree handler.
func edgeRefsToJSON(refs []index.EdgeRef) []map[string]any {
	out := make([]map[string]any, 0, len(refs))
	for _, r := range refs {
		out = append(out, map[string]any{
			"edge_id":   r.EdgeID,
			"target_id": r.TargetID,
			"label":     r.Label,
			"weight":    r.Weight,
			"source":    r.Source,
		})
	}
	return out
}
