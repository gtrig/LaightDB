package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		w.Header().Set("X-Request-ID", id)
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("http", "method", r.Method, "path", r.URL.Path, "dur_ms", time.Since(start).Milliseconds(), "req_id", id)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic", "err", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
