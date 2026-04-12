package config

import (
	"testing"
	"time"
)

func TestFromEnv_defaults(t *testing.T) {
	// Clear relevant env so defaults apply (cannot use t.Parallel with t.Setenv).
	for _, k := range []string{
		"LAIGHTDB_DATA_DIR", "LAIGHTDB_HTTP_ADDR", "LAIGHTDB_MCP_TRANSPORT",
		"LAIGHTDB_MEMTABLE_SIZE", "LAIGHTDB_CORS_ORIGIN",
	} {
		t.Setenv(k, "")
	}
	cfg := FromEnv()
	if cfg.DataDir != "./data" {
		t.Errorf("DataDir = %q", cfg.DataDir)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.MCPTransport != "stdio" {
		t.Errorf("MCPTransport = %q", cfg.MCPTransport)
	}
	if cfg.MemtableBytes != 4<<20 {
		t.Errorf("MemtableBytes = %d", cfg.MemtableBytes)
	}
	if cfg.CorsOrigin != "" {
		t.Errorf("CorsOrigin = %q", cfg.CorsOrigin)
	}
}

func TestFromEnv_overrides(t *testing.T) {
	t.Setenv("LAIGHTDB_DATA_DIR", "/data/db")
	t.Setenv("LAIGHTDB_HTTP_ADDR", ":9999")
	t.Setenv("LAIGHTDB_MCP_TRANSPORT", "http")
	t.Setenv("LAIGHTDB_MEMTABLE_SIZE", "8388608")
	t.Setenv("LAIGHTDB_SESSION_TTL", "48h")
	t.Setenv("LAIGHTDB_CORS_ORIGIN", "https://app.example.com")
	cfg := FromEnv()
	if cfg.DataDir != "/data/db" {
		t.Errorf("DataDir = %q", cfg.DataDir)
	}
	if cfg.HTTPAddr != ":9999" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.MCPTransport != "http" {
		t.Errorf("MCPTransport = %q", cfg.MCPTransport)
	}
	if cfg.MemtableBytes != 8388608 {
		t.Errorf("MemtableBytes = %d", cfg.MemtableBytes)
	}
	if cfg.SessionTTL != 48*time.Hour {
		t.Errorf("SessionTTL = %v", cfg.SessionTTL)
	}
	if cfg.CorsOrigin != "https://app.example.com" {
		t.Errorf("CorsOrigin = %q", cfg.CorsOrigin)
	}
}

func TestFromEnv_invalidIntFallsBack(t *testing.T) {
	t.Setenv("LAIGHTDB_MEMTABLE_SIZE", "not-a-number")
	cfg := FromEnv()
	if cfg.MemtableBytes != 4<<20 {
		t.Errorf("expected default memtable size, got %d", cfg.MemtableBytes)
	}
}
