package summarize

import (
	"context"
	"testing"
)

func TestNoop(t *testing.T) {
	t.Parallel()
	s, err := Noop().Summarize(context.Background(), "hello")
	if err != nil || s != "" {
		t.Fatal(s, err)
	}
}
