package index

import "testing"

func TestRRF(t *testing.T) {
	t.Parallel()
	a := []RankedID{{ID: "x", Score: 1}, {ID: "y", Score: 0.5}}
	b := []RankedID{{ID: "y", Score: 1}, {ID: "z", Score: 0.5}}
	out := RRF([][]RankedID{a, b}, 10)
	if len(out) < 2 {
		t.Fatalf("%+v", out)
	}
}
