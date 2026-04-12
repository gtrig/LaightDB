package summarize

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicSummarize(t *testing.T) {
	t.Parallel()
	var gotMethod string
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "hello") {
			t.Errorf("expected content in body: %s", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{{"text": "short summary"}},
		})
	}))
	t.Cleanup(srv.Close)

	a := &anthropic{
		apiKey:   "test-key",
		messages: srv.URL,
		client:   srv.Client(),
	}
	out, err := a.Summarize(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if out != "short summary" {
		t.Errorf("got %q", out)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method %s", gotMethod)
	}
	if gotPath != "/" {
		t.Errorf("path %s", gotPath)
	}
}
