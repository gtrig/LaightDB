package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(10, 5) // 10 rps, burst of 5

	// First 5 should succeed (burst)
	for i := range 5 {
		if !rl.Allow("test") {
			t.Fatalf("request %d should be allowed", i)
		}
	}
	// 6th should fail
	if rl.Allow("test") {
		t.Fatal("6th request should be rejected")
	}
}

func TestRateLimiterRefill(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1000, 1) // 1000 rps, burst of 1

	if !rl.Allow("key") {
		t.Fatal("first should pass")
	}
	if rl.Allow("key") {
		t.Fatal("second should fail")
	}

	time.Sleep(5 * time.Millisecond) // enough for 5 tokens at 1000 rps

	if !rl.Allow("key") {
		t.Fatal("should be allowed after refill")
	}
}

func TestRateLimiterPerKeyIsolation(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(10, 2)

	rl.Allow("a")
	rl.Allow("a")
	if rl.Allow("a") {
		t.Fatal("a should be exhausted")
	}

	// b should still work
	if !rl.Allow("b") {
		t.Fatal("b should be allowed")
	}
}

func TestRateLimiterRetryAfter(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(1, 1) // 1 rps, burst 1

	rl.Allow("key")
	retry := rl.RetryAfter("key")
	if retry < 1 {
		t.Fatalf("expected retry >= 1, got %d", retry)
	}
}

func TestRateLimiterPrune(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(10, 5)
	rl.Allow("old")
	time.Sleep(10 * time.Millisecond)
	rl.Allow("new")

	pruned := rl.Prune(5 * time.Millisecond)
	if pruned != 1 {
		t.Fatalf("expected 1 pruned, got %d", pruned)
	}
}

func TestRateLimitMiddleware429(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(10, 1)
	handler := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	// First request OK
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Second should be rate limited
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}
