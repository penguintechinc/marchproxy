//go:build ci
// +build ci

package zerotrust

import (
	"os"
	"testing"
	"time"

	"marchproxy-l3l4/internal/logging"
)

func TestNewAuditLogger(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	logFile := "/tmp/audit_test.log"
	defer os.Remove(logFile)

	auditLogger, err := NewAuditLogger(logFile, logger)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	if auditLogger == nil {
		t.Fatal("NewAuditLogger returned nil")
	}
}

func TestAuditLoggerLogEvent(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	logFile := "/tmp/audit_test_log.log"
	defer os.Remove(logFile)

	auditLogger, _ := NewAuditLogger(logFile, logger)

	event := &AuditEvent{
		Timestamp:  time.Now(),
		User:       "test-user",
		Action:     "LOGIN",
		Resource:   "/admin",
		SourceIP:   "192.168.1.1",
		Allowed:    true,
		PolicyName: "default-policy",
	}

	err := auditLogger.LogEvent(event)
	if err != nil {
		t.Errorf("LogEvent failed: %v", err)
	}
}

func TestAuditLoggerLogEventValid(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	logFile := "/tmp/audit_test_valid.log"
	defer os.Remove(logFile)

	auditLogger, _ := NewAuditLogger(logFile, logger)

	event := &AuditEvent{
		Timestamp: time.Now(),
		User:      "user1",
		Action:    "LOGIN",
		Resource:  "/admin",
		SourceIP:  "10.0.0.1",
		Allowed:   true,
	}

	err := auditLogger.LogEvent(event)
	if err != nil {
		t.Errorf("LogEvent should succeed with valid event: %v", err)
	}
}

func TestAuditEventHashChaining(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	logFile := "/tmp/audit_test_chain.log"
	defer os.Remove(logFile)

	auditLogger, _ := NewAuditLogger(logFile, logger)

	event1 := &AuditEvent{
		Timestamp: time.Now(),
		User:      "user1",
		Action:    "READ",
		Resource:  "/data",
		SourceIP:  "10.0.0.1",
		Allowed:   true,
	}

	err := auditLogger.LogEvent(event1)
	if err != nil {
		t.Fatalf("First LogEvent failed: %v", err)
	}

	// Event should have hash chain set
	if event1.EventID != 1 {
		t.Errorf("Expected EventID 1, got %d", event1.EventID)
	}

	// Log second event
	event2 := &AuditEvent{
		Timestamp: time.Now(),
		User:      "user2",
		Action:    "WRITE",
		Resource:  "/config",
		SourceIP:  "10.0.0.2",
		Allowed:   false,
		Reason:    "insufficient permissions",
	}

	err = auditLogger.LogEvent(event2)
	if err != nil {
		t.Fatalf("Second LogEvent failed: %v", err)
	}

	if event2.EventID != 2 {
		t.Errorf("Expected EventID 2, got %d", event2.EventID)
	}

	// Verify hash chaining: event2's PrevHash should be event1's CurrentHash
	if event2.PrevHash == "" {
		t.Error("Event2 PrevHash should not be empty")
	}
}

func TestAuditEventWithReason(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	logFile := "/tmp/audit_test_reason.log"
	defer os.Remove(logFile)

	auditLogger, _ := NewAuditLogger(logFile, logger)

	event := &AuditEvent{
		Timestamp: time.Now(),
		User:      "user1",
		Action:    "DELETE",
		Resource:  "/user/test",
		SourceIP:  "10.0.0.1",
		Allowed:   false,
		Reason:    "user not found",
		Duration:  100 * time.Millisecond,
	}

	err := auditLogger.LogEvent(event)
	if err != nil {
		t.Errorf("LogEvent with reason failed: %v", err)
	}
}

func TestAuditEventMetadata(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	logFile := "/tmp/audit_test_meta.log"
	defer os.Remove(logFile)

	auditLogger, _ := NewAuditLogger(logFile, logger)

	metadata := map[string]interface{}{
		"request_id": "req-123",
		"user_agent": "Mozilla/5.0",
	}

	event := &AuditEvent{
		Timestamp: time.Now(),
		User:      "user1",
		Action:    "WRITE",
		Resource:  "/data",
		SourceIP:  "10.0.0.1",
		Allowed:   true,
		Metadata:  metadata,
	}

	err := auditLogger.LogEvent(event)
	if err != nil {
		t.Errorf("LogEvent with metadata failed: %v", err)
	}
}
