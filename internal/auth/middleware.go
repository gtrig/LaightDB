package auth

import (
	"net/http"
	"strings"
)

const SessionCookieName = "ldb_session"

// readOnlyRoutes that ReadOnly role is permitted to access.
var readOnlyMethods = map[string]bool{
	"GET": true,
}

var readOnlyPOSTRoutes = map[string]bool{
	"/v1/search":      true,
	"/v1/auth/login":  true,
	"/v1/auth/logout": true,
}

var publicRoutes = map[string]bool{
	"/v1/health":      true,
	"/v1/auth/status": true,
	"/v1/auth/login":  true,
	"/v1/auth/logout": true,
}

// Middleware returns an HTTP middleware that enforces authentication.
// When no users exist in the store (open mode), all requests pass through.
func Middleware(store *FileAuthStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Public routes always pass
			if publicRoutes[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Open mode: no users exist yet
			if store.UserCount() == 0 {
				next.ServeHTTP(w, r)
				return
			}

			user, role, ok := authenticate(r, store)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if !permitted(r, role) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			ctx := WithUser(r.Context(), user)
			ctx = WithRole(ctx, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func authenticate(r *http.Request, store *FileAuthStore) (*User, Role, bool) {
	// Try session cookie first
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		sess, err := store.GetSession(r.Context(), cookie.Value)
		if err == nil {
			u, err := store.GetUser(r.Context(), sess.UserID)
			if err == nil {
				return u, u.Role, true
			}
		}
	}

	// Try Bearer token
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		u, tok, err := store.AuthenticateToken(r.Context(), token)
		if err == nil {
			return u, tok.Role, true
		}
	}

	return nil, "", false
}

func permitted(r *http.Request, role Role) bool {
	if role == RoleAdmin {
		return true
	}
	// ReadOnly: allow GET and specific POST routes
	if readOnlyMethods[r.Method] {
		return true
	}
	if r.Method == "POST" && readOnlyPOSTRoutes[r.URL.Path] {
		return true
	}
	return false
}
