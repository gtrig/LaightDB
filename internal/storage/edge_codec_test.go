package storage

import (
	"testing"
	"time"
)

func TestEncodeDecodeEdge_RoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Unix(1700000000, 999999000).UTC()
	e := Edge{
		ID:       "edge-1",
		FromID:   "node-a",
		ToID:     "node-b",
		Label:    "child",
		Weight:   0.85,
		Source:   "user",
		Metadata: map[string]string{"note": "important", "rank": "1"},
		CreatedAt: now,
	}
	data := EncodeEdge(e)
	got, err := DecodeEdge(data)
	if err != nil {
		t.Fatalf("DecodeEdge: %v", err)
	}
	if got.ID != e.ID {
		t.Errorf("ID: got %q want %q", got.ID, e.ID)
	}
	if got.FromID != e.FromID {
		t.Errorf("FromID: got %q want %q", got.FromID, e.FromID)
	}
	if got.ToID != e.ToID {
		t.Errorf("ToID: got %q want %q", got.ToID, e.ToID)
	}
	if got.Label != e.Label {
		t.Errorf("Label: got %q want %q", got.Label, e.Label)
	}
	if got.Weight != e.Weight {
		t.Errorf("Weight: got %v want %v", got.Weight, e.Weight)
	}
	if got.Source != e.Source {
		t.Errorf("Source: got %q want %q", got.Source, e.Source)
	}
	if got.CreatedAt.UnixNano() != e.CreatedAt.UnixNano() {
		t.Errorf("CreatedAt: got %v want %v", got.CreatedAt, e.CreatedAt)
	}
	if len(got.Metadata) != len(e.Metadata) {
		t.Fatalf("Metadata len: got %d want %d", len(got.Metadata), len(e.Metadata))
	}
	for k, v := range e.Metadata {
		if got.Metadata[k] != v {
			t.Errorf("Metadata[%q]: got %q want %q", k, got.Metadata[k], v)
		}
	}
}

func TestEncodeDecodeEdge_ZeroWeight(t *testing.T) {
	t.Parallel()
	e := Edge{
		ID:     "edge-2",
		FromID: "a",
		ToID:   "b",
		Label:  "related_to",
		Weight: 0.0,
		Source: "auto",
	}
	got, err := DecodeEdge(EncodeEdge(e))
	if err != nil {
		t.Fatalf("DecodeEdge: %v", err)
	}
	if got.Weight != 0.0 {
		t.Errorf("Weight: got %v want 0.0", got.Weight)
	}
	if got.Metadata != nil {
		t.Errorf("Metadata should be nil for empty input, got %v", got.Metadata)
	}
}

func TestEncodeDecodeEdge_EmptyStrings(t *testing.T) {
	t.Parallel()
	e := Edge{ID: "x"}
	got, err := DecodeEdge(EncodeEdge(e))
	if err != nil {
		t.Fatalf("DecodeEdge: %v", err)
	}
	if got.ID != "x" {
		t.Errorf("ID: got %q want %q", got.ID, "x")
	}
	if got.FromID != "" || got.ToID != "" || got.Label != "" || got.Source != "" {
		t.Errorf("unexpected non-empty fields: %+v", got)
	}
}

func TestDecodeEdge_TruncatedData(t *testing.T) {
	t.Parallel()
	_, err := DecodeEdge(nil)
	if err == nil {
		t.Error("expected error for nil data")
	}
	_, err = DecodeEdge([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestDecodeEdge_WrongVersion(t *testing.T) {
	t.Parallel()
	_, err := DecodeEdge([]byte{0x99})
	if err == nil {
		t.Error("expected error for wrong version byte")
	}
}

func TestEncodeDecodeEdge_LargeWeight(t *testing.T) {
	t.Parallel()
	e := Edge{ID: "e", Weight: 1e308}
	got, err := DecodeEdge(EncodeEdge(e))
	if err != nil {
		t.Fatalf("DecodeEdge: %v", err)
	}
	if got.Weight != 1e308 {
		t.Errorf("Weight: got %v want 1e308", got.Weight)
	}
}

func TestEncodeDecodeEdge_NegativeWeight(t *testing.T) {
	t.Parallel()
	e := Edge{ID: "e", Weight: -0.5}
	got, err := DecodeEdge(EncodeEdge(e))
	if err != nil {
		t.Fatalf("DecodeEdge: %v", err)
	}
	if got.Weight != -0.5 {
		t.Errorf("Weight: got %v want -0.5", got.Weight)
	}
}
