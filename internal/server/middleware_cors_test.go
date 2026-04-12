package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_noOpWhenEmpty(t *testing.T) {
	t.Parallel()
	mw := CORSMiddleware("")
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unexpected CORS header")
	}
}

func TestCORSMiddleware_preflight(t *testing.T) {
	t.Parallel()
	mw := CORSMiddleware("https://app.example.com")
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler should not run for OPTIONS")
	}))
	req := httptest.NewRequest(http.MethodOptions, "/v1/contexts", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("origin header: %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
