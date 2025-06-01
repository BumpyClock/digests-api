// +build gofuzz

package sqlite

import (
	"context"
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
		if err == nil && len(retrieved) != len(data) {
			panic("Data corruption detected")
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
	_ = ValidateKey(part1)
	_ = ValidateValue(data)
	
	return 1
}