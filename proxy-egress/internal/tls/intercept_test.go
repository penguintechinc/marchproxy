package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// Helper function to generate test CA for tests
func generateTestCA(t *testing.T) (certPath, keyPath string, cleanup func()) {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to generate CA key: %v", err)
	}

	// Create CA certificate template
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	caTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "Test CA",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create CA certificate: %v", err)
	}

	// Write CA certificate
	certPath = filepath.Join(tempDir, "ca.crt")
	certFile, err := os.Create(certPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create cert file: %v", err)
	}
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	certFile.Close()

	// Write CA private key
	keyPath = filepath.Join(tempDir, "ca.key")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create key file: %v", err)
	}
	pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})
	keyFile.Close()

	cleanup = func() {
		os.RemoveAll(tempDir)
	}

	return certPath, keyPath, cleanup
}

func TestNewInterceptManager_Disabled(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled: false,
	}

	manager, err := NewInterceptManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewInterceptManager returned nil")
	}

	if manager.IsEnabled() {
		t.Error("Expected manager to be disabled")
	}
}

func TestNewInterceptManager_MITM(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 1000,
	}

	manager, err := NewInterceptManager(cfg, logger)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	if !manager.IsEnabled() {
		t.Error("Expected manager to be enabled")
	}

	if manager.GetMode() != InterceptModeMITM {
		t.Errorf("Expected mode MITM, got %s", manager.GetMode())
	}
}

func TestNewInterceptManager_InvalidCAPath(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: "/nonexistent/ca.crt",
		CAKeyPath:  "/nonexistent/ca.key",
	}

	_, err := NewInterceptManager(cfg, logger)
	if err == nil {
		t.Error("Expected error for invalid CA path")
	}
}

func TestInterceptManager_ShouldIntercept_Disabled(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled: false,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	if manager.ShouldIntercept("example.com", "192.168.1.1") {
		t.Error("Expected ShouldIntercept to return false when disabled")
	}
}

func TestInterceptManager_ShouldIntercept_DefaultEnabled(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	// By default, all hosts should be intercepted when enabled
	if !manager.ShouldIntercept("example.com", "192.168.1.1") {
		t.Error("Expected ShouldIntercept to return true by default")
	}
}

func TestInterceptManager_ShouldIntercept_DomainConfig(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
		DomainConfig: map[string]bool{
			"skip.example.com":      false, // Don't intercept
			"intercept.example.com": true,  // Intercept
		},
	}

	manager, _ := NewInterceptManager(cfg, logger)

	tests := []struct {
		host      string
		intercept bool
	}{
		{"skip.example.com", false},
		{"intercept.example.com", true},
		{"other.example.com", true}, // Default: intercept
	}

	for _, tt := range tests {
		result := manager.ShouldIntercept(tt.host, "")
		if result != tt.intercept {
			t.Errorf("Host %s: expected intercept=%v, got %v", tt.host, tt.intercept, result)
		}
	}
}

func TestInterceptManager_ShouldIntercept_IPConfig(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
		IPConfig: map[string]bool{
			"192.168.1.100": false, // Don't intercept
			"10.0.0.1":      true,  // Intercept
		},
	}

	manager, _ := NewInterceptManager(cfg, logger)

	tests := []struct {
		ip        string
		intercept bool
	}{
		{"192.168.1.100", false},
		{"10.0.0.1", true},
		{"172.16.0.1", true}, // Default: intercept
	}

	for _, tt := range tests {
		result := manager.ShouldIntercept("example.com", tt.ip)
		if result != tt.intercept {
			t.Errorf("IP %s: expected intercept=%v, got %v", tt.ip, tt.intercept, result)
		}
	}
}

func TestInterceptManager_GetCertificate_MITM(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 1000,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	cert, err := manager.GetCertificate("example.com")
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}

	if cert == nil {
		t.Fatal("GetCertificate returned nil certificate")
	}

	// Verify certificate has correct CN
	if cert.Leaf == nil {
		t.Fatal("Certificate Leaf is nil")
	}

	if cert.Leaf.Subject.CommonName != "example.com" {
		t.Errorf("Expected CN 'example.com', got '%s'", cert.Leaf.Subject.CommonName)
	}
}

func TestInterceptManager_GetCertificate_Cached(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 1000,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	// First call - should generate
	cert1, _ := manager.GetCertificate("example.com")

	// Second call - should use cache
	cert2, _ := manager.GetCertificate("example.com")

	// Verify both return same certificate (from cache)
	if cert1 != cert2 {
		t.Error("Expected same certificate from cache")
	}

	stats := manager.GetStats()
	if stats["cache_hits"] != 1 {
		t.Errorf("Expected 1 cache hit, got %d", stats["cache_hits"])
	}
}

