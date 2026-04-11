package mcpdev

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ListenAndServeHTTP serves the dev MCP at /mcp (streamable HTTP) and GET /health.
// We avoid registering "GET /" alongside "/mcp": Go 1.22+ ServeMux rejects that pattern pair (method-specific root vs all-methods path).
func (s *Server) ListenAndServeHTTP(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("GET /health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintf(w, "ok\nstreamable MCP: POST %s/mcp\n", addr)
	}))
	mux.Handle("/mcp", s.StreamableHTTPHandler())

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		slog.Info("dev mcp streamable http (do not expose to untrusted networks)", "addr", addr, "mcp_path", "/mcp", "data_dir", s.DataDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("dev mcp http", "err", err)
		}
	}()

	<-ctx.Done()
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shCtx)
}
