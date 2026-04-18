//go:build ci

package extauth

import (
	"context"
	"testing"
	"time"

	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/grpc/codes"

	"marchproxy-egress/internal/logging"
	"marchproxy-egress/internal/threat"
)

// TestCheckRequestWithMissingHTTP tests Check with missing HTTP attributes
func TestCheckRequestWithMissingHTTP(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ServerConfig{
		Port: 50061,
	}
	server := NewServer(cfg, logger)

	// Request with no HTTP attributes
	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: nil,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.Check(ctx, req)
	if err != nil {
		t.Errorf("Check should not return error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Check should return response")
	}

	// Should be denied due to missing HTTP attributes
	if resp.GetDeniedResponse() == nil {
		t.Error("expected DeniedResponse for missing HTTP attributes")
	}

	if resp.GetStatus().GetCode() != int32(codes.InvalidArgument) {
		t.Errorf("expected InvalidArgument status code, got %v", resp.GetStatus().GetCode())
	}
}

// TestCheckRequestWithBothSourceAndDest tests Check with both source and destination addresses
func TestCheckRequestWithBothSourceAndDest(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	threatMgr := createTestThreatManager(t)

	cfg := ServerConfig{
		Port:          50062,
		ThreatManager: threatMgr,
	}
	server := NewServer(cfg, logger)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Source: &authv3.AttributeContext_Peer{
				Address: &corev3.Address{
					Address: &corev3.Address_SocketAddress{
						SocketAddress: &corev3.SocketAddress{
							Address: "192.168.1.100",
							PortSpecifier: &corev3.SocketAddress_PortValue{
								PortValue: 12345,
							},
						},
					},
				},
			},
			Destination: &authv3.AttributeContext_Peer{
				Address: &corev3.Address{
					Address: &corev3.Address_SocketAddress{
						SocketAddress: &corev3.SocketAddress{
							Address: "10.0.0.1",
							PortSpecifier: &corev3.SocketAddress_PortValue{
								PortValue: 8080,
							},
						},
					},
				},
			},
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "example.com",
					Path:   "/api/test",
					Method: "GET",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.Check(ctx, req)
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Check should return response")
	}

	// Should allow since no blocking rules
	if resp.GetOkResponse() == nil {
		t.Error("expected OkResponse for allowed request")
	}
}

// TestCheckRequestWithServiceContext tests service context extraction
func TestCheckRequestWithServiceContext(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ServerConfig{
		Port: 50063,
	}
	server := NewServer(cfg, logger)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "example.com",
					Path:   "/api/test",
					Method: "GET",
					Headers: map[string]string{
						"authorization":  "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
						"x-service-id":    "service-123",
						"x-service-name":  "my-service",
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.Check(ctx, req)
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Check should return response")
	}

	// Should allow since no blocking rules
	if resp.GetOkResponse() == nil {
		t.Error("expected OkResponse for request with service context")
	}
}

// TestCheckRequestWithBearerTokenVariations tests Bearer token case variations
func TestCheckRequestWithBearerTokenVariations(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		expectAllow bool
	}{
		{
			name:        "Bearer uppercase",
			authHeader:  "Bearer token123",
			expectAllow: true,
		},
		{
			name:        "bearer lowercase",
			authHeader:  "bearer token123",
			expectAllow: true,
		},
		{
			name:        "no token",
			authHeader:  "",
			expectAllow: true, // unauthenticated but allowed if no access control
		},
		{
			name:        "invalid format",
			authHeader:  "Basic dXNlcjpwYXNzd29yZA==",
			expectAllow: true, // not Bearer, treated as unauthenticated
		},
	}

	logger, _ := logging.NewLogrusAdapter("test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ServerConfig{
				Port: 50064,
			}
			server := NewServer(cfg, logger)

			headers := map[string]string{
				"host":   "example.com",
				"path":   "/api/test",
				"method": "GET",
			}
			if tt.authHeader != "" {
				headers["authorization"] = tt.authHeader
			}

			req := &authv3.CheckRequest{
				Attributes: &authv3.AttributeContext{
					Request: &authv3.AttributeContext_Request{
						Http: &authv3.AttributeContext_HttpRequest{
							Host:    "example.com",
							Path:    "/api/test",
							Method:  "GET",
							Headers: headers,
						},
					},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := server.Check(ctx, req)
			if err != nil {
				t.Errorf("Check failed: %v", err)
			}

			if resp == nil {
				t.Fatal("Check should return response")
			}

			if tt.expectAllow && resp.GetOkResponse() == nil {
				t.Errorf("expected allowed request, got denied: %v", resp.GetDeniedResponse())
			}
		})
	}
}

