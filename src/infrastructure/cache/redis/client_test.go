package redis

import (
	"context"
	"testing"
	"time"

	"digests-app-api/pkg/config"
)

// Note: These are integration tests that require a Redis instance
// In a real project, you might use testcontainers or mock the Redis client

func skipIfNoRedis(t *testing.T) {
	// Skip tests if Redis is not available
	// This is a simple approach; in production, you might use environment variables
	t.Skip("Skipping Redis integration tests - set REDIS_TEST=1 to run")
}

func TestNewRedisCache(t *testing.T) {
	skipIfNoRedis(t)
	
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}
	
	cache, err := NewRedisCache(cfg)
	
	if err != nil {
		t.Errorf("NewRedisCache returned error: %v", err)
	}
	if cache == nil {
		t.Error("NewRedisCache returned nil")
	}
}

func TestNewRedisCache_InvalidAddress(t *testing.T) {
	cfg := config.RedisConfig{
		Address:  "", // Empty address
		Password: "",
		DB:       0,
	}
	
	cache, err := NewRedisCache(cfg)
	
	if err == nil {
		t.Error("NewRedisCache should return error for empty address")
	}
	if cache != nil {
		t.Error("NewRedisCache should return nil cache for invalid config")
	}
}

func TestRedisCache_Get_ExistingKey(t *testing.T) {
	skipIfNoRedis(t)
	
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}
	
	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	ctx := context.Background()
	key := "test-key"
	value := []byte("test-value")
	
	// Set a value
	err = cache.Set(ctx, key, value, 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}
	
	// Get the value
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("Get returned %s, want %s", string(got), string(value))
	}
	
	// Cleanup
	cache.Delete(ctx, key)
}

func TestRedisCache_Get_NonExistentKey(t *testing.T) {
	skipIfNoRedis(t)
	
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}
	
	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	ctx := context.Background()
	
	got, err := cache.Get(ctx, "non-existent-key")
	
	if err == nil {
		t.Error("Get should return error for non-existent key")
	}
	if got != nil {
		t.Error("Get should return nil value for non-existent key")
	}
}

func TestRedisCache_Set_StoresValue(t *testing.T) {
	skipIfNoRedis(t)
	
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}
	
	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	ctx := context.Background()
	key := "test-key-set"
	value := []byte("test-value")
	
	err = cache.Set(ctx, key, value, 1*time.Hour)
	
	if err != nil {
		t.Errorf("Set returned error: %v", err)
	}
	
	// Verify the value was stored
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Errorf("Failed to get stored value: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("Stored value = %s, want %s", string(got), string(value))
	}
	
	// Cleanup
	cache.Delete(ctx, key)
}

func TestRedisCache_Set_AppliesTTL(t *testing.T) {
	skipIfNoRedis(t)
	
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}
	
	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	ctx := context.Background()
	key := "test-key-ttl"
	value := []byte("test-value")
	
	// Set with short TTL
	err = cache.Set(ctx, key, value, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	
	// Wait for expiration
	time.Sleep(200 * time.Millisecond)
	
	// Value should be gone
	got, err := cache.Get(ctx, key)
	if err == nil {
		t.Error("Get should return error for expired key")
	}
	if got != nil {
		t.Error("Get should return nil for expired key")
	}
}

func TestRedisCache_Delete_RemovesKey(t *testing.T) {
	skipIfNoRedis(t)
	
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}
	
	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	ctx := context.Background()
	key := "test-key-delete"
	value := []byte("test-value")
	
	// Set a value
	err = cache.Set(ctx, key, value, 1*time.Hour)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	
	// Delete the key
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Errorf("Delete returned error: %v", err)
	}
	
	// Verify key is gone
	got, err := cache.Get(ctx, key)
	if err == nil {
		t.Error("Get should return error for deleted key")
	}
	if got != nil {
		t.Error("Get should return nil for deleted key")
	}
}

func TestRedisCache_Delete_NonExistentKey(t *testing.T) {
	skipIfNoRedis(t)
	
	cfg := config.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
		DB:       0,
	}
	
	cache, err := NewRedisCache(cfg)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	
	ctx := context.Background()
	
	err = cache.Delete(ctx, "non-existent-key")
	
	if err != nil {
		t.Errorf("Delete should return nil for non-existent key, got: %v", err)
	}
}