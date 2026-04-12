package server

import (
	"encoding/json"
	"net/http"
)

func (s *HTTPServer) handleStorageDiagnostics(w http.ResponseWriter, r *http.Request) {
	diag, err := s.Store.StorageDiagnostics(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(diag)
}
