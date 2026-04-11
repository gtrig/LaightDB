package summarize

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ollama struct {
	base   string
	client *http.Client
	model  string
}

// NewOllama creates an Ollama summarizer.
func NewOllama() Summarizer {
	base := os.Getenv("LAIGHTDB_OLLAMA_URL")
	if base == "" {
		base = "http://localhost:11434"
	}
	return &ollama{
		base:   strings.TrimRight(base, "/"),
		client: &http.Client{Timeout: 120 * time.Second},
		model:  "llama3.2",
	}
}

func (o *ollama) Summarize(ctx context.Context, content string) (string, error) {
	body := map[string]any{
		"model":  o.model,
		"prompt": "Summarize in 2-4 sentences:\n\n" + content,
		"stream": false,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.base+"/api/generate", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("summarize ollama: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("summarize ollama: status %d: %s", resp.StatusCode, raw)
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Response), nil
}
