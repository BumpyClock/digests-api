// ABOUTME: Standard HTTP client implementation with retry logic and timeout support
// ABOUTME: Provides HTTP functionality with exponential backoff for resilient external API calls

package standard

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"digests-app-api/core/interfaces"
)

const (
	maxRetries = 3
	userAgent  = "DigestsAPI/1.0"
)

// StandardHTTPClient implements the HTTPClient interface using standard library
type StandardHTTPClient struct {
	client *http.Client
}

// NewStandardHTTPClient creates a new HTTP client with the specified timeout
func NewStandardHTTPClient(timeout time.Duration) *StandardHTTPClient {
	return &StandardHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Get performs an HTTP GET request
func (c *StandardHTTPClient) Get(ctx context.Context, url string) (interfaces.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent
	req.Header.Set("User-Agent", userAgent)

	// Perform request with retry logic
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms
			backoff := time.Duration(100*(1<<(attempt-1))) * time.Millisecond
			select {
			case <-time.After(backoff):
				// Continue with retry
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err = c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Don't retry on success or 4xx errors
		if resp.StatusCode < 500 {
			break
		}

		// Close body for retry
		resp.Body.Close()
		lastErr = fmt.Errorf("server returned %d", resp.StatusCode)
	}

	if resp == nil {
		return nil, lastErr
	}

	return &httpResponse{
		statusCode: resp.StatusCode,
		body:       resp.Body,
		headers:    resp.Header,
	}, nil
}

// Post performs an HTTP POST request
func (c *StandardHTTPClient) Post(ctx context.Context, url string, body io.Reader) (interfaces.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}

	// Set User-Agent
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	return &httpResponse{
		statusCode: resp.StatusCode,
		body:       resp.Body,
		headers:    resp.Header,
	}, nil
}

// httpResponse implements the Response interface
type httpResponse struct {
	statusCode int
	body       io.ReadCloser
	headers    http.Header
}

// StatusCode returns the HTTP status code
func (r *httpResponse) StatusCode() int {
	return r.statusCode
}

// Body returns the response body
func (r *httpResponse) Body() io.ReadCloser {
	return r.body
}

// Header returns the value of the specified header
func (r *httpResponse) Header(key string) string {
	return r.headers.Get(key)
}