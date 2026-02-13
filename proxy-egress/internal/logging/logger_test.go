package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewLogger(t *testing.T) {
	// Test creating logger with default level
	logger, err := NewLogger("info", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created, got nil")
	}

	if logger.Logger.Level != logrus.InfoLevel {
		t.Errorf("Expected log level to be Info, got %v", logger.Logger.Level)
	}
}

func TestNewLoggerWithLevels(t *testing.T) {
	testCases := []struct {
		level    string
		expected logrus.Level
	}{
		{"debug", logrus.DebugLevel},
		{"info", logrus.InfoLevel},
		{"warn", logrus.WarnLevel},
		{"error", logrus.ErrorLevel},
		{"fatal", logrus.FatalLevel},
		{"panic", logrus.PanicLevel},
		{"DEBUG", logrus.DebugLevel}, // Test case insensitive
		{"INFO", logrus.InfoLevel},
		{"invalid", logrus.InfoLevel}, // Test fallback to info for invalid levels
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			logger, err := NewLogger(tc.level, "")
			if err != nil {
				t.Fatalf("Failed to create logger with level %s: %v", tc.level, err)
			}

			if logger.Logger.Level != tc.expected {
				t.Errorf("Expected log level to be %v, got %v", tc.expected, logger.Logger.Level)
			}
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	// Capture output
	var buf bytes.Buffer

	logger, err := NewLogger("info", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Set output to our buffer
	logger.Logger.SetOutput(&buf)

	// Log a message
	logger.Info("test message")

	// Check output
	output := buf.String()
	if output == "" {
		t.Error("Expected log output, got empty string")
	}

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("Failed to parse JSON log output: %v", err)
	}

	// Check required fields
	if logEntry["level"] != "info" {
		t.Errorf("Expected level 'info', got %v", logEntry["level"])
	}

	if logEntry["msg"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", logEntry["msg"])
	}

	if logEntry["time"] == nil {
		t.Error("Expected timestamp field")
	}
}

func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer

	logger, err := NewLogger("info", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Logger.SetOutput(&buf)

	// Log with fields
	logger.WithFields(logrus.Fields{
		"user_id": "123",
		"action":  "login",
	}).Info("User logged in")

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &logEntry); err != nil {
		t.Errorf("Failed to parse JSON log output: %v", err)
	}

	// Check fields
	if logEntry["user_id"] != "123" {
		t.Errorf("Expected user_id '123', got %v", logEntry["user_id"])
	}

	if logEntry["action"] != "login" {
		t.Errorf("Expected action 'login', got %v", logEntry["action"])
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	// Create logger with WARN level
	logger, err := NewLogger("warn", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Logger.SetOutput(&buf)

	// Log at different levels
	logger.Debug("debug message")  // Should be filtered out
	logger.Info("info message")    // Should be filtered out
	logger.Warn("warn message")    // Should appear
	logger.Error("error message") // Should appear

	output := buf.String()

	// Check that debug and info messages are filtered out
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out at WARN level")
	}

	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out at WARN level")
	}

	// Check that warn and error messages appear
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should appear at WARN level")
	}

	if !strings.Contains(output, "error message") {
		t.Error("Error message should appear at WARN level")
	}
}

func TestNewLoggerWithSyslog(t *testing.T) {
	// Create logger with syslog endpoint - the warning is logged during creation
	// but goes to stdout. We can't capture it in the buffer since buffer is set after.
	// This test just verifies the logger is created successfully with a syslog endpoint.
	logger, err := NewLogger("info", "udp://localhost:514")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created, got nil")
	}

	// Verify logger functions correctly after creation
	var buf bytes.Buffer
	logger.Logger.SetOutput(&buf)
	logger.Info("test message after syslog config")

	output := buf.String()
	if !strings.Contains(output, "test message after syslog config") {
		t.Error("Expected log message to be written")
	}
}

func TestRequestLogging(t *testing.T) {
	var buf bytes.Buffer

	logger, err := NewLogger("info", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Logger.SetOutput(&buf)

	// Log a request
	logger.LogRequest("GET", "/api/test", "200", "100ms", "192.168.1.1")

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &logEntry); err != nil {
		t.Errorf("Failed to parse JSON log output: %v", err)
	}

	// Check request fields
	if logEntry["method"] != "GET" {
		t.Errorf("Expected method 'GET', got %v", logEntry["method"])
	}

	if logEntry["path"] != "/api/test" {
		t.Errorf("Expected path '/api/test', got %v", logEntry["path"])
	}

	if logEntry["status"] != "200" {
		t.Errorf("Expected status '200', got %v", logEntry["status"])
	}

	if logEntry["duration"] != "100ms" {
		t.Errorf("Expected duration '100ms', got %v", logEntry["duration"])
	}

	if logEntry["client_ip"] != "192.168.1.1" {
		t.Errorf("Expected client_ip '192.168.1.1', got %v", logEntry["client_ip"])
	}
}

func TestAuthenticationLogging(t *testing.T) {
	var buf bytes.Buffer

	logger, err := NewLogger("info", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Logger.SetOutput(&buf)

	// Log authentication event
	logger.LogAuthentication("user123", "192.168.1.1", true, "")

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &logEntry); err != nil {
		t.Errorf("Failed to parse JSON log output: %v", err)
	}

	// Check authentication fields
	if logEntry["user"] != "user123" {
		t.Errorf("Expected user 'user123', got %v", logEntry["user"])
	}

	if logEntry["client_ip"] != "192.168.1.1" {
		t.Errorf("Expected client_ip '192.168.1.1', got %v", logEntry["client_ip"])
	}

	if logEntry["success"] != true {
		t.Errorf("Expected success true, got %v", logEntry["success"])
	}
}

func TestErrorLogging(t *testing.T) {
	var buf bytes.Buffer

	logger, err := NewLogger("info", "")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Logger.SetOutput(&buf)

	// Log error event
	logger.LogError("database_error", "Failed to connect to database", "connection timeout")

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &logEntry); err != nil {
		t.Errorf("Failed to parse JSON log output: %v", err)
	}

	// Check error fields
	if logEntry["error_type"] != "database_error" {
		t.Errorf("Expected error_type 'database_error', got %v", logEntry["error_type"])
	}

	if logEntry["error_message"] != "Failed to connect to database" {
		t.Errorf("Expected error_message 'Failed to connect to database', got %v", logEntry["error_message"])
	}

	if logEntry["details"] != "connection timeout" {
		t.Errorf("Expected details 'connection timeout', got %v", logEntry["details"])
	}
}

func BenchmarkLogInfo(b *testing.B) {
	logger, err := NewLogger("info", "")
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	// Discard output for benchmark
	logger.Logger.SetOutput(&bytes.Buffer{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLogWithFields(b *testing.B) {
	logger, err := NewLogger("info", "")
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}

	// Discard output for benchmark
	logger.Logger.SetOutput(&bytes.Buffer{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(logrus.Fields{
			"user_id": "123",
			"action":  "test",
		}).Info("benchmark message")
	}
}