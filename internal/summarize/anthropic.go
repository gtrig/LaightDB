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

type anthropic struct {
	apiKey   string
	messages string // full URL for POST (default https://api.anthropic.com/v1/messages)
	client   *http.Client
}

// NewAnthropic creates an Anthropic summarizer.
func NewAnthropic() Summarizer {
	return &anthropic{
		apiKey:   os.Getenv("LAIGHTDB_ANTHROPIC_API_KEY"),
		messages: "https://api.anthropic.com/v1/messages",
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (a *anthropic) Summarize(ctx context.Context, content string) (string, error) {
	if strings.TrimSpace(a.apiKey) == "" {
		return "", fmt.Errorf("summarize anthropic: missing LAIGHTDB_ANTHROPIC_API_KEY")
	}
	body := map[string]any{
		"model":      "claude-3-5-haiku-20241022",
		"max_tokens": 512,
		"messages": []map[string]string{
			{"role": "user", "content": "Summarize in 2-4 sentences:\n\n" + content},
		},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.messages, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("summarize anthropic: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("summarize anthropic: status %d: %s", resp.StatusCode, raw)
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("summarize anthropic: empty content")
	}
	return strings.TrimSpace(out.Content[0].Text), nil
}
