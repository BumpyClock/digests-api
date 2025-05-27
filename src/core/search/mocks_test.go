package search

import (
	"context"
	"io"
	"strings"
	"time"

	"digests-app-api/core/interfaces"
)

// mockHTTPClient is a mock implementation of the HTTPClient interface
type mockHTTPClient struct {
	getFunc  func(ctx context.Context, url string) (interfaces.Response, error)
	postFunc func(ctx context.Context, url string, body io.Reader) (interfaces.Response, error)
}

func (m *mockHTTPClient) Get(ctx context.Context, url string) (interfaces.Response, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, url)
	}
	return nil, nil
}

func (m *mockHTTPClient) Post(ctx context.Context, url string, body io.Reader) (interfaces.Response, error) {
	if m.postFunc != nil {
		return m.postFunc(ctx, url, body)
	}
	return nil, nil
}

// mockResponse is a mock implementation of the Response interface
type mockResponse struct {
	statusCode int
	body       string
	headers    map[string]string
}

func (m *mockResponse) StatusCode() int {
	return m.statusCode
}

func (m *mockResponse) Body() io.ReadCloser {
	return io.NopCloser(strings.NewReader(m.body))
}

func (m *mockResponse) Header(key string) string {
	if m.headers != nil {
		return m.headers[key]
	}
	return ""
}

// mockCache is a mock implementation of the Cache interface
type mockCache struct {
	getFunc    func(ctx context.Context, key string) ([]byte, error)
	setFunc    func(ctx context.Context, key string, value []byte, ttl time.Duration) error
	deleteFunc func(ctx context.Context, key string) error
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return nil, nil
}

func (m *mockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, ttl)
	}
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
	return nil
}