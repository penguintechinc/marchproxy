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
		t.Fatal("adapter should not be nil")
	}
	if adapter.logger == nil {
		t.Fatal("logger should not be nil")
	}
}

func TestWithFields(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	fields := map[string]interface{}{
		"string": "value",
		"int":    42,
		"int64":  int64(999),
		"float":  3.14,
		"bool":   true,
		"custom": struct{}{},
	}

	newAdapter := adapter.WithFields(fields)
	if newAdapter == nil {
		t.Fatal("WithFields should return adapter")
	}
	if len(newAdapter.fields) != len(fields) {
		t.Errorf("expected %d fields, got %d", len(fields), len(newAdapter.fields))
	}
	// Original adapter should not be modified
	if len(adapter.fields) != 0 {
		t.Fatal("original adapter should not be modified")
	}
}

func TestWithError(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	testErr := &errorType{"test error"}
	newAdapter := adapter.WithError(testErr)
	if newAdapter == nil {
		t.Fatal("WithError should return adapter")
	}
	if len(newAdapter.fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(newAdapter.fields))
	}
}

type errorType struct {
	msg string
}

func (e *errorType) Error() string {
	return e.msg
}

func TestWithField(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	newAdapter := adapter.WithField("key", "value")
	if newAdapter == nil {
		t.Fatal("WithField should return adapter")
	}
	if len(newAdapter.fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(newAdapter.fields))
	}
}

func TestDebug(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	// Should not panic
	adapter.Debug("test message")
	adapter.Debug("test", "multiple", "args")
	adapter.Debug() // empty args
}

func TestDebugf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Debugf("test %s", "message")
}

func TestInfo(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Info("test message")
	adapter.Info("test", "multiple", "args")
	adapter.Info()
}

func TestInfof(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Infof("test %s", "message")
}

func TestWarn(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Warn("test message")
	adapter.Warn("test", "multiple", "args")
	adapter.Warn()
}

func TestWarnf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Warnf("test %s", "message")
}

func TestError(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Error("test message")
	adapter.Error("test", "multiple", "args")
	adapter.Error()
}

func TestErrorf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Errorf("test %s", "message")
}

func TestFatal(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Fatal("test message")
	adapter.Fatal("test", "multiple", "args")
	adapter.Fatal()
}

func TestFatalf(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	adapter.Fatalf("test %s", "message")
}

func TestChainedWithFields(t *testing.T) {
	adapter, err := NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("NewLogrusAdapter failed: %v", err)
	}

	chained := adapter.WithField("field1", "value1").
		WithField("field2", "value2").
		WithField("field3", "value3")

	if chained == nil {
		t.Fatal("chained adapter should not be nil")
	}
	if len(chained.fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(chained.fields))
	}
}