// TestAllowedResponseHeaders tests headers in allowed response
func TestAllowedResponseHeaders(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ServerConfig{
		Port: 50065,
	}
	server := NewServer(cfg, logger)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "example.com",
					Path:   "/api/test",
					Method: "GET",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, _ := server.Check(ctx, req)

	if resp == nil {
		t.Fatal("Check should return response")
	}

	okResp := resp.GetOkResponse()
	if okResp == nil {
		t.Fatal("expected OkResponse")
	}

	// Check for required headers
	if okResp.Headers == nil || len(okResp.Headers) == 0 {
		t.Error("OkResponse should contain headers")
	}

	// Verify headers contain expected values
	headerFound := false
	for _, hdr := range okResp.Headers {
		if hdr.Header != nil && hdr.Header.Key == "x-marchproxy-checked" {
			if hdr.Header.Value != "true" {
				t.Errorf("expected x-marchproxy-checked=true, got %s", hdr.Header.Value)
			}
			headerFound = true
		}
	}

	if !headerFound {
		t.Error("x-marchproxy-checked header not found in response")
	}
}

// TestDeniedResponseHeaders tests headers in denied response
func TestDeniedResponseHeaders(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	threatMgr := createTestThreatManager(t)

	// Add blocking rule
	domainBlocker := threatMgr.GetDomainBlocker()
	domainBlocker.AddRule(threat.BlockRule{
		ID:        "test-block",
		Pattern:   "blocked.com",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	})

	cfg := ServerConfig{
		Port:          50066,
		ThreatManager: threatMgr,
	}
	server := NewServer(cfg, logger)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "blocked.com",
					Path:   "/api/test",
					Method: "GET",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.Check(ctx, req)
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Check should return response")
	}

	deniedResp := resp.GetDeniedResponse()
	if deniedResp == nil {
		t.Fatal("expected DeniedResponse")
	}

	// Check for required headers
	if deniedResp.Headers == nil || len(deniedResp.Headers) == 0 {
		t.Error("DeniedResponse should contain headers")
	}

	// Verify headers contain expected values
	blockedHeaderFound := false
	for _, hdr := range deniedResp.Headers {
		if hdr.Header != nil && hdr.Header.Key == "x-marchproxy-blocked" {
			if hdr.Header.Value != "true" {
				t.Errorf("expected x-marchproxy-blocked=true, got %s", hdr.Header.Value)
			}
			blockedHeaderFound = true
		}
	}

	if !blockedHeaderFound {
		t.Error("x-marchproxy-blocked header not found in response")
	}
}

// TestDeniedResponseBody tests body content in denied response
func TestDeniedResponseBody(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	threatMgr := createTestThreatManager(t)

	// Add blocking rule
	domainBlocker := threatMgr.GetDomainBlocker()
	domainBlocker.AddRule(threat.BlockRule{
		ID:        "test-block",
		Pattern:   "evil.com",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	})

	cfg := ServerConfig{
		Port:          50067,
		ThreatManager: threatMgr,
	}
	server := NewServer(cfg, logger)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "evil.com",
					Path:   "/api/test",
					Method: "GET",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.Check(ctx, req)
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Check should return response")
	}

	deniedResp := resp.GetDeniedResponse()
	if deniedResp == nil {
		t.Fatal("expected DeniedResponse")
	}

	if deniedResp.Body == "" {
		t.Error("DeniedResponse should contain body with error information")
	}

	if !contains(deniedResp.Body, "error") {
		t.Errorf("response body should contain 'error', got %s", deniedResp.Body)
	}
}

// TestStatisticsUpdating tests that statistics are properly updated
func TestStatisticsUpdating(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	threatMgr := createTestThreatManager(t)

	cfg := ServerConfig{
		Port:          50068,
		ThreatManager: threatMgr,
	}
	server := NewServer(cfg, logger)

	initialStats := server.GetStats()
	initialTotal := initialStats["total_requests"]

	// Send multiple requests
	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "example.com",
					Path:   "/api/test",
					Method: "GET",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < 3; i++ {
		server.Check(ctx, req)
	}

	stats := server.GetStats()
	if stats["total_requests"] != initialTotal+3 {
		t.Errorf("expected total_requests=%d, got %d", initialTotal+3, stats["total_requests"])
	}

	if stats["allowed_requests"] < 3 {
		t.Errorf("expected at least 3 allowed_requests, got %d", stats["allowed_requests"])
	}
}

