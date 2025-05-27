package memory

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkMemoryCache_Get(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	// Pre-populate cache
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		cache.Set(ctx, key, value, 1*time.Hour)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		_, _ = cache.Get(ctx, key)
	}
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		_ = cache.Set(ctx, key, value, 1*time.Hour)
	}
}

func BenchmarkMemoryCache_Delete(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	// Pre-populate cache
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		cache.Set(ctx, key, value, 1*time.Hour)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		_ = cache.Delete(ctx, key)
	}
}

func BenchmarkMemoryCache_ConcurrentGet(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	// Pre-populate cache
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		cache.Set(ctx, key, value, 1*time.Hour)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%100)
			_, _ = cache.Get(ctx, key)
			i++
		}
	})
}

func BenchmarkMemoryCache_ConcurrentSet(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			value := []byte(fmt.Sprintf("value-%d", i))
			_ = cache.Set(ctx, key, value, 1*time.Hour)
			i++
		}
	})
}

func BenchmarkMemoryCache_ExpiredItemCleanup(b *testing.B) {
	cache := NewMemoryCache()
	ctx := context.Background()
	
	// Add items with very short TTL
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		cache.Set(ctx, key, value, 1*time.Nanosecond) // Expire immediately
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		_, _ = cache.Get(ctx, key) // This triggers cleanup of expired items
	}
}