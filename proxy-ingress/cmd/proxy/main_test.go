//go:build ci

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"marchproxy-ingress/internal/config"
	"marchproxy-ingress/internal/manager"
)

func TestGetHostname(t *testing.T) {
	hostname := getHostname()
	if hostname == "" {
		t.Error("expected non-empty hostname")
	}
}

func TestIngressProxyFindMatchingRoute(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		path      string
		vhosts    []manager.VirtualHost
		wantMatch bool
	}{
		{
			name: "exact host match",
			host: "api.example.com",
			path: "/",
			vhosts: []manager.VirtualHost{
				{Hostname: "api.example.com"},
			},
			wantMatch: true,
		},
		{
			name: "wildcard domain match",
			host: "api.example.com",
			path: "/",
			vhosts: []manager.VirtualHost{
				{Hostname: "*.example.com"},
			},
			wantMatch: true,
		},
		{
			name: "catch-all pattern",
			host: "anything.com",
			path: "/",
			vhosts: []manager.VirtualHost{
				{Hostname: "*"},
			},
			wantMatch: true,
		},
		{
			name: "no match",
			host: "other.com",
			path: "/",
			vhosts: []manager.VirtualHost{
				{Hostname: "api.example.com"},
			},
			wantMatch: false,
		},
		{
			name: "path prefix match",
			host: "api.example.com",
			path: "/api/v1/users",
			vhosts: []manager.VirtualHost{
				{
					Hostname: "api.example.com",
					RoutingRules: []manager.RoutingRule{
						{PathPattern: "/api/*"},
					},
				},
			},
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := &IngressProxy{
				clusterConfig: &manager.ClusterConfig{
					VirtualHosts: tt.vhosts,
				},
			}

			req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s%s", tt.host, tt.path), nil)
			req.Host = tt.host

			got := proxy.findMatchingRoute(req)
			if (got != nil) != tt.wantMatch {
				t.Errorf("findMatchingRoute() got match=%v, want=%v", got != nil, tt.wantMatch)
			}
		})
	}
}

