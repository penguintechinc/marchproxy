//go:build ci

package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"
)

// generateTestCert creates a test X.509 certificate for testing.
func generateTestCert(t *testing.T, cn string, ou []string, notBefore, notAfter time.Time) (*x509.Certificate, *rsa.PrivateKey, []byte) {
	t.Helper()

	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:         cn,
			OrganizationalUnit: ou,
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, priv, certDER
}

func TestNewMTLSAuthenticatorDisabled(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, err := NewMTLSAuthenticator(config)
	if err != nil {
		t.Fatalf("failed to create authenticator: %v", err)
	}

	if auth == nil {
		t.Fatal("expected non-nil authenticator")
	}
	if auth.config.Enabled {
		t.Error("expected disabled config")
	}
	if auth.initialized {
		t.Error("expected uninitialized state for disabled config")
	}
}

func TestNewMTLSAuthenticatorMissingPaths(t *testing.T) {
	config := MTLSConfig{
		Enabled: true,
	}

	_, err := NewMTLSAuthenticator(config)
	if err == nil {
		t.Fatal("expected error for missing certificate paths")
	}
}

func TestGetTLSConfigDisabled(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)
	tlsConfig := auth.GetTLSConfig()

	if tlsConfig != nil {
		t.Error("expected nil tls.Config for disabled authenticator")
	}
}

func TestGetMetrics(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)

	// Record some metrics
	auth.metrics.recordSuccess()
	auth.metrics.recordFailure()
	auth.metrics.recordExpiredCert()

	snapshot := auth.GetMetrics()

	if snapshot.SuccessfulAuths != 1 {
		t.Errorf("expected 1 successful auth, got %d", snapshot.SuccessfulAuths)
	}
	if snapshot.FailedAuths != 2 {
		t.Errorf("expected 2 failed auths, got %d", snapshot.FailedAuths)
	}
	if snapshot.ExpiredCerts != 1 {
		t.Errorf("expected 1 expired cert, got %d", snapshot.ExpiredCerts)
	}
}

func TestExtractCertInfo(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "test.example.com", []string{"engineering"}, now, now.Add(365*24*time.Hour))

	info := auth.extractCertInfo(cert)

	if info.CommonName != "test.example.com" {
		t.Errorf("expected CN 'test.example.com', got '%s'", info.CommonName)
	}
	if len(info.OrganizationalUnit) != 1 || info.OrganizationalUnit[0] != "engineering" {
		t.Errorf("expected OU 'engineering', got %v", info.OrganizationalUnit)
	}
	if info.IsExpired {
		t.Error("expected certificate to not be expired")
	}
	if !info.IsCA && cert.IsCA {
		t.Error("expected IsCA to match certificate")
	}
}

func TestExtractCertInfoExpired(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "expired.example.com", []string{}, now.Add(-48*time.Hour), now.Add(-24*time.Hour))

	info := auth.extractCertInfo(cert)

	if !info.IsExpired {
		t.Error("expected certificate to be expired")
	}
}

func TestValidateClientCertificateExpired(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "test", []string{}, now.Add(-48*time.Hour), now.Add(-24*time.Hour))

	err := auth.validateClientCertificate(cert, nil)
	if err == nil {
		t.Fatal("expected error for expired certificate")
	}
}

func TestValidateClientCertificateNotYetValid(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "test", []string{}, now.Add(24*time.Hour), now.Add(48*time.Hour))

	err := auth.validateClientCertificate(cert, nil)
	if err == nil {
		t.Fatal("expected error for not-yet-valid certificate")
	}
}

func TestValidateClientCertificateCNAllowed(t *testing.T) {
	config := MTLSConfig{
		Enabled:       false,
		AllowedCNs:    []string{"allowed.example.com"},
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "allowed.example.com", []string{}, now, now.Add(365*24*time.Hour))

	err := auth.validateClientCertificate(cert, nil)
	if err != nil {
		t.Errorf("expected no error for allowed CN, got: %v", err)
	}
}

