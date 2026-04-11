package server

import (
	"encoding/json"
	"net/http"

	"github.com/gtrig/laightdb/internal/auth"
	"github.com/gtrig/laightdb/internal/stress"
)

func (s *HTTPServer) handleGetStressQueries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string][]string{"queries": stress.StandardQueries})
}

func (s *HTTPServer) handlePostStress(w http.ResponseWriter, r *http.Request) {
	if !stressRunAllowed(r, s.AuthStore) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req struct {
		Collection        string `json:"collection"`
		Writes            int    `json:"writes"`
		WriteConcurrency  int    `json:"write_concurrency"`
		Searches          int    `json:"searches"`
		SearchConcurrency int    `json:"search_concurrency"`
		TopK              int    `json:"top_k"`
		Detail            string `json:"detail"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rep, err := stress.RunStore(r.Context(), s.Store, stress.StoreConfig{
		Collection:        req.Collection,
		Writes:            req.Writes,
		WriteConcurrency:  req.WriteConcurrency,
		Searches:          req.Searches,
		SearchConcurrency: req.SearchConcurrency,
		TopK:              req.TopK,
		Detail:            req.Detail,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rep)
}

// stressRunAllowed: open mode (no users) or admin session/token.
func stressRunAllowed(r *http.Request, store *auth.FileAuthStore) bool {
	if store.UserCount() == 0 {
		return true
	}
	role, ok := auth.RoleFromContext(r.Context())
	return ok && role == auth.RoleAdmin
}
