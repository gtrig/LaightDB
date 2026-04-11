package stress

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Config drives the HTTP stress workload against a running LaightDB server.
type Config struct {
	BaseURL string // e.g. http://127.0.0.1:8080 (no trailing slash)
	Token   string // optional Bearer token

	Collection string

	Writes            int
	WriteConcurrency  int
	Searches          int
	SearchConcurrency int

	TopK   int
	Detail string

	HTTPTimeout time.Duration
}

// Report aggregates timing and error counts for writes and searches.
type Report struct {
	BaseURL    string        `json:"base_url"`
	Collection string        `json:"collection"`
	Writes     PhaseStat     `json:"writes"`
	Searches   PhaseStat     `json:"searches"`
	TotalWall  time.Duration `json:"total_wall"`
}

// PhaseStat is timing for one workload phase (writes or searches).
type PhaseStat struct {
	Requested int           `json:"requested"`
	OK        int           `json:"ok"`
	Errors    int           `json:"errors"`
	Wall      time.Duration `json:"wall"`
	P50       time.Duration `json:"p50"`
	P95       time.Duration `json:"p95"`
	P99       time.Duration `json:"p99"`
	OpsPerSec float64       `json:"ops_per_sec"`
}

// Run executes the load phase (POST /v1/contexts) then the search phase (POST /v1/search).
func Run(ctx context.Context, cfg Config) (*Report, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("stress: base URL is required")
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Collection == "" {
		cfg.Collection = "stress"
	}
	if cfg.Writes < 0 || cfg.Searches < 0 {
		return nil, fmt.Errorf("stress: writes and searches must be non-negative")
	}
	if cfg.WriteConcurrency < 1 {
		cfg.WriteConcurrency = 1
	}
	if cfg.SearchConcurrency < 1 {
		cfg.SearchConcurrency = 1
	}
	if cfg.TopK < 1 {
		cfg.TopK = 10
	}
	if cfg.Detail == "" {
		cfg.Detail = "summary"
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 5 * time.Minute
	}

	client := &http.Client{Timeout: cfg.HTTPTimeout}
	r := &Report{BaseURL: base, Collection: cfg.Collection}

	t0 := time.Now()

	writeLat, writeErr, err := runWrites(ctx, client, base, cfg.Token, cfg.Collection, cfg.Writes, cfg.WriteConcurrency)
	if err != nil {
		return nil, err
	}
	fillPhase(&r.Writes, cfg.Writes, writeLat, writeErr, time.Since(t0))

	t1 := time.Now()
	searchLat, searchErr, err := runSearches(ctx, client, base, cfg.Token, cfg.Collection, cfg.Searches, cfg.SearchConcurrency, cfg.TopK, cfg.Detail)
	if err != nil {
		return nil, err
	}
	fillPhase(&r.Searches, cfg.Searches, searchLat, searchErr, time.Since(t1))

	r.TotalWall = time.Since(t0)
	return r, nil
}

func fillPhase(dst *PhaseStat, requested int, latencies []time.Duration, errs int, wall time.Duration) {
	dst.Requested = requested
	dst.Errors = errs
	dst.Wall = wall
	dst.OK = len(latencies)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	dst.P50 = percentile(latencies, 0.50)
	dst.P95 = percentile(latencies, 0.95)
	dst.P99 = percentile(latencies, 0.99)
	if wall > 0 && len(latencies) > 0 {
		dst.OpsPerSec = float64(len(latencies)) / wall.Seconds()
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Round(float64(len(sorted)-1) * p))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func runWrites(ctx context.Context, client *http.Client, base, token, collection string, n, workers int) ([]time.Duration, int, error) {
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
				body := map[string]any{
					"collection":   collection,
					"content_type": "doc",
					"content":      writeBody(i),
					"metadata": map[string]string{
						"stress_id": fmt.Sprintf("%d", i),
						"workload":  "laightdb-stress",
					},
				}
				start := time.Now()
				code, err := postJSON(ctx, client, base+"/v1/contexts", token, body)
				d := time.Since(start)
				if err != nil || code != http.StatusOK {
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

func writeBody(i int) string {
	return fmt.Sprintf(`Stress document %d. Topics: error handling, database storage, concurrent access, vector search, full text indexing, authentication, REST API, compaction, WAL, memtable. Keywords: implementation configuration middleware embedding chunking summarization metadata filters hybrid retrieval.`, i)
}

func runSearches(ctx context.Context, client *http.Client, base, token, collection string, n, workers, topK int, detail string) ([]time.Duration, int, error) {
	if n == 0 {
		return nil, 0, nil
	}
	queries := StandardQueries
	if len(queries) == 0 {
		return nil, 0, fmt.Errorf("stress: no standard queries")
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
				q := queries[i%len(queries)]
				body := map[string]any{
					"query":      q,
					"collection": collection,
					"top_k":      topK,
					"detail":     detail,
				}
				start := time.Now()
				code, err := postJSON(ctx, client, base+"/v1/search", token, body)
				d := time.Since(start)
				if err != nil || code != http.StatusOK {
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

func postJSON(ctx context.Context, client *http.Client, url, token string, payload any) (int, error) {
	buf, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return 0, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_, copyErr := io.Copy(io.Discard, resp.Body)
	closeErr := resp.Body.Close()
	if copyErr != nil {
		return resp.StatusCode, fmt.Errorf("read body: %w", copyErr)
	}
	if closeErr != nil {
		return resp.StatusCode, fmt.Errorf("close body: %w", closeErr)
	}
	return resp.StatusCode, nil
}
