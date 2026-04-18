//go:build ci
// +build ci

package rtmp

import (
	"testing"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
)

// TestNewServer tests RTMP server creation
func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		Host:     "127.0.0.1",
		Port:     1935,
		GRPCPort: 50053,
	}

	server, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create RTMP server: %v", err)
	}

	if server == nil {
		t.Fatal("expected non-nil server")
	}
	if server.config != cfg {
		t.Error("config not set correctly")
	}
	if server.running {
		t.Error("server should not be running initially")
	}
	if len(server.sessions) != 0 {
		t.Error("sessions should be empty initially")
	}
}

// TestNewServer_DifferentConfigs tests RTMP server with different configurations
func TestNewServer_DifferentConfigs(t *testing.T) {
	configs := []struct {
		name string
		host string
		port int
	}{
		{"localhost", "127.0.0.1", 1935},
		{"all interfaces", "0.0.0.0", 1935},
		{"custom port", "127.0.0.1", 2000},
		{"ipv6", "::", 1935},
	}

	for _, tt := range configs {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Host: tt.host,
				Port: tt.port,
			}

			server, err := NewServer(cfg, nil)
			if err != nil {
				t.Fatalf("failed to create server: %v", err)
			}

			if server.config.Host != tt.host {
				t.Errorf("host mismatch: expected %s, got %s", tt.host, server.config.Host)
			}
			if server.config.Port != tt.port {
				t.Errorf("port mismatch: expected %d, got %d", tt.port, server.config.Port)
			}
		})
	}
}

// TestNewServer_WithFFmpegManager tests RTMP server with FFmpeg manager
func TestNewServer_WithFFmpegManager(t *testing.T) {
	cfg := &config.Config{
		Host: "127.0.0.1",
		Port: 1935,
	}

	server, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create RTMP server: %v", err)
	}

	if server.ffmpegManager != nil {
		t.Error("ffmpegManager should be nil when nil is passed")
	}
}

// TestNewServer_MultipleInstances tests creating multiple RTMP servers
func TestNewServer_MultipleInstances(t *testing.T) {
	configs := []*config.Config{
		{Host: "127.0.0.1", Port: 1935},
		{Host: "127.0.0.1", Port: 1936},
		{Host: "127.0.0.1", Port: 1937},
	}

	servers := make([]*Server, len(configs))
	for i, cfg := range configs {
		server, err := NewServer(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create server %d: %v", i, err)
		}
		servers[i] = server
	}

	// Verify each server has independent config
	for i, srv := range servers {
		if srv.config.Port != configs[i].Port {
			t.Errorf("server %d port mismatch: expected %d, got %d", i, configs[i].Port, srv.config.Port)
		}
	}
}

// TestServer_SessionsInitialization tests that sessions map is properly initialized
func TestServer_SessionsInitialization(t *testing.T) {
	cfg := &config.Config{
		Host: "127.0.0.1",
		Port: 1935,
	}

	server, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server.sessions == nil {
		t.Fatal("sessions map should be initialized")
	}
	if len(server.sessions) != 0 {
		t.Errorf("sessions should be empty, got %d sessions", len(server.sessions))
	}
}

// TestServer_MutexInitialization tests that synchronization primitives exist
func TestServer_MutexInitialization(t *testing.T) {
	cfg := &config.Config{
		Host: "127.0.0.1",
		Port: 1935,
	}

	server, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// These should not panic even with zero-initialized mutexes
	server.sessionsMutex.RLock()
	server.sessionsMutex.RUnlock()

	server.runningMutex.RLock()
	server.runningMutex.RUnlock()
}

// TestServer_Config tests that server stores config correctly
func TestServer_Config(t *testing.T) {
	cfg := &config.Config{
		Host:             "192.168.1.1",
		Port:             2000,
		GRPCPort:         50053,
		Encoder:          "x265",
		LogLevel:         "debug",
		EnableHLS:        true,
		EnableDASH:       false,
		SegmentDuration:  8,
		MaxBitrate:       50,
		MaxStreams:       200,
	}

	server, err := NewServer(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if server.config.Host != cfg.Host {
		t.Errorf("host mismatch: expected %s, got %s", cfg.Host, server.config.Host)
	}
	if server.config.Port != cfg.Port {
		t.Errorf("port mismatch: expected %d, got %d", cfg.Port, server.config.Port)
	}
	if server.config.Encoder != cfg.Encoder {
		t.Errorf("encoder mismatch: expected %s, got %s", cfg.Encoder, server.config.Encoder)
	}
}

