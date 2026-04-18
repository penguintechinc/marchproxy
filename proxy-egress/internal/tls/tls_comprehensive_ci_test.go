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

	"marchproxy-egress/internal/config"
)

// generateTestCertAndKey generates test certificate and key files
func generateTestCertAndKey(t *testing.T, tempDir string, name string) (certPath, keyPath string) {
	t.Helper()

	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Create certificate template
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   name,
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Write certificate
	certPath = filepath.Join(tempDir, name+".crt")
	certFile, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	// Write key
	keyPath = filepath.Join(tempDir, name+".key")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyFile.Close()

	return certPath, keyPath
}

// TestTLSManager_NewTLSManager tests TLS manager creation
func TestTLSManager_NewTLSManager(t *testing.T) {
	config := TLSConfig{
		EnableTLS: true,
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}

	tm, err := NewTLSManager(config)
	if err != nil {
		t.Fatalf("NewTLSManager failed: %v", err)
	}
	if tm == nil {
		t.Fatal("Expected TLS manager, got nil")
	}
}

// TestTLSManager_GetTLSConfig tests TLS config generation
func TestTLSManager_GetTLSConfig(t *testing.T) {
	config := TLSConfig{
		EnableTLS: true,
	}

	tm, _ := NewTLSManager(config)
	tlsConfig := tm.GetTLSConfig()

	if tlsConfig == nil {
		t.Fatal("Expected TLS config, got nil")
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion: got %d, want %d", tlsConfig.MinVersion, tls.VersionTLS12)
	}
	if tlsConfig.MaxVersion != tls.VersionTLS13 {
		t.Errorf("MaxVersion: got %d, want %d", tlsConfig.MaxVersion, tls.VersionTLS13)
	}
}

// TestTLSManager_LoadCertificates tests certificate loading
func TestTLSManager_LoadCertificates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-cert-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCertAndKey(t, tempDir, "server")

	config := TLSConfig{
		EnableTLS: true,
		CertFile:  certPath,
		KeyFile:   keyPath,
	}

	tm, err := NewTLSManager(config)
	if err != nil {
		t.Fatalf("NewTLSManager failed: %v", err)
	}
	if tm == nil {
		t.Fatal("Expected TLS manager, got nil")
	}
}

// TestTLSManager_LoadMultipleCertificates tests loading multiple certificates
func TestTLSManager_LoadMultipleCertificates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tls-multi-cert-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cert1Path, key1Path := generateTestCertAndKey(t, tempDir, "cert1")
	cert2Path, key2Path := generateTestCertAndKey(t, tempDir, "cert2")

	config := TLSConfig{
		EnableTLS: true,
		Certificates: map[string]CertificateConfig{
			"cert1": {
				CertFile: cert1Path,
				KeyFile:  key1Path,
				Domains:  []string{"example1.com", "www.example1.com"},
			},
			"cert2": {
				CertFile: cert2Path,
				KeyFile:  key2Path,
				Domains:  []string{"example2.com"},
			},
		},
	}

	tm, err := NewTLSManager(config)
	if err != nil {
		t.Fatalf("NewTLSManager failed: %v", err)
	}
	if tm == nil {
		t.Fatal("Expected TLS manager, got nil")
	}
}

// TestTLSManager_InvalidCertificate tests error handling for invalid certificates
func TestTLSManager_InvalidCertificate(t *testing.T) {
	config := TLSConfig{
		EnableTLS: true,
		CertFile:  "/nonexistent/cert.pem",
		KeyFile:   "/nonexistent/key.pem",
	}

	_, err := NewTLSManager(config)
	if err == nil {
		t.Error("Expected error for invalid certificate")
	}
}

// TestTLSManager_DefaultCipherSuites tests default cipher suite configuration
func TestTLSManager_DefaultCipherSuites(t *testing.T) {
	config := TLSConfig{}

	tm, _ := NewTLSManager(config)
	tlsConfig := tm.GetTLSConfig()

	if len(tlsConfig.CipherSuites) == 0 {
		t.Error("Expected cipher suites to be configured")
	}
}

