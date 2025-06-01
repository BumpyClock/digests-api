package sqlite

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestClient_WithLogger tests that the client properly logs suspicious patterns
func TestClient_WithLogger(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test_cache_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create mock logger
	logger := &MockLogger{}

	// Create client with logger
	client, err := NewSQLiteCacheWithLogger(tmpFile.Name(), logger)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test with a key containing suspicious patterns
	suspiciousKey := "user_data';DROP TABLE cache;--"
	value := []byte("test value")

	// Set should work but log a warning
	err = client.Set(ctx, suspiciousKey, value, time.Hour)
	if err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Verify warning was logged
	if len(logger.warnings) == 0 {
		t.Error("Expected warning to be logged for suspicious key")
	} else {
		// Check that multiple warnings were logged (for each suspicious pattern)
		patterns := make(map[string]bool)
		for _, w := range logger.warnings {
			if w.msg == "Suspicious pattern detected in cache key" {
				if pattern, ok := w.fields["pattern"].(string); ok {
					patterns[pattern] = true
				}
			}
		}
		
		// Should have detected at least the semicolon and single quote
		if !patterns["'"] {
			t.Error("Expected warning for single quote pattern")
		}
		if !patterns[";"] {
			t.Error("Expected warning for semicolon pattern")
		}
		if !patterns["--"] {
			t.Error("Expected warning for SQL comment pattern")
		}
	}

	// Get should also log warnings
	logger.warnings = nil // Reset warnings
	_, err = client.Get(ctx, suspiciousKey)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}

	if len(logger.warnings) == 0 {
		t.Error("Expected warning to be logged for Get operation")
	}

	// Delete should also log warnings
	logger.warnings = nil // Reset warnings
	err = client.Delete(ctx, suspiciousKey)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	if len(logger.warnings) == 0 {
		t.Error("Expected warning to be logged for Delete operation")
	}
}

// TestClient_WithoutLogger tests that the client works without a logger
func TestClient_WithoutLogger(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test_cache_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create client without logger (using the original constructor)
	client, err := NewSQLiteCache(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test with a key containing suspicious patterns - should not panic
	suspiciousKey := "user_data';DROP TABLE cache;--"
	value := []byte("test value")

	err = client.Set(ctx, suspiciousKey, value, time.Hour)
	if err != nil {
		t.Errorf("Set() error = %v", err)
	}

	retrievedValue, err := client.Get(ctx, suspiciousKey)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}

	if string(retrievedValue) != string(value) {
		t.Errorf("Retrieved value doesn't match: got %q, want %q", retrievedValue, value)
	}
}