// Package tls provides TLS interception functionality for the egress proxy
// This allows the proxy to terminate and re-originate TLS connections
// for deep packet inspection and content filtering
package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
)

// InterceptMode defines the TLS interception mode
type InterceptMode string

const (
	// InterceptModeMITM enables full man-in-the-middle with dynamic cert generation
	InterceptModeMITM InterceptMode = "mitm"
	// InterceptModePreconfigured uses pre-configured certificates for specific domains
	InterceptModePreconfigured InterceptMode = "preconfigured"
)

// InterceptConfig holds configuration for TLS interception
type InterceptConfig struct {
	Enabled       bool
	Mode          InterceptMode
	CACertPath    string
	CAKeyPath     string
	CertCacheSize int
	// Per-domain and per-IP configuration
	DomainConfig map[string]bool // domain -> intercept on/off
	IPConfig     map[string]bool // IP -> intercept on/off
}

// InterceptManager manages TLS interception
type InterceptManager struct {
	enabled      bool
	mode         InterceptMode
	caCert       *x509.Certificate
	caKey        *rsa.PrivateKey
	certCache    *lru.Cache
	domainConfig map[string]bool
	ipConfig     map[string]bool

	// Pre-configured certificates for specific domains
	preconfigCerts map[string]*tls.Certificate

	mu     sync.RWMutex
	logger *logrus.Logger

	// Statistics
	stats struct {
		CertsGenerated   int64
		CacheHits        int64
		CacheMisses      int64
		InterceptedConns int64
		PassthroughConns int64
	}
}

// NewInterceptManager creates a new TLS intercept manager
func NewInterceptManager(cfg InterceptConfig, logger *logrus.Logger) (*InterceptManager, error) {
	if logger == nil {
		logger = logrus.New()
	}

	m := &InterceptManager{
		enabled:        cfg.Enabled,
		mode:           cfg.Mode,
		domainConfig:   make(map[string]bool),
		ipConfig:       make(map[string]bool),
		preconfigCerts: make(map[string]*tls.Certificate),
		logger:         logger,
	}

	// Copy domain/IP config
	if cfg.DomainConfig != nil {
		for k, v := range cfg.DomainConfig {
			m.domainConfig[k] = v
		}
	}
	if cfg.IPConfig != nil {
		for k, v := range cfg.IPConfig {
			m.ipConfig[k] = v
		}
	}

	// Initialize certificate cache
	cacheSize := cfg.CertCacheSize
	if cacheSize == 0 {
		cacheSize = 10000
	}
	cache, err := lru.New(cacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate cache: %w", err)
	}
	m.certCache = cache

	// Load CA certificate and key for MITM mode
	if cfg.Enabled && cfg.Mode == InterceptModeMITM {
		if err := m.loadCA(cfg.CACertPath, cfg.CAKeyPath); err != nil {
			return nil, fmt.Errorf("failed to load CA: %w", err)
		}
		logger.Info("TLS interception CA loaded successfully")
	}

	return m, nil
}

// loadCA loads the CA certificate and private key for signing
func (m *InterceptManager) loadCA(certPath, keyPath string) error {
	// Load CA certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Load CA private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %w", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode CA private key PEM")
	}

	// Try PKCS1 first, then PKCS8
	caKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		keyInterface, err2 := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err2 != nil {
			return fmt.Errorf("failed to parse CA private key: PKCS1 error: %v, PKCS8 error: %v", err, err2)
		}
		var ok bool
		caKey, ok = keyInterface.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("CA private key is not RSA")
		}
	}

	m.caCert = caCert
	m.caKey = caKey

	return nil
}

// ShouldIntercept determines if TLS should be intercepted for a target
func (m *InterceptManager) ShouldIntercept(host string, ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.enabled {
		return false
	}

	// Check domain-specific config
	if intercept, ok := m.domainConfig[host]; ok {
		return intercept
	}

	// Check IP-specific config
	if ip != "" {
		if intercept, ok := m.ipConfig[ip]; ok {
			return intercept
		}
	}

	// Default: intercept if enabled
	return true
}

