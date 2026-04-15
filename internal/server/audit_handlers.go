package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gtrig/laightdb/internal/auth"
)

func (s *HTTPServer) handleAuditCalls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.CallLog == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if s.AuthStore.UserCount() == 0 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	role, ok := auth.RoleFromContext(r.Context())
	if !ok || role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	calls := s.CallLog.List(limit)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"calls": calls})
}
