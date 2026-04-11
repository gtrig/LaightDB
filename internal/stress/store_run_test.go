package stress

import (
	"testing"

	lctx "github.com/gtrig/laightdb/internal/context"
	"github.com/gtrig/laightdb/internal/summarize"
)

func TestRunStore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := lctx.OpenStore(t.Context(), dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	rep, err := RunStore(t.Context(), st, StoreConfig{
		Collection:        "bench",
		Writes:            4,
		WriteConcurrency:  2,
		Searches:          6,
		SearchConcurrency: 3,
		TopK:              5,
		Detail:            "summary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.BaseURL != "in-process" {
		t.Fatalf("base_url: %q", rep.BaseURL)
	}
	if rep.Collection != "bench" {
		t.Fatalf("collection: %q", rep.Collection)
	}
	if rep.Writes.Requested != 4 || rep.Writes.OK != 4 || rep.Writes.Errors != 0 {
		t.Fatalf("writes: %+v", rep.Writes)
	}
	if rep.Searches.Requested != 6 || rep.Searches.OK != 6 || rep.Searches.Errors != 0 {
		t.Fatalf("searches: %+v", rep.Searches)
	}
	if rep.TotalWall <= 0 {
		t.Fatal("total wall")
	}
}

func TestRunStoreRejectsExcess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	st, err := lctx.OpenStore(t.Context(), dir, 1<<20, nil, summarize.Noop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	_, err = RunStore(t.Context(), st, StoreConfig{
		Writes: MaxStoreWrites + 1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
