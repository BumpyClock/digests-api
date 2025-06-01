package sqlite

import (
	"strings"
	"testing"
)

// MockLogger captures log calls for testing
type MockLogger struct {
	warnings []struct {
		msg    string
		fields map[string]interface{}
	}
}

func (ml *MockLogger) Warn(msg string, fields map[string]interface{}) {
	ml.warnings = append(ml.warnings, struct {
		msg    string
		fields map[string]interface{}
	}{msg: msg, fields: fields})
}

func TestValidateKey_LogsSuspiciousPatterns(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		expectedPattern string
		shouldLog       bool
	}{
		{
			name:            "Normal key without suspicious patterns",
			key:             "normal_cache_key_123",
			expectedPattern: "",
			shouldLog:       false,
		},
		{
			name:            "Key with SQL comment pattern",
			key:             "key--with--comments",
			expectedPattern: "--",
			shouldLog:       true,
		},
		{
			name:            "Key with semicolon",
			key:             "key;with;semicolons",
			expectedPattern: ";",
			shouldLog:       true,
		},
		{
			name:            "Key with single quote",
			key:             "key'with'quotes",
			expectedPattern: "'",
			shouldLog:       true,
		},
		{
			name:            "Key with newline",
			key:             "key\nwith\nnewlines",
			expectedPattern: "\n",
			shouldLog:       true,
		},
		{
			name:            "Key with multiple suspicious patterns",
			key:             "key';--DROP",
			expectedPattern: "'", // Should log for the first pattern found
			shouldLog:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with logger
			logger := &MockLogger{}
			err := ValidateKey(tt.key, logger)
			
			// ValidateKey should not return error for suspicious patterns
			if err != nil {
				t.Errorf("ValidateKey() unexpected error = %v", err)
			}
			
			if tt.shouldLog {
				if len(logger.warnings) == 0 {
					t.Errorf("Expected warning to be logged, but no warnings were logged")
					return
				}
				
				// Check that at least one warning contains the expected pattern
				foundExpectedPattern := false
				for _, warning := range logger.warnings {
					if warning.msg == "Suspicious pattern detected in cache key" {
						if pattern, ok := warning.fields["pattern"].(string); ok && pattern == tt.expectedPattern {
							foundExpectedPattern = true
							break
						}
					}
				}
				
				if !foundExpectedPattern {
					t.Errorf("Expected warning for pattern %q, but it was not found in logs", tt.expectedPattern)
				}
			} else {
				if len(logger.warnings) > 0 {
					t.Errorf("Expected no warnings, but got %d warnings", len(logger.warnings))
				}
			}
		})
	}
}

func TestValidateKey_WithNilLogger(t *testing.T) {
	// Should not panic when logger is nil
	suspiciousKey := "key';DROP TABLE cache;--"
	err := ValidateKey(suspiciousKey, nil)
	
	if err != nil {
		t.Errorf("ValidateKey() unexpected error = %v", err)
	}
}

func TestValidateKey_TruncatesLongKeys(t *testing.T) {
	logger := &MockLogger{}
	
	// Create a long key with suspicious pattern
	longKey := strings.Repeat("a", 100) + "';DROP TABLE cache;--"
	
	err := ValidateKey(longKey, logger)
	if err != nil {
		t.Errorf("ValidateKey() unexpected error = %v", err)
	}
	
	// Check that the key was truncated in the log
	if len(logger.warnings) == 0 {
		t.Fatal("Expected warning to be logged")
	}
	
	keyPreview, ok := logger.warnings[0].fields["key_preview"].(string)
	if !ok {
		t.Fatal("key_preview field not found in log")
	}
	
	// Should be truncated to 50 chars + "..."
	if len(keyPreview) != 53 {
		t.Errorf("Expected truncated key length of 53, got %d", len(keyPreview))
	}
	
	if !strings.HasSuffix(keyPreview, "...") {
		t.Errorf("Expected truncated key to end with '...', got %q", keyPreview)
	}
}