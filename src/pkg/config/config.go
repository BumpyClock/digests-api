// ABOUTME: Configuration management for the application with environment variable support
// ABOUTME: Defines configuration structures for server, cache, and other settings

package config

import (
	"errors"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	// Server contains HTTP server configuration
	Server ServerConfig

	// Cache contains cache configuration
	Cache CacheConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	// Port is the HTTP server port
	Port string

	// RefreshTimer is the interval in seconds for feed refresh
	RefreshTimer int
}

// CacheConfig holds cache backend configuration
type CacheConfig struct {
	// Type specifies the cache backend (redis/memory/sqlite)
	Type string

	// Redis contains Redis-specific configuration
	Redis RedisConfig

	// Memory contains in-memory cache configuration
	Memory MemoryConfig
	
	// SQLite contains SQLite-specific configuration
	SQLite SQLiteConfig
	
	// ColorCacheDays is the number of days to cache thumbnail colors
	ColorCacheDays int
}

// SQLiteConfig holds SQLite-specific configuration
type SQLiteConfig struct {
	// FilePath is the path to the SQLite database file
	FilePath string
}

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	// Address is the Redis server address
	Address string

	// Password is the Redis authentication password
	Password string

	// DB is the Redis database number
	DB int
}

// MemoryConfig holds in-memory cache configuration
type MemoryConfig struct {
	// DefaultExpiration is the default TTL for cache entries in seconds
	DefaultExpiration int
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnvOrDefault("PORT", "8000"),
			RefreshTimer: getEnvAsIntOrDefault("REFRESH_TIMER", 60),
		},
		Cache: CacheConfig{
			Type: getEnvOrDefault("CACHE_TYPE", "memory"),
			Redis: RedisConfig{
				Address:  getEnvOrDefault("REDIS_ADDRESS", "localhost:6379"),
				Password: getEnvOrDefault("REDIS_PASSWORD", ""),
				DB:       getEnvAsIntOrDefault("REDIS_DB", 0),
			},
			Memory: MemoryConfig{
				DefaultExpiration: getEnvAsIntOrDefault("MEMORY_CACHE_EXPIRATION", 3600),
			},
			SQLite: SQLiteConfig{
				FilePath: getEnvOrDefault("SQLITE_CACHE_PATH", "cache.db"),
			},
			ColorCacheDays: getEnvAsIntOrDefault("COLOR_CACHE_DAYS", 7),
		},
	}

	return cfg, nil
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsIntOrDefault returns the environment variable as int or a default
func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return errors.New("port cannot be empty")
	}

	if c.Server.RefreshTimer < 1 {
		return errors.New("refresh timer must be at least 1 second")
	}

	if c.Cache.Type != "redis" && c.Cache.Type != "memory" && c.Cache.Type != "sqlite" {
		return errors.New("cache type must be 'redis', 'memory', or 'sqlite'")
	}

	if c.Cache.Type == "redis" && c.Cache.Redis.Address == "" {
		return errors.New("redis address cannot be empty when using redis cache")
	}

	return nil
}