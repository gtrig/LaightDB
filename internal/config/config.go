package config

import (
	"flag"
	"os"
	"strconv"
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
	flag.Parse()
	return cfg
}
