package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
	config LogConfig
}

type LogConfig struct {
	Level       string
	Format      string
	Output      string
	File        string
	MaxSize     int64
	MaxAge      int
	MaxBackups  int
	Compress    bool
	Structured  bool
	Fields      map[string]interface{}
	SyslogAddr  string
	SyslogNet   string
}

type MTLSLogEntry struct {
	Timestamp      time.Time `json:"timestamp"`
	Level          string    `json:"level"`
	Message        string    `json:"message"`
	ClientCN       string    `json:"client_cn,omitempty"`
	ClientOU       string    `json:"client_ou,omitempty"`
	ClientSerial   string    `json:"client_serial,omitempty"`
	ServerName     string    `json:"server_name,omitempty"`
	TLSVersion     string    `json:"tls_version,omitempty"`
	CipherSuite    string    `json:"cipher_suite,omitempty"`
	Result         string    `json:"result"`
	Error          string    `json:"error,omitempty"`
	VirtualHost    string    `json:"virtual_host,omitempty"`
	Backend        string    `json:"backend,omitempty"`
	RequestID      string    `json:"request_id,omitempty"`
	RemoteAddr     string    `json:"remote_addr,omitempty"`
}

type RequestLogEntry struct {
	Timestamp       time.Time         `json:"timestamp"`
	Level           string            `json:"level"`
	Message         string            `json:"message"`
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Path            string            `json:"path"`
	StatusCode      int               `json:"status_code"`
	ResponseTime    time.Duration     `json:"response_time"`
	RequestSize     int64             `json:"request_size"`
	ResponseSize    int64             `json:"response_size"`
	UserAgent       string            `json:"user_agent,omitempty"`
	Referer         string            `json:"referer,omitempty"`
	XForwardedFor   string            `json:"x_forwarded_for,omitempty"`
	VirtualHost     string            `json:"virtual_host"`
	Backend         string            `json:"backend"`
	BackendEndpoint string            `json:"backend_endpoint,omitempty"`
	RequestID       string            `json:"request_id"`
	RemoteAddr      string            `json:"remote_addr"`
	Headers         map[string]string `json:"headers,omitempty"`
	Error           string            `json:"error,omitempty"`
}

