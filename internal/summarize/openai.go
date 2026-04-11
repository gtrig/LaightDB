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

type openAI struct {
	apiKey string
	client *http.Client
	model  string
}

// NewOpenAI creates an OpenAI chat summarizer.
func NewOpenAI() Summarizer {
	return &openAI{
		apiKey: os.Getenv("LAIGHTDB_OPENAI_API_KEY"),
		client: &http.Client{Timeout: 60 * time.Second},
		model:  "gpt-4o-mini",
	}
}

func (o *openAI) Summarize(ctx context.Context, content string) (string, error) {
	if strings.TrimSpace(o.apiKey) == "" {
		return "", fmt.Errorf("summarize openai: missing LAIGHTDB_OPENAI_API_KEY")
	}
	body := map[string]any{
		"model": o.model,
		"messages": []map[string]string{
			{"role": "system", "content": "Summarize the following in 2-4 sentences."},
			{"role": "user", "content": content},
		},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("summarize openai: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("summarize openai: status %d: %s", resp.StatusCode, raw)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("summarize openai: empty choices")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