func TestValidateClientCertificateCNDenied(t *testing.T) {
	config := MTLSConfig{
		Enabled:    false,
		AllowedCNs: []string{"allowed.example.com"},
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "denied.example.com", []string{}, now, now.Add(365*24*time.Hour))

	err := auth.validateClientCertificate(cert, nil)
	if err == nil {
		t.Fatal("expected error for denied CN")
	}
}

func TestValidateClientCertificateOUAllowed(t *testing.T) {
	config := MTLSConfig{
		Enabled:     false,
		AllowedOUs:  []string{"engineering"},
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "test.example.com", []string{"engineering"}, now, now.Add(365*24*time.Hour))

	err := auth.validateClientCertificate(cert, nil)
	if err != nil {
		t.Errorf("expected no error for allowed OU, got: %v", err)
	}
}

func TestValidateClientCertificateOUDenied(t *testing.T) {
	config := MTLSConfig{
		Enabled:    false,
		AllowedOUs: []string{"engineering"},
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "test.example.com", []string{"sales"}, now, now.Add(365*24*time.Hour))

	err := auth.validateClientCertificate(cert, nil)
	if err == nil {
		t.Fatal("expected error for denied OU")
	}
}

func TestValidateClientCertificateCustomValidator(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
		CustomVerifyFunc: func(cert *x509.Certificate) error {
			return &x509.UnhandledCriticalExtension{}
		},
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, _ := generateTestCert(t, "test", []string{}, now, now.Add(365*24*time.Hour))

	err := auth.validateClientCertificate(cert, nil)
	if err == nil {
		t.Error("expected custom validation error")
	}
	// Error should be wrapped with "custom certificate validation failed"
	if err != nil && !strings.Contains(err.Error(), "custom") {
		t.Errorf("expected 'custom' in error message, got: %v", err)
	}
}

func TestVerifyClientCertificateNoCerts(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)

	err := auth.verifyClientCertificate([][]byte{}, [][]*x509.Certificate{})
	if err == nil {
		t.Fatal("expected error for no certificates")
	}
}

func TestVerifyClientCertificateChainTooLong(t *testing.T) {
	config := MTLSConfig{
		Enabled:           false,
		MaxCertChainDepth: 1,
	}

	auth, _ := NewMTLSAuthenticator(config)

	now := time.Now()
	cert, _, certDER := generateTestCert(t, "test", []string{}, now, now.Add(365*24*time.Hour))

	longChain := [][]*x509.Certificate{
		{cert, cert, cert}, // Chain of 3
	}

	err := auth.verifyClientCertificate([][]byte{certDER}, longChain)
	if err == nil {
		t.Fatal("expected error for chain too long")
	}
}

func TestMetricsRecordSuccess(t *testing.T) {
	metrics := &MTLSMetrics{}

	metrics.recordSuccess()
	metrics.recordSuccess()

	if metrics.SuccessfulAuths != 2 {
		t.Errorf("expected 2 successful auths, got %d", metrics.SuccessfulAuths)
	}
	if metrics.FailedAuths != 0 {
		t.Errorf("expected 0 failed auths, got %d", metrics.FailedAuths)
	}
}

func TestMetricsRecordFailure(t *testing.T) {
	metrics := &MTLSMetrics{}

	metrics.recordFailure()
	metrics.recordFailure()

	if metrics.FailedAuths != 2 {
		t.Errorf("expected 2 failed auths, got %d", metrics.FailedAuths)
	}
}

func TestMetricsRecordExpiredCert(t *testing.T) {
	metrics := &MTLSMetrics{}

	metrics.recordExpiredCert()

	if metrics.ExpiredCerts != 1 {
		t.Errorf("expected 1 expired cert, got %d", metrics.ExpiredCerts)
	}
	if metrics.FailedAuths != 1 {
		t.Errorf("expected 1 failed auth, got %d", metrics.FailedAuths)
	}
}

func TestMetricsRecordInvalidCert(t *testing.T) {
	metrics := &MTLSMetrics{}

	metrics.recordInvalidCert()

	if metrics.InvalidCerts != 1 {
		t.Errorf("expected 1 invalid cert, got %d", metrics.InvalidCerts)
	}
	if metrics.FailedAuths != 1 {
		t.Errorf("expected 1 failed auth, got %d", metrics.FailedAuths)
	}
}

