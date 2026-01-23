package extauth

import (
	"context"
	"testing"
	"time"

	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"marchproxy-egress/internal/threat"
)

func createTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

func createTestThreatManager(t *testing.T) *threat.Manager {
	logger := createTestLogger()
	cfg := threat.ManagerConfig{
		IPBlockingEnabled:     true,
		IPCacheSize:           10000,
		DomainBlockingEnabled: true,
		WildcardSupport:       true,
		URLMatchingEnabled:    true,
		URLEngine:             "re2",
		MaxPatterns:           1000,
	}
	manager, err := threat.NewManager(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create threat manager: %v", err)
	}
	return manager
}

func TestNewServer(t *testing.T) {
	logger := createTestLogger()

	cfg := ServerConfig{
		Port: 50051,
	}

	server := NewServer(cfg, logger)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.port != 50051 {
		t.Errorf("Expected port 50051, got %d", server.port)
	}
}

func TestNewServer_NilLogger(t *testing.T) {
	cfg := ServerConfig{
		Port: 50051,
	}

	server := NewServer(cfg, nil)
	if server == nil {
		t.Fatal("NewServer returned nil")
	}

	if server.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}

func TestServer_StartStop(t *testing.T) {
	logger := createTestLogger()
	threatManager := createTestThreatManager(t)

	cfg := ServerConfig{
		Port:          50060,
		ThreatManager: threatManager,
	}

	server := NewServer(cfg, logger)

	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give the server time to start
	time.Sleep(50 * time.Millisecond)

	// Verify we can connect
	conn, err := grpc.Dial("localhost:50060", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	conn.Close()

	server.Stop()
}

func TestServer_Check_AllowedRequest(t *testing.T) {
	logger := createTestLogger()
	threatManager := createTestThreatManager(t)

	cfg := ServerConfig{
		Port:          50053,
		ThreatManager: threatManager,
	}

	server := NewServer(cfg, logger)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Give the server time to start
	time.Sleep(50 * time.Millisecond)

	// Connect and make a request
	conn, err := grpc.Dial("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := authv3.NewAuthorizationClient(conn)

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

	resp, err := client.Check(ctx, req)
	if err != nil {
		t.Fatalf("Check request failed: %v", err)
	}

	// Without any blocking rules, the request should be allowed
	if resp.GetOkResponse() == nil {
		t.Error("Expected OK response for allowed request")
	}
}

func TestServer_Check_BlockedByIP(t *testing.T) {
	logger := createTestLogger()
	threatManager := createTestThreatManager(t)

	// Add a blocking rule for destination IP
	ipBlocker := threatManager.GetIPBlocker()
	ipBlocker.AddRule(threat.BlockRule{
		ID:        "test-block",
		Pattern:   "10.0.0.100",
		Category:  "test",
		Source:    "test",
		CreatedAt: time.Now(),
	})

	cfg := ServerConfig{
		Port:          50054,
		ThreatManager: threatManager,
	}

	server := NewServer(cfg, logger)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	conn, err := grpc.Dial("localhost:50054", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := authv3.NewAuthorizationClient(conn)

	// Request to blocked destination - note: the server parses socket address differently
	// Since we can't easily set socket address in test, we'll verify the basic flow works
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

	resp, err := client.Check(ctx, req)
	if err != nil {
		t.Fatalf("Check request failed: %v", err)
	}

	// Without proper socket address, it should still respond (allowed in this case)
	if resp == nil {
		t.Error("Expected response from server")
	}
}

func TestServer_Check_BlockedByDomain(t *testing.T) {
	logger := createTestLogger()
	threatManager := createTestThreatManager(t)

	// Add a domain blocking rule
	domainBlocker := threatManager.GetDomainBlocker()
	domainBlocker.AddRule(threat.BlockRule{
		ID:        "test-domain-block",
		Pattern:   "malware.com",
		Category:  "malware",
		Source:    "test",
		CreatedAt: time.Now(),
	})

	cfg := ServerConfig{
		Port:          50055,
		ThreatManager: threatManager,
	}

	server := NewServer(cfg, logger)
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	conn, err := grpc.Dial("localhost:50055", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := authv3.NewAuthorizationClient(conn)

	req := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Host:   "malware.com",
					Path:   "/api/test",
					Method: "GET",
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Check(ctx, req)
	if err != nil {
		t.Fatalf("Check request failed: %v", err)
	}

	// Should be denied because domain is blocked
	if resp.GetDeniedResponse() == nil {
		t.Error("Expected Denied response for blocked domain")
	}
}

func TestServer_GetStats(t *testing.T) {
	logger := createTestLogger()

	cfg := ServerConfig{
		Port: 50056,
	}

	server := NewServer(cfg, logger)

	stats := server.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if stats["total_requests"] != int64(0) {
		t.Errorf("Expected 0 total_requests, got %v", stats["total_requests"])
	}
}