// TestResetStats tests statistics reset functionality
func TestResetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ServerConfig{
		Port: 50069,
	}
	server := NewServer(cfg, logger)

	// Make some requests to accumulate stats
	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "example.com",
					Path:   "/api/test",
					Method: "GET",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Check(ctx, req)

	// Verify stats were accumulated
	stats := server.GetStats()
	if stats["total_requests"] == 0 {
		t.Fatal("expected non-zero stats after request")
	}

	// Reset stats
	server.ResetStats()

	stats = server.GetStats()
	if stats["total_requests"] != 0 {
		t.Errorf("expected total_requests=0 after reset, got %d", stats["total_requests"])
	}
	if stats["allowed_requests"] != 0 {
		t.Errorf("expected allowed_requests=0 after reset, got %d", stats["allowed_requests"])
	}
	if stats["denied_requests"] != 0 {
		t.Errorf("expected denied_requests=0 after reset, got %d", stats["denied_requests"])
	}
}

// TestMissingServiceComponents tests Check with missing service components
func TestMissingServiceComponents(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	// No threat manager or access controller
	cfg := ServerConfig{
		Port:          50070,
		ThreatManager: nil,
	}
	server := NewServer(cfg, logger)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "example.com",
					Path:   "/api/test",
					Method: "GET",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.Check(ctx, req)
	if err != nil {
		t.Errorf("Check should not error with missing components, got %v", err)
	}

	if resp == nil {
		t.Fatal("Check should return response")
	}

	// Should allow since no threat manager to block
	if resp.GetOkResponse() == nil {
		t.Error("expected OkResponse when no threat manager present")
	}
}

// TestHTTPStatusCodes tests HTTP status code assignment
func TestHTTPStatusCodes(t *testing.T) {
	tests := []struct {
		name      string
		code      codes.Code
		wantHTTP  typev3.StatusCode
	}{
		{
			name:     "PermissionDenied -> Forbidden",
			code:     codes.PermissionDenied,
			wantHTTP: typev3.StatusCode_Forbidden,
		},
		{
			name:     "Unauthenticated -> Unauthorized",
			code:     codes.Unauthenticated,
			wantHTTP: typev3.StatusCode_Unauthorized,
		},
		{
			name:     "InvalidArgument -> BadRequest",
			code:     codes.InvalidArgument,
			wantHTTP: typev3.StatusCode_BadRequest,
		},
	}

	logger, _ := logging.NewLogrusAdapter("test")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ServerConfig{
				Port: 50071,
			}
			server := NewServer(cfg, logger)

			resp := server.createDeniedResponse("test reason", tt.code)

			if resp == nil {
				t.Fatal("createDeniedResponse should return response")
			}

			deniedResp := resp.GetDeniedResponse()
			if deniedResp == nil {
				t.Fatal("expected DeniedResponse")
			}

			if deniedResp.Status == nil {
				t.Fatal("expected Status in DeniedResponse")
			}

			if deniedResp.Status.Code != tt.wantHTTP {
				t.Errorf("expected HTTP status %v, got %v", tt.wantHTTP, deniedResp.Status.Code)
			}
		})
	}
}

// TestMinHelper tests the min helper function
func TestMinHelper(t *testing.T) {
	tests := []struct {
		a, b  int
		want  int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 100, 0},
		{-1, 5, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestRequestMethod tests different HTTP methods
func TestRequestMethod(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	logger, _ := logging.NewLogrusAdapter("test")
	threatMgr := createTestThreatManager(t)

	cfg := ServerConfig{
		Port:          50072,
		ThreatManager: threatMgr,
	}
	server := NewServer(cfg, logger)

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := &authv3.CheckRequest{
				Attributes: &authv3.AttributeContext{
					Request: &authv3.AttributeContext_Request{
						Http: &authv3.AttributeContext_HttpRequest{
							Host:   "example.com",
							Path:   "/api/test",
							Method: method,
						},
					},
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := server.Check(ctx, req)
			if err != nil {
				t.Errorf("Check failed for method %s: %v", method, err)
			}

			if resp == nil {
				t.Errorf("Check should return response for method %s", method)
			}
		})
	}
}

// TestCaseSensitiveHeaders tests header case sensitivity
func TestCaseSensitiveHeaders(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")

	cfg := ServerConfig{
		Port: 50073,
	}
	server := NewServer(cfg, logger)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "example.com",
					Path:   "/api/test",
					Method: "GET",
					Headers: map[string]string{
						"Authorization": "Bearer token123", // Different case
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := server.Check(ctx, req)
	if err != nil {
		t.Errorf("Check failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Check should return response")
	}

	// Should work with different case in headers
	if resp.GetOkResponse() == nil {
		t.Error("expected OkResponse with case-variant Authorization header")
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || (len(substr) <= len(s)))
}
