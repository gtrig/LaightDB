package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/summarize"
)

func TestHealth(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := context.OpenStore(t.Context(), dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s := NewHTTPServer(st)
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
}

func TestPostGetContext(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := context.OpenStore(t.Context(), dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s := NewHTTPServer(st)

	body := `{"collection":"c","content":"hello world test","content_type":"doc"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/contexts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
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
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatal(rec.Code)
	}
}
