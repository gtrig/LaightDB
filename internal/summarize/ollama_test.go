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

func TestOllamaSummarize(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "topic") {
			t.Errorf("body: %s", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"response": "ollama summary"})
	}))
	t.Cleanup(srv.Close)

	o := &ollama{
		base:   strings.TrimSuffix(srv.URL, "/"),
		client: srv.Client(),
		model:  "llama3.2",
	}
	out, err := o.Summarize(context.Background(), "topic text")
	if err != nil {
		t.Fatal(err)
	}
	if out != "ollama summary" {
		t.Errorf("got %q", out)
	}
}
