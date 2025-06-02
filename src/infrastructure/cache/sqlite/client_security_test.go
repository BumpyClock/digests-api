package sqlite

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestSQLiteCache_SQLInjectionAttempts(t *testing.T) {
	// Create temporary database for testing
	tempFile, err := os.CreateTemp("", "test_cache_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	
	cache, err := NewSQLiteCache(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	ctx := context.Background()
	
	// Test various SQL injection attempts in cache keys
	injectionKeys := []string{
		// Basic SQL injection attempts
		"key'; DROP TABLE cache; --",
		"key' OR '1'='1",
		"key\" OR \"1\"=\"1",
		"key`; DROP TABLE cache; --",
		
		// Union-based injection
		"key' UNION SELECT null, null, null--",
		"key' UNION ALL SELECT 'a',2,3--",
		
		// Comment variations
		"key'/**/OR/**/1=1--",
		"key'#",
		"key'-- -",
		
		// Encoding attempts
		"key%27%20OR%20%271%27%3D%271",
		"key\\' OR \\'1\\'=\\'1",
		
		// Nested queries
		"key'; SELECT * FROM (SELECT * FROM cache); --",
		"key'); INSERT INTO cache VALUES ('hack', 'data', 9999999999); --",
		
		// Time-based blind SQL injection
		"key' OR SLEEP(5)--",
		"key' OR pg_sleep(5)--",
		
		// Special characters
		"key with spaces",
		"key\twith\ttabs",
		"key\nwith\nnewlines",
		"key\rwith\rcarriage\rreturns",
		"key;with;semicolons",
		"key--with--comments",
		"key/*with*/comments",
		"key#with#hashes",
		
		// Unicode and special encoding
		"key™",
		"key🔥emoji",
		"key\x00nullbyte",
		"key\\'escaped",
		
		// Very long keys that might cause buffer issues
		string(make([]byte, 1000)), // 1000 null bytes
		"key" + string(make([]byte, 1000, 1000)) + "end",
	}
	
	testValue := []byte("test value")
	
	// Test Set operations with injection attempts
	for _, key := range injectionKeys {
		t.Run("Set_"+key[:min(20, len(key))], func(t *testing.T) {
			err := cache.Set(ctx, key, testValue, 1*time.Hour)
			// We expect some keys to fail (empty, null bytes), but no SQL injection should occur
			_ = err
			
			// Verify database is still functional
			err = cache.Set(ctx, "test_after_injection", testValue, 1*time.Hour)
			if err != nil {
				t.Errorf("Cache broken after injection attempt with key %q: %v", key, err)
			}
			
			// Verify we can still read
			_, err = cache.Get(ctx, "test_after_injection")
			if err != nil {
				t.Errorf("Cache read broken after injection attempt with key %q: %v", key, err)
			}
			
			// Verify table still exists
			stats, err := cache.Stats()
			if err != nil {
				t.Errorf("Stats broken after injection attempt with key %q: %v", key, err)
			}
			if stats["total_entries"] == nil {
				t.Errorf("Table might be dropped after injection attempt with key %q", key)
			}
		})
	}
	
	// Test Get operations with injection attempts
	for _, key := range injectionKeys {
		t.Run("Get_"+key[:min(20, len(key))], func(t *testing.T) {
			_, err := cache.Get(ctx, key)
			// We expect errors for non-existent keys, but no SQL injection
			_ = err
			
			// Verify database is still functional
			err = cache.Set(ctx, "test_get_after", testValue, 1*time.Hour)
			if err != nil {
				t.Errorf("Cache broken after GET injection attempt with key %q: %v", key, err)
			}
		})
	}
	
	// Test Delete operations with injection attempts
	for _, key := range injectionKeys {
		t.Run("Delete_"+key[:min(20, len(key))], func(t *testing.T) {
			err := cache.Delete(ctx, key)
			_ = err
			
			// Verify database is still functional
			err = cache.Set(ctx, "test_delete_after", testValue, 1*time.Hour)
			if err != nil {
				t.Errorf("Cache broken after DELETE injection attempt with key %q: %v", key, err)
			}
		})
	}
	
	// Test injection attempts in values
	injectionValues := [][]byte{
		[]byte("'); DROP TABLE cache; --"),
		[]byte("' OR '1'='1"),
		[]byte("\x00\x01\x02\x03"), // Binary data
		[]byte(string(make([]byte, 10000))), // Large value
		[]byte("value with 'quotes'"),
		[]byte(`value with "double quotes"`),
		[]byte("value with `backticks`"),
	}
	
	for i, value := range injectionValues {
		t.Run("Value_Injection_"+string(rune(i)), func(t *testing.T) {
			err := cache.Set(ctx, "safe_key", value, 1*time.Hour)
			if err != nil && len(value) > 0 { // Empty values should fail
				t.Errorf("Failed to set value with potential injection: %v", err)
			}
			
			// Verify we can read it back correctly
			if err == nil {
				retrieved, err := cache.Get(ctx, "safe_key")
				if err != nil {
					t.Errorf("Failed to get value after injection attempt: %v", err)
				}
				if len(retrieved) != len(value) {
					t.Errorf("Value corrupted: expected %d bytes, got %d", len(value), len(retrieved))
				}
			}
		})
	}
}

func TestSQLiteCache_KeyValidation(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_cache_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	
	cache, err := NewSQLiteCache(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	ctx := context.Background()
	testValue := []byte("test")
	
	// Test empty key validation
	err = cache.Set(ctx, "", testValue, 1*time.Hour)
	if err == nil {
		t.Error("Expected error for empty key in Set")
	}
	
	_, err = cache.Get(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key in Get")
	}
	
	err = cache.Delete(ctx, "")
	if err == nil {
		t.Error("Expected error for empty key in Delete")
	}
}

func TestSQLiteCache_ValueValidation(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_cache_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	
	cache, err := NewSQLiteCache(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	defer cache.Close()
	
	ctx := context.Background()
	
	// Test empty value validation
	err = cache.Set(ctx, "key", []byte{}, 1*time.Hour)
	if err == nil {
		t.Error("Expected error for empty value")
	}
	
	err = cache.Set(ctx, "key", nil, 1*time.Hour)
	if err == nil {
		t.Error("Expected error for nil value")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}