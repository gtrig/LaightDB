package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gtrig/laightdb/internal/auth"
)

func TestAuditCallsForbiddenOpenMode(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	req := httptest.NewRequest(http.MethodGet, "/v1/audit/calls", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", rec.Code, rec.Body.String())
	}
	_ = as
}

func TestAuditCallsForbiddenReadOnly(t *testing.T) {
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

	req = httptest.NewRequest(http.MethodGet, "/v1/audit/calls", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestAuditCallsOKAdminAndMCPEntry(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)
	ctx := t.Context()

	u, _ := as.CreateUser(ctx, "admin", "pass123", auth.RoleAdmin)

	loginBody := `{"username":"admin","password":"pass123"}`
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

	// MCP-only log: inject an entry as the MCP layer would (same user as session).
	uctx := auth.WithUser(ctx, u)
	s.CallLog.RecordMCP(uctx, time.Now(), "get_stats", "{}", `{"entries":0}`, false, time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/v1/audit/calls?limit=50", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("audit: %d %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Calls []struct {
			Channel string `json:"channel"`
			Tool    string `json:"tool"`
		} `json:"calls"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range out.Calls {
		if c.Channel == "mcp" && c.Tool == "get_stats" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no mcp get_stats entry in %#v", out.Calls)
	}
}
