package sqlite

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestDataIntegrity verifies that data stored in the cache is retrieved exactly as stored
func TestDataIntegrity(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test_cache_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cache, err := NewSQLiteCache(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Simple text",
			data: []byte("Hello, World!"),
		},
		{
			name: "Binary data",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		},
		{
			name: "Empty data",
			data: []byte{},
		},
		{
			name: "Large data",
			data: make([]byte, 10000), // 10KB of zeros
		},
		{
			name: "All possible bytes",
			data: func() []byte {
				data := make([]byte, 256)
				for i := 0; i < 256; i++ {
					data[i] = byte(i)
				}
				return data
			}(),
		},
		{
			name: "UTF-8 text with special characters",
			data: []byte("Hello 世界 🌍 \n\t\r"),
		},
		{
			name: "Data with null bytes",
			data: []byte("before\x00middle\x00after"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "test_key_" + tt.name

			// Skip empty data test as cache doesn't allow empty values
			if len(tt.data) == 0 {
				err := cache.Set(ctx, key, tt.data, time.Hour)
				if err == nil {
					t.Error("Expected error for empty data, but got none")
				}
				return
			}

			// Store the data
			err := cache.Set(ctx, key, tt.data, time.Hour)
			if err != nil {
				t.Fatalf("Failed to set data: %v", err)
			}

			// Retrieve the data
			retrieved, err := cache.Get(ctx, key)
			if err != nil {
				t.Fatalf("Failed to get data: %v", err)
			}

			// Verify data integrity
			if len(retrieved) != len(tt.data) {
				t.Errorf("Length mismatch: expected %d bytes, got %d bytes", len(tt.data), len(retrieved))
				return
			}

			// Byte-by-byte comparison
			for i := 0; i < len(tt.data); i++ {
				if retrieved[i] != tt.data[i] {
					t.Errorf("Byte mismatch at position %d: expected %#x, got %#x", i, tt.data[i], retrieved[i])
					// Show context around the mismatch
					start := i - 5
					if start < 0 {
						start = 0
					}
					end := i + 5
					if end > len(tt.data) {
						end = len(tt.data)
					}
					t.Errorf("Context: expected %#x, got %#x", tt.data[start:end], retrieved[start:end])
					return
				}
			}
		})
	}
}

// TestDataIntegrityStress performs a stress test with many different data patterns
func TestDataIntegrityStress(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test_cache_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cache, err := NewSQLiteCache(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test with various data sizes
	sizes := []int{1, 10, 100, 1000, 10000, 100000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			// Create data with a pattern
			data := make([]byte, size)
			for i := 0; i < size; i++ {
				// Create a pattern that will help detect corruption
				data[i] = byte((i * 7) % 256)
			}

			key := fmt.Sprintf("stress_test_%d", size)

			// Store the data
			err := cache.Set(ctx, key, data, time.Hour)
			if err != nil {
				t.Fatalf("Failed to set data of size %d: %v", size, err)
			}

			// Retrieve the data
			retrieved, err := cache.Get(ctx, key)
			if err != nil {
				t.Fatalf("Failed to get data of size %d: %v", size, err)
			}

			// Verify integrity
			if !bytes.Equal(retrieved, data) {
				// Find first mismatch
				for i := 0; i < len(data); i++ {
					if i >= len(retrieved) {
						t.Errorf("Retrieved data is shorter: %d vs %d bytes", len(retrieved), len(data))
						break
					}
					if retrieved[i] != data[i] {
						t.Errorf("First mismatch at position %d: expected %#x, got %#x", i, data[i], retrieved[i])
						break
					}
				}
			}
		})
	}
}