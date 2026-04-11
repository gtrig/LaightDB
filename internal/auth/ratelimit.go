package auth

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"sync"
	"time"
)

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// RateLimiter implements a per-key token bucket rate limiter.
type RateLimiter struct {
	rps   float64
	burst float64

	mu      sync.Mutex
	buckets map[string]*bucket
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		rps:     rps,
		burst:   float64(burst),
		buckets: make(map[string]*bucket),
	}
}

// Allow checks whether a request for the given key is allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{tokens: rl.burst, lastFill: now}
		rl.buckets[key] = b
	}

	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens = min(rl.burst, b.tokens+elapsed*rl.rps)
	b.lastFill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// RetryAfter returns how many seconds until 1 token is available for the key.
func (rl *RateLimiter) RetryAfter(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		return 0
	}
	deficit := 1.0 - b.tokens
	if deficit <= 0 {
		return 0
	}
	return int(math.Ceil(deficit / rl.rps))
}

// Prune removes buckets not accessed for the given duration.
func (rl *RateLimiter) Prune(staleAfter time.Duration) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-staleAfter)
	pruned := 0
	for k, b := range rl.buckets {
		if b.lastFill.Before(cutoff) {
			delete(rl.buckets, k)
			pruned++
		}
	}
	return pruned
}

// RateLimitMiddleware returns HTTP middleware applying rate limiting.
// Keys by authenticated user ID or by remote IP for unauthenticated requests.
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(r)
			if !rl.Allow(key) {
				retry := rl.RetryAfter(key)
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retry))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitKey(r *http.Request) string {
	if u, ok := UserFromContext(r.Context()); ok {
		return "user:" + u.ID
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "ip:" + r.RemoteAddr
	}
	return "ip:" + host
}
