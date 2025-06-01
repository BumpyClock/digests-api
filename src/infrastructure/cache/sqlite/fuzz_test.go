// +build gofuzz

package sqlite

import (
	"bytes"
	"context"
	"fmt"
	"time"
)

// FuzzCacheKey tests the cache with fuzzing inputs for keys
// To run: go-fuzz-build && go-fuzz -func FuzzCacheKey
func FuzzCacheKey(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	
	// Create in-memory SQLite for fuzzing
	cache, err := NewSQLiteCache(":memory:")
	if err != nil {
		return -1
	}
	defer cache.Close()
	
	ctx := context.Background()
	key := string(data)
	value := []byte("test value")
	
	// Try to set with fuzzed key
	err = cache.Set(ctx, key, value, 1*time.Hour)
	
	// Try to get with fuzzed key
	_, _ = cache.Get(ctx, key)
	
	// Try to delete with fuzzed key
	_ = cache.Delete(ctx, key)
	
	// If we haven't panicked, the input was handled safely
	return 1
}

// FuzzCacheValue tests the cache with fuzzing inputs for values
func FuzzCacheValue(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	
	cache, err := NewSQLiteCache(":memory:")
	if err != nil {
		return -1
	}
	defer cache.Close()
	
	ctx := context.Background()
	key := "test_key"
	
	// Try to set with fuzzed value
	err = cache.Set(ctx, key, data, 1*time.Hour)
	
	if err == nil {
		// Try to get it back
		retrieved, err := cache.Get(ctx, key)
		if err == nil {
			// First do a quick comparison using bytes.Equal
			if !bytes.Equal(retrieved, data) {
				// If not equal, find the exact mismatch for debugging
				if len(retrieved) != len(data) {
					panic(fmt.Sprintf("Data corruption detected: length mismatch (expected %d bytes, got %d bytes)", len(data), len(retrieved)))
				}
				
				// Perform byte-by-byte comparison to find exact mismatch
				for i := 0; i < len(data); i++ {
					if retrieved[i] != data[i] {
						panic(fmt.Sprintf("Data corruption detected: byte mismatch at position %d (expected %#x, got %#x)", i, data[i], retrieved[i]))
					}
				}
				
				// This should never be reached if bytes.Equal returned false
				panic("Data corruption detected: bytes.Equal returned false but no mismatch found")
			}
		}
	}
	
	return 1
}

// FuzzQueryBuilder tests the query builder with fuzzing inputs
func FuzzQueryBuilder(data []byte) int {
	if len(data) < 3 {
		return -1
	}
	
	qb := NewQueryBuilder()
	
	// Split data into parts for different inputs
	part1 := string(data[:len(data)/3])
	part2 := string(data[len(data)/3 : 2*len(data)/3])
	part3 := string(data[2*len(data)/3:])
	
	// Try various operations
	qb.Select(part1, part2)
	qb.From(part1)
	qb.Where(part2, "=", part3)
	
	// Build should never panic
	query, params := qb.Build()
	_ = query
	_ = params
	
	// Validate functions should never panic
	_ = ValidateKey(part1, nil)
	_ = ValidateValue(data)
	
	return 1
}