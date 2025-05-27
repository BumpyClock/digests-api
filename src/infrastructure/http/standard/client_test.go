package standard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewStandardHTTPClient(t *testing.T) {
	timeout := 10 * time.Second
	client := NewStandardHTTPClient(timeout)
	
	if client == nil {
		t.Error("NewStandardHTTPClient returned nil")
	}
	
	if client.client.Timeout != timeout {
		t.Errorf("Client timeout = %v, want %v", client.client.Timeout, timeout)
	}
}

func TestStandardHTTPClient_Get_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()
	
	client := NewStandardHTTPClient(10 * time.Second)
	ctx := context.Background()
	
	resp, err := client.Get(ctx, server.URL)
	
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Get returned nil response")
	}
	if resp.StatusCode() != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode(), http.StatusOK)
	}
	
	// Read body
	body, err := io.ReadAll(resp.Body())
	resp.Body().Close()
	if err != nil {
		t.Errorf("Failed to read body: %v", err)
	}
	if string(body) != "test response" {
		t.Errorf("Body = %s, want 'test response'", string(body))
	}
}

func TestStandardHTTPClient_Get_UserAgent(t *testing.T) {
	var capturedUserAgent string
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserAgent = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewStandardHTTPClient(10 * time.Second)
	ctx := context.Background()
	
	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	resp.Body().Close()
	
	if capturedUserAgent == "" {
		t.Error("User-Agent header not set")
	}
	if !strings.Contains(capturedUserAgent, "DigestsAPI") {
		t.Errorf("User-Agent = %s, should contain 'DigestsAPI'", capturedUserAgent)
	}
}

func TestStandardHTTPClient_Get_ContextTimeout(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewStandardHTTPClient(10 * time.Second)
	
	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	
	resp, err := client.Get(ctx, server.URL)
	
	if err == nil {
		resp.Body().Close()
		t.Error("Get should return error for context timeout")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Error should mention context deadline, got: %v", err)
	}
}

func TestStandardHTTPClient_Get_InvalidURL(t *testing.T) {
	client := NewStandardHTTPClient(10 * time.Second)
	ctx := context.Background()
	
	resp, err := client.Get(ctx, "not a valid url")
	
	if err == nil {
		resp.Body().Close()
		t.Error("Get should return error for invalid URL")
	}
}

func TestHTTPResponse_StatusCode(t *testing.T) {
	resp := &httpResponse{
		statusCode: 201,
	}
	
	if resp.StatusCode() != 201 {
		t.Errorf("StatusCode() = %d, want 201", resp.StatusCode())
	}
}

func TestHTTPResponse_Body(t *testing.T) {
	bodyContent := "test body content"
	resp := &httpResponse{
		body: io.NopCloser(strings.NewReader(bodyContent)),
	}
	
	body := resp.Body()
	content, err := io.ReadAll(body)
	body.Close()
	
	if err != nil {
		t.Errorf("Failed to read body: %v", err)
	}
	if string(content) != bodyContent {
		t.Errorf("Body content = %s, want %s", string(content), bodyContent)
	}
}

func TestHTTPResponse_Header(t *testing.T) {
	resp := &httpResponse{
		headers: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Custom":     []string{"value1", "value2"},
		},
	}
	
	// Test existing header
	if resp.Header("Content-Type") != "application/json" {
		t.Errorf("Header(Content-Type) = %s, want application/json", resp.Header("Content-Type"))
	}
	
	// Test case-insensitive
	if resp.Header("content-type") != "application/json" {
		t.Errorf("Header(content-type) = %s, want application/json", resp.Header("content-type"))
	}
	
	// Test non-existent header
	if resp.Header("Non-Existent") != "" {
		t.Errorf("Header(Non-Existent) = %s, want empty string", resp.Header("Non-Existent"))
	}
}

func TestStandardHTTPClient_Get_Retry503(t *testing.T) {
	attempts := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewStandardHTTPClient(10 * time.Second)
	ctx := context.Background()
	
	resp, err := client.Get(ctx, server.URL)
	
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Get returned nil response")
	}
	resp.Body().Close()
	
	if attempts != 3 {
		t.Errorf("Attempts = %d, want 3", attempts)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode(), http.StatusOK)
	}
}

func TestStandardHTTPClient_Get_MaxRetries(t *testing.T) {
	attempts := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	
	client := NewStandardHTTPClient(10 * time.Second)
	ctx := context.Background()
	
	resp, err := client.Get(ctx, server.URL)
	
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Get returned nil response")
	}
	resp.Body().Close()
	
	if attempts != 3 {
		t.Errorf("Attempts = %d, want 3 (max retries)", attempts)
	}
	if resp.StatusCode() != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode(), http.StatusServiceUnavailable)
	}
}

func TestStandardHTTPClient_Get_NoRetryOn4xx(t *testing.T) {
	attempts := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	
	client := NewStandardHTTPClient(10 * time.Second)
	ctx := context.Background()
	
	resp, err := client.Get(ctx, server.URL)
	
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Get returned nil response")
	}
	resp.Body().Close()
	
	if attempts != 1 {
		t.Errorf("Attempts = %d, want 1 (no retry on 4xx)", attempts)
	}
}

func TestStandardHTTPClient_Post_Success(t *testing.T) {
	var capturedBody string
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()
	
	client := NewStandardHTTPClient(10 * time.Second)
	ctx := context.Background()
	
	postBody := strings.NewReader("test post data")
	resp, err := client.Post(ctx, server.URL, postBody)
	
	if err != nil {
		t.Errorf("Post returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("Post returned nil response")
	}
	defer resp.Body().Close()
	
	if resp.StatusCode() != http.StatusCreated {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode(), http.StatusCreated)
	}
	
	if capturedBody != "test post data" {
		t.Errorf("Captured body = %s, want 'test post data'", capturedBody)
	}
}