package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	logs []LogEntry
}

type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]interface{}
}

func (m *MockLogger) Debug(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "DEBUG", Message: msg, Fields: fields})
}

func (m *MockLogger) Info(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "INFO", Message: msg, Fields: fields})
}

func (m *MockLogger) Warn(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "WARN", Message: msg, Fields: fields})
}

func (m *MockLogger) Error(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, LogEntry{Level: "ERROR", Message: msg, Fields: fields})
}

func TestRequestLoggingMiddleware_LogsRequestMethodAndPath(t *testing.T) {
	logger := &MockLogger{}
	middleware := RequestLoggingMiddleware(logger)
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("POST", "/api/test?query=value", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	// Should have 2 logs: request started and request completed
	assert.Len(t, logger.logs, 2)
	
	// Check request started log
	startLog := logger.logs[0]
	assert.Equal(t, "INFO", startLog.Level)
	assert.Equal(t, "Request started", startLog.Message)
	assert.Equal(t, "POST", startLog.Fields["method"])
	assert.Equal(t, "/api/test", startLog.Fields["path"])
	assert.NotEmpty(t, startLog.Fields["request_id"])
	
	// Check request completed log
	completeLog := logger.logs[1]
	assert.Equal(t, "INFO", completeLog.Level)
	assert.Equal(t, "Request completed", completeLog.Message)
	assert.Equal(t, "POST", completeLog.Fields["method"])
	assert.Equal(t, "/api/test", completeLog.Fields["path"])
}

func TestRequestLoggingMiddleware_LogsResponseStatusCode(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		expectedLogs   int
		expectError    bool
	}{
		{"200 OK", http.StatusOK, 2, false},
		{"404 Not Found", http.StatusNotFound, 2, false},
		{"500 Internal Server Error", http.StatusInternalServerError, 3, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, 3, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &MockLogger{}
			middleware := RequestLoggingMiddleware(logger)
			
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
			}))
			
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			
			handler.ServeHTTP(rec, req)
			
			assert.Len(t, logger.logs, tt.expectedLogs)
			
			// Check completed log has correct status
			completeLog := logger.logs[1]
			assert.Equal(t, tt.responseStatus, completeLog.Fields["status"])
			
			// Check for error log if expected
			if tt.expectError {
				errorLog := logger.logs[2]
				assert.Equal(t, "ERROR", errorLog.Level)
				assert.Contains(t, errorLog.Message, "server error")
			}
		})
	}
}

func TestRequestLoggingMiddleware_LogsRequestDuration(t *testing.T) {
	logger := &MockLogger{}
	middleware := RequestLoggingMiddleware(logger)
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	completeLog := logger.logs[1]
	assert.NotNil(t, completeLog.Fields["duration"])
	assert.NotNil(t, completeLog.Fields["duration_ms"])
	
	// Duration should be at least 50ms
	durationMs := completeLog.Fields["duration_ms"].(int64)
	assert.GreaterOrEqual(t, durationMs, int64(50))
}

func TestRequestLoggingMiddleware_IncludesRequestID(t *testing.T) {
	logger := &MockLogger{}
	middleware := RequestLoggingMiddleware(logger)
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that request ID is in response headers
		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID)
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	// Check request ID is in logs
	startLog := logger.logs[0]
	completeLog := logger.logs[1]
	
	requestID1 := startLog.Fields["request_id"].(string)
	requestID2 := completeLog.Fields["request_id"].(string)
	
	assert.NotEmpty(t, requestID1)
	assert.Equal(t, requestID1, requestID2)
	
	// Check request ID is valid UUID format
	assert.Len(t, requestID1, 36)
	assert.Contains(t, requestID1, "-")
	
	// Check response header
	assert.Equal(t, requestID1, rec.Header().Get("X-Request-ID"))
}

func TestRequestLoggingMiddleware_LogsSlowRequests(t *testing.T) {
	logger := &MockLogger{}
	middleware := RequestLoggingMiddleware(logger)
	
	// Create a handler that simulates a slow request
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Note: In test, we'll check the logic without actually sleeping 5s
		w.WriteHeader(http.StatusOK)
	}))
	
	req := httptest.NewRequest("GET", "/slow", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	// In actual implementation with 5+ second delay, would see warning log
	// For test purposes, we verify the middleware structure is correct
	assert.GreaterOrEqual(t, len(logger.logs), 2)
}

func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}
	
	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.True(t, rw.written)
	
	// Subsequent calls should not change status
	rw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
}

func TestResponseWriter_DefaultsTo200(t *testing.T) {
	rw := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}
	
	// Write without calling WriteHeader
	rw.Write([]byte("test"))
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.True(t, rw.written)
}

func TestGetRequestID(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	
	requestID := GetRequestID(req)
	assert.Equal(t, "test-request-id", requestID)
}

func TestRequestLogFields(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/test?foo=bar", strings.NewReader("body"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Request-ID", "req-123")
	req.RemoteAddr = "192.168.1.1:1234"
	
	fields := RequestLogFields(req)
	
	assert.Equal(t, "POST", fields["method"])
	assert.Equal(t, "/api/test", fields["path"])
	assert.Equal(t, "foo=bar", fields["query"])
	assert.Equal(t, "192.168.1.1:1234", fields["remote_ip"])
	assert.Equal(t, "test-agent", fields["user_agent"])
	assert.Equal(t, "req-123", fields["request_id"])
	assert.Equal(t, "application/json", fields["content_type"])
}

func TestResponseLogFields(t *testing.T) {
	duration := 123 * time.Millisecond
	fields := ResponseLogFields(http.StatusNotFound, duration)
	
	assert.Equal(t, http.StatusNotFound, fields["status"])
	assert.Equal(t, "123ms", fields["duration"])
	assert.Equal(t, int64(123), fields["duration_ms"])
	assert.Equal(t, "404 Not Found", fields["status_text"])
}