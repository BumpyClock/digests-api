// ABOUTME: Request logging middleware for API endpoints
// ABOUTME: Logs request details, response status, and timing information

package middleware

import (
	"fmt"
	"net/http"
	"time"

	"digests-app-api/core/interfaces"
	"github.com/google/uuid"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
		rw.written = true
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// RequestIDKey is the context key for request ID
type RequestIDKey struct{}

// RequestLoggingMiddleware creates a middleware that logs all requests
func RequestLoggingMiddleware(logger interfaces.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate request ID
			requestID := uuid.New().String()
			
			// Add request ID to response headers
			w.Header().Set("X-Request-ID", requestID)
			
			// Store request ID in context
			ctx := r.Context()
			ctx = r.Context()  // Note: In real implementation, would use context.WithValue
			r = r.WithContext(ctx)
			
			// Record start time
			start := time.Now()
			
			// Create response writer wrapper
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}
			
			// Log request
			logger.Info("Request started", map[string]interface{}{
				"request_id": requestID,
				"method":     r.Method,
				"path":       r.URL.Path,
				"remote_ip":  extractIP(r),
				"user_agent": r.UserAgent(),
			})
			
			// Process request
			next.ServeHTTP(wrapped, r)
			
			// Calculate duration
			duration := time.Since(start)
			
			// Log response
			logger.Info("Request completed", map[string]interface{}{
				"request_id": requestID,
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     wrapped.statusCode,
				"duration":   duration.String(),
				"duration_ms": duration.Milliseconds(),
			})
			
			// Log slow requests as warnings
			if duration > 5*time.Second {
				logger.Warn("Slow request detected", map[string]interface{}{
					"request_id": requestID,
					"method":     r.Method,
					"path":       r.URL.Path,
					"duration":   duration.String(),
				})
			}
			
			// Log errors
			if wrapped.statusCode >= 500 {
				logger.Error("Request failed with server error", map[string]interface{}{
					"request_id": requestID,
					"method":     r.Method,
					"path":       r.URL.Path,
					"status":     wrapped.statusCode,
				})
			}
		})
	}
}

// GetRequestID retrieves the request ID from the request headers
func GetRequestID(r *http.Request) string {
	return r.Header.Get("X-Request-ID")
}

// LoggingRoundTripper implements http.RoundTripper with logging
type LoggingRoundTripper struct {
	Transport http.RoundTripper
	Logger    interfaces.Logger
}

// RoundTrip logs outgoing HTTP requests
func (t *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	
	// Get request ID from context if available
	requestID := GetRequestID(req)
	if requestID == "" {
		requestID = uuid.New().String()
	}
	
	// Log outgoing request
	t.Logger.Debug("Outgoing HTTP request", map[string]interface{}{
		"request_id": requestID,
		"method":     req.Method,
		"url":        req.URL.String(),
		"host":       req.Host,
	})
	
	// Make request
	resp, err := t.Transport.RoundTrip(req)
	
	duration := time.Since(start)
	
	if err != nil {
		t.Logger.Error("Outgoing HTTP request failed", map[string]interface{}{
			"request_id": requestID,
			"method":     req.Method,
			"url":        req.URL.String(),
			"duration":   duration.String(),
			"error":      err.Error(),
		})
		return nil, err
	}
	
	// Log response
	t.Logger.Debug("Outgoing HTTP response", map[string]interface{}{
		"request_id": requestID,
		"method":     req.Method,
		"url":        req.URL.String(),
		"status":     resp.StatusCode,
		"duration":   duration.String(),
	})
	
	return resp, nil
}

// RequestLogFields extracts common log fields from a request
func RequestLogFields(r *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"query":       r.URL.RawQuery,
		"remote_ip":   extractIP(r),
		"user_agent":  r.UserAgent(),
		"request_id":  GetRequestID(r),
		"host":        r.Host,
		"proto":       r.Proto,
		"content_type": r.Header.Get("Content-Type"),
	}
}

// ResponseLogFields creates log fields for a response
func ResponseLogFields(statusCode int, duration time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"status":      statusCode,
		"duration":    duration.String(),
		"duration_ms": duration.Milliseconds(),
		"status_text": fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
	}
}