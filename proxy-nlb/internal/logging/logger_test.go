//go:build ci

package logging

import (
	"testing"
)

func TestNewLogrusAdapter(t *testing.T) {
	logger, err := NewLogrusAdapter("test")
	if err != nil {
		t.Errorf("NewLogrusAdapter() returned error: %v", err)
	}
	if logger == nil {
		t.Errorf("NewLogrusAdapter() returned nil")
	}
}

func TestLogrusAdapter_Info(t *testing.T) {
	logger, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() error: %v", err)
	}

	// Should not panic
	logger.Info("test message")
}

func TestLogrusAdapter_WithField(t *testing.T) {
	logger, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() error: %v", err)
	}

	// Should not panic
	entry := logger.WithField("key", "value")
	if entry == nil {
		t.Errorf("WithField() returned nil")
	}

	// Chain additional fields
	entry2 := entry.WithField("key2", "value2")
	if entry2 == nil {
		t.Errorf("Chained WithField() returned nil")
	}
}

func TestLogrusAdapter_WithFields(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	fields := Fields{
		"key1": "value1",
		"key2": "value2",
		"key3": 123,
	}

	entry := logger.WithFields(fields)
	if entry == nil {
		t.Errorf("WithFields() returned nil")
	}
}

func TestLogrusAdapter_Warn(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	// Should not panic
	logger.Warn("warning message")

	entry := logger.WithField("module", "test")
	entry.Warn("warning with context")
}

func TestLogrusAdapter_Error(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	// Should not panic
	logger.Error("error message")

	entry := logger.WithField("error_code", 500)
	entry.Error("error with context")
}

func TestLogrusAdapter_Debug(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	// Should not panic
	logger.Debug("debug message")

	entry := logger.WithField("trace_id", "abc123")
	entry.Debug("debug with context")
}

func TestLogrusAdapter_Chaining(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	// Test method chaining
	logger.
		WithField("request_id", "123").
		WithField("user", "testuser").
		Info("User action completed")

	// Verify it doesn't panic and continues to work
	logger.WithFields(Fields{
		"status": "ok",
		"count":  5,
	}).Info("Operation successful")
}

func TestLogrusAdapter_VariousDataTypes(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"String value", "msg", "hello"},
		{"Integer value", "count", 42},
		{"Float value", "rate", 3.14},
		{"Boolean value", "active", true},
		{"Nil value", "null", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic with various types
			entry := logger.WithField(tt.key, tt.value)
			if entry == nil {
				t.Errorf("WithField() returned nil for value type %T", tt.value)
			}
		})
	}
}

func TestLogrusAdapter_FieldsMap(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	fields := Fields{
		"app":     "proxy-nlb",
		"version": "1.0.0",
		"pod":     "nlb-pod-123",
		"port":    50051,
	}

	entry := logger.WithFields(fields)
	if entry == nil {
		t.Errorf("WithFields() returned nil")
	}

	// Chain additional operation
	entry.Info("Service started")
}

func TestLogrusAdapter_EmptyFields(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	// Empty fields should not panic
	empty := Fields{}
	entry := logger.WithFields(empty)
	if entry == nil {
		t.Errorf("WithFields(empty) returned nil")
	}

	entry.Info("Empty fields test")
}

func TestLogrusAdapter_LargeFieldSets(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	// Create a large field set
	fields := Fields{}
	for i := 0; i < 50; i++ {
		fields["field"+string(rune(i))] = i
	}

	entry := logger.WithFields(fields)
	if entry == nil {
		t.Errorf("WithFields(large) returned nil")
	}

	entry.Info("Large field set")
}

func TestLogrusAdapter_LogLevels(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	// Test all log levels
	logger.Debug("This is a debug message")
	logger.Info("This is an info message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")

	// With fields
	withFields := logger.WithField("component", "router")
	withFields.Debug("Debug with field")
	withFields.Info("Info with field")
	withFields.Warn("Warn with field")
	withFields.Error("Error with field")
}

func TestLogrusAdapter_ContextPreservation(t *testing.T) {
	logger, _ := NewLogrusAdapter("test")

	// Create entry with context
	ctx := logger.
		WithField("request_id", "req-123").
		WithField("user_id", "user-456").
		WithField("action", "deploy")

	// Use in different scenarios
	ctx.Info("Started deployment")
	ctx.Debug("Deployment in progress")
	ctx.Warn("High resource usage detected")

	// Verify it still works after multiple calls
	ctx.Info("Deployment completed")
}