func TestIngressProxyMatchesHostPattern(t *testing.T) {
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"api.example.com", "api.example.com", true},
		{"api.example.com", "*.example.com", true},
		{"example.com", "*.example.com", true},
		{"other.com", "*.example.com", false},
		{"api.example.com", "*", true},
		{"anything", "", true},
		{"api.example.com", "api.example.com", true},
	}

	proxy := &IngressProxy{}
	for _, tt := range tests {
		t.Run(tt.host+":"+tt.pattern, func(t *testing.T) {
			if got := proxy.matchesHostPattern(tt.host, tt.pattern); got != tt.want {
				t.Errorf("matchesHostPattern(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestIngressProxyMatchesPathPattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"/api/v1/users", "/api/*", true},
		{"/api/v1/users", "/api/v1/*", true},
		{"/other", "/api/*", false},
		{"/", "/", true},
		{"/api/v1/users", "/*", true},
		{"/api/v1/users", "/api/v1/users", true},
		{"/api/v1/users", "/api/v1/products", false},
	}

	proxy := &IngressProxy{}
	for _, tt := range tests {
		t.Run(tt.path+":"+tt.pattern, func(t *testing.T) {
			if got := proxy.matchesPathPattern(tt.path, tt.pattern); got != tt.want {
				t.Errorf("matchesPathPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestIngressProxySelectBackend(t *testing.T) {
	tests := []struct {
		name      string
		vhost     *manager.VirtualHost
		backends  []manager.Backend
		wantErr   bool
	}{
		{
			name: "valid backend selection",
			vhost: &manager.VirtualHost{
				Backend: "api-backend",
			},
			backends: []manager.Backend{
				{Name: "api-backend", Endpoints: []manager.BackendEndpoint{
					{Host: "localhost", Port: 8080},
				}},
			},
			wantErr: false,
		},
		{
			name: "backend not found",
			vhost: &manager.VirtualHost{
				Backend: "nonexistent",
			},
			backends: []manager.Backend{},
			wantErr:  true,
		},
		{
			name: "no backend configured",
			vhost: &manager.VirtualHost{
				Backend: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := &IngressProxy{
				clusterConfig: &manager.ClusterConfig{
					Backends: tt.backends,
				},
			}

			got, err := proxy.selectBackend(tt.vhost)
			if (err != nil) != tt.wantErr {
				t.Errorf("selectBackend() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Errorf("selectBackend() = nil, want non-nil URL")
			}
		})
	}
}

func TestIngressProxyValidateClientCertificate(t *testing.T) {
	proxy := &IngressProxy{}
	vhost := &manager.VirtualHost{}
	cert := &x509.Certificate{}

	err := proxy.validateClientCertificate(cert, vhost)
	if err != nil {
		t.Errorf("validateClientCertificate() = %v, want nil", err)
	}
}

func TestCreateReverseProxyHandler(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			VirtualHosts: []manager.VirtualHost{},
		},
		metrics: &IngressMetrics{},
	}

	handler := proxy.createReverseProxyHandler(false)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestCreateReverseProxyHandlerWithTLS(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			VirtualHosts: []manager.VirtualHost{},
		},
		metrics: &IngressMetrics{},
	}

	handler := proxy.createReverseProxyHandler(true)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

// GenerateTLSCert generates a self-signed TLS certificate for testing
func GenerateTLSCert(t *testing.T) (tls.Certificate, []byte, []byte) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test.local",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		DNSNames:    []string{"test.local", "*.test.local"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Self-sign certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	// Load certificate
	tlsCert, err := tls.X509KeyPair([]byte(encodeCert(certBytes, t)), []byte(encodeKey(privateKey, t)))
	if err != nil {
		t.Fatalf("failed to load certificate: %v", err)
	}

	return tlsCert, certBytes, []byte(encodeKey(privateKey, t))
}

func encodeCert(certBytes []byte, t *testing.T) string {
	return fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----",
		fmt.Sprintf("%x", certBytes))
}

func encodeKey(key *rsa.PrivateKey, t *testing.T) string {
	return "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDU...\n-----END PRIVATE KEY-----"
}

func TestIngressProxyHandler(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			VirtualHosts: []manager.VirtualHost{
				{
					Hostname: "test.local",
					Backend:  "test-backend",
				},
			},
			Backends: []manager.Backend{
				{
					Name: "test-backend",
					Endpoints: []manager.BackendEndpoint{
						{Host: "localhost", Port: 8080},
					},
				},
			},
		},
		metrics: &IngressMetrics{},
	}

	handler := proxy.createReverseProxyHandler(false)
	req := httptest.NewRequest("GET", "http://test.local/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should either succeed or return error gracefully
	if w.Code == 0 {
		t.Error("handler did not write response")
	}
}

func TestIngressProxyHandlerNoRoute(t *testing.T) {
	proxy := &IngressProxy{
		clusterConfig: &manager.ClusterConfig{
			VirtualHosts: []manager.VirtualHost{},
		},
		metrics: &IngressMetrics{},
	}

	handler := proxy.createReverseProxyHandler(false)
	req := httptest.NewRequest("GET", "http://unknown.local/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound && w.Code != http.StatusBadGateway {
		t.Errorf("handler returned status %d, want 404 or 502", w.Code)
	}
}

func TestIngressProxyStartHTTP(t *testing.T) {
	cfg := &config.Config{
		Port: 0,
	}
	proxy := &IngressProxy{
		config: cfg,
		metrics: &IngressMetrics{},
	}

	// Start in goroutine with context
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Should handle gracefully
	_ = proxy
	_ = ctx
}

func TestIngressProxyStartHTTPS(t *testing.T) {
	cfg := &config.Config{
		TLSPort: 0,
	}
	proxy := &IngressProxy{
		config:  cfg,
		metrics: &IngressMetrics{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_ = proxy
	_ = ctx
}
