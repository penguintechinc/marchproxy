package logging

import (
	"fmt"
	"time"

	"github.com/penguintechinc/penguin-libs/packages/go-common/logging"
	"go.uber.org/zap"
)

type Logger struct {
	*logging.SanitizedLogger
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
	logger, err := logging.NewSanitizedLogger("marchproxy")
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Note: File output configuration via SanitizedLogger is handled internally.
	// Log level and other configuration can be set via environment variables.

	l := &Logger{
		SanitizedLogger: logger,
		config:          config,
	}

	return l, nil
}

func (l *Logger) LogMTLSAuth(entry MTLSLogEntry) {
	fields := []zap.Field{
		zap.String("component", "mtls_auth"),
		zap.String("client_cn", entry.ClientCN),
		zap.String("client_ou", entry.ClientOU),
		zap.String("client_serial", entry.ClientSerial),
		zap.String("server_name", entry.ServerName),
		zap.String("tls_version", entry.TLSVersion),
		zap.String("cipher_suite", entry.CipherSuite),
		zap.String("result", entry.Result),
		zap.String("virtual_host", entry.VirtualHost),
		zap.String("backend", entry.Backend),
		zap.String("request_id", entry.RequestID),
		zap.String("remote_addr", entry.RemoteAddr),
	}

	if entry.Error != "" {
		fields = append(fields, zap.String("error", entry.Error))
	}

	if entry.Result == "success" {
		l.SanitizedLogger.Info(entry.Message, fields...)
	} else {
		l.SanitizedLogger.Warn(entry.Message, fields...)
	}
}

func (l *Logger) LogRequest(entry RequestLogEntry) {
	fields := []zap.Field{
		zap.String("component", "request"),
		zap.String("method", entry.Method),
		zap.String("url", entry.URL),
		zap.String("path", entry.Path),
		zap.Int("status_code", entry.StatusCode),
		zap.Int64("response_time_ms", entry.ResponseTime.Milliseconds()),
		zap.Int64("request_size", entry.RequestSize),
		zap.Int64("response_size", entry.ResponseSize),
		zap.String("user_agent", entry.UserAgent),
		zap.String("referer", entry.Referer),
		zap.String("x_forwarded_for", entry.XForwardedFor),
		zap.String("virtual_host", entry.VirtualHost),
		zap.String("backend", entry.Backend),
		zap.String("backend_endpoint", entry.BackendEndpoint),
		zap.String("request_id", entry.RequestID),
		zap.String("remote_addr", entry.RemoteAddr),
	}

	if entry.Error != "" {
		fields = append(fields, zap.String("error", entry.Error))
	}

	if len(entry.Headers) > 0 {
		fields = append(fields, zap.Any("headers", entry.Headers))
	}

	if entry.StatusCode >= 200 && entry.StatusCode < 400 {
		l.SanitizedLogger.Info(entry.Message, fields...)
	} else if entry.StatusCode >= 400 && entry.StatusCode < 500 {
		l.SanitizedLogger.Warn(entry.Message, fields...)
	} else {
		l.SanitizedLogger.Error(entry.Message, fields...)
	}
}

func (l *Logger) LogHealth(entry HealthLogEntry) {
	fields := []zap.Field{
		zap.String("component", "health_check"),
		zap.String("check_type", entry.CheckType),
		zap.String("target", entry.Target),
		zap.String("status", entry.Status),
		zap.Int64("response_time_ms", entry.ResponseTime.Milliseconds()),
		zap.String("virtual_host", entry.VirtualHost),
		zap.String("backend", entry.Backend),
		zap.String("backend_endpoint", entry.BackendEndpoint),
	}

	if entry.Error != "" {
		fields = append(fields, zap.String("error", entry.Error))
	}

	if entry.Metadata != nil {
		fields = append(fields, zap.Any("metadata", entry.Metadata))
	}

	switch entry.Status {
	case "healthy":
		l.SanitizedLogger.Debug(entry.Message, fields...)
	case "degraded":
		l.SanitizedLogger.Warn(entry.Message, fields...)
	case "unhealthy":
		l.SanitizedLogger.Error(entry.Message, fields...)
	default:
		l.SanitizedLogger.Info(entry.Message, fields...)
	}
}

func (l *Logger) LogConfigUpdate(message string, fields map[string]interface{}) {
	zapFields := []zap.Field{zap.String("component", "config")}

	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	l.SanitizedLogger.Info(message, zapFields...)
}

func (l *Logger) LogCertificateEvent(message string, certInfo map[string]interface{}) {
	zapFields := []zap.Field{zap.String("component", "certificate")}

	for k, v := range certInfo {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	l.SanitizedLogger.Info(message, zapFields...)
}

func (l *Logger) LogLoadBalancer(message string, backend, algorithm, endpoint string) {
	fields := []zap.Field{
		zap.String("component", "load_balancer"),
		zap.String("backend", backend),
		zap.String("algorithm", algorithm),
		zap.String("endpoint", endpoint),
	}

	l.SanitizedLogger.Info(message, fields...)
}

func (l *Logger) LogCircuitBreaker(message string, backend, state string, errorRate float64) {
	fields := []zap.Field{
		zap.String("component", "circuit_breaker"),
		zap.String("backend", backend),
		zap.String("state", state),
		zap.Float64("error_rate", errorRate),
	}

	l.SanitizedLogger.Warn(message, fields...)
}

func (l *Logger) LogRateLimit(message string, clientIP, reason string, limit int) {
	fields := []zap.Field{
		zap.String("component", "rate_limit"),
		zap.String("client_ip", clientIP),
		zap.String("reason", reason),
		zap.Int("limit", limit),
	}

	l.SanitizedLogger.Warn(message, fields...)
}

func (l *Logger) LogError(err error, context string, fields map[string]interface{}) {
	zapFields := []zap.Field{
		zap.String("component", context),
		zap.String("error", err.Error()),
	}

	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	l.SanitizedLogger.Error("Error occurred", zapFields...)
}

func (l *Logger) LogStartup(version, buildTime string) {
	fields := []zap.Field{
		zap.String("component", "startup"),
		zap.String("version", version),
		zap.String("build_time", buildTime),
		zap.String("proxy_type", "ingress"),
	}

	l.SanitizedLogger.Info("MarchProxy Ingress starting up", fields...)
}

func (l *Logger) LogShutdown(reason string) {
	fields := []zap.Field{
		zap.String("component", "shutdown"),
		zap.String("reason", reason),
	}

	l.SanitizedLogger.Info("MarchProxy Ingress shutting down", fields...)
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
