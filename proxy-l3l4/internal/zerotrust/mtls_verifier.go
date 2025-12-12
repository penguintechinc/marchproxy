package zerotrust

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MTLSVerifier handles enhanced mTLS certificate validation
type MTLSVerifier struct {
	mu            sync.RWMutex
	caCertPool    *x509.CertPool
	crlList       []*x509.RevocationList
	ocspEnabled   bool
	ocspResponder string
	logger        *logrus.Logger
	strictMode    bool
	checkExpiry   bool
}

// CertificateValidationResult contains the result of certificate validation
type CertificateValidationResult struct {
	Valid          bool                   `json:"valid"`
	Subject        string                 `json:"subject"`
	Issuer         string                 `json:"issuer"`
	SerialNumber   string                 `json:"serial_number"`
	NotBefore      time.Time              `json:"not_before"`
	NotAfter       time.Time              `json:"not_after"`
	DNSNames       []string               `json:"dns_names"`
	Errors         []string               `json:"errors,omitempty"`
	Warnings       []string               `json:"warnings,omitempty"`
	RevokedStatus  string                 `json:"revoked_status,omitempty"`
	ValidationTime time.Time              `json:"validation_time"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// NewMTLSVerifier creates a new mTLS certificate verifier
func NewMTLSVerifier(logger *logrus.Logger) *MTLSVerifier {
	return &MTLSVerifier{
		caCertPool:  x509.NewCertPool(),
		crlList:     []*x509.RevocationList{},
		logger:      logger,
		strictMode:  true,
		checkExpiry: true,
	}
}

// LoadCACertificate loads a CA certificate for verification
func (mv *MTLSVerifier) LoadCACertificate(certPEM []byte) error {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	mv.caCertPool.AddCert(cert)
	mv.logger.WithField("subject", cert.Subject.CommonName).Info("Loaded CA certificate")

	return nil
}

// LoadCRL loads a Certificate Revocation List
func (mv *MTLSVerifier) LoadCRL(crlPEM []byte) error {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	block, _ := pem.Decode(crlPEM)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	crl, err := x509.ParseRevocationList(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CRL: %w", err)
	}

	mv.crlList = append(mv.crlList, crl)
	mv.logger.WithField("issuer", crl.Issuer.CommonName).Info("Loaded CRL")

	return nil
}

// EnableOCSP enables OCSP checking with specified responder
func (mv *MTLSVerifier) EnableOCSP(responderURL string) {
	mv.mu.Lock()
	defer mv.mu.Unlock()

	mv.ocspEnabled = true
	mv.ocspResponder = responderURL
	mv.logger.WithField("responder", responderURL).Info("Enabled OCSP checking")
}

// VerifyCertificate performs comprehensive certificate validation
func (mv *MTLSVerifier) VerifyCertificate(certPEM []byte, intermediates [][]byte) (*CertificateValidationResult, error) {
	result := &CertificateValidationResult{
		ValidationTime: time.Now(),
		Errors:         []string{},
		Warnings:       []string{},
		Metadata:       make(map[string]interface{}),
	}

	// Parse the certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Extract basic information
	result.Subject = cert.Subject.CommonName
	result.Issuer = cert.Issuer.CommonName
	result.SerialNumber = cert.SerialNumber.String()
	result.NotBefore = cert.NotBefore
	result.NotAfter = cert.NotAfter
	result.DNSNames = cert.DNSNames

	// Check expiry
	if mv.checkExpiry {
		now := time.Now()
		if now.Before(cert.NotBefore) {
			result.Errors = append(result.Errors, "certificate not yet valid")
		}
		if now.After(cert.NotAfter) {
			result.Errors = append(result.Errors, "certificate expired")
		}

		// Warn if expiring soon (within 30 days)
		if now.Add(30 * 24 * time.Hour).After(cert.NotAfter) {
			result.Warnings = append(result.Warnings, "certificate expiring soon")
		}
	}

	// Build intermediate pool
	intermediatePool := x509.NewCertPool()
	for _, intermPEM := range intermediates {
		block, _ := pem.Decode(intermPEM)
		if block != nil {
			intermCert, err := x509.ParseCertificate(block.Bytes)
			if err == nil {
				intermediatePool.AddCert(intermCert)
			}
		}
	}

	// Verify certificate chain
	mv.mu.RLock()
	caCertPool := mv.caCertPool
	mv.mu.RUnlock()

	opts := x509.VerifyOptions{
		Roots:         caCertPool,
		Intermediates: intermediatePool,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	chains, err := cert.Verify(opts)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("chain verification failed: %v", err))
	} else {
		result.Metadata["chain_length"] = len(chains[0])
	}

	// Check CRL
	if revoked, reason := mv.checkCRL(cert); revoked {
		result.Errors = append(result.Errors, "certificate revoked")
		result.RevokedStatus = reason
	}

	// Check OCSP if enabled
	if mv.ocspEnabled {
		if revoked, reason := mv.checkOCSP(cert); revoked {
			result.Errors = append(result.Errors, "certificate revoked (OCSP)")
			result.RevokedStatus = reason
		}
	}

	// Determine overall validity
	result.Valid = len(result.Errors) == 0

	if !result.Valid && mv.strictMode {
		return result, fmt.Errorf("certificate validation failed: %v", result.Errors)
	}

	return result, nil
}

// checkCRL checks if certificate is revoked using CRL
func (mv *MTLSVerifier) checkCRL(cert *x509.Certificate) (bool, string) {
	mv.mu.RLock()
	defer mv.mu.RUnlock()

	for _, crl := range mv.crlList {
		// Check if CRL is from the certificate's issuer
		if crl.Issuer.CommonName != cert.Issuer.CommonName {
			continue
		}

		// Check if CRL is still valid
		if time.Now().After(crl.NextUpdate) {
			mv.logger.Warn("CRL expired, skipping check")
			continue
		}

		// Check revoked certificates
		for _, revokedCert := range crl.RevokedCertificateEntries {
			if revokedCert.SerialNumber.Cmp(cert.SerialNumber) == 0 {
				reason := "unspecified"
				if revokedCert.ReasonCode != 0 {
					reason = fmt.Sprintf("reason code: %d", revokedCert.ReasonCode)
				}
				return true, reason
			}
		}
	}

	return false, ""
}

// checkOCSP checks certificate revocation status using OCSP
func (mv *MTLSVerifier) checkOCSP(cert *x509.Certificate) (bool, string) {
	// Simple OCSP check implementation
	// In production, use crypto/ocsp package for full implementation

	if mv.ocspResponder == "" {
		return false, ""
	}

	// Create OCSP request
	// Note: This is a simplified implementation
	// Full implementation would use crypto/ocsp.CreateRequest and ParseResponse

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(mv.ocspResponder)
	if err != nil {
		mv.logger.WithError(err).Warn("OCSP check failed")
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mv.logger.WithField("status", resp.StatusCode).Warn("OCSP responder error")
		return false, ""
	}

	// Parse OCSP response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		mv.logger.WithError(err).Warn("Failed to read OCSP response")
		return false, ""
	}

	// Note: Full implementation would parse the OCSP response here
	// For now, assume certificate is not revoked
	_ = body

	return false, ""
}

// SetStrictMode enables or disables strict mode
func (mv *MTLSVerifier) SetStrictMode(strict bool) {
	mv.mu.Lock()
	defer mv.mu.Unlock()
	mv.strictMode = strict
}

// SetCheckExpiry enables or disables expiry checking
func (mv *MTLSVerifier) SetCheckExpiry(check bool) {
	mv.mu.Lock()
	defer mv.mu.Unlock()
	mv.checkExpiry = check
}

// GetCertificateInfo extracts certificate information for audit logging
func GetCertificateInfo(cert *x509.Certificate) *CertificateInfo {
	if cert == nil {
		return nil
	}

	return &CertificateInfo{
		Subject:      cert.Subject.CommonName,
		Issuer:       cert.Issuer.CommonName,
		SerialNumber: cert.SerialNumber.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		DNSNames:     cert.DNSNames,
	}
}
