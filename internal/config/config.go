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

	// CorsOrigin is the value for Access-Control-Allow-Origin (e.g. https://app.example.com). Empty disables CORS.
	CorsOrigin string
}

// FromEnv loads configuration from environment variables only (no CLI flags).
// Used by tests and as defaults before flag registration in Parse.
func FromEnv() *Config {
	cfg := &Config{}
	cfg.DataDir = envOr("LAIGHTDB_DATA_DIR", "./data")
	cfg.HTTPAddr = envOr("LAIGHTDB_HTTP_ADDR", ":8080")
	cfg.MCPTransport = envOr("LAIGHTDB_MCP_TRANSPORT", "stdio")
	cfg.OllamaURL = envOr("LAIGHTDB_OLLAMA_URL", "http://localhost:11434")
	cfg.Summarizer = envOr("LAIGHTDB_SUMMARIZER", "noop")
	cfg.MemtableBytes = envInt("LAIGHTDB_MEMTABLE_SIZE", 4<<20)
	cfg.SearchTopK = envInt("LAIGHTDB_SEARCH_TOP_K", 10)
	cfg.BootstrapUser = envOr("LAIGHTDB_BOOTSTRAP_USER", "")
	cfg.SessionTTL = envDuration("LAIGHTDB_SESSION_TTL", 24*time.Hour)
	cfg.RateLimitRPS = envFloat("LAIGHTDB_RATE_LIMIT_RPS", 100)
	cfg.RateLimitBurst = envInt("LAIGHTDB_RATE_LIMIT_BURST", 200)
	cfg.DevMCPHTTPAddr = envOr("LAIGHTDB_DEV_MCP_HTTP_ADDR", "")
	cfg.DevMCPSkipStore = envBool("LAIGHTDB_DEV_MCP_SKIP_STORE", false)
	cfg.CorsOrigin = strings.TrimSpace(envOr("LAIGHTDB_CORS_ORIGIN", ""))
	return cfg
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
	cfg := FromEnv()
	flag.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "data directory")
	flag.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	flag.StringVar(&cfg.MCPTransport, "mcp-transport", cfg.MCPTransport, "MCP transport: stdio or http")
	flag.StringVar(&cfg.OllamaURL, "ollama-url", cfg.OllamaURL, "Ollama base URL")
	flag.StringVar(&cfg.Summarizer, "summarizer", cfg.Summarizer, "summarizer: noop, openai, anthropic, ollama")
	flag.IntVar(&cfg.MemtableBytes, "memtable-size", cfg.MemtableBytes, "memtable flush threshold (bytes)")
	flag.IntVar(&cfg.SearchTopK, "search-top-k", cfg.SearchTopK, "default search top-k")

	flag.StringVar(&cfg.DevMCPHTTPAddr, "dev-mcp-http-addr", cfg.DevMCPHTTPAddr, "laightdb-dev-mcp only: streamable HTTP listen addr (empty = stdio)")
	flag.BoolVar(&cfg.DevMCPSkipStore, "dev-mcp-skip-store", cfg.DevMCPSkipStore, "laightdb-dev-mcp only: do not open DB files (set true when API already uses the data dir)")

	flag.Parse()
	return cfg
}
