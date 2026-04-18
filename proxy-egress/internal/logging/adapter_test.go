package logging

import (
	"testing"
)

func TestNewLogrusAdapter(t *testing.T) {
	adapter, err := NewLogrusAdapter("test-logger")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	if adapter == nil {
		t.Fatal("expected adapter to be created, got nil")
	}

	if adapter.logger == nil {
		t.Fatal("expected logger to be initialized")
	}
}

func TestLogrusAdapterWithFields(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	tests := []struct {
		name   string
		fields map[string]interface{}
	}{
		{
			name: "string field",
			fields: map[string]interface{}{
				"user_id": "12345",
			},
		},
		{
			name: "int field",
			fields: map[string]interface{}{
				"count": 42,
			},
		},
		{
			name: "int64 field",
			fields: map[string]interface{}{
				"duration": int64(1000),
			},
		},
		{
			name: "float64 field",
			fields: map[string]interface{}{
				"latency": 1.234,
			},
		},
		{
			name: "bool field",
			fields: map[string]interface{}{
				"success": true,
			},
		},
		{
			name: "mixed fields",
			fields: map[string]interface{}{
				"user_id":  "user123",
				"count":    42,
				"latency":  1.234,
				"success":  true,
				"duration": int64(5000),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.WithFields(tt.fields)
			if result == nil {
				t.Fatal("expected WithFields to return non-nil adapter")
			}
			if len(result.fields) == 0 {
				t.Errorf("expected fields to be added, got empty")
			}
		})
	}
}

func TestLogrusAdapterWithField(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	result := adapter.WithField("key", "value")
	if result == nil {
		t.Fatal("expected WithField to return non-nil adapter")
	}
	if len(result.fields) == 0 {
		t.Error("expected field to be added")
	}
}

func TestLogrusAdapterWithError(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	testErr := "test error message"
	result := adapter.WithError(Err{msg: testErr})
	if result == nil {
		t.Fatal("expected WithError to return non-nil adapter")
	}
	if len(result.fields) == 0 {
		t.Error("expected error field to be added")
	}
}

// Err is a simple test error type
type Err struct {
	msg string
}

func (e Err) Error() string {
	return e.msg
}

func TestLogrusAdapterInfo(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic
	adapter.Info("test message")
	adapter.Info("test", "multiple", "args")
}

func TestLogrusAdapterInfof(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic
	adapter.Infof("formatted %s message", "test")
}

func TestLogrusAdapterWarn(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic
	adapter.Warn("warning message")
	adapter.Warn("multiple", "warning", "args")
}

func TestLogrusAdapterWarnf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic
	adapter.Warnf("formatted %s warning", "test")
}

func TestLogrusAdapterError(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic
	adapter.Error("error message")
	adapter.Error("multiple", "error", "args")
}

func TestLogrusAdapterErrorf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic
	adapter.Errorf("formatted %s error", "test")
}

func TestLogrusAdapterDebug(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic
	adapter.Debug("debug message")
	adapter.Debug("multiple", "debug", "args")
}

func TestLogrusAdapterDebugf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic
	adapter.Debugf("formatted %s debug", "test")
}

func TestLogrusAdapterFatal(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic during call (logging only, no exit in test)
	adapter.Fatal("fatal message")
}

func TestLogrusAdapterFatalf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should not panic during call (logging only, no exit in test)
	adapter.Fatalf("formatted %s fatal", "test")
}

func TestLogrusAdapterChaining(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Test method chaining
	result := adapter.
		WithField("user_id", "123").
		WithFields(map[string]interface{}{"action": "login"}).
		WithError(Err{msg: "test error"})

	if result == nil {
		t.Fatal("expected chained result to be non-nil")
	}

	// All methods should work on result
	result.Info("chained message")
}

func TestLogrusAdapterEmptyArgs(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter() failed: %v", err)
	}

	// Should handle empty args gracefully
	adapter.Info()
	adapter.Warn()
	adapter.Error()
	adapter.Debug()
}
