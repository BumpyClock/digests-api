package standard

import (
	"testing"
)

func TestNewStandardLogger(t *testing.T) {
	logger := NewStandardLogger()
	
	if logger == nil {
		t.Error("NewStandardLogger returned nil")
	}
	
	if logger.debug == nil {
		t.Error("Debug logger not initialized")
	}
	
	if logger.info == nil {
		t.Error("Info logger not initialized")
	}
	
	if logger.warn == nil {
		t.Error("Warn logger not initialized")
	}
	
	if logger.error == nil {
		t.Error("Error logger not initialized")
	}
}

func TestStandardLogger_LogMethods(t *testing.T) {
	logger := NewStandardLogger()
	
	// Test that methods don't panic
	t.Run("Debug", func(t *testing.T) {
		logger.Debug("test debug", nil)
		logger.Debug("test debug with fields", map[string]interface{}{
			"key": "value",
			"num": 42,
		})
	})
	
	t.Run("Info", func(t *testing.T) {
		logger.Info("test info", nil)
		logger.Info("test info with fields", map[string]interface{}{
			"user": "john",
		})
	})
	
	t.Run("Warn", func(t *testing.T) {
		logger.Warn("test warn", nil)
		logger.Warn("test warn with fields", map[string]interface{}{
			"error": "something wrong",
		})
	})
	
	t.Run("Error", func(t *testing.T) {
		logger.Error("test error", nil)
		logger.Error("test error with fields", map[string]interface{}{
			"code": 500,
		})
	})
}