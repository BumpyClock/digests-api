// ABOUTME: Rate limiting middleware for API endpoints
// ABOUTME: Implements per-IP rate limiting with configurable limits

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter tracks request counts per IP
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]*bucket
	limit    int
	window   time.Duration
}

// bucket tracks requests for a specific key
type bucket struct {
	count      int
	windowStart time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*bucket),
		limit:    limit,
		window:   window,
	}
	
	// Start cleanup goroutine
	go rl.cleanup()
	
	return rl
}

// cleanup removes expired buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.requests {
			if now.Sub(b.windowStart) > rl.window {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request from the given key is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	b, exists := rl.requests[key]
	
	if !exists || now.Sub(b.windowStart) > rl.window {
		// New bucket or window expired
		rl.requests[key] = &bucket{
			count:       1,
			windowStart: now,
		}
		return true
	}
	
	if b.count < rl.limit {
		b.count++
		return true
	}
	
	return false
}

// extractIP gets the client IP from the request
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		if idx := len(xff) - 1; idx >= 0 {
			for i := idx; i >= 0; i-- {
				if xff[i] == ',' || xff[i] == ' ' {
					return xff[i+1:]
				}
			}
			return xff
		}
	}
	
	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// RateLimitMiddleware creates a middleware that enforces rate limits
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			
			if !limiter.Allow(ip) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.limit))
				w.Header().Set("X-RateLimit-Window", limiter.window.String())
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(limiter.window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"Too many requests","message":"Rate limit exceeded. Please try again later."}`))
				return
			}
			
			// Add rate limit headers to successful responses
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.limit))
			w.Header().Set("X-RateLimit-Window", limiter.window.String())
			
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitContext allows passing rate limit info through context
type rateLimitContextKey struct{}

// GetRateLimitInfo retrieves rate limit info from context
func GetRateLimitInfo(ctx context.Context) (limit int, window time.Duration, ok bool) {
	val := ctx.Value(rateLimitContextKey{})
	if val == nil {
		return 0, 0, false
	}
	
	info, ok := val.(struct {
		Limit  int
		Window time.Duration
	})
	
	return info.Limit, info.Window, ok
}