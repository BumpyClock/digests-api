// ABOUTME: Redis cache implementation using go-redis client
// ABOUTME: Provides distributed caching with TTL support and connection pooling

package redis

import (
	"context"
	"errors"
	"time"

	"digests-app-api/pkg/config"
	"github.com/redis/go-redis/v9"
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg config.RedisConfig) (*RedisCache, error) {
	if cfg.Address == "" {
		return nil, errors.New("redis address cannot be empty")
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisCache{
		client: client,
	}, nil
}

// Get retrieves a value from Redis
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("key not found")
		}
		return nil, err
	}

	return val, nil
}

// Set stores a value in Redis with the given TTL
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Redis SET with 0 TTL means no expiration
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a key from Redis
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	// Redis DEL returns number of keys deleted, but we ignore it
	// as deleting non-existent key is not an error for our use case
	c.client.Del(ctx, key)
	return nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}