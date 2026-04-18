//go:build ci

package nlb_test

import (
	"testing"

	"marchproxy-nlb/internal/logging"
	"marchproxy-nlb/internal/nlb"
)

func TestNewRouter(t *testing.T) {
	logger, err := logging.NewLogrusAdapter("test")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	router := nlb.NewRouter(logger)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
}

func TestNewRouterNilLogger(t *testing.T) {
	// Router should handle nil logger
	router := nlb.NewRouter(nil)
	if router == nil {
		t.Fatal("expected non-nil router even with nil logger")
	}
}

func TestProtocol(t *testing.T) {
	tests := []nlb.Protocol{
		nlb.ProtocolUnknown,
		nlb.ProtocolHTTP,
		nlb.ProtocolMySQL,
		nlb.ProtocolPostgreSQL,
		nlb.ProtocolMongoDB,
		nlb.ProtocolRedis,
		nlb.ProtocolRTMP,
	}

	for _, p := range tests {
		// Verify String() method works
		str := p.String()
		if str == "" {
			t.Errorf("protocol string should not be empty for %v", p)
		}
	}
}

func TestProtocolString(t *testing.T) {
	tests := []struct {
		protocol     nlb.Protocol
		expectedName string
	}{
		{nlb.ProtocolHTTP, "HTTP"},
		{nlb.ProtocolMySQL, "MySQL"},
		{nlb.ProtocolPostgreSQL, "PostgreSQL"},
		{nlb.ProtocolMongoDB, "MongoDB"},
		{nlb.ProtocolRedis, "Redis"},
		{nlb.ProtocolRTMP, "RTMP"},
		{nlb.ProtocolUnknown, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			if tt.protocol.String() != tt.expectedName {
				t.Errorf("expected %q, got %q", tt.expectedName, tt.protocol.String())
			}
		})
	}
}
