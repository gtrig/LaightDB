// Command laightdb-stress loads synthetic context and runs fixed search queries against a running server, printing latency metrics.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/gtrig/laightdb/internal/stress"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	baseURL := flag.String("url", "http://127.0.0.1:8080", "LaightDB base URL (no trailing slash)")
	token := flag.String("token", "", "Bearer API token (if auth is enabled)")
	collection := flag.String("collection", "stress", "Collection name for stored documents")
	writes := flag.Int("writes", 50, "Number of POST /v1/contexts requests")
	writeConc := flag.Int("write-concurrency", 4, "Concurrent writers")
	searches := flag.Int("searches", 200, "Number of POST /v1/search requests")
	searchConc := flag.Int("search-concurrency", 8, "Concurrent search requests")
	topK := flag.Int("top-k", 10, "search top_k")
	detail := flag.String("detail", "summary", "search detail level (summary|full)")
	timeout := flag.Duration("timeout", 5*time.Minute, "Per-request HTTP timeout")
	jsonOut := flag.Bool("json", false, "Print report as JSON to stdout")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	rep, err := stress.Run(ctx, stress.Config{
		BaseURL:           *baseURL,
		Token:             *token,
		Collection:        *collection,
		Writes:            *writes,
		WriteConcurrency:  *writeConc,
		Searches:          *searches,
		SearchConcurrency: *searchConc,
		TopK:              *topK,
		Detail:            *detail,
		HTTPTimeout:       *timeout,
	})
	if err != nil {
		slog.Error("stress run", "err", err)
		os.Exit(1)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			slog.Error("encode json", "err", err)
			os.Exit(1)
		}
		return
	}

	printText(rep)
}

func printText(r *stress.Report) {
	fmt.Printf("LaightDB stress — %s (collection %q)\n\n", r.BaseURL, r.Collection)
	printPhase("Writes", r.Writes)
	printPhase("Searches", r.Searches)
	fmt.Printf("Total wall: %s\n", r.TotalWall.Round(time.Millisecond))
}

func printPhase(name string, p stress.PhaseStat) {
	fmt.Printf("%s: requested=%d ok=%d errors=%d wall=%s\n",
		name, p.Requested, p.OK, p.Errors, p.Wall.Round(time.Millisecond))
	if p.OK > 0 {
		fmt.Printf("  p50=%s p95=%s p99=%s  %.2f ops/s\n",
			p.P50.Round(time.Millisecond),
			p.P95.Round(time.Millisecond),
			p.P99.Round(time.Millisecond),
			p.OpsPerSec,
		)
	}
	fmt.Println()
}
