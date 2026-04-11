package context

import "testing"

func TestEstimateTokens(t *testing.T) {
	t.Parallel()
	if EstimateTokens("") != 0 {
		t.Fatal()
	}
	if EstimateTokens("abcd") != 1 {
		t.Fatal()
	}
}
