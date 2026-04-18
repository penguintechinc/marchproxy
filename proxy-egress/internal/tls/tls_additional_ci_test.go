//go:build ci

package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Helper to create test certificates
func generateTestCert(t *testing.T, tempDir string, name string, isCA bool) (certPath, keyPath string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   name,
			Organization: []string{"Test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
		template.DNSNames = []string{name, "*.example.com"}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	certPath = filepath.Join(tempDir, name+".crt")
	certFile, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	keyPath = filepath.Join(tempDir, name+".key")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	keyFile.Close()

	return certPath, keyPath
}

// TestInterceptManagerNewWithoutCA tests creation without CA
func TestInterceptManagerNewWithoutCA(t *testing.T) {
	cfg := InterceptConfig{
		Enabled: false,
		Mode:    InterceptModePreconfigured,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
}

// TestInterceptManagerMITMMode tests MITM mode initialization
func TestInterceptManagerMITMMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 100,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager MITM failed: %v", err)
	}

	if !manager.IsEnabled() {
		t.Error("expected manager to be enabled")
	}
	if manager.GetMode() != InterceptModeMITM {
		t.Errorf("expected mode MITM, got %s", manager.GetMode())
	}
}

// TestShouldInterceptDomainConfig tests domain-specific interception
func TestShouldInterceptDomainConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:      true,
		Mode:         InterceptModeMITM,
		CACertPath:   certPath,
		CAKeyPath:    keyPath,
		DomainConfig: map[string]bool{
			"blocked.com":    false,
			"allowed.com":    true,
		},
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	// Blocked domain should not intercept
	if manager.ShouldIntercept("blocked.com", "") {
		t.Error("expected blocked.com to not be intercepted")
	}

	// Allowed domain should intercept
	if !manager.ShouldIntercept("allowed.com", "") {
		t.Error("expected allowed.com to be intercepted")
	}

	// Unknown domain should use default (true)
	if !manager.ShouldIntercept("unknown.com", "") {
		t.Error("expected unknown domain to default to interception")
	}
}

// TestShouldInterceptIPConfig tests IP-specific interception
func TestShouldInterceptIPConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
		IPConfig: map[string]bool{
			"192.168.1.1": false,
			"10.0.0.1":    true,
		},
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	// Blocked IP
	if manager.ShouldIntercept("example.com", "192.168.1.1") {
		t.Error("expected 192.168.1.1 to not be intercepted")
	}

	// Allowed IP
	if !manager.ShouldIntercept("example.com", "10.0.0.1") {
		t.Error("expected 10.0.0.1 to be intercepted")
	}
}

// TestGetCertificateCacheMiss tests certificate generation on cache miss
func TestGetCertificateCacheMiss(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 100,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	cert, err := manager.GetCertificate("test.example.com")
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if cert == nil {
		t.Fatal("expected non-nil certificate")
	}

	stats := manager.GetStats()
	if stats["certs_generated"] != 1 {
		t.Errorf("expected 1 cert generated, got %d", stats["certs_generated"])
	}
	if stats["cache_misses"] != 1 {
		t.Errorf("expected 1 cache miss, got %d", stats["cache_misses"])
	}
}

// TestGetCertificateCacheHit tests certificate caching
func TestGetCertificateCacheHit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 100,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	// First call - cache miss
	cert1, _ := manager.GetCertificate("test.example.com")

	// Second call - cache hit
	cert2, _ := manager.GetCertificate("test.example.com")

	stats := manager.GetStats()
	if stats["cache_hits"] != 1 {
		t.Errorf("expected 1 cache hit, got %d", stats["cache_hits"])
	}
	if stats["cache_misses"] != 1 {
		t.Errorf("expected 1 cache miss, got %d", stats["cache_misses"])
	}

	// Both certs should be the same instance
	if cert1 != cert2 {
		t.Error("expected cached cert to be same instance")
	}
}

// TestAddPreconfiguredCert tests adding pre-configured certificates
func TestAddPreconfiguredCert(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModePreconfigured,
		CertCacheSize: 100,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	// Create a preconfigured cert
	precertPath, prekeyPath := generateTestCert(t, tempDir, "preconfig", false)
	cert, _ := tls.LoadX509KeyPair(precertPath, prekeyPath)

	manager.AddPreconfiguredCert("api.example.com", &cert)

	retrieved, err := manager.GetCertificate("api.example.com")
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil certificate")
	}
}

// TestLoadPreconfiguredCert tests loading pre-configured certificates from files
func TestLoadPreconfiguredCert(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModePreconfigured,
		CertCacheSize: 100,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	precertPath, prekeyPath := generateTestCert(t, tempDir, "preconfig", false)

	err = manager.LoadPreconfiguredCert("loaded.example.com", precertPath, prekeyPath)
	if err != nil {
		t.Fatalf("LoadPreconfiguredCert failed: %v", err)
	}

	retrieved, err := manager.GetCertificate("loaded.example.com")
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil certificate")
	}
}

