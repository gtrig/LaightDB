package index

import "testing"

func TestMetadataSetMatchRemove(t *testing.T) {
	t.Parallel()
	m := NewMetadataIndex()
	m.Set("doc1", map[string]string{"lang": "go", "env": "prod"})
	m.Set("doc2", map[string]string{"lang": "python", "env": "prod"})
	m.Set("doc3", map[string]string{"lang": "go", "env": "staging"})

	got := m.Match(map[string]string{"lang": "go", "env": "prod"})
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d: %v", len(got), got)
	}
	if _, ok := got["doc1"]; !ok {
		t.Error("expected doc1 in match result")
	}

	m.Remove("doc1")
	got2 := m.Match(map[string]string{"lang": "go", "env": "prod"})
	if len(got2) != 0 {
		t.Errorf("expected empty after remove, got %v", got2)
	}
}

// TestMetadataSnapshotRoundtrip verifies that EncodeSnapshot/DecodeMetadataSnapshot
// produces a functionally identical index.
func TestMetadataSnapshotRoundtrip(t *testing.T) {
	t.Parallel()
	m := NewMetadataIndex()
	m.Set("doc1", map[string]string{"lang": "go", "env": "prod"})
	m.Set("doc2", map[string]string{"lang": "python", "env": "prod"})
	m.Set("doc3", map[string]string{"lang": "go", "env": "staging"})

	snap := m.EncodeSnapshot()
	if len(snap) == 0 {
		t.Fatal("empty snapshot")
	}

	m2, err := DecodeMetadataSnapshot(snap)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	cases := []struct {
		filters map[string]string
		wantIDs []string
	}{
		{map[string]string{"lang": "go"}, []string{"doc1", "doc3"}},
		{map[string]string{"lang": "python"}, []string{"doc2"}},
		{map[string]string{"env": "prod"}, []string{"doc1", "doc2"}},
		{map[string]string{"lang": "go", "env": "prod"}, []string{"doc1"}},
		{map[string]string{"lang": "rust"}, nil},
	}
	for _, tc := range cases {
		got := m2.Match(tc.filters)
		if len(got) != len(tc.wantIDs) {
			t.Errorf("Match(%v): got %d results, want %d: %v", tc.filters, len(got), len(tc.wantIDs), got)
			continue
		}
		for _, id := range tc.wantIDs {
			if _, ok := got[id]; !ok {
				t.Errorf("Match(%v): missing %s in %v", tc.filters, id, got)
			}
		}
	}
}

// TestMetadataSnapshotEmpty verifies that an empty index encodes and decodes cleanly.
func TestMetadataSnapshotEmpty(t *testing.T) {
	t.Parallel()
	m := NewMetadataIndex()
	snap := m.EncodeSnapshot()
	m2, err := DecodeMetadataSnapshot(snap)
	if err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if got := m2.Match(map[string]string{"k": "v"}); len(got) != 0 {
		t.Errorf("expected empty match on decoded empty index, got %v", got)
	}
}
