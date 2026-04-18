//go:build ci
// +build ci

package xdp

import (
	"testing"

	"marchproxy-l3l4/internal/logging"
)

func TestNewXDPProgram(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	if program == nil {
		t.Fatal("NewXDPProgram returned nil")
	}

	if program.device != "eth0" {
		t.Errorf("Program device = %q, want %q", program.device, "eth0")
	}
}

func TestXDPProgramLoad(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	err := program.Load("")
	if err != nil {
		t.Errorf("Load failed: %v", err)
	}

	if !program.IsLoaded() {
		t.Error("Program should be loaded after Load()")
	}
}

func TestXDPProgramLoadAlreadyLoaded(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	program.Load("")
	err := program.Load("")
	if err == nil {
		t.Error("Load should fail when already loaded")
	}
}

func TestXDPProgramUnload(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	program.Load("")
	err := program.Unload()
	if err != nil {
		t.Errorf("Unload failed: %v", err)
	}

	if program.IsLoaded() {
		t.Error("Program should not be loaded after Unload()")
	}
}

func TestXDPProgramUnloadNotLoaded(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	// Unload without Load should succeed (idempotent)
	err := program.Unload()
	if err != nil {
		t.Errorf("Unload without load should succeed: %v", err)
	}
}

func TestXDPProgramIsLoaded(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	if program.IsLoaded() {
		t.Error("Newly created program should not be loaded")
	}

	program.Load("")
	if !program.IsLoaded() {
		t.Error("Program should be loaded after Load()")
	}

	program.Unload()
	if program.IsLoaded() {
		t.Error("Program should not be loaded after Unload()")
	}
}

func TestXDPProgramGetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	stats := program.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	// Check expected fields
	expectedFields := []string{"device", "loaded", "packets_processed", "packets_dropped", "bytes_processed", "stub"}
	for _, field := range expectedFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("GetStats missing field: %s", field)
		}
	}

	// Check device field value
	if device, ok := stats["device"]; ok {
		if device != "eth0" {
			t.Errorf("Stats device = %v, want eth0", device)
		}
	}

	// Check loaded field
	if loaded, ok := stats["loaded"].(bool); ok {
		if loaded {
			t.Error("Stats loaded should be false before Load()")
		}
	}

	// Check stub flag
	if stub, ok := stats["stub"].(bool); ok {
		if !stub {
			t.Error("Stats stub should be true for CI build")
		}
	}
}

func TestXDPProgramGetStatsAfterLoad(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	program.Load("")
	stats := program.GetStats()

	if loaded, ok := stats["loaded"].(bool); ok {
		if !loaded {
			t.Error("Stats loaded should be true after Load()")
		}
	}
}

func TestXDPProgramUpdateStatsNotLoaded(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	err := program.UpdateStats()
	if err == nil {
		t.Error("UpdateStats should fail when not loaded")
	}
}

func TestXDPProgramUpdateStatsLoaded(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	program.Load("")
	err := program.UpdateStats()
	if err != nil {
		t.Errorf("UpdateStats should succeed when loaded: %v", err)
	}
}

func TestXDPProgramThreadSafety(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	program := NewXDPProgram("eth0", logger)

	// Concurrent calls should not panic
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			program.Load("")
			program.IsLoaded()
			program.GetStats()
			program.UpdateStats()
			program.Unload()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