// TestSetDomainIntercept tests runtime domain configuration
func TestSetDomainIntercept(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}
	_ = keyPath  // Use keyPath to satisfy compiler

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	// Initially allows
	if !manager.ShouldIntercept("test.com", "") {
		t.Error("expected test.com to be intercepted initially")
	}

	// Disable it
	manager.SetDomainIntercept("test.com", false)
	if manager.ShouldIntercept("test.com", "") {
		t.Error("expected test.com to not be intercepted after SetDomainIntercept(false)")
	}

	// Re-enable
	manager.SetDomainIntercept("test.com", true)
	if !manager.ShouldIntercept("test.com", "") {
		t.Error("expected test.com to be intercepted after SetDomainIntercept(true)")
	}
}

// TestSetIPIntercept tests runtime IP configuration
func TestSetIPIntercept(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}
	_ = certPath  // Use certPath to satisfy compiler

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	manager.SetIPIntercept("192.168.1.100", false)
	if manager.ShouldIntercept("any.com", "192.168.1.100") {
		t.Error("expected IP to not be intercepted after SetIPIntercept(false)")
	}
}

// TestEnableDisable tests enabling/disabling interception
func TestEnableDisable(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	if !manager.IsEnabled() {
		t.Error("expected manager to be enabled")
	}

	manager.Disable()
	if manager.IsEnabled() {
		t.Error("expected manager to be disabled after Disable()")
	}
	if manager.ShouldIntercept("test.com", "") {
		t.Error("expected no interception when disabled")
	}

	manager.Enable()
	if !manager.IsEnabled() {
		t.Error("expected manager to be enabled after Enable()")
	}
}

// TestGetDomainConfig returns domain configuration
func TestGetDomainConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
		DomainConfig: map[string]bool{
			"blocked.com": false,
			"allowed.com": true,
		},
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	config := manager.GetDomainConfig()
	blockedVal, blockedExists := config["blocked.com"]
	if !blockedExists || blockedVal != false {
		t.Errorf("expected blocked.com=false in config, got: %v (exists: %v)", blockedVal, blockedExists)
	}
	allowedVal, allowedExists := config["allowed.com"]
	if !allowedExists || allowedVal != true {
		t.Errorf("expected allowed.com=true in config, got: %v (exists: %v)", allowedVal, allowedExists)
	}
}

// TestGetIPConfig returns IP configuration
func TestGetIPConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
		IPConfig: map[string]bool{
			"10.0.0.1":   true,
			"10.0.0.2":   false,
		},
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	config := manager.GetIPConfig()
	if config["10.0.0.1"] != true {
		t.Error("expected 10.0.0.1=true in config")
	}
	if config["10.0.0.2"] != false {
		t.Error("expected 10.0.0.2=false in config")
	}
}

// TestClearCache clears certificate cache
func TestClearCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 100,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	// Generate cert to populate cache
	manager.GetCertificate("test.example.com")

	statsBefore := manager.GetStats()
	if statsBefore["certs_generated"] != 1 {
		t.Errorf("expected 1 cert generated before clear, got %d", statsBefore["certs_generated"])
	}

	// Clear cache
	manager.ClearCache()

	// Get same cert again - should regenerate
	manager.GetCertificate("test.example.com")

	statsAfter := manager.GetStats()
	if statsAfter["certs_generated"] != 2 {
		t.Errorf("expected 2 certs generated after clear, got %d", statsAfter["certs_generated"])
	}
}

// TestGetTLSConfig returns TLS config for interception
func TestGetTLSConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	tlsConfig := manager.GetTLSConfig()
	if tlsConfig == nil {
		t.Fatal("expected non-nil TLS config")
	}
	if tlsConfig.GetCertificate == nil {
		t.Error("expected GetCertificate callback to be set")
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 minimum, got %d", tlsConfig.MinVersion)
	}
}

// TestRecordInterceptedAndPassthrough tests statistics recording
func TestRecordInterceptedAndPassthrough(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:    true,
		Mode:       InterceptModeMITM,
		CACertPath: certPath,
		CAKeyPath:  keyPath,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	manager.RecordIntercepted()
	manager.RecordPassthrough()
	manager.RecordIntercepted()

	stats := manager.GetStats()
	if stats["intercepted_conns"] != 2 {
		t.Errorf("expected 2 intercepted connections, got %d", stats["intercepted_conns"])
	}
	if stats["passthrough_conns"] != 1 {
		t.Errorf("expected 1 passthrough connection, got %d", stats["passthrough_conns"])
	}
}

// TestGenerateCertificateForIP tests certificate generation for IP addresses
func TestGenerateCertificateForIP(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCert(t, tempDir, "ca", true)

	cfg := InterceptConfig{
		Enabled:       true,
		Mode:          InterceptModeMITM,
		CACertPath:    certPath,
		CAKeyPath:     keyPath,
		CertCacheSize: 100,
	}

	manager, err := NewInterceptManager(cfg, nil)
	if err != nil {
		t.Fatalf("NewInterceptManager failed: %v", err)
	}

	// Generate cert for IP address
	cert, err := manager.GetCertificate("192.168.1.1")
	if err != nil {
		t.Fatalf("GetCertificate for IP failed: %v", err)
	}
	if cert == nil {
		t.Fatal("expected non-nil certificate for IP")
	}

	// Parse cert to verify IP is included
	if len(cert.Certificate) == 0 {
		t.Fatal("expected certificate bytes")
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	// Check that IP address is in the certificate
	found := false
	for _, ip := range parsed.IPAddresses {
		if ip.String() == "192.168.1.1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected IP address to be in certificate")
	}
}

