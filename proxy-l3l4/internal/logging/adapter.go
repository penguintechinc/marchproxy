package logging

import (
	"fmt"

	"github.com/penguintechinc/penguin-libs/packages/go-common/logging"
	"go.uber.org/zap"
)

// LogrusAdapter wraps SanitizedLogger to provide a logrus-compatible interface.
type LogrusAdapter struct {
	logger *logging.SanitizedLogger
	fields []zap.Field
}

// NewLogrusAdapter creates an adapter providing logrus-style API on SanitizedLogger.
func NewLogrusAdapter(name string) (*LogrusAdapter, error) {
	logger, err := logging.NewSanitizedLogger(name)
	if err != nil {
		return nil, err
	}
	return &LogrusAdapter{logger: logger}, nil
}

// WithFields returns a new adapter with additional fields.
func (la *LogrusAdapter) WithFields(fields map[string]interface{}) *LogrusAdapter {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			zapFields = append(zapFields, zap.String(k, val))
		case int:
			zapFields = append(zapFields, zap.Int(k, val))
		case int64:
			zapFields = append(zapFields, zap.Int64(k, val))
		case float64:
			zapFields = append(zapFields, zap.Float64(k, val))
		case bool:
			zapFields = append(zapFields, zap.Bool(k, val))
		default:
			zapFields = append(zapFields, zap.Any(k, val))
		}
	}

	newAdapter := *la
	newAdapter.fields = append([]zap.Field{}, la.fields...)
	newAdapter.fields = append(newAdapter.fields, zapFields...)
	return &newAdapter
}

// WithError returns a new adapter with error field.
func (la *LogrusAdapter) WithError(err error) *LogrusAdapter {
	return la.WithFields(map[string]interface{}{"error": err.Error()})
}

// Info logs an info message.
func (la *LogrusAdapter) Info(args ...interface{}) {
	msg := ""
	if len(args) > 0 {
		msg = fmt.Sprint(args...)
	}
	la.logger.Info(msg, la.fields...)
}

// Infof logs a formatted info message.
func (la *LogrusAdapter) Infof(format string, args ...interface{}) {
	la.logger.Info(fmt.Sprintf(format, args...), la.fields...)
}

// Warn logs a warning message.
func (la *LogrusAdapter) Warn(args ...interface{}) {
	msg := ""
	if len(args) > 0 {
		msg = fmt.Sprint(args...)
	}
	la.logger.Warn(msg, la.fields...)
}

// Warnf logs a formatted warning message.
func (la *LogrusAdapter) Warnf(format string, args ...interface{}) {
	la.logger.Warn(fmt.Sprintf(format, args...), la.fields...)
}

// Error logs an error message.
func (la *LogrusAdapter) Error(args ...interface{}) {
	msg := ""
	if len(args) > 0 {
		msg = fmt.Sprint(args...)
	}
	la.logger.Error(msg, la.fields...)
}

// Errorf logs a formatted error message.
func (la *LogrusAdapter) Errorf(format string, args ...interface{}) {
	la.logger.Error(fmt.Sprintf(format, args...), la.fields...)
}

// Fatal logs an error and exits.
func (la *LogrusAdapter) Fatal(args ...interface{}) {
	msg := ""
	if len(args) > 0 {
		msg = fmt.Sprint(args...)
	}
	la.logger.Error(msg, la.fields...)
}

// Fatalf logs a formatted error and exits.
func (la *LogrusAdapter) Fatalf(format string, args ...interface{}) {
	la.logger.Error(fmt.Sprintf(format, args...), la.fields...)
}

// Debug logs a debug message.
func (la *LogrusAdapter) Debug(args ...interface{}) {
	msg := ""
	if len(args) > 0 {
		msg = fmt.Sprint(args...)
	}
	la.logger.Debug(msg, la.fields...)
}

// Debugf logs a formatted debug message.
func (la *LogrusAdapter) Debugf(format string, args ...interface{}) {
	la.logger.Debug(fmt.Sprintf(format, args...), la.fields...)
}

// WithField returns a new adapter with a single field.
func (la *LogrusAdapter) WithField(key string, value interface{}) *LogrusAdapter {
	return la.WithFields(map[string]interface{}{key: value})
}