// TestTLSManager_CustomCipherSuites tests custom cipher suite configuration
func TestTLSManager_CustomCipherSuites(t *testing.T) {
	customSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	}

	config := TLSConfig{
		CipherSuites: customSuites,
	}

	tm, _ := NewTLSManager(config)
	tlsConfig := tm.GetTLSConfig()

	if len(tlsConfig.CipherSuites) != len(customSuites) {
		t.Errorf("CipherSuites: got %d, want %d", len(tlsConfig.CipherSuites), len(customSuites))
	}
}

// TestTLSManager_SessionCache tests session cache configuration
func TestTLSManager_SessionCache(t *testing.T) {
	config := TLSConfig{
		SessionCacheSize: 1000,
	}

	tm, _ := NewTLSManager(config)
	if tm.sessionCache == nil {
		t.Error("Expected session cache to be initialized")
	}
}

// TestOCSPCache_GetSet tests OCSP cache operations
func TestOCSPCache_GetSet(t *testing.T) {
	cache := NewOCSPCache(24 * time.Hour)
	defer cache.cleanup()

	response := &OCSPResponse{
		Response:   []byte("test"),
		NextUpdate: time.Now().Add(24 * time.Hour),
		Status:     0,
	}

	serial := "12345"
	cache.Set(serial, response)

	retrieved := cache.Get(serial)
	if retrieved == nil {
		t.Error("Expected to retrieve cached response")
	}
	if retrieved.Status != 0 {
		t.Errorf("Status: got %d, want 0", retrieved.Status)
	}
}

// TestOCSPCache_Expired tests OCSP cache expiration
func TestOCSPCache_Expired(t *testing.T) {
	cache := NewOCSPCache(1 * time.Millisecond)
	defer cache.cleanup()

	response := &OCSPResponse{
		Response:   []byte("test"),
		NextUpdate: time.Now().Add(-1 * time.Hour),
		Status:     0,
	}

	serial := "12345"
	cache.Set(serial, response)
	time.Sleep(2 * time.Millisecond)

	retrieved := cache.Get(serial)
	if retrieved != nil {
		t.Error("Expected expired response to be nil")
	}
}

// TestSessionCache_PutGet tests session cache operations
func TestSessionCache_PutGet(t *testing.T) {
	cache := NewSessionCache(10, 1*time.Hour)

	cs := &tls.ConnectionState{}
	cache.Put("session1", cs)

	sessionID, exists := cache.Get("session1")
	if !exists {
		t.Error("Expected session to exist")
	}
	if sessionID == nil {
		t.Error("Expected session ID, got nil")
	}
}

// TestSessionCache_MaxSize tests session cache size limits
func TestSessionCache_MaxSize(t *testing.T) {
	cache := NewSessionCache(3, 1*time.Hour)
	cs := &tls.ConnectionState{}

	for i := 1; i <= 5; i++ {
		cache.Put("session"+string(rune(i)), cs)
	}

	if len(cache.sessions) > 3 {
		t.Errorf("Cache size: got %d, want max 3", len(cache.sessions))
	}
}

// TestCertificateRotator_Start tests certificate rotator lifecycle
func TestCertificateRotator_Start(t *testing.T) {
	config := TLSConfig{}
	tm, _ := NewTLSManager(config)

	rotator := NewCertificateRotator(tm, 1*time.Second)
	rotator.Start()

	if !rotator.running {
		t.Error("Expected rotator to be running")
	}

	rotator.Stop()
	if rotator.running {
		t.Error("Expected rotator to be stopped")
	}
}

// TestCertificateRotator_StartIdempotent tests that starting twice doesn't cause issues
func TestCertificateRotator_StartIdempotent(t *testing.T) {
	config := TLSConfig{}
	tm, _ := NewTLSManager(config)

	rotator := NewCertificateRotator(tm, 1*time.Second)
	rotator.Start()
	rotator.Start() // Should be idempotent

	if !rotator.running {
		t.Error("Expected rotator to be running")
	}

	rotator.Stop()
}

// TestACMEManager_Initialize tests ACME manager initialization
func TestACMEManager_Initialize(t *testing.T) {
	am := NewACMEManager("https://acme.example.com", "test@example.com", []string{"example.com"})
	err := am.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
}