type HealthLogEntry struct {
	Timestamp       time.Time `json:"timestamp"`
	Level           string    `json:"level"`
	Message         string    `json:"message"`
	CheckType       string    `json:"check_type"`
	Target          string    `json:"target"`
	Status          string    `json:"status"`
	ResponseTime    time.Duration `json:"response_time"`
	Error           string    `json:"error,omitempty"`
	VirtualHost     string    `json:"virtual_host,omitempty"`
	Backend         string    `json:"backend,omitempty"`
	BackendEndpoint string    `json:"backend_endpoint,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

func NewLogger(config LogConfig) (*Logger, error) {
	logger := logrus.New()

	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	if config.Structured {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	if config.File != "" {
		if err := os.MkdirAll(filepath.Dir(config.File), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		if config.Output == "both" {
			logger.SetOutput(io.MultiWriter(os.Stdout, file))
		} else {
			logger.SetOutput(file)
		}
	} else {
		logger.SetOutput(os.Stdout)
	}

	l := &Logger{
		Logger: logger,
		config: config,
	}

	if len(config.Fields) > 0 {
		l.Logger = l.Logger.WithFields(config.Fields).Logger
	}

	return l, nil
}

func (l *Logger) LogMTLSAuth(entry MTLSLogEntry) {
	fields := logrus.Fields{
		"component":       "mtls_auth",
		"client_cn":       entry.ClientCN,
		"client_ou":       entry.ClientOU,
		"client_serial":   entry.ClientSerial,
		"server_name":     entry.ServerName,
		"tls_version":     entry.TLSVersion,
		"cipher_suite":    entry.CipherSuite,
		"result":          entry.Result,
		"virtual_host":    entry.VirtualHost,
		"backend":         entry.Backend,
		"request_id":      entry.RequestID,
		"remote_addr":     entry.RemoteAddr,
	}

	if entry.Error != "" {
		fields["error"] = entry.Error
	}

	if entry.Result == "success" {
		l.WithFields(fields).Info(entry.Message)
	} else {
		l.WithFields(fields).Warn(entry.Message)
	}
}

func (l *Logger) LogRequest(entry RequestLogEntry) {
	fields := logrus.Fields{
		"component":         "request",
		"method":            entry.Method,
		"url":               entry.URL,
		"path":              entry.Path,
		"status_code":       entry.StatusCode,
		"response_time_ms":  entry.ResponseTime.Milliseconds(),
		"request_size":      entry.RequestSize,
		"response_size":     entry.ResponseSize,
		"user_agent":        entry.UserAgent,
		"referer":           entry.Referer,
		"x_forwarded_for":   entry.XForwardedFor,
		"virtual_host":      entry.VirtualHost,
		"backend":           entry.Backend,
		"backend_endpoint":  entry.BackendEndpoint,
		"request_id":        entry.RequestID,
		"remote_addr":       entry.RemoteAddr,
	}

	if entry.Error != "" {
		fields["error"] = entry.Error
	}

	if len(entry.Headers) > 0 {
		fields["headers"] = entry.Headers
	}

	if entry.StatusCode >= 200 && entry.StatusCode < 400 {
		l.WithFields(fields).Info(entry.Message)
	} else if entry.StatusCode >= 400 && entry.StatusCode < 500 {
		l.WithFields(fields).Warn(entry.Message)
	} else {
		l.WithFields(fields).Error(entry.Message)
	}
}

func (l *Logger) LogHealth(entry HealthLogEntry) {
	fields := logrus.Fields{
		"component":         "health_check",
		"check_type":        entry.CheckType,
		"target":            entry.Target,
		"status":            entry.Status,
		"response_time_ms":  entry.ResponseTime.Milliseconds(),
		"virtual_host":      entry.VirtualHost,
		"backend":           entry.Backend,
		"backend_endpoint":  entry.BackendEndpoint,
	}

	if entry.Error != "" {
		fields["error"] = entry.Error
	}

	if entry.Metadata != nil {
		fields["metadata"] = entry.Metadata
	}

	switch entry.Status {
	case "healthy":
		l.WithFields(fields).Debug(entry.Message)
	case "degraded":
		l.WithFields(fields).Warn(entry.Message)
	case "unhealthy":
		l.WithFields(fields).Error(entry.Message)
	default:
		l.WithFields(fields).Info(entry.Message)
	}
}

func (l *Logger) LogConfigUpdate(message string, fields map[string]interface{}) {
	logFields := logrus.Fields{
		"component": "config",
	}

	for k, v := range fields {
		logFields[k] = v
	}

	l.WithFields(logFields).Info(message)
}

func (l *Logger) LogCertificateEvent(message string, certInfo map[string]interface{}) {
	fields := logrus.Fields{
		"component": "certificate",
	}

	for k, v := range certInfo {
		fields[k] = v
	}

	l.WithFields(fields).Info(message)
}

func (l *Logger) LogLoadBalancer(message string, backend, algorithm, endpoint string) {
	fields := logrus.Fields{
		"component": "load_balancer",
		"backend":   backend,
		"algorithm": algorithm,
		"endpoint":  endpoint,
	}

	l.WithFields(fields).Info(message)
}

func (l *Logger) LogCircuitBreaker(message string, backend, state string, errorRate float64) {
	fields := logrus.Fields{
		"component":  "circuit_breaker",
		"backend":    backend,
		"state":      state,
		"error_rate": errorRate,
	}

	l.WithFields(fields).Warn(message)
}

func (l *Logger) LogRateLimit(message string, clientIP, reason string, limit int) {
	fields := logrus.Fields{
		"component": "rate_limit",
		"client_ip": clientIP,
		"reason":    reason,
		"limit":     limit,
	}

	l.WithFields(fields).Warn(message)
}

func (l *Logger) LogError(err error, context string, fields map[string]interface{}) {
	logFields := logrus.Fields{
		"component": context,
		"error":     err.Error(),
	}

	for k, v := range fields {
		logFields[k] = v
	}

	l.WithFields(logFields).Error("Error occurred")
}

func (l *Logger) LogStartup(version, buildTime string) {
	fields := logrus.Fields{
		"component":  "startup",
		"version":    version,
		"build_time": buildTime,
		"proxy_type": "ingress",
	}

	l.WithFields(fields).Info("MarchProxy Ingress starting up")
}

func (l *Logger) LogShutdown(reason string) {
	fields := logrus.Fields{
		"component": "shutdown",
		"reason":    reason,
	}

	l.WithFields(fields).Info("MarchProxy Ingress shutting down")
}

func (l *Logger) WithRequestID(requestID string) *logrus.Entry {
	return l.WithField("request_id", requestID)
}

func (l *Logger) WithVirtualHost(vhost string) *logrus.Entry {
	return l.WithField("virtual_host", vhost)
}

func (l *Logger) WithBackend(backend string) *logrus.Entry {
	return l.WithField("backend", backend)
}

func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.WithField("component", component)
}

func DefaultLogConfig() LogConfig {
	return LogConfig{
		Level:      "info",
		Format:     "text",
		Output:     "stdout",
		Structured: false,
		Fields: map[string]interface{}{
			"service": "marchproxy-ingress",
		},
	}
}