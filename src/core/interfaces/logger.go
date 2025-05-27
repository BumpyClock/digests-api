package interfaces

// Logger defines the interface for logging throughout the application.
// This abstraction allows for different logging implementations (logrus, zap, etc.)
// while maintaining a consistent interface.
//
// Example usage:
//
//	logger.Info("Processing feed", map[string]interface{}{
//		"url": "https://example.com/feed.xml",
//		"items": 42,
//	})
//
//	logger.Error("Failed to parse feed", map[string]interface{}{
//		"url": "https://example.com/feed.xml",
//		"error": err.Error(),
//	})
type Logger interface {
	// Debug logs a debug level message with optional structured fields.
	// Debug messages are typically used for detailed troubleshooting information.
	Debug(msg string, fields map[string]interface{})

	// Info logs an info level message with optional structured fields.
	// Info messages are used for general informational messages.
	Info(msg string, fields map[string]interface{})

	// Warn logs a warning level message with optional structured fields.
	// Warning messages indicate potential issues that don't prevent operation.
	Warn(msg string, fields map[string]interface{})

	// Error logs an error level message with optional structured fields.
	// Error messages indicate failures that need attention.
	Error(msg string, fields map[string]interface{})
}