// TestACMEManager_Fields tests ACME manager fields
func TestACMEManager_Fields(t *testing.T) {
	am := NewACMEManager("https://acme.example.com", "test@example.com", []string{"example.com", "www.example.com"})

	if am.directory != "https://acme.example.com" {
		t.Errorf("directory: got %q, want %q", am.directory, "https://acme.example.com")
	}
	if am.email != "test@example.com" {
		t.Errorf("email: got %q, want %q", am.email, "test@example.com")
	}
	if len(am.domains) != 2 {
		t.Errorf("domains: got %d, want 2", len(am.domains))
	}
}

// TestTLSMetrics_RecordHandshake tests metrics recording
func TestTLSMetrics_RecordHandshake(t *testing.T) {
	metrics := &TLSMetrics{}
	metrics.recordHandshake()

	if metrics.Handshakes != 1 {
		t.Errorf("Handshakes: got %d, want 1", metrics.Handshakes)
	}
}

// TestTLSMetrics_RecordClientCert tests client certificate metrics
func TestTLSMetrics_RecordClientCert(t *testing.T) {
	metrics := &TLSMetrics{}
	metrics.recordClientCertProvided()
	metrics.recordClientCertValid()

	if metrics.ClientCertProvided != 1 {
		t.Errorf("ClientCertProvided: got %d, want 1", metrics.ClientCertProvided)
	}
	if metrics.ClientCertValid != 1 {
		t.Errorf("ClientCertValid: got %d, want 1", metrics.ClientCertValid)
	}
}

// TestTLSMetrics_RecordOCSP tests OCSP metrics
func TestTLSMetrics_RecordOCSP(t *testing.T) {
	metrics := &TLSMetrics{}
	metrics.recordOCSPRequest()
	metrics.recordOCSPRequest()
	metrics.recordOCSPCacheHit()

	if metrics.OCSPRequests != 2 {
		t.Errorf("OCSPRequests: got %d, want 2", metrics.OCSPRequests)
	}
	if metrics.OCSPCacheHits != 1 {
		t.Errorf("OCSPCacheHits: got %d, want 1", metrics.OCSPCacheHits)
	}
}

// TestTLSMetrics_RecordCertRotation tests cert rotation metrics
func TestTLSMetrics_RecordCertRotation(t *testing.T) {
	metrics := &TLSMetrics{}
	metrics.recordCertRotation()
	metrics.recordCertRotation()

	if metrics.CertRotations != 2 {
		t.Errorf("CertRotations: got %d, want 2", metrics.CertRotations)
	}
}

// TestTLSManager_GenerateSelfSignedCertificate tests self-signed cert generation
func TestTLSManager_GenerateSelfSignedCertificate(t *testing.T) {
	config := TLSConfig{}
	tm, _ := NewTLSManager(config)

	hosts := []string{"example.com", "www.example.com"}
	cert, err := tm.GenerateSelfSignedCertificate(hosts)

	if err != nil {
		t.Fatalf("GenerateSelfSignedCertificate failed: %v", err)
	}
	if cert == nil {
		t.Fatal("Expected certificate, got nil")
	}
	if len(cert.Certificate) == 0 {
		t.Error("Expected certificate bytes")
	}
}

// TestTLSConfig_PreferServerCiphers tests server cipher preference
func TestTLSConfig_PreferServerCiphers(t *testing.T) {
	config := TLSConfig{
		PreferServerCiphers: true,
	}

	tm, _ := NewTLSManager(config)
	tlsConfig := tm.GetTLSConfig()

	if !tlsConfig.PreferServerCipherSuites {
		t.Error("Expected PreferServerCipherSuites to be true")
	}
}

// TestTLSConfig_SessionTicketsDisabled tests session ticket configuration
func TestTLSConfig_SessionTicketsDisabled(t *testing.T) {
	config := TLSConfig{
		SessionTicketsDisabled: true,
	}

	tm, _ := NewTLSManager(config)
	tlsConfig := tm.GetTLSConfig()

	if !tlsConfig.SessionTicketsDisabled {
		t.Error("Expected SessionTicketsDisabled to be true")
	}
}

