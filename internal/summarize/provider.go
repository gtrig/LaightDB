package summarize

import "context"

// Summarizer produces short summaries for stored content.
type Summarizer interface {
	Summarize(ctx context.Context, content string) (string, error)
}