func TestMetricsRecordClientCertMissing(t *testing.T) {
	metrics := &MTLSMetrics{}

	metrics.recordClientCertMissing()

	if metrics.ClientCertMissing != 1 {
		t.Errorf("expected 1 missing cert, got %d", metrics.ClientCertMissing)
	}
	if metrics.FailedAuths != 1 {
		t.Errorf("expected 1 failed auth, got %d", metrics.FailedAuths)
	}
}

func TestMetricsRecordCertChainTooLong(t *testing.T) {
	metrics := &MTLSMetrics{}

	metrics.recordCertChainTooLong()

	if metrics.CertChainTooLong != 1 {
		t.Errorf("expected 1 chain too long, got %d", metrics.CertChainTooLong)
	}
	if metrics.FailedAuths != 1 {
		t.Errorf("expected 1 failed auth, got %d", metrics.FailedAuths)
	}
}

func TestMetricsRecordCustomValidationError(t *testing.T) {
	metrics := &MTLSMetrics{}

	metrics.recordCustomValidationError()

	if metrics.CustomValidationErr != 1 {
		t.Errorf("expected 1 custom validation error, got %d", metrics.CustomValidationErr)
	}
	if metrics.FailedAuths != 1 {
		t.Errorf("expected 1 failed auth, got %d", metrics.FailedAuths)
	}
}

func TestAuthenticateRequestDisabled(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)

	req := &http.Request{}
	info, err := auth.AuthenticateRequest(req)

	if err != nil {
		t.Errorf("expected no error for disabled auth, got: %v", err)
	}
	if info != nil {
		t.Error("expected nil cert info for disabled auth")
	}
}

func TestAuthenticateRequestNoTLS(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
		RequireClientCert: true,
	}

	auth, _ := NewMTLSAuthenticator(config)

	req := &http.Request{
		TLS: nil,
	}

	// With disabled config, should return nil error
	_, err := auth.AuthenticateRequest(req)
	if err != nil {
		t.Errorf("expected no error for disabled config, got: %v", err)
	}
}

func TestAuthenticateRequestNoCertificate(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
		RequireClientCert: true,
	}

	auth, _ := NewMTLSAuthenticator(config)

	req := &http.Request{
		TLS: &tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{},
		},
	}

	_, err := auth.AuthenticateRequest(req)
	// Disabled config returns nil
	if err != nil {
		t.Errorf("expected no error for disabled config, got: %v", err)
	}
}

func TestCertExpiredGracePeriod(t *testing.T) {
	config := MTLSConfig{
		Enabled:           false,
		CertExpiredGrace:  1 * time.Hour,
	}

	auth, _ := NewMTLSAuthenticator(config)

	// Create cert expired 30 minutes ago
	now := time.Now()
	cert, _, _ := generateTestCert(t, "test", []string{}, now.Add(-48*time.Hour), now.Add(-30*time.Minute))

	// Should not error due to grace period
	err := auth.validateClientCertificate(cert, nil)
	if err != nil {
		t.Errorf("expected no error within grace period, got: %v", err)
	}
}

func TestCertExpiredBeyondGracePeriod(t *testing.T) {
	config := MTLSConfig{
		Enabled:           false,
		CertExpiredGrace:  10 * time.Minute,
	}

	auth, _ := NewMTLSAuthenticator(config)

	// Create cert expired 30 minutes ago
	now := time.Now()
	cert, _, _ := generateTestCert(t, "test", []string{}, now.Add(-48*time.Hour), now.Add(-30*time.Minute))

	err := auth.validateClientCertificate(cert, nil)
	if err == nil {
		t.Fatal("expected error for expired cert beyond grace period")
	}
}

func TestUpdateMetricsAverageLatency(t *testing.T) {
	config := MTLSConfig{
		Enabled: false,
	}

	auth, _ := NewMTLSAuthenticator(config)

	start := time.Now()
	time.Sleep(10 * time.Millisecond)
	auth.updateMetrics(start)

	snapshot := auth.GetMetrics()
	if snapshot.AverageLatency == 0 {
		t.Error("expected non-zero average latency")
	}
}