// TestTLSManager_ShouldRotateCertificate tests rotation decision logic
func TestTLSManager_ShouldRotateCertificate(t *testing.T) {
	config := TLSConfig{}
	tm, _ := NewTLSManager(config)

	// Create an expired certificate
	expiredTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "expired",
		},
		NotBefore: time.Now().Add(-48 * time.Hour),
		NotAfter:  time.Now().Add(-1 * time.Hour),
	}

	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	certDER, _ := x509.CreateCertificate(rand.Reader, expiredTemplate, expiredTemplate, &priv.PublicKey, priv)

	cert := &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}

	if !tm.shouldRotateCertificate(cert) {
		t.Error("Expected rotation to be needed for expired certificate")
	}
}

// TestMTLSManager_NewMTLSManager tests mTLS manager creation
func TestMTLSManager_NewMTLSManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "mtls-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCertAndKey(t, tempDir, "server")
	caPath, _ := generateTestCertAndKey(t, tempDir, "ca")

	cfg := &config.Config{
		EnableMTLS: true,
		MTLSServerCertPath: certPath,
		MTLSServerKeyPath: keyPath,
		MTLSClientCAPath: caPath,
		MTLSRequireClientCert: true,
	}

	mm, err := NewMTLSManager(cfg)
	if err != nil {
		t.Fatalf("NewMTLSManager failed: %v", err)
	}
	if mm == nil {
		t.Fatal("Expected mTLS manager, got nil")
	}
}

// TestMTLSManager_GetTLSConfig tests mTLS config retrieval
func TestMTLSManager_GetTLSConfig(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "mtls-config-test")
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCertAndKey(t, tempDir, "server")
	caPath, _ := generateTestCertAndKey(t, tempDir, "ca")

	cfg := &config.Config{
		EnableMTLS: true,
		MTLSServerCertPath: certPath,
		MTLSServerKeyPath: keyPath,
		MTLSClientCAPath: caPath,
	}

	mm, err := NewMTLSManager(cfg)
	if err != nil {
		t.Fatalf("NewMTLSManager failed: %v", err)
	}
	tlsConfig := mm.GetTLSConfig()

	if tlsConfig != nil && len(tlsConfig.Certificates) > 0 {
		// Successfully loaded
		return
	}
	if tlsConfig == nil {
		t.Fatal("Expected TLS config, got nil")
	}
}

// TestMTLSManager_ValidateConfiguration tests configuration validation
func TestMTLSManager_ValidateConfiguration(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "mtls-validate-test")
	defer os.RemoveAll(tempDir)

	certPath, keyPath := generateTestCertAndKey(t, tempDir, "server")

	cfg := &config.Config{
		EnableMTLS: true,
		MTLSServerCertPath: certPath,
		MTLSServerKeyPath: keyPath,
	}

	mm, _ := NewMTLSManager(cfg)
	err := mm.ValidateConfiguration()
	if err != nil {
		t.Fatalf("ValidateConfiguration failed: %v", err)
	}
}

// TestMTLSManager_ValidateConfiguration_Disabled tests validation with mTLS disabled
func TestMTLSManager_ValidateConfiguration_Disabled(t *testing.T) {
	cfg := &config.Config{
		EnableMTLS: false,
	}

	mm, err := NewMTLSManager(cfg)
	if err != nil {
		t.Fatalf("NewMTLSManager failed: %v", err)
	}
	// Validation should pass when mTLS is disabled
	err = mm.ValidateConfiguration()
	if err != nil {
		t.Fatalf("ValidateConfiguration failed: %v", err)
	}
}

// BenchmarkTLSManagerGetConfig benchmarks TLS config retrieval
func BenchmarkTLSManagerGetConfig(b *testing.B) {
	config := TLSConfig{}
	tm, _ := NewTLSManager(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.GetTLSConfig()
	}
}

// BenchmarkGenerateSelfSignedCertificate benchmarks certificate generation
func BenchmarkGenerateSelfSignedCertificate(b *testing.B) {
	config := TLSConfig{}
	tm, _ := NewTLSManager(config)
	hosts := []string{"example.com"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.GenerateSelfSignedCertificate(hosts)
	}
}
