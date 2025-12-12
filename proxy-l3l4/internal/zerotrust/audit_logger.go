package zerotrust

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AuditLogger provides immutable audit logging with SHA-256 chaining
type AuditLogger struct {
	mu            sync.Mutex
	logPath       string
	logFile       *os.File
	previousHash  string
	eventCount    int64
	logger        *logrus.Logger
	rotateSize    int64
	rotateEnabled bool
	chainBroken   bool
}

// AuditEvent represents an audit log event
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventID     int64                  `json:"event_id"`
	EventType   string                 `json:"event_type"`
	Service     string                 `json:"service,omitempty"`
	User        string                 `json:"user,omitempty"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	SourceIP    string                 `json:"source_ip"`
	Allowed     bool                   `json:"allowed"`
	Reason      string                 `json:"reason,omitempty"`
	PolicyName  string                 `json:"policy_name,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	PrevHash    string                 `json:"prev_hash"`
	CurrentHash string                 `json:"current_hash"`
}

// AuditChainEntry represents a single entry in the audit chain
type AuditChainEntry struct {
	Event *AuditEvent `json:"event"`
	Hash  string      `json:"hash"`
}

// NewAuditLogger creates a new immutable audit logger
func NewAuditLogger(logPath string, logger *logrus.Logger) (*AuditLogger, error) {
	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file in append mode
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	al := &AuditLogger{
		logPath:       logPath,
		logFile:       file,
		previousHash:  "0000000000000000000000000000000000000000000000000000000000000000", // Genesis hash
		eventCount:    0,
		logger:        logger,
		rotateSize:    100 * 1024 * 1024, // 100MB default
		rotateEnabled: true,
		chainBroken:   false,
	}

	// Load previous hash if file exists and has content
	if err := al.loadPreviousHash(); err != nil {
		logger.WithError(err).Warn("Failed to load previous hash, starting new chain")
	}

	logger.WithField("log_path", logPath).Info("Audit logger initialized")

	return al, nil
}

// LogEvent logs an audit event with hash chaining
func (al *AuditLogger) LogEvent(event *AuditEvent) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Increment event count
	al.eventCount++
	event.EventID = al.eventCount

	// Set previous hash
	event.PrevHash = al.previousHash

	// Calculate current hash
	hash, err := al.calculateHash(event)
	if err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	event.CurrentHash = hash

	// Create chain entry
	entry := &AuditChainEntry{
		Event: event,
		Hash:  hash,
	}

	// Serialize to JSON
	eventJSON, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	// Write to file (append-only, immutable)
	if _, err := al.logFile.WriteString(string(eventJSON) + "\n"); err != nil {
		al.chainBroken = true
		return fmt.Errorf("failed to write audit event: %w", err)
	}

	// Flush to ensure durability
	if err := al.logFile.Sync(); err != nil {
		al.logger.WithError(err).Warn("Failed to sync audit log")
	}

	// Update previous hash for next event
	al.previousHash = hash

	// Check if rotation is needed
	if al.rotateEnabled {
		if err := al.checkRotation(); err != nil {
			al.logger.WithError(err).Error("Failed to rotate audit log")
		}
	}

	return nil
}

// calculateHash calculates SHA-256 hash of the audit event
func (al *AuditLogger) calculateHash(event *AuditEvent) (string, error) {
	// Create deterministic string representation
	data := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%t|%s|%s",
		event.EventID,
		event.Timestamp.Format(time.RFC3339Nano),
		event.EventType,
		event.Service,
		event.User,
		event.Action,
		event.Resource,
		event.Allowed,
		event.PrevHash,
		event.PolicyName,
	)

	// Calculate SHA-256 hash
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:]), nil
}

