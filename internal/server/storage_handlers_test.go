package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gtrig/laightdb/internal/storage"
)

func TestStorageDiagnostics_Basic(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	req := httptest.NewRequest(http.MethodGet, "/v1/storage/diagnostics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var diag storage.EngineDiagnostics
	if err := json.NewDecoder(rec.Body).Decode(&diag); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diag.DataDir == "" {
		t.Error("expected non-empty data_dir")
	}
	if diag.WALBytes < 0 {
		t.Error("wal_bytes must be non-negative")
	}
	if diag.MemEntries < 0 {
		t.Error("mem_entries must be non-negative")
	}
	if diag.SSTables == nil {
		t.Error("sstables must not be nil (empty slice ok)")
	}
}
