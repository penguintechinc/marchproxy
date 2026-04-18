//go:build ci

package nlb

import (
	"testing"
	"time"
)

func TestModuleEndpoint_HealthStatus(t *testing.T) {
	endpoint := &ModuleEndpoint{
		Name:        "test-module",
		Protocol:    ProtocolHTTP,
		Address:     "localhost",
		GRPCPort:    50051,
		Healthy:     false,
		MaxConns:    100,
		LastHealthy: time.Now(),
	}

	// Test initial state
	if endpoint.IsHealthy() {
		t.Errorf("IsHealthy() should be false initially")
	}

	// Set healthy
	endpoint.SetHealthy(true)
	if !endpoint.IsHealthy() {
		t.Errorf("IsHealthy() should be true after SetHealthy(true)")
	}

	// Set unhealthy
	endpoint.SetHealthy(false)
	if endpoint.IsHealthy() {
		t.Errorf("IsHealthy() should be false after SetHealthy(false)")
	}
}

func TestModuleEndpoint_ConnectionManagement(t *testing.T) {
	endpoint := &ModuleEndpoint{
		Name:     "test-module",
		Protocol: ProtocolHTTP,
		Address:  "localhost",
		GRPCPort: 50051,
		Healthy:  true,
		MaxConns: 10,
	}

	tests := []struct {
		name      string
		conns     int
		increment bool
		wantErr   bool
	}{
		{"Increment from 0", 0, true, false},
		{"Increment to max", 10, true, true},
		{"Decrement from 5", 5, false, false},
		{"Decrement from 0", 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint.ActiveConns = tt.conns

			if tt.increment {
				err := endpoint.IncrementConns()
				if (err != nil) != tt.wantErr {
					t.Errorf("IncrementConns() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				endpoint.DecrementConns()
				// Decrement should not error, just be safe
			}
		})
	}
}

func TestModuleEndpoint_GetActiveConns(t *testing.T) {
	endpoint := &ModuleEndpoint{
		Name:         "test-module",
		Protocol:     ProtocolHTTP,
		ActiveConns:  5,
		MaxConns:     100,
	}

	got := endpoint.GetActiveConns()
	if got != 5 {
		t.Errorf("GetActiveConns() = %d, want 5", got)
	}
}

func TestModuleEndpoint_ConnsCappedAtMax(t *testing.T) {
	endpoint := &ModuleEndpoint{
		Name:     "test-module",
		Protocol: ProtocolHTTP,
		MaxConns: 3,
	}

	// Fill to max
	_ = endpoint.IncrementConns()
	_ = endpoint.IncrementConns()
	_ = endpoint.IncrementConns()

	// Try to exceed
	err := endpoint.IncrementConns()
	if err == nil {
		t.Errorf("IncrementConns() should error at max capacity")
	}

	if endpoint.GetActiveConns() != 3 {
		t.Errorf("ActiveConns should be 3 (max), got %d", endpoint.GetActiveConns())
	}
}

func TestModuleEndpoint_LastHealthyTime(t *testing.T) {
	endpoint := &ModuleEndpoint{
		Name:    "test-module",
		Protocol: ProtocolHTTP,
		Healthy: false,
	}

	before := time.Now()
	endpoint.SetHealthy(true)
	after := time.Now()

	if endpoint.LastHealthy.Before(before) || endpoint.LastHealthy.After(after.Add(time.Second)) {
		t.Errorf("LastHealthy time not updated correctly")
	}
}