// GetCertificate returns a certificate for the given host
func (m *InterceptManager) GetCertificate(host string) (*tls.Certificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check cache first
	if cached, ok := m.certCache.Get(host); ok {
		m.stats.CacheHits++
		return cached.(*tls.Certificate), nil
	}

	m.stats.CacheMisses++

	// Check preconfigured certificates
	if cert, ok := m.preconfigCerts[host]; ok {
		m.certCache.Add(host, cert)
		return cert, nil
	}

	// Generate a new certificate for MITM mode
	if m.mode == InterceptModeMITM {
		cert, err := m.generateCertificate(host)
		if err != nil {
			return nil, fmt.Errorf("failed to generate certificate: %w", err)
		}
		m.certCache.Add(host, cert)
		m.stats.CertsGenerated++
		return cert, nil
	}

	return nil, fmt.Errorf("no certificate available for %s", host)
}

// generateCertificate generates a certificate signed by the CA for the given host
func (m *InterceptManager) generateCertificate(host string) (*tls.Certificate, error) {
	if m.caCert == nil || m.caKey == nil {
		return nil, fmt.Errorf("CA not loaded")
	}

	// Generate a new key for this certificate
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Determine if host is IP or domain
	var dnsNames []string
	var ipAddresses []net.IP

	if ip := net.ParseIP(host); ip != nil {
		ipAddresses = append(ipAddresses, ip)
	} else {
		dnsNames = append(dnsNames, host)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"MarchProxy TLS Interception"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	// Sign with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, m.caCert, &priv.PublicKey, m.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert := &tls.Certificate{
		Certificate: [][]byte{certDER, m.caCert.Raw},
		PrivateKey:  priv,
		Leaf:        template,
	}

	m.logger.WithField("host", host).Debug("Generated TLS interception certificate")

	return cert, nil
}

// AddPreconfiguredCert adds a pre-configured certificate for a domain
func (m *InterceptManager) AddPreconfiguredCert(domain string, cert *tls.Certificate) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preconfigCerts[domain] = cert
	m.logger.WithField("domain", domain).Debug("Added pre-configured certificate")
}

// LoadPreconfiguredCert loads a pre-configured certificate from files
func (m *InterceptManager) LoadPreconfiguredCert(domain, certPath, keyPath string) error {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate for %s: %w", domain, err)
	}

	m.AddPreconfiguredCert(domain, &cert)
	return nil
}

// SetDomainIntercept sets whether to intercept a specific domain
func (m *InterceptManager) SetDomainIntercept(domain string, intercept bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.domainConfig[domain] = intercept
	m.logger.WithFields(logrus.Fields{
		"domain":    domain,
		"intercept": intercept,
	}).Debug("Set domain intercept config")
}

// SetIPIntercept sets whether to intercept a specific IP
func (m *InterceptManager) SetIPIntercept(ip string, intercept bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ipConfig[ip] = intercept
	m.logger.WithFields(logrus.Fields{
		"ip":        ip,
		"intercept": intercept,
	}).Debug("Set IP intercept config")
}

// Enable enables TLS interception
func (m *InterceptManager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
	m.logger.Info("TLS interception enabled")
}

// Disable disables TLS interception
func (m *InterceptManager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
	m.logger.Info("TLS interception disabled")
}

// IsEnabled returns whether TLS interception is enabled
func (m *InterceptManager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// GetMode returns the current interception mode
func (m *InterceptManager) GetMode() InterceptMode {
	return m.mode
}

// GetStats returns interception statistics
func (m *InterceptManager) GetStats() map[string]int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]int64{
		"certs_generated":    m.stats.CertsGenerated,
		"cache_hits":         m.stats.CacheHits,
		"cache_misses":       m.stats.CacheMisses,
		"intercepted_conns":  m.stats.InterceptedConns,
		"passthrough_conns":  m.stats.PassthroughConns,
	}
}

// GetDomainConfig returns the current domain intercept configuration
func (m *InterceptManager) GetDomainConfig() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := make(map[string]bool)
	for k, v := range m.domainConfig {
		config[k] = v
	}
	return config
}

// GetIPConfig returns the current IP intercept configuration
func (m *InterceptManager) GetIPConfig() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := make(map[string]bool)
	for k, v := range m.ipConfig {
		config[k] = v
	}
	return config
}

// ClearCache clears the certificate cache
func (m *InterceptManager) ClearCache() {
	m.certCache.Purge()
	m.logger.Info("Certificate cache cleared")
}

// GetTLSConfig creates a TLS config that uses dynamic certificate generation
func (m *InterceptManager) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return m.GetCertificate(hello.ServerName)
		},
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
}

// RecordIntercepted records an intercepted connection
func (m *InterceptManager) RecordIntercepted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.InterceptedConns++
}

// RecordPassthrough records a passthrough connection
func (m *InterceptManager) RecordPassthrough() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.PassthroughConns++
}
