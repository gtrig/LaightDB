package launch

import (
	"testing"
)

func TestPickSummarizer(t *testing.T) {
	t.Parallel()
	if pickSummarizer("noop") == nil {
		t.Fatal("noop")
	}
	if pickSummarizer("openai") == nil {
		t.Fatal("openai")
	}
	if pickSummarizer("anthropic") == nil {
		t.Fatal("anthropic")
	}
	if pickSummarizer("ollama") == nil {
		t.Fatal("ollama")
	}
}
