package logging

import (
	"testing"
)

func TestNewLogrusAdapter(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if adapter.logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestWithFields(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	tests := []struct {
		name   string
		fields map[string]interface{}
	}{
		{
			name: "string field",
			fields: map[string]interface{}{
				"key": "value",
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
				"large": int64(9999999),
			},
		},
		{
			name: "float64 field",
			fields: map[string]interface{}{
				"ratio": 3.14,
			},
		},
		{
			name: "bool field",
			fields: map[string]interface{}{
				"enabled": true,
			},
		},
		{
			name: "mixed fields",
			fields: map[string]interface{}{
				"str":    "text",
				"num":    100,
				"flag":   false,
				"ratio":  2.71,
				"big":    int64(123456),
			},
		},
		{
			name:   "empty fields",
			fields: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newAdapter := adapter.WithFields(tt.fields)
			if newAdapter == nil {
				t.Fatal("expected non-nil adapter")
			}
			if newAdapter.logger == nil {
				t.Fatal("expected non-nil logger")
			}
			// Original adapter should be unchanged
			if len(adapter.fields) != 0 {
				t.Error("original adapter fields should be empty")
			}
			// New adapter should have fields
			if len(newAdapter.fields) != len(tt.fields) {
				t.Errorf("expected %d fields, got %d", len(tt.fields), len(newAdapter.fields))
			}
		})
	}
}

func TestWithError(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	testErr := &testError{msg: "test error"}
	newAdapter := adapter.WithError(testErr)
	if newAdapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if len(newAdapter.fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(newAdapter.fields))
	}
}

func TestWithField(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	newAdapter := adapter.WithField("key", "value")
	if newAdapter == nil {
		t.Fatal("expected non-nil adapter")
	}
	if len(newAdapter.fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(newAdapter.fields))
	}
}

func TestInfo(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	// Should not panic
	adapter.Info("test message")
	adapter.Info("test", "message")
	adapter.Info()
}

func TestInfof(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Infof("test message: %s", "value")
	adapter.Infof("multiple: %s %d", "string", 42)
}

func TestWarn(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Warn("warning message")
	adapter.Warn("warning", "message")
	adapter.Warn()
}

func TestWarnf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Warnf("warning: %s", "value")
	adapter.Warnf("multiple: %s %d", "string", 42)
}

func TestError(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Error("error message")
	adapter.Error("error", "message")
	adapter.Error()
}

func TestErrorf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Errorf("error: %s", "value")
	adapter.Errorf("multiple: %s %d", "string", 42)
}

func TestFatal(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	// Fatal doesn't exit in tests, just logs
	adapter.Fatal("fatal message")
	adapter.Fatal("fatal", "message")
	adapter.Fatal()
}

func TestFatalf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Fatalf("fatal: %s", "value")
	adapter.Fatalf("multiple: %s %d", "string", 42)
}

func TestDebug(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Debug("debug message")
	adapter.Debug("debug", "message")
	adapter.Debug()
}

func TestDebugf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Debugf("debug: %s", "value")
	adapter.Debugf("multiple: %s %d", "string", 42)
}

func TestChainedWithFields(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	chained := adapter.
		WithField("field1", "value1").
		WithField("field2", 42).
		WithError(&testError{msg: "test"})

	if chained == nil {
		t.Fatal("expected non-nil adapter")
	}
	if len(chained.fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(chained.fields))
	}

	// Original should still be unchanged
	if len(adapter.fields) != 0 {
		t.Error("original adapter should not be modified")
	}
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
