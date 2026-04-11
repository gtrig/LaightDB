package stress

import (
	"testing"
	"time"
)

func TestPercentile(t *testing.T) {
	t.Parallel()
	d := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
	}
	if got := percentile(d, 0.50); got != 3*time.Millisecond {
		t.Fatalf("p50: got %v want 3ms", got)
	}
	if got := percentile(d, 0.95); got != 5*time.Millisecond {
		t.Fatalf("p95: got %v want 5ms", got)
	}
	if got := percentile(nil, 0.50); got != 0 {
		t.Fatalf("empty: got %v", got)
	}
}

func TestStandardQueriesNonEmpty(t *testing.T) {
	t.Parallel()
	if len(StandardQueries) == 0 {
		t.Fatal("StandardQueries must not be empty")
	}
}
