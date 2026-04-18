package logging_test

import (
	"bytes"
	"errors"
	"testing"

	"marchproxy-dblb/internal/logging"
)

func TestNewLogrusAdapter(t *testing.T) {
	adapter, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter == nil {
		t.Fatal("adapter should not be nil")
	}
}

func TestLogrusAdapterDebug(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Debug("test debug message")
	adapter.Debug("test", "debug", "args")
}

func TestLogrusAdapterDebugf(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Debugf("debug %s", "message")
	adapter.Debugf("debug %d", 42)
}

func TestLogrusAdapterInfo(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Info("test info message")
	adapter.Info("test", "info", "args")
}

func TestLogrusAdapterInfof(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Infof("info %s", "message")
	adapter.Infof("info %d", 42)
}

func TestLogrusAdapterWarn(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Warn("test warn message")
	adapter.Warn("test", "warn", "args")
}

func TestLogrusAdapterWarnf(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Warnf("warn %s", "message")
	adapter.Warnf("warn %d", 42)
}

func TestLogrusAdapterError(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Error("test error message")
	adapter.Error("test", "error", "args")
}

func TestLogrusAdapterErrorf(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Errorf("error %s", "message")
	adapter.Errorf("error %d", 42)
}

func TestLogrusAdapterFatal(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Fatal("test fatal message")
	adapter.Fatal("test", "fatal", "args")
}

func TestLogrusAdapterFatalf(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Fatalf("fatal %s", "message")
	adapter.Fatalf("fatal %d", 42)
}

func TestLogrusAdapterWithField(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	newAdapter := adapter.WithField("key", "value")
	if newAdapter == nil {
		t.Fatal("WithField should return non-nil adapter")
	}
	newAdapter.Info("test with field")
}

func TestLogrusAdapterWithFieldMultiple(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	a1 := adapter.WithField("key1", "value1")
	a2 := a1.WithField("key2", "value2")
	if a2 == nil {
		t.Fatal("chained WithField should return non-nil adapter")
	}
	a2.Info("test chained fields")
}

func TestLogrusAdapterWithFields(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	fields := map[string]interface{}{
		"field1": "string_value",
		"field2": 42,
		"field3": int64(100),
		"field4": 3.14,
		"field5": true,
		"field6": bytes.Buffer{},
	}
	newAdapter := adapter.WithFields(fields)
	if newAdapter == nil {
		t.Fatal("WithFields should return non-nil adapter")
	}
	newAdapter.Info("test with fields")
}

func TestLogrusAdapterWithFieldsEmpty(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	newAdapter := adapter.WithFields(map[string]interface{}{})
	if newAdapter == nil {
		t.Fatal("WithFields with empty map should return non-nil adapter")
	}
	newAdapter.Info("test with empty fields")
}

func TestLogrusAdapterWithError(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	testErr := errors.New("test error")
	newAdapter := adapter.WithError(testErr)
	if newAdapter == nil {
		t.Fatal("WithError should return non-nil adapter")
	}
	newAdapter.Info("test with error")
}

func TestLogrusAdapterChainedOperations(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.
		WithField("user", "alice").
		WithField("action", "login").
		Info("user action")
}

func TestLogrusAdapterFieldTypes(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	fields := map[string]interface{}{
		"string":  "hello",
		"int":     123,
		"int64":   int64(456),
		"float64": 2.71828,
		"bool":    true,
	}
	adapter.WithFields(fields).Info("test all types")
}

func TestLogrusAdapterNoArgs(t *testing.T) {
	adapter, _ := logging.NewLogrusAdapter("test")
	adapter.Debug()
	adapter.Info()
	adapter.Warn()
	adapter.Error()
	adapter.Fatal()
}
