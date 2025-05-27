package interfaces

import (
	"context"
	"io"
)

// HTTPClient defines the interface for making HTTP requests.
// This abstraction allows for easy mocking in tests and switching between
// different HTTP client implementations (standard library, retryable client, etc.)
type HTTPClient interface {
	// Get performs an HTTP GET request to the specified URL.
	// Returns a Response interface or an error if the request fails.
	Get(ctx context.Context, url string) (Response, error)

	// Post performs an HTTP POST request to the specified URL with the given body.
	// The body should be closed by the caller after use.
	Post(ctx context.Context, url string, body io.Reader) (Response, error)
}

// Response defines the interface for HTTP responses.
// This abstraction allows different HTTP client implementations to provide
// their own response types while maintaining a consistent interface.
type Response interface {
	// StatusCode returns the HTTP status code of the response.
	StatusCode() int

	// Body returns the response body as an io.ReadCloser.
	// The caller is responsible for closing the body when done.
	Body() io.ReadCloser

	// Header returns the value of the specified header.
	// Returns an empty string if the header is not present.
	// Header names are case-insensitive.
	Header(key string) string
}