// loadPreviousHash loads the hash of the last event from the log file
func (al *AuditLogger) loadPreviousHash() error {
	// Get file size
	fileInfo, err := al.logFile.Stat()
	if err != nil {
		return err
	}

	// If file is empty, use genesis hash
	if fileInfo.Size() == 0 {
		return nil
	}

	// Read last line from file
	// Note: This is a simplified implementation
	// Production code should use a more efficient last-line reader

	content, err := os.ReadFile(al.logPath)
	if err != nil {
		return err
	}

	if len(content) == 0 {
		return nil
	}

	// Find last newline
	lines := string(content)
	lastNewline := len(lines) - 1
	for lastNewline >= 0 && lines[lastNewline] == '\n' {
		lastNewline--
	}

	// Find previous newline
	prevNewline := lastNewline
	for prevNewline >= 0 && lines[prevNewline] != '\n' {
		prevNewline--
	}

	if prevNewline < lastNewline {
		lastLine := lines[prevNewline+1 : lastNewline+1]

		var entry AuditChainEntry
		if err := json.Unmarshal([]byte(lastLine), &entry); err != nil {
			return fmt.Errorf("failed to parse last audit entry: %w", err)
		}

		al.previousHash = entry.Hash
		al.eventCount = entry.Event.EventID
		al.logger.WithFields(logrus.Fields{
			"event_id":      al.eventCount,
			"previous_hash": al.previousHash[:16] + "...",
		}).Info("Loaded audit chain state")
	}

	return nil
}

// checkRotation checks if log rotation is needed
func (al *AuditLogger) checkRotation() error {
	fileInfo, err := al.logFile.Stat()
	if err != nil {
		return err
	}

	if fileInfo.Size() >= al.rotateSize {
		return al.rotateLog()
	}

	return nil
}

// rotateLog rotates the audit log file
func (al *AuditLogger) rotateLog() error {
	// Close current file
	if err := al.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	// Rename current file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", al.logPath, timestamp)

	if err := os.Rename(al.logPath, rotatedPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Create new log file
	newFile, err := os.OpenFile(al.logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	al.logFile = newFile

	al.logger.WithFields(logrus.Fields{
		"rotated_file": rotatedPath,
		"new_file":     al.logPath,
	}).Info("Rotated audit log")

	return nil
}

// VerifyChain verifies the integrity of the audit log chain
func (al *AuditLogger) VerifyChain() (bool, error) {
	content, err := os.ReadFile(al.logPath)
	if err != nil {
		return false, fmt.Errorf("failed to read audit log: %w", err)
	}

	lines := []string{}
	currentLine := ""
	for _, ch := range string(content) {
		if ch == '\n' {
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
				currentLine = ""
			}
		} else {
			currentLine += string(ch)
		}
	}

	if len(lines) == 0 {
		return true, nil // Empty log is valid
	}

	previousHash := "0000000000000000000000000000000000000000000000000000000000000000"

	for i, line := range lines {
		var entry AuditChainEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return false, fmt.Errorf("failed to parse entry %d: %w", i, err)
		}

		// Verify hash chain
		if entry.Event.PrevHash != previousHash {
			return false, fmt.Errorf("chain broken at entry %d: expected prev hash %s, got %s",
				i, previousHash[:16], entry.Event.PrevHash[:16])
		}

		// Recalculate hash
		calculatedHash, err := al.calculateHash(entry.Event)
		if err != nil {
			return false, fmt.Errorf("failed to calculate hash for entry %d: %w", i, err)
		}

		if calculatedHash != entry.Hash {
			return false, fmt.Errorf("hash mismatch at entry %d: expected %s, got %s",
				i, calculatedHash[:16], entry.Hash[:16])
		}

		previousHash = entry.Hash
	}

	al.logger.WithField("entries_verified", len(lines)).Info("Audit chain verification successful")
	return true, nil
}

// GetEvents retrieves audit events within a time range
func (al *AuditLogger) GetEvents(startTime, endTime time.Time) ([]*AuditEvent, error) {
	content, err := os.ReadFile(al.logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	lines := []string{}
	currentLine := ""
	for _, ch := range string(content) {
		if ch == '\n' {
			if len(currentLine) > 0 {
				lines = append(lines, currentLine)
				currentLine = ""
			}
		} else {
			currentLine += string(ch)
		}
	}

	events := []*AuditEvent{}

	for _, line := range lines {
		var entry AuditChainEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if entry.Event.Timestamp.After(startTime) && entry.Event.Timestamp.Before(endTime) {
			events = append(events, entry.Event)
		}
	}

	return events, nil
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.logFile != nil {
		return al.logFile.Close()
	}

	return nil
}

// SetRotateSize sets the file size threshold for rotation
func (al *AuditLogger) SetRotateSize(size int64) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.rotateSize = size
}

// IsChainBroken returns whether the audit chain has been broken
func (al *AuditLogger) IsChainBroken() bool {
	al.mu.Lock()
	defer al.mu.Unlock()
	return al.chainBroken
}
