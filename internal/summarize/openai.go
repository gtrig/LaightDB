package summarize

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
)

type openAI struct {
	apiKey  string
	baseURL string
	client  *http.Client
	model   string
}

// NewOpenAI creates an OpenAI-compatible chat summarizer (OpenAI API, LM Studio, etc.).
//
// Environment:
//   - LAIGHTDB_OPENAI_BASE_URL — API root including /v1 (default: https://api.openai.com/v1).
//     For LM Studio: http://127.0.0.1:1234/v1 (or the port shown in Local Server).
//   - LAIGHTDB_OPENAI_MODEL — chat model id (default: gpt-4o-mini). In LM Studio, use the
//     exact model id of the loaded model.
//   - LAIGHTDB_OPENAI_API_KEY — required for api.openai.com; optional for local servers
//     (LM Studio often ignores it).
func NewOpenAI() Summarizer {
	base := strings.TrimSpace(os.Getenv("LAIGHTDB_OPENAI_BASE_URL"))
	if base == "" {
		base = defaultOpenAIBaseURL
	}
	base = strings.TrimRight(base, "/")
	model := strings.TrimSpace(os.Getenv("LAIGHTDB_OPENAI_MODEL"))
	if model == "" {
		model = defaultOpenAIModel
	}
	return &openAI{
		apiKey:  os.Getenv("LAIGHTDB_OPENAI_API_KEY"),
		baseURL: base,
		client:  &http.Client{Timeout: 120 * time.Second},
		model:   model,
	}
}

func openAIRequiresAPIKey(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return true
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return true
	}
	return host == "api.openai.com"
}

func (o *openAI) Summarize(ctx context.Context, content string) (string, error) {
	if strings.TrimSpace(o.apiKey) == "" && openAIRequiresAPIKey(o.baseURL) {
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
	chatURL := o.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(o.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+o.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("summarize openai: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
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
