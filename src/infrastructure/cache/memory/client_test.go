package memory

import (
	"context"
	"testing"
	"time"
)

func TestNewMemoryCache(t *testing.T) {
	cache := NewMemoryCache()
	
	if cache == nil {
		t.Error("NewMemoryCache returned nil")
	}
}

func TestMemoryCache_Get_ExistingKey(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	// Set a value
	key := "test-key"
	value := []byte("test-value")
	err := cache.Set(ctx, key, value, 1*time.Hour)
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
}

func TestMemoryCache_Get_NonExistentKey(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	got, err := cache.Get(ctx, "non-existent")
	
	if err == nil {
		t.Error("Get should return error for non-existent key")
	}
	if got != nil {
		t.Error("Get should return nil value for non-existent key")
	}
}

func TestMemoryCache_Get_ExpiredKey(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	// Set a value with short TTL
	key := "test-key"
	value := []byte("test-value")
	err := cache.Set(ctx, key, value, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to set value: %v", err)
	}
	
	// Wait for expiration
	time.Sleep(20 * time.Millisecond)
	
	// Try to get the expired value
	got, err := cache.Get(ctx, key)
	
	if err == nil {
		t.Error("Get should return error for expired key")
	}
	if got != nil {
		t.Error("Get should return nil value for expired key")
	}
}

func TestMemoryCache_Set_StoresValue(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	key := "test-key"
	value := []byte("test-value")
	
	err := cache.Set(ctx, key, value, 1*time.Hour)
	
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
}

func TestMemoryCache_Set_WithZeroTTL(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	key := "test-key"
	value := []byte("test-value")
	
	// Set with zero TTL (should not expire)
	err := cache.Set(ctx, key, value, 0)
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	
	// Wait a bit
	time.Sleep(50 * time.Millisecond)
	
	// Value should still be there
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if string(got) != string(value) {
		t.Errorf("Get returned %s, want %s", string(got), string(value))
	}
}

func TestMemoryCache_Set_UpdatesExisting(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	key := "test-key"
	value1 := []byte("value1")
	value2 := []byte("value2")
	
	// Set initial value
	err := cache.Set(ctx, key, value1, 1*time.Hour)
	if err != nil {
		t.Fatalf("First set failed: %v", err)
	}
	
	// Update with new value
	err = cache.Set(ctx, key, value2, 1*time.Hour)
	if err != nil {
		t.Fatalf("Second set failed: %v", err)
	}
	
	// Verify updated value
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Errorf("Get returned error: %v", err)
	}
	if string(got) != string(value2) {
		t.Errorf("Get returned %s, want %s", string(got), string(value2))
	}
}

func TestMemoryCache_Delete_RemovesKey(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	key := "test-key"
	value := []byte("test-value")
	
	// Set a value
	err := cache.Set(ctx, key, value, 1*time.Hour)
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

func TestMemoryCache_Delete_NonExistentKey(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	err := cache.Delete(ctx, "non-existent")
	
	if err != nil {
		t.Errorf("Delete should return nil for non-existent key, got: %v", err)
	}
}

func TestMemoryCache_Cleanup(t *testing.T) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	// Set multiple values with different expiration times
	cache.Set(ctx, "key1", []byte("value1"), 10*time.Millisecond)
	cache.Set(ctx, "key2", []byte("value2"), 100*time.Millisecond)
	cache.Set(ctx, "key3", []byte("value3"), 1*time.Hour)
	
	// Wait for first key to expire
	time.Sleep(20 * time.Millisecond)
	
	// Access key2 to trigger cleanup
	cache.Get(ctx, "key2")
	
	// key1 should be gone
	_, err := cache.Get(ctx, "key1")
	if err == nil {
		t.Error("key1 should have been cleaned up")
	}
	
	// key2 and key3 should still exist
	_, err = cache.Get(ctx, "key2")
	if err != nil {
		t.Error("key2 should still exist")
	}
	
	_, err = cache.Get(ctx, "key3")
	if err != nil {
		t.Error("key3 should still exist")
	}
}