package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func TestMiddlewareOpenMode(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	handler := Middleware(store)(okHandler())

	req := httptest.NewRequest("POST", "/v1/contexts", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("open mode should allow all requests, got %d", rec.Code)
	}
}

func TestMiddlewareHealthAlwaysPublic(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	ctx := t.Context()
	_, _ = store.CreateUser(ctx, "admin", "pass", RoleAdmin)

	handler := Middleware(store)(okHandler())
	req := httptest.NewRequest("GET", "/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("health should be public, got %d", rec.Code)
	}
}

func TestMiddlewareRejectsUnauthenticated(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	ctx := t.Context()
	_, _ = store.CreateUser(ctx, "admin", "pass", RoleAdmin)

	handler := Middleware(store)(okHandler())
	req := httptest.NewRequest("GET", "/v1/contexts", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestMiddlewareBearerToken(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	ctx := t.Context()
	u, _ := store.CreateUser(ctx, "admin", "pass", RoleAdmin)
	plain, _, _ := store.CreateToken(ctx, u.ID, "key", RoleAdmin)

	handler := Middleware(store)(okHandler())
	req := httptest.NewRequest("GET", "/v1/contexts", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", rec.Code)
	}
}

func TestMiddlewareSessionCookie(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	ctx := t.Context()
	u, _ := store.CreateUser(ctx, "admin", "pass", RoleAdmin)
	sess, _ := store.CreateSession(ctx, u.ID)

	handler := Middleware(store)(okHandler())
	req := httptest.NewRequest("GET", "/v1/contexts", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with session cookie, got %d", rec.Code)
	}
}

func TestMiddlewareReadOnlyRole(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	ctx := t.Context()
	u, _ := store.CreateUser(ctx, "reader", "pass", RoleReadOnly)
	plain, _, _ := store.CreateToken(ctx, u.ID, "key", RoleReadOnly)

	handler := Middleware(store)(okHandler())

	// GET should work
	req := httptest.NewRequest("GET", "/v1/contexts", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readonly GET should work, got %d", rec.Code)
	}

	// POST /v1/search should work
	req = httptest.NewRequest("POST", "/v1/search", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("readonly POST /v1/search should work, got %d", rec.Code)
	}

	// POST /v1/contexts should be forbidden
	req = httptest.NewRequest("POST", "/v1/contexts", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("readonly POST /v1/contexts should be forbidden, got %d", rec.Code)
	}

	// DELETE should be forbidden
	req = httptest.NewRequest("DELETE", "/v1/contexts/abc", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("readonly DELETE should be forbidden, got %d", rec.Code)
	}
}

func TestMiddlewareExpiredSession(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store, _ := NewFileAuthStore(dir, 1*time.Millisecond)
	ctx := t.Context()
	u, _ := store.CreateUser(ctx, "admin", "pass", RoleAdmin)
	sess, _ := store.CreateSession(ctx, u.ID)

	time.Sleep(5 * time.Millisecond)

	handler := Middleware(store)(okHandler())
	req := httptest.NewRequest("GET", "/v1/contexts", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sess.ID})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired session should be rejected, got %d", rec.Code)
	}
}

func TestMiddlewareSetsContext(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	ctx := t.Context()
	u, _ := store.CreateUser(ctx, "admin", "pass", RoleAdmin)
	plain, _, _ := store.CreateToken(ctx, u.ID, "key", RoleAdmin)

	var gotUser *User
	var gotRole Role
	inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotUser, _ = UserFromContext(r.Context())
		gotRole, _ = RoleFromContext(r.Context())
	})
	handler := Middleware(store)(inner)

	req := httptest.NewRequest("GET", "/v1/contexts", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotUser == nil || gotUser.ID != u.ID {
		t.Fatal("user not set in context")
	}
	if gotRole != RoleAdmin {
		t.Fatalf("expected admin role, got %s", gotRole)
	}
}
