package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gtrig/laightdb/internal/auth"
)

// --- Auth endpoints ---

func (s *HTTPServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}

	u, err := s.AuthStore.AuthenticatePassword(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	sess, err := s.AuthStore.CreateSession(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"user": userResponse(u),
	})
}

func (s *HTTPServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(auth.SessionCookieName)
	if err == nil {
		_ = s.AuthStore.DeleteSession(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleMe(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": userResponse(u),
	})
}

func (s *HTTPServer) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"auth_required": s.AuthStore.UserCount() > 0,
	})
}

// --- User management endpoints ---

func (s *HTTPServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string    `json:"username"`
		Password string    `json:"password"`
		Role     auth.Role `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}
	if !req.Role.Valid() {
		req.Role = auth.RoleReadOnly
	}

	// First user creation is allowed without auth (bootstraps the system).
	// Subsequent users require admin.
	if s.AuthStore.UserCount() > 0 {
		caller, ok := auth.UserFromContext(r.Context())
		if !ok || caller.Role != auth.RoleAdmin {
			http.Error(w, "admin required", http.StatusForbidden)
			return
		}
	} else {
		req.Role = auth.RoleAdmin
	}

	u, err := s.AuthStore.CreateUser(r.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		if errors.Is(err, auth.ErrDuplicate) {
			http.Error(w, "username already exists", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user": userResponse(u),
	})
}

func (s *HTTPServer) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.AuthStore.ListUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]map[string]any, len(users))
	for i, u := range users {
		out[i] = userResponse(u)
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}

func (s *HTTPServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	caller, ok := auth.UserFromContext(r.Context())
	if !ok || caller.Role != auth.RoleAdmin {
		http.Error(w, "admin required", http.StatusForbidden)
		return
	}
	if caller.ID == id {
		http.Error(w, "cannot delete yourself", http.StatusBadRequest)
		return
	}

	if err := s.AuthStore.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	caller, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	// Admin or self
	if caller.Role != auth.RoleAdmin && caller.ID != id {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "password required", http.StatusBadRequest)
		return
	}

	if err := s.AuthStore.UpdatePassword(r.Context(), id, req.Password); err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *HTTPServer) handleChangeRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	caller, ok := auth.UserFromContext(r.Context())
	if !ok || caller.Role != auth.RoleAdmin {
		http.Error(w, "admin required", http.StatusForbidden)
		return
	}

	var req struct {
		Role auth.Role `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !req.Role.Valid() {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	if err := s.AuthStore.UpdateRole(r.Context(), id, req.Role); err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Token management endpoints ---

func (s *HTTPServer) handleCreateToken(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name string    `json:"name"`
		Role auth.Role `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if !req.Role.Valid() {
		req.Role = auth.RoleReadOnly
	}
	if !caller.Role.CanEscalate(req.Role) {
		http.Error(w, "cannot create token with higher role than your own", http.StatusForbidden)
		return
	}

	plaintext, tok, err := s.AuthStore.CreateToken(r.Context(), caller.ID, req.Name, req.Role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"token":     plaintext,
		"id":        tok.ID,
		"name":      tok.Name,
		"prefix":    tok.Prefix,
		"role":      tok.Role,
		"created_at": tok.CreatedAt.Format(time.RFC3339),
	})
}

func (s *HTTPServer) handleListTokens(w http.ResponseWriter, r *http.Request) {
	caller, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID := caller.ID
	if caller.Role == auth.RoleAdmin {
		userID = "" // admin sees all
	}

	tokens, err := s.AuthStore.ListTokens(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out := make([]map[string]any, len(tokens))
	for i, t := range tokens {
		out[i] = tokenResponse(t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": out})
}

func (s *HTTPServer) handleRevokeToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	caller, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	tokens, _ := s.AuthStore.ListTokens(r.Context(), "")
	var target *auth.APIToken
	for _, t := range tokens {
		if t.ID == id {
			target = t
			break
		}
	}
	if target == nil {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	if caller.Role != auth.RoleAdmin && target.UserID != caller.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := s.AuthStore.RevokeToken(r.Context(), id); err != nil {
		if errors.Is(err, auth.ErrNotFound) {
			http.Error(w, "token not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func userResponse(u *auth.User) map[string]any {
	return map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"role":       u.Role,
		"created_at": u.CreatedAt.Format(time.RFC3339),
		"updated_at": u.UpdatedAt.Format(time.RFC3339),
	}
}

func tokenResponse(t *auth.APIToken) map[string]any {
	resp := map[string]any{
		"id":         t.ID,
		"user_id":    t.UserID,
		"name":       t.Name,
		"prefix":     t.Prefix,
		"role":       t.Role,
		"created_at": t.CreatedAt.Format(time.RFC3339),
		"active":     t.Active(),
	}
	if t.RevokedAt != nil {
		resp["revoked_at"] = t.RevokedAt.Format(time.RFC3339)
	}
	return resp
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
