// ABOUTME: Default implementations for library dependencies
// ABOUTME: Provides factory functions for creating default service implementations

package digests

import (
	"log"
	"net/http"
	"os"
	"time"
	
	"digests-app-api/core/interfaces"
	"digests-app-api/infrastructure/cache/memory"
	"digests-app-api/infrastructure/cache/sqlite"
	httpInfra "digests-app-api/infrastructure/http/standard"
	loggerInfra "digests-app-api/infrastructure/logger/standard"
)

// DefaultHTTPClient creates a default HTTP client with sensible timeouts
func DefaultHTTPClient() interfaces.HTTPClient {
	return httpInfra.NewStandardHTTPClient(30 * time.Second)
}

// DefaultMemoryCache creates a default in-memory cache
func DefaultMemoryCache() interfaces.Cache {
	return memory.NewMemoryCache()
}

// DefaultSQLiteCache creates a default SQLite cache with the given file path
func DefaultSQLiteCache(filePath string) (interfaces.Cache, error) {
	return sqlite.NewClient(filePath)
}

// DefaultLogger creates a default logger that writes to stdout
func DefaultLogger() interfaces.Logger {
	return loggerInfra.NewStandardLogger()
}

// DefaultLoggerWithPrefix creates a default logger with a custom prefix
func DefaultLoggerWithPrefix(prefix string) interfaces.Logger {
	// For now, return standard logger - prefix support can be added later
	return loggerInfra.NewStandardLogger()
}

// QuietLogger creates a logger that discards all output
func QuietLogger() interfaces.Logger {
	// Create a custom logger that discards output
	return &quietLogger{}
}

// quietLogger is a logger that discards all output
type quietLogger struct{}

func (q *quietLogger) Debug(msg string, fields map[string]interface{}) {}
func (q *quietLogger) Info(msg string, fields map[string]interface{})  {}
func (q *quietLogger) Warn(msg string, fields map[string]interface{})  {}
func (q *quietLogger) Error(msg string, fields map[string]interface{}) {}

// CacheOption represents cache configuration options
type CacheOption struct {
	Type     CacheType
	FilePath string // For SQLite cache
}

// CacheType represents the type of cache
type CacheType string

const (
	CacheTypeMemory CacheType = "memory"
	CacheTypeSQLite CacheType = "sqlite"
)

// WithCacheOption creates a cache based on the provided options
func WithCacheOption(opt CacheOption) Option {
	return func(c *Config) error {
		switch opt.Type {
		case CacheTypeMemory:
			c.Cache = DefaultMemoryCache()
		case CacheTypeSQLite:
			if opt.FilePath == "" {
				opt.FilePath = "digests_cache.db"
			}
			cache, err := DefaultSQLiteCache(opt.FilePath)
			if err != nil {
				return err
			}
			c.Cache = cache
		default:
			return NewError(ErrorTypeConfiguration, "invalid cache type").
				WithContext("type", string(opt.Type))
		}
		return nil
	}
}

// WithDefaultDependencies configures the client with all default dependencies
func WithDefaultDependencies() Option {
	return func(c *Config) error {
		if c.HTTPClient == nil {
			c.HTTPClient = DefaultHTTPClient()
		}
		if c.Cache == nil {
			c.Cache = DefaultMemoryCache()
		}
		if c.Logger == nil {
			c.Logger = DefaultLogger()
		}
		return nil
	}
}

// WithQuietMode configures the client to suppress all log output
func WithQuietMode() Option {
	return func(c *Config) error {
		c.Logger = QuietLogger()
		return nil
	}
}

// HTTPClientConfig holds configuration for HTTP client
type HTTPClientConfig struct {
	Timeout             time.Duration
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	UserAgent           string
}

// DefaultHTTPClientConfig returns default HTTP client configuration
func DefaultHTTPClientConfig() HTTPClientConfig {
	return HTTPClientConfig{
		Timeout:             30 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		UserAgent:           "Digests-Library/1.0",
	}
}

// WithHTTPClientConfig creates an HTTP client with custom configuration
func WithHTTPClientConfig(config HTTPClientConfig) Option {
	return func(c *Config) error {
		// For now, just use the standard client with custom timeout
		// More advanced configuration can be added later
		c.HTTPClient = httpInfra.NewStandardHTTPClient(config.Timeout)
		return nil
	}
}