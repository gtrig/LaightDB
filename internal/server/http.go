package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gtrig/laightdb/internal/auth"
	"github.com/gtrig/laightdb/internal/context"
)

// HTTPServer wires REST routes to the Store.
type HTTPServer struct {
	Store     *context.Store
	AuthStore *auth.FileAuthStore
	Mux       *http.ServeMux
}

// NewHTTPServer registers v1 routes.
func NewHTTPServer(store *context.Store, authStore *auth.FileAuthStore) *HTTPServer {
	m := http.NewServeMux()
	s := &HTTPServer{Store: store, AuthStore: authStore, Mux: m}

	// Context routes
	m.HandleFunc("POST /v1/contexts", s.handlePostContexts)
	m.HandleFunc("GET /v1/contexts", s.handleListContexts)
	m.HandleFunc("GET /v1/contexts/{id}", s.handleGetContext)
	m.HandleFunc("DELETE /v1/contexts/{id}", s.handleDeleteContext)
	m.HandleFunc("POST /v1/search", s.handleSearch)
	m.HandleFunc("GET /v1/collections", s.handleCollections)
	m.HandleFunc("DELETE /v1/collections/{name}", s.handleDeleteCollection)
	m.HandleFunc("POST /v1/collections/{name}/compact", s.handleCompact)
	m.HandleFunc("GET /v1/stats", s.handleStats)
	m.HandleFunc("GET /v1/health", s.handleHealth)

	m.HandleFunc("GET /v1/stress/queries", s.handleGetStressQueries)
	m.HandleFunc("POST /v1/stress", s.handlePostStress)

	// Graph / Edge routes
	m.HandleFunc("POST /v1/edges", s.handlePostEdge)
	m.HandleFunc("GET /v1/edges", s.handleListEdges)
	m.HandleFunc("GET /v1/edges/{id}", s.handleGetEdge)
	m.HandleFunc("DELETE /v1/edges/{id}", s.handleDeleteEdge)
	m.HandleFunc("GET /v1/graph/{id}/neighbors", s.handleGraphNeighbors)
	m.HandleFunc("GET /v1/graph/{id}/subtree", s.handleGraphSubtree)
	m.HandleFunc("POST /v1/graph/search", s.handleGraphSearch)
	m.HandleFunc("GET /v1/graph/{id}/suggest-links", s.handleGraphSuggestLinks)

	// Auth routes
	m.HandleFunc("POST /v1/auth/login", s.handleLogin)
	m.HandleFunc("POST /v1/auth/logout", s.handleLogout)
	m.HandleFunc("GET /v1/auth/me", s.handleMe)
	m.HandleFunc("GET /v1/auth/status", s.handleAuthStatus)

	// User management routes
	m.HandleFunc("POST /v1/users", s.handleCreateUser)
	m.HandleFunc("GET /v1/users", s.handleListUsers)
	m.HandleFunc("DELETE /v1/users/{id}", s.handleDeleteUser)
	m.HandleFunc("PUT /v1/users/{id}/password", s.handleChangePassword)
	m.HandleFunc("PUT /v1/users/{id}/role", s.handleChangeRole)

	// Token management routes
	m.HandleFunc("POST /v1/tokens", s.handleCreateToken)
	m.HandleFunc("GET /v1/tokens", s.handleListTokens)
	m.HandleFunc("DELETE /v1/tokens/{id}", s.handleRevokeToken)

	return s
}

// BuildHandler constructs the middleware chain. Called by launch after wiring.
func (s *HTTPServer) BuildHandler(middlewares ...func(http.Handler) http.Handler) http.Handler {
	var h http.Handler = s.Mux
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return recoveryMiddleware(loggingMiddleware(h))
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *HTTPServer) handleListContexts(w http.ResponseWriter, r *http.Request) {
	collection := r.URL.Query().Get("collection")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.Store.ListEntries(r.Context(), collection, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"entries": items})
}

func (s *HTTPServer) handlePostContexts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Collection  string            `json:"collection"`
		Content     string            `json:"content"`
		ContentType string            `json:"content_type"`
		Metadata    map[string]string `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := s.Store.Put(r.Context(), context.PutRequest{
		Collection:  req.Collection,
		Content:     req.Content,
		ContentType: req.ContentType,
		Metadata:    req.Metadata,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func (s *HTTPServer) handleGetContext(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail := r.URL.Query().Get("detail")
	if detail == "" {
		detail = "summary"
	}
	d := context.DetailLevel(detail)
	ent, err := s.Store.Get(r.Context(), id, d)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	out, err := context.ProjectJSON(ent, d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}

func (s *HTTPServer) handleDeleteContext(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.Store.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query      string            `json:"query"`
		Collection string            `json:"collection"`
		Filters    map[string]string `json:"filters"`
		TopK       int               `json:"top_k"`
		Detail     string            `json:"detail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	hits, err := s.Store.Search(r.Context(), context.SearchRequest{
		Query:      req.Query,
		Collection: req.Collection,
		Filters:    req.Filters,
		TopK:       req.TopK,
		Detail:     context.DetailLevel(req.Detail),
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

func (s *HTTPServer) handleCollections(w http.ResponseWriter, r *http.Request) {
	cols, err := s.Store.ListCollections(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"collections": cols})
}

func (s *HTTPServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.Store.Stats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *HTTPServer) handleDeleteCollection(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	deleted, err := s.Store.DeleteCollection(r.Context(), name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"deleted": deleted})
}

func (s *HTTPServer) handleCompact(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_ = name
	if err := s.Store.Engine().RunCompaction(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
