//go:build ci
// +build ci

package xdp

import (
	"testing"

	"marchproxy-l3l4/internal/logging"
)

func TestNewHandler(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	handler, err := NewHandler("eth0", logger)

	if err != nil {
		t.Errorf("NewHandler failed: %v", err)
	}

	if handler == nil {
		t.Fatal("NewHandler returned nil handler")
	}
}

func TestHandlerStart(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	handler, _ := NewHandler("eth0", logger)

	err := handler.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}
}

func TestHandlerStop(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	handler, _ := NewHandler("eth0", logger)
	handler.Start()

	handler.Stop()
	// Stop should not error
}

func TestHandlerGetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	handler, _ := NewHandler("eth0", logger)
	handler.Start()

	stats := handler.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	// Verify expected keys exist
	expectedKeys := []string{"device", "loaded", "packets_processed"}
	for _, key := range expectedKeys {
		if _, ok := stats[key]; !ok {
			t.Errorf("GetStats missing key: %s", key)
		}
	}
}

func TestHandlerStartMultiple(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	handler, _ := NewHandler("eth0", logger)

	// First start should succeed
	err := handler.Start()
	if err != nil {
		t.Errorf("First Start failed: %v", err)
	}

	// Second start should fail (already loaded)
	err = handler.Start()
	if err == nil {
		t.Error("Second Start should fail with already loaded error")
	}
}

func TestHandlerStopWithoutStart(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	handler, _ := NewHandler("eth0", logger)

	// Stop without start should not panic
	handler.Stop()
}

func TestHandlerStatsContainDevice(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	deviceName := "eth1"
	handler, _ := NewHandler(deviceName, logger)
	handler.Start()

	stats := handler.GetStats()
	if device, ok := stats["device"]; ok {
		if device != deviceName {
			t.Errorf("Stats device %v != expected %s", device, deviceName)
		}
	} else {
		t.Error("Stats missing device field")
	}
}

func TestHandlerStatsLoaded(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	handler, _ := NewHandler("eth0", logger)

	// Before start, should not be loaded
	stats := handler.GetStats()
	if loaded, ok := stats["loaded"].(bool); ok && loaded {
		t.Error("Handler should not be loaded before Start()")
	}

	// After start, should be loaded
	handler.Start()
	stats = handler.GetStats()
	if loaded, ok := stats["loaded"].(bool); !ok || !loaded {
		t.Error("Handler should be loaded after Start()")
	}

	// After stop, should not be loaded
	handler.Stop()
	stats = handler.GetStats()
	if loaded, ok := stats["loaded"].(bool); ok && loaded {
		t.Error("Handler should not be loaded after Stop()")
	}
}
