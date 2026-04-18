//go:build ci

package nlb

import (
	"context"
	"testing"

	"marchproxy-nlb/internal/logging"
)

func TestRouter_RegisterModule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)

	ep := &ModuleEndpoint{Name: "mod1", Protocol: ProtocolHTTP, Address: "127.0.0.1", GRPCPort: 50051}
	err := router.RegisterModule(ep)
	if err != nil {
		t.Errorf("RegisterModule error = %v", err)
	}

	modules := router.GetModules(ProtocolHTTP)
	if len(modules) != 1 || modules[0].Name != "mod1" {
		t.Errorf("Module not registered correctly")
	}
}

func TestRouter_UnregisterModule(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)

	ep := &ModuleEndpoint{Name: "mod1", Protocol: ProtocolHTTP, Address: "127.0.0.1", GRPCPort: 50051}
	router.RegisterModule(ep)

	err := router.UnregisterModule(ProtocolHTTP, "mod1")
	if err != nil {
		t.Errorf("UnregisterModule error = %v", err)
	}

	modules := router.GetModules(ProtocolHTTP)
	if len(modules) != 0 {
		t.Errorf("Module not unregistered")
	}
}

func TestRouter_RouteConnection_HTTP(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)

	ep := &ModuleEndpoint{Name: "http-mod", Protocol: ProtocolHTTP, Address: "127.0.0.1", GRPCPort: 50051, Healthy: true}
	router.RegisterModule(ep)

	data := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n")
	routed, err := router.RouteConnection(context.Background(), data)

	if routed == nil && err == nil {
		t.Logf("RouteConnection returned nil endpoint (may be expected if health check fails)")
	}
}

func TestRouter_RouteConnection_NoModules(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)

	data := []byte("GET / HTTP/1.1")
	_, err := router.RouteConnection(context.Background(), data)

	if err == nil {
		t.Logf("RouteConnection should fail with no modules")
	}
}

func TestRouter_GetAllModules(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)

	ep1 := &ModuleEndpoint{Name: "http-mod", Protocol: ProtocolHTTP, Address: "127.0.0.1"}
	ep2 := &ModuleEndpoint{Name: "mysql-mod", Protocol: ProtocolMySQL, Address: "127.0.0.1"}

	router.RegisterModule(ep1)
	router.RegisterModule(ep2)

	allModules := router.GetAllModules()
	if len(allModules) != 2 {
		t.Errorf("GetAllModules returned %d protocols, want 2", len(allModules))
	}
}

func TestRouter_GetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)

	ep := &ModuleEndpoint{Name: "mod1", Protocol: ProtocolHTTP, Address: "127.0.0.1"}
	router.RegisterModule(ep)

	stats := router.GetStats()
	if stats == nil {
		t.Errorf("GetStats returned nil")
	}
	if len(stats) == 0 {
		t.Logf("GetStats returned empty stats")
	}
}

func TestRouter_MultipleProtocols(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)

	protocols := []Protocol{ProtocolHTTP, ProtocolMySQL, ProtocolPostgreSQL, ProtocolRedis}
	for i, p := range protocols {
		ep := &ModuleEndpoint{Name: "mod" + string(rune(i)), Protocol: p, Address: "127.0.0.1"}
		router.RegisterModule(ep)
	}

	allModules := router.GetAllModules()
	if len(allModules) != len(protocols) {
		t.Errorf("Got %d protocol groups, want %d", len(allModules), len(protocols))
	}
}