func TestInterceptManager_GetCertificate_IPAddress(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 1000,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	cert, err := manager.GetCertificate("192.168.1.100")
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}

	if cert == nil {
		t.Fatal("GetCertificate returned nil certificate")
	}

	// Verify certificate has IP in IPAddresses
	if len(cert.Leaf.IPAddresses) == 0 {
		t.Error("Expected IP address in certificate")
	}
}

func TestInterceptManager_GetCertificate_Preconfigured_NotFound(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled: true,
		Mode:    InterceptModePreconfigured,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	_, err := manager.GetCertificate("example.com")
	if err == nil {
		t.Error("Expected error when no preconfigured cert available")
	}
}

func TestInterceptManager_SetDomainIntercept(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	// By default should intercept
	if !manager.ShouldIntercept("example.com", "") {
		t.Error("Expected to intercept by default")
	}

	// Disable for specific domain
	manager.SetDomainIntercept("example.com", false)

	if manager.ShouldIntercept("example.com", "") {
		t.Error("Expected not to intercept after setting to false")
	}

	// Re-enable
	manager.SetDomainIntercept("example.com", true)

	if !manager.ShouldIntercept("example.com", "") {
		t.Error("Expected to intercept after setting to true")
	}
}

func TestInterceptManager_SetIPIntercept(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	// By default should intercept
	if !manager.ShouldIntercept("example.com", "192.168.1.100") {
		t.Error("Expected to intercept by default")
	}

	// Disable for specific IP
	manager.SetIPIntercept("192.168.1.100", false)

	if manager.ShouldIntercept("example.com", "192.168.1.100") {
		t.Error("Expected not to intercept after setting to false")
	}
}

func TestInterceptManager_Enable_Disable(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	if !manager.IsEnabled() {
		t.Error("Expected manager to be enabled")
	}

	manager.Disable()

	if manager.IsEnabled() {
		t.Error("Expected manager to be disabled")
	}

	if manager.ShouldIntercept("example.com", "") {
		t.Error("Expected ShouldIntercept to return false when disabled")
	}

	manager.Enable()

	if !manager.IsEnabled() {
		t.Error("Expected manager to be enabled after Enable()")
	}
}

func TestInterceptManager_ClearCache(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 1000,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	// Generate some certificates
	manager.GetCertificate("example1.com")
	manager.GetCertificate("example2.com")

	// Clear cache
	manager.ClearCache()

	// Get stats - cache should be empty now, but certs_generated stays
	stats := manager.GetStats()
	if stats["certs_generated"] != 2 {
		t.Errorf("Expected 2 certs generated, got %d", stats["certs_generated"])
	}
}

func TestInterceptManager_GetDomainConfig(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
		DomainConfig: map[string]bool{
			"domain1.com": true,
			"domain2.com": false,
		},
	}

	manager, _ := NewInterceptManager(cfg, logger)

	config := manager.GetDomainConfig()
	if len(config) != 2 {
		t.Errorf("Expected 2 domain configs, got %d", len(config))
	}

	if config["domain1.com"] != true {
		t.Error("Expected domain1.com to be true")
	}

	if config["domain2.com"] != false {
		t.Error("Expected domain2.com to be false")
	}
}

func TestInterceptManager_GetIPConfig(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
		IPConfig: map[string]bool{
			"10.0.0.1":      true,
			"192.168.1.100": false,
		},
	}

	manager, _ := NewInterceptManager(cfg, logger)

	config := manager.GetIPConfig()
	if len(config) != 2 {
		t.Errorf("Expected 2 IP configs, got %d", len(config))
	}
}

func TestInterceptManager_GetTLSConfig(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	tlsConfig := manager.GetTLSConfig()
	if tlsConfig == nil {
		t.Fatal("GetTLSConfig returned nil")
	}

	if tlsConfig.GetCertificate == nil {
		t.Error("Expected GetCertificate callback to be set")
	}
}

func TestInterceptManager_RecordStats(t *testing.T) {
	certPath, keyPath, cleanup := generateTestCA(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, _ := NewInterceptManager(cfg, logger)

	manager.RecordIntercepted()
	manager.RecordIntercepted()
	manager.RecordPassthrough()

	stats := manager.GetStats()
	if stats["intercepted_conns"] != 2 {
		t.Errorf("Expected 2 intercepted conns, got %d", stats["intercepted_conns"])
	}
	if stats["passthrough_conns"] != 1 {
		t.Errorf("Expected 1 passthrough conn, got %d", stats["passthrough_conns"])
	}
}
