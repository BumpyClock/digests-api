// ABOUTME: Standard logger implementation using Go's standard log package
// ABOUTME: Provides structured logging with level support

package standard

import (
	"encoding/json"
	"log"
	"os"
)

// StandardLogger implements the Logger interface using standard library
type StandardLogger struct {
	debug  *log.Logger
	info   *log.Logger
	warn   *log.Logger
	error  *log.Logger
}

// NewStandardLogger creates a new standard logger
func NewStandardLogger() *StandardLogger {
	return &StandardLogger{
		debug: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags),
		info:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		warn:  log.New(os.Stdout, "[WARN] ", log.LstdFlags),
		error: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
	}
}

// Debug logs a debug message
func (l *StandardLogger) Debug(msg string, fields map[string]interface{}) {
	l.logWithFields(l.debug, msg, fields)
}

// Info logs an info message
func (l *StandardLogger) Info(msg string, fields map[string]interface{}) {
	l.logWithFields(l.info, msg, fields)
}

// Warn logs a warning message
func (l *StandardLogger) Warn(msg string, fields map[string]interface{}) {
	l.logWithFields(l.warn, msg, fields)
}

// Error logs an error message
func (l *StandardLogger) Error(msg string, fields map[string]interface{}) {
	l.logWithFields(l.error, msg, fields)
}

// logWithFields logs a message with structured fields
func (l *StandardLogger) logWithFields(logger *log.Logger, msg string, fields map[string]interface{}) {
	if fields == nil || len(fields) == 0 {
		logger.Println(msg)
		return
	}

	// Convert fields to JSON for structured logging
	fieldsJSON, err := json.Marshal(fields)
	if err != nil {
		logger.Printf("%s (failed to marshal fields: %v)", msg, err)
		return
	}

	logger.Printf("%s %s", msg, string(fieldsJSON))
}