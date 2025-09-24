// Package tls provides mTLS functionality for the proxy
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"marchproxy-egress/internal/config"
)

// MTLSManager handles mTLS configuration and certificate management
type MTLSManager struct {
	config    *config.Config
	tlsConfig *tls.Config
}

// NewMTLSManager creates a new mTLS manager with the given configuration
func NewMTLSManager(cfg *config.Config) (*MTLSManager, error) {
	manager := &MTLSManager{
		config: cfg,
	}

	if cfg.IsMTLSEnabled() {
		tlsConfig, err := manager.setupMTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to setup mTLS config: %w", err)
		}
		manager.tlsConfig = tlsConfig
	}

	return manager, nil
}

// setupMTLSConfig creates the TLS configuration for mTLS
func (m *MTLSManager) setupMTLSConfig() (*tls.Config, error) {
	serverCert, serverKey, clientCA := m.config.GetMTLSConfig()

	// Load server certificate and key
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate and key: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	// Configure client certificate validation if required
	if m.config.RequiresClientCert() {
		// Load client CA certificate
		clientCAData, err := ioutil.ReadFile(clientCA)
		if err != nil {
			return nil, fmt.Errorf("failed to read client CA certificate: %w", err)
		}

		clientCertPool := x509.NewCertPool()
		if !clientCertPool.AppendCertsFromPEM(clientCAData) {
			return nil, fmt.Errorf("failed to parse client CA certificate")
		}

		tlsConfig.ClientCAs = clientCertPool

		if m.config.ShouldVerifyClientCert() {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConfig.ClientAuth = tls.RequireAnyClientCert
		}
	}

	return tlsConfig, nil
}

// GetTLSConfig returns the configured TLS config
func (m *MTLSManager) GetTLSConfig() *tls.Config {
	return m.tlsConfig
}

// WrapHandler wraps an HTTP handler with mTLS middleware
func (m *MTLSManager) WrapHandler(handler http.Handler) http.Handler {
	if !m.config.IsMTLSEnabled() {
		return handler
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if connection is using TLS
		if r.TLS == nil {
			http.Error(w, "mTLS required", http.StatusUpgradeRequired)
			return
		}

		// Verify client certificate if required
		if m.config.RequiresClientCert() {
			if len(r.TLS.PeerCertificates) == 0 {
				http.Error(w, "Client certificate required", http.StatusUnauthorized)
				return
			}

			// Log client certificate info for debugging
			clientCert := r.TLS.PeerCertificates[0]
			log.Printf("mTLS: Client certificate CN=%s, Serial=%s", clientCert.Subject.CommonName, clientCert.SerialNumber)
		}

		// Set security headers
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		handler.ServeHTTP(w, r)
	})
}

// CreateHTTPClient creates an HTTP client configured for mTLS
func (m *MTLSManager) CreateHTTPClient() (*http.Client, error) {
	if !m.config.IsMTLSEnabled() {
		return &http.Client{}, nil
	}

	// Load client certificate if configured
	var clientCerts []tls.Certificate
	if m.config.MTLSClientCertPath != "" && m.config.MTLSClientKeyPath != "" {
		clientCert, err := tls.LoadX509KeyPair(m.config.MTLSClientCertPath, m.config.MTLSClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		clientCerts = append(clientCerts, clientCert)
	}

	// Create TLS config for client
	clientTLSConfig := &tls.Config{
		Certificates: clientCerts,
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}

	// Load server CA if configured for verification
	if m.config.MTLSClientCAPath != "" {
		serverCAData, err := ioutil.ReadFile(m.config.MTLSClientCAPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read server CA certificate: %w", err)
		}

		serverCertPool := x509.NewCertPool()
		if !serverCertPool.AppendCertsFromPEM(serverCAData) {
			return nil, fmt.Errorf("failed to parse server CA certificate")
		}
		clientTLSConfig.RootCAs = serverCertPool
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: clientTLSConfig,
		},
	}, nil
}

// ValidateConfiguration validates the mTLS configuration
func (m *MTLSManager) ValidateConfiguration() error {
	if !m.config.IsMTLSEnabled() {
		return nil
	}

	serverCert, serverKey, clientCA := m.config.GetMTLSConfig()

	// Validate server certificate and key
	_, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return fmt.Errorf("invalid server certificate or key: %w", err)
	}

	// Validate client CA if client certificates are required
	if m.config.RequiresClientCert() {
		clientCAData, err := ioutil.ReadFile(clientCA)
		if err != nil {
			return fmt.Errorf("failed to read client CA certificate: %w", err)
		}

		clientCertPool := x509.NewCertPool()
		if !clientCertPool.AppendCertsFromPEM(clientCAData) {
			return fmt.Errorf("failed to parse client CA certificate")
		}
	}

	log.Printf("mTLS configuration validated successfully")
	return nil
}

// GetCertificateInfo returns information about the loaded certificates
func (m *MTLSManager) GetCertificateInfo() map[string]interface{} {
	info := make(map[string]interface{})

	if !m.config.IsMTLSEnabled() {
		info["enabled"] = false
		return info
	}

	info["enabled"] = true
	info["require_client_cert"] = m.config.RequiresClientCert()
	info["verify_client_cert"] = m.config.ShouldVerifyClientCert()

	// Get server certificate info
	if m.tlsConfig != nil && len(m.tlsConfig.Certificates) > 0 {
		serverCert := m.tlsConfig.Certificates[0]
		if len(serverCert.Certificate) > 0 {
			cert, err := x509.ParseCertificate(serverCert.Certificate[0])
			if err == nil {
				info["server_cert"] = map[string]interface{}{
					"subject":    cert.Subject.String(),
					"issuer":     cert.Issuer.String(),
					"not_before": cert.NotBefore,
					"not_after":  cert.NotAfter,
					"serial":     cert.SerialNumber.String(),
				}
			}
		}
	}

	return info
}