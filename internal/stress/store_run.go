package stress

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	lctx "github.com/gtrig/laightdb/internal/context"
)

// StoreConfig drives in-process stress against a Store (same workload as Run over HTTP).
type StoreConfig struct {
	Collection string

	Writes            int
	WriteConcurrency  int
	Searches          int
	SearchConcurrency int

	TopK   int
	Detail string
}

const (
	// MaxStoreWrites caps POST /v1/stress and RunStore.
	MaxStoreWrites = 50000
	// MaxStoreSearches caps search operations in one run.
	MaxStoreSearches = 500000
	// MaxStoreConcurrency caps worker goroutines per phase.
	MaxStoreConcurrency = 128
)

// RunStore runs the load phase (Put) then the search phase (Search) against the store.
// It does not use HTTP; metrics reflect database work only (no JSON/HTTP overhead).
func RunStore(ctx context.Context, store *lctx.Store, cfg StoreConfig) (*Report, error) {
	if err := normalizeStoreConfig(&cfg); err != nil {
		return nil, err
	}

	r := &Report{
		BaseURL:    "in-process",
		Collection: cfg.Collection,
	}

	t0 := time.Now()
	writeLat, writeErr, err := runWritesStore(ctx, store, cfg.Collection, cfg.Writes, cfg.WriteConcurrency)
	if err != nil {
		return nil, err
	}
	fillPhase(&r.Writes, cfg.Writes, writeLat, writeErr, time.Since(t0))

	t1 := time.Now()
	searchLat, searchErr, err := runSearchesStore(ctx, store, cfg.Collection, cfg.Searches, cfg.SearchConcurrency, cfg.TopK, cfg.Detail)
	if err != nil {
		return nil, err
	}
	fillPhase(&r.Searches, cfg.Searches, searchLat, searchErr, time.Since(t1))

	r.TotalWall = time.Since(t0)
	return r, nil
}

func normalizeStoreConfig(cfg *StoreConfig) error {
	if cfg.Collection == "" {
		cfg.Collection = "stress"
	}
	if cfg.Writes < 0 || cfg.Searches < 0 {
		return fmt.Errorf("stress: writes and searches must be non-negative")
	}
	if cfg.Writes > MaxStoreWrites || cfg.Searches > MaxStoreSearches {
		return fmt.Errorf("stress: writes max %d, searches max %d", MaxStoreWrites, MaxStoreSearches)
	}
	if cfg.WriteConcurrency < 1 {
		cfg.WriteConcurrency = 1
	}
	if cfg.SearchConcurrency < 1 {
		cfg.SearchConcurrency = 1
	}
	if cfg.WriteConcurrency > MaxStoreConcurrency {
		cfg.WriteConcurrency = MaxStoreConcurrency
	}
	if cfg.SearchConcurrency > MaxStoreConcurrency {
		cfg.SearchConcurrency = MaxStoreConcurrency
	}
	if cfg.TopK < 1 {
		cfg.TopK = 10
	}
	if cfg.Detail == "" {
		cfg.Detail = "summary"
	}
	return nil
}

func runWritesStore(ctx context.Context, store *lctx.Store, collection string, n, workers int) ([]time.Duration, int, error) {
	if n == 0 {
		return nil, 0, nil
	}
	var (
		mu       sync.Mutex
		lat      []time.Duration
		errCount int32
		next     atomic.Int64
	)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for {
				i := int(next.Add(1)) - 1
				if i >= n {
					return
				}
				start := time.Now()
				_, err := store.Put(ctx, lctx.PutRequest{
					Collection:  collection,
					ContentType: "doc",
					Content:     writeBody(i),
					Metadata: map[string]string{
						"stress_id": fmt.Sprintf("%d", i),
						"workload":  "laightdb-stress",
					},
				})
				d := time.Since(start)
				if err != nil {
					atomic.AddInt32(&errCount, 1)
					continue
				}
				mu.Lock()
				lat = append(lat, d)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return lat, int(errCount), nil
}

func runSearchesStore(ctx context.Context, store *lctx.Store, collection string, n, workers, topK int, detail string) ([]time.Duration, int, error) {
	if n == 0 {
		return nil, 0, nil
	}
	queries := StandardQueries
	if len(queries) == 0 {
		return nil, 0, fmt.Errorf("stress: no standard queries")
	}
	dl := lctx.DetailLevel(detail)
	var (
		mu       sync.Mutex
		lat      []time.Duration
		errCount int32
		next     atomic.Int64
	)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for {
				i := int(next.Add(1)) - 1
				if i >= n {
					return
				}
				q := queries[i%len(queries)]
				start := time.Now()
				_, err := store.Search(ctx, lctx.SearchRequest{
					Query:      q,
					Collection: collection,
					TopK:       topK,
					Detail:     dl,
				})
				d := time.Since(start)
				if err != nil {
					atomic.AddInt32(&errCount, 1)
					continue
				}
				mu.Lock()
				lat = append(lat, d)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return lat, int(errCount), nil
}
