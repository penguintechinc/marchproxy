// Package logging provides structured logging functionality for MarchProxy
package logging

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger is a structured logger interface
type Logger struct {
	*logrus.Entry
}

// NewLogger creates a new structured logger
func NewLogger(level string, syslogEndpoint string) (*Logger, error) {
	logger := logrus.New()
	
	// Set log level
	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)
	
	// Set JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	
	// Set output to stdout by default
	logger.SetOutput(os.Stdout)
	
	// TODO: Add syslog hook if syslogEndpoint is provided
	if syslogEndpoint != "" {
		// This would require additional syslog integration
		logger.WithField("syslog_endpoint", syslogEndpoint).Warn("Syslog integration not yet implemented")
	}
	
	entry := logger.WithFields(logrus.Fields{
		"service": "marchproxy",
		"version": "1.0.0",
	})
	
	return &Logger{Entry: entry}, nil
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{Entry: l.Entry.WithField(key, value)}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{Entry: l.Entry.WithFields(fields)}
}

// Info logs an info message with optional key-value pairs
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	fields := parseKeysAndValues(keysAndValues...)
	l.Entry.WithFields(fields).Info(msg)
}

// Error logs an error message with optional key-value pairs
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	fields := parseKeysAndValues(keysAndValues...)
	l.Entry.WithFields(fields).Error(msg)
}

// Warn logs a warning message with optional key-value pairs
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	fields := parseKeysAndValues(keysAndValues...)
	l.Entry.WithFields(fields).Warn(msg)
}

// Debug logs a debug message with optional key-value pairs
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	fields := parseKeysAndValues(keysAndValues...)
	l.Entry.WithFields(fields).Debug(msg)
}

// parseKeysAndValues converts alternating key-value pairs to a map
func parseKeysAndValues(keysAndValues ...interface{}) logrus.Fields {
	fields := logrus.Fields{}
	
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key := fmt.Sprintf("%v", keysAndValues[i])
			value := keysAndValues[i+1]
			fields[key] = value
		}
	}
	
	return fields
}