package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gtrig/laightdb/internal/auth"
	"github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/summarize"
)

func newTestServer(t *testing.T) (*HTTPServer, *auth.FileAuthStore) {
	t.Helper()
	dir := t.TempDir()
	st, err := context.OpenStore(t.Context(), dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	as, err := auth.NewFileAuthStore(filepath.Join(dir, "auth"), 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return NewHTTPServer(st, as), as
}

func testHandler(s *HTTPServer, as *auth.FileAuthStore) http.Handler {
	return s.BuildHandler(auth.Middleware(as))
}

func TestHealth(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
}

func TestPostGetContext(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	body := `{"collection":"c","content":"hello world test","content_type":"doc"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/contexts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil || out.ID == "" {
		t.Fatal(out, err)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/contexts/"+out.ID+"?detail=metadata", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
}

func TestListContexts(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	body := `{"collection":"alpha","content":"list test one","content_type":"doc"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/contexts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("post %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/contexts?limit=50", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list %d %s", rec.Code, rec.Body.String())
	}
	var listOut struct {
		Entries []struct {
			ID         string `json:"id"`
			Collection string `json:"collection"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listOut); err != nil {
		t.Fatal(err)
	}
	if len(listOut.Entries) != 1 || listOut.Entries[0].Collection != "alpha" {
		t.Fatalf("entries: %+v", listOut.Entries)
	}
}

// --- Auth endpoint tests ---

func TestLoginLogout(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)
	ctx := t.Context()

	_, _ = as.CreateUser(ctx, "admin", "pass123", auth.RoleAdmin)

	// Login
	body := `{"username":"admin","password":"pass123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("no session cookie")
	}

	// /auth/me with session cookie
	req = httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("me: %d", rec.Code)
	}

	// Logout
	req = httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout: %d", rec.Code)
	}
}

func TestLoginBadCredentials(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)
	_, _ = as.CreateUser(t.Context(), "admin", "pass", auth.RoleAdmin)

	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestUserCRUDEndpoints(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	// First user creation (no auth required, becomes admin)
	body := `{"username":"admin","password":"pass123","role":"readonly"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create first user: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		User struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"user"`
	}
	json.NewDecoder(rec.Body).Decode(&created)
	if created.User.Role != "admin" {
		t.Fatal("first user should be admin")
	}

	// Login as admin to get session
	loginBody := `{"username":"admin","password":"pass123"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(loginBody))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}

	// Create second user (requires admin)
	body = `{"username":"reader","password":"pass","role":"readonly"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create second user: %d %s", rec.Code, rec.Body.String())
	}

	// List users
	req = httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list users: %d", rec.Code)
	}
}

func TestTokenCRUDEndpoints(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)
	ctx := t.Context()

	u, _ := as.CreateUser(ctx, "admin", "pass", auth.RoleAdmin)
	sess, _ := as.CreateSession(ctx, u.ID)
	cookie := &http.Cookie{Name: auth.SessionCookieName, Value: sess.ID}

	// Create token
	body := `{"name":"ci-key","role":"readonly"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/tokens", strings.NewReader(body))
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create token: %d %s", rec.Code, rec.Body.String())
	}
	var tokenResp struct {
		Token string `json:"token"`
		ID    string `json:"id"`
	}
	json.NewDecoder(rec.Body).Decode(&tokenResp)
	if tokenResp.Token == "" {
		t.Fatal("expected plaintext token")
	}

	// List tokens
	req = httptest.NewRequest(http.MethodGet, "/v1/tokens", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list tokens: %d", rec.Code)
	}

	// Revoke token
	req = httptest.NewRequest(http.MethodDelete, "/v1/tokens/"+tokenResp.ID, nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("revoke token: %d %s", rec.Code, rec.Body.String())
	}
}

func TestAuthStatus(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	// No users -- auth not required
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	var statusResp struct {
		AuthRequired bool `json:"auth_required"`
	}
	json.NewDecoder(rec.Body).Decode(&statusResp)
	if statusResp.AuthRequired {
		t.Fatal("should not require auth with no users")
	}
}

func TestStressQueriesOpenMode(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	req := httptest.NewRequest(http.MethodGet, "/v1/stress/queries", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("queries: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Queries []string `json:"queries"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil || len(out.Queries) == 0 {
		t.Fatalf("decode: %v queries=%v", err, out.Queries)
	}
}

func TestStressRunOpenMode(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	body := `{"collection":"tstress","writes":2,"write_concurrency":1,"searches":3,"search_concurrency":1,"top_k":5,"detail":"summary"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/stress", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stress: %d %s", rec.Code, rec.Body.String())
	}
	var rep struct {
		BaseURL    string `json:"base_url"`
		Writes     struct{ OK int `json:"ok"` } `json:"writes"`
		Searches   struct{ OK int `json:"ok"` } `json:"searches"`
		TotalWall  int64  `json:"total_wall"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	if rep.BaseURL != "in-process" || rep.Writes.OK != 2 || rep.Searches.OK != 3 {
		t.Fatalf("report: %+v", rep)
	}
}

func TestStressRunForbiddenForReadOnly(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)
	ctx := t.Context()

	_, _ = as.CreateUser(ctx, "admin", "pass", auth.RoleAdmin)
	_, _ = as.CreateUser(ctx, "reader", "pass", auth.RoleReadOnly)

	loginBody := `{"username":"reader","password":"pass"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(loginBody))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d", rec.Code)
	}
	var sessionCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			sessionCookie = c
		}
	}

	stressBody := `{"writes":1,"searches":0}`
	req = httptest.NewRequest(http.MethodPost, "/v1/stress", strings.NewReader(stressBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", rec.Code, rec.Body.String())
	}
}
