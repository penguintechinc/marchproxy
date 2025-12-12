package zerotrust

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CertRotator handles automated certificate rotation with zero-downtime
type CertRotator struct {
	mu                sync.RWMutex
	currentCert       *tls.Certificate
	nextCert          *tls.Certificate
	certPath          string
	keyPath           string
	checkInterval     time.Duration
	rotationThreshold time.Duration
	logger            *logrus.Logger
	stopCh            chan struct{}
	callbacks         []func(*tls.Certificate)
}

// RotationEvent represents a certificate rotation event
type RotationEvent struct {
	Timestamp   time.Time
	OldCert     *CertificateInfo
	NewCert     *CertificateInfo
	Reason      string
	Success     bool
	Error       string
}

// NewCertRotator creates a new certificate rotator
func NewCertRotator(certPath, keyPath string, logger *logrus.Logger) (*CertRotator, error) {
	// Load initial certificate
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial certificate: %w", err)
	}

	rotator := &CertRotator{
		currentCert:       &cert,
		certPath:          certPath,
		keyPath:           keyPath,
		checkInterval:     1 * time.Hour,
		rotationThreshold: 30 * 24 * time.Hour, // 30 days before expiry
		logger:            logger,
		stopCh:            make(chan struct{}),
		callbacks:         []func(*tls.Certificate){},
	}

	return rotator, nil
}

// Start begins the automatic certificate rotation monitoring
func (cr *CertRotator) Start() {
	go cr.monitorCertificates()
	cr.logger.Info("Certificate rotation monitor started")
}

// Stop stops the certificate rotation monitoring
func (cr *CertRotator) Stop() {
	close(cr.stopCh)
	cr.logger.Info("Certificate rotation monitor stopped")
}

// GetCertificate returns the current certificate (thread-safe)
func (cr *CertRotator) GetCertificate() *tls.Certificate {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.currentCert
}

// GetClientHelloInfo returns a GetCertificate function for TLS config
func (cr *CertRotator) GetClientHelloInfo(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return cr.GetCertificate(), nil
}

// RegisterCallback registers a callback function to be called on cert rotation
func (cr *CertRotator) RegisterCallback(callback func(*tls.Certificate)) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.callbacks = append(cr.callbacks, callback)
}

// monitorCertificates periodically checks certificates and rotates if needed
func (cr *CertRotator) monitorCertificates() {
	ticker := time.NewTicker(cr.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cr.checkAndRotate(); err != nil {
				cr.logger.WithError(err).Error("Certificate rotation check failed")
			}
		case <-cr.stopCh:
			return
		}
	}
}

// checkAndRotate checks if rotation is needed and performs it
func (cr *CertRotator) checkAndRotate() error {
	cr.mu.RLock()
	currentCert := cr.currentCert
	cr.mu.RUnlock()

	// Parse the certificate
	x509Cert, err := x509.ParseCertificate(currentCert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check if rotation is needed
	timeUntilExpiry := time.Until(x509Cert.NotAfter)

	if timeUntilExpiry <= cr.rotationThreshold {
		cr.logger.WithFields(logrus.Fields{
			"expires_in": timeUntilExpiry,
			"threshold":  cr.rotationThreshold,
		}).Info("Certificate rotation threshold reached")

		return cr.rotateCertificate("approaching expiry")
	}

	// Also check if certificate file has been updated externally
	newCert, err := tls.LoadX509KeyPair(cr.certPath, cr.keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate from disk: %w", err)
	}

	newX509Cert, err := x509.ParseCertificate(newCert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse new certificate: %w", err)
	}

	// Check if certificate has changed (different serial number)
	if newX509Cert.SerialNumber.Cmp(x509Cert.SerialNumber) != 0 {
		cr.logger.Info("Detected external certificate update")
		return cr.rotateCertificate("external update")
	}

	return nil
}

// rotateCertificate performs the actual certificate rotation
func (cr *CertRotator) rotateCertificate(reason string) error {
	cr.logger.WithField("reason", reason).Info("Starting certificate rotation")

	// Load new certificate
	newCert, err := tls.LoadX509KeyPair(cr.certPath, cr.keyPath)
	if err != nil {
		return fmt.Errorf("failed to load new certificate: %w", err)
	}

	// Parse certificates for logging
	oldX509Cert, _ := x509.ParseCertificate(cr.currentCert.Certificate[0])
	newX509Cert, err := x509.ParseCertificate(newCert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse new certificate: %w", err)
	}

	// Create rotation event
	event := &RotationEvent{
		Timestamp: time.Now(),
		OldCert:   GetCertificateInfo(oldX509Cert),
		NewCert:   GetCertificateInfo(newX509Cert),
		Reason:    reason,
		Success:   true,
	}

	// Atomic swap
	cr.mu.Lock()
	cr.nextCert = &newCert
	cr.currentCert = cr.nextCert
	cr.mu.Unlock()

	// Call registered callbacks
	cr.notifyCallbacks(&newCert)

	cr.logger.WithFields(logrus.Fields{
		"old_serial": event.OldCert.SerialNumber,
		"new_serial": event.NewCert.SerialNumber,
		"reason":     reason,
	}).Info("Certificate rotation completed successfully")

	return nil
}

// notifyCallbacks notifies all registered callbacks
func (cr *CertRotator) notifyCallbacks(cert *tls.Certificate) {
	cr.mu.RLock()
	callbacks := make([]func(*tls.Certificate), len(cr.callbacks))
	copy(callbacks, cr.callbacks)
	cr.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(cert)
	}
}

// ForceRotation forces an immediate certificate rotation
func (cr *CertRotator) ForceRotation() error {
	cr.logger.Info("Forcing certificate rotation")
	return cr.rotateCertificate("manual rotation")
}

// GetExpiryInfo returns information about certificate expiry
func (cr *CertRotator) GetExpiryInfo() (time.Time, time.Duration, error) {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	x509Cert, err := x509.ParseCertificate(cr.currentCert.Certificate[0])
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("failed to parse certificate: %w", err)
	}

	expiryTime := x509Cert.NotAfter
	timeUntilExpiry := time.Until(expiryTime)

	return expiryTime, timeUntilExpiry, nil
}

// SetRotationThreshold sets the threshold for automatic rotation
func (cr *CertRotator) SetRotationThreshold(threshold time.Duration) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.rotationThreshold = threshold
	cr.logger.WithField("threshold", threshold).Info("Updated rotation threshold")
}

// SetCheckInterval sets the interval for checking certificate status
func (cr *CertRotator) SetCheckInterval(interval time.Duration) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.checkInterval = interval
	cr.logger.WithField("interval", interval).Info("Updated check interval")
}
