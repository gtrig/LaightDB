package summarize

import "context"

type noop struct{}

// Noop returns a summarizer that always returns empty string.
func Noop() Summarizer { return noop{} }

func (noop) Summarize(ctx context.Context, content string) (string, error) {
	_ = ctx
	_ = content
	return "", nil
}
