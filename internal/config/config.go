package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime options (env + flags).
type Config struct {
	DataDir       string
	HTTPAddr      string
	MCPTransport  string
	OllamaURL     string
	Summarizer    string
	MemtableBytes int
	SearchTopK    int

	BootstrapUser  string
	SessionTTL     time.Duration
	RateLimitRPS   float64
	RateLimitBurst int

	// DevMCPHTTPAddr is used by cmd/laightdb-dev-mcp only: if non-empty, serve dev MCP over streamable HTTP instead of stdio.
	DevMCPHTTPAddr string
	// DevMCPSkipStore avoids opening the storage engine (safe next to a running laightdb; disables debug_store_stats).
	DevMCPSkipStore bool
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func envFloat(key string, def float64) float64 {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func envDuration(key string, def time.Duration) time.Duration {
	s := os.Getenv(key)
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}

func envBool(key string, def bool) bool {
	s := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if s == "" {
		return def
	}
	return s == "1" || s == "true" || s == "yes" || s == "on"
}

// Parse loads configuration from flags and environment.
func Parse() *Config {
	cfg := &Config{}
	flag.StringVar(&cfg.DataDir, "data-dir", envOr("LAIGHTDB_DATA_DIR", "./data"), "data directory")
	flag.StringVar(&cfg.HTTPAddr, "http-addr", envOr("LAIGHTDB_HTTP_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&cfg.MCPTransport, "mcp-transport", envOr("LAIGHTDB_MCP_TRANSPORT", "stdio"), "MCP transport: stdio or http")
	flag.StringVar(&cfg.OllamaURL, "ollama-url", envOr("LAIGHTDB_OLLAMA_URL", "http://localhost:11434"), "Ollama base URL")
	flag.StringVar(&cfg.Summarizer, "summarizer", envOr("LAIGHTDB_SUMMARIZER", "noop"), "summarizer: noop, openai, anthropic, ollama")
	flag.IntVar(&cfg.MemtableBytes, "memtable-size", envInt("LAIGHTDB_MEMTABLE_SIZE", 4<<20), "memtable flush threshold (bytes)")
	flag.IntVar(&cfg.SearchTopK, "search-top-k", envInt("LAIGHTDB_SEARCH_TOP_K", 10), "default search top-k")

	cfg.BootstrapUser = envOr("LAIGHTDB_BOOTSTRAP_USER", "")
	cfg.SessionTTL = envDuration("LAIGHTDB_SESSION_TTL", 24*time.Hour)
	cfg.RateLimitRPS = envFloat("LAIGHTDB_RATE_LIMIT_RPS", 100)
	cfg.RateLimitBurst = envInt("LAIGHTDB_RATE_LIMIT_BURST", 200)

	defaultDevMCPHTTP := envOr("LAIGHTDB_DEV_MCP_HTTP_ADDR", "")
	flag.StringVar(&cfg.DevMCPHTTPAddr, "dev-mcp-http-addr", defaultDevMCPHTTP, "laightdb-dev-mcp only: streamable HTTP listen addr (empty = stdio)")
	defaultSkipStore := envBool("LAIGHTDB_DEV_MCP_SKIP_STORE", false)
	flag.BoolVar(&cfg.DevMCPSkipStore, "dev-mcp-skip-store", defaultSkipStore, "laightdb-dev-mcp only: do not open DB files (set true when API already uses the data dir)")

	flag.Parse()
	return cfg
}
