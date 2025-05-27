package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Second)
	
	// First 3 requests should be allowed
	assert.True(t, rl.Allow("127.0.0.1"))
	assert.True(t, rl.Allow("127.0.0.1"))
	assert.True(t, rl.Allow("127.0.0.1"))
	
	// 4th request should be denied
	assert.False(t, rl.Allow("127.0.0.1"))
	assert.False(t, rl.Allow("127.0.0.1"))
	
	// Different IP should be allowed
	assert.True(t, rl.Allow("192.168.1.1"))
	
	// Wait for window to expire
	time.Sleep(1100 * time.Millisecond)
	
	// Should be allowed again
	assert.True(t, rl.Allow("127.0.0.1"))
}

func TestRateLimitMiddleware_AllowsRequestsUnderLimit(t *testing.T) {
	limiter := NewRateLimiter(5, 1*time.Minute)
	middleware := RateLimitMiddleware(limiter)
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	
	// Make 5 requests (at the limit)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		
		handler.ServeHTTP(rec, req)
		
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
		assert.Equal(t, "5", rec.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimitMiddleware_Returns429ForExceededLimit(t *testing.T) {
	limiter := NewRateLimiter(2, 1*time.Minute)
	middleware := RateLimitMiddleware(limiter)
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	// Make 2 requests (at the limit)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
	
	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Body.String(), "Rate limit exceeded")
	assert.Equal(t, "2", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
}

func TestRateLimitMiddleware_UsesIPAddressForLimiting(t *testing.T) {
	limiter := NewRateLimiter(1, 1*time.Minute)
	middleware := RateLimitMiddleware(limiter)
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	// First request from IP1
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "127.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)
	
	// Second request from IP1 should be limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	
	// First request from IP2 should be allowed
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.1:5678"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)
}

func TestRateLimitMiddleware_ResetsAfterTimeWindow(t *testing.T) {
	limiter := NewRateLimiter(1, 100*time.Millisecond)
	middleware := RateLimitMiddleware(limiter)
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	// First request
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "127.0.0.1:1234"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)
	
	// Second request should be limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "127.0.0.1:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	
	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)
	
	// Third request should be allowed
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "127.0.0.1:1234"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusOK, rec3.Code)
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name        string
		setupReq    func(*http.Request)
		expectedIP  string
	}{
		{
			name: "uses X-Forwarded-For header",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.2")
				r.RemoteAddr = "10.0.0.1:1234"
			},
			expectedIP: "198.51.100.2",
		},
		{
			name: "uses X-Real-IP header",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "203.0.113.1")
				r.RemoteAddr = "10.0.0.1:1234"
			},
			expectedIP: "203.0.113.1",
		},
		{
			name: "falls back to RemoteAddr",
			setupReq: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:1234"
			},
			expectedIP: "192.168.1.1:1234",
		},
		{
			name: "prefers X-Forwarded-For over X-Real-IP",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "203.0.113.1")
				r.Header.Set("X-Real-IP", "198.51.100.1")
				r.RemoteAddr = "10.0.0.1:1234"
			},
			expectedIP: "203.0.113.1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupReq(req)
			
			ip := extractIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}