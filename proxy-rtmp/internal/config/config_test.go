package config_test

import (
	"os"
	"testing"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
)

// TestLoadDefaults verifies that Load("") returns expected default values
// when no config file and no conflicting env vars are present.
func TestLoadDefaults(t *testing.T) {
	// Create a temp dir for the output directory so MkdirAll succeeds.
	tmpDir := t.TempDir()
	t.Setenv("RTMP_OUTPUT_DIR", tmpDir)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected Host '0.0.0.0', got %q", cfg.Host)
	}
	if cfg.Port != 1935 {
		t.Errorf("expected Port 1935, got %d", cfg.Port)
	}
	if cfg.GRPCPort != 50053 {
		t.Errorf("expected GRPCPort 50053, got %d", cfg.GRPCPort)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel 'info', got %q", cfg.LogLevel)
	}
}

func TestEncoderTypeConstants(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RTMP_OUTPUT_DIR", tmpDir)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Encoder != "auto" {
		t.Errorf("expected default Encoder 'auto', got %q", cfg.Encoder)
	}
	if cfg.Preset != "medium" {
		t.Errorf("expected default Preset 'medium', got %q", cfg.Preset)
	}
}

func TestSRTDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RTMP_OUTPUT_DIR", tmpDir)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.EnableSRT {
		t.Error("expected EnableSRT false by default")
	}
	if cfg.SRTPort != 8890 {
		t.Errorf("expected SRTPort 8890, got %d", cfg.SRTPort)
	}
	if cfg.SRTLatency != 120 {
		t.Errorf("expected SRTLatency 120, got %d", cfg.SRTLatency)
	}
	if cfg.SRTPBKLen != 16 {
		t.Errorf("expected SRTPBKLen 16, got %d", cfg.SRTPBKLen)
	}
}

func TestWebRTCDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RTMP_OUTPUT_DIR", tmpDir)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.EnableWebRTC {
		t.Error("expected EnableWebRTC false by default")
	}
	if cfg.WHIPPort != 8080 {
		t.Errorf("expected WHIPPort 8080, got %d", cfg.WHIPPort)
	}
	if cfg.WHEPPort != 8081 {
		t.Errorf("expected WHEPPort 8081, got %d", cfg.WHEPPort)
	}
	if cfg.WebRTCICEPolicy != "all" {
		t.Errorf("expected WebRTCICEPolicy 'all', got %q", cfg.WebRTCICEPolicy)
	}
}

func TestHLSDASHDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RTMP_OUTPUT_DIR", tmpDir)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if !cfg.EnableHLS {
		t.Error("expected EnableHLS true by default")
	}
	if !cfg.EnableDASH {
		t.Error("expected EnableDASH true by default")
	}
	if cfg.SegmentDuration != 6 {
		t.Errorf("expected SegmentDuration 6, got %d", cfg.SegmentDuration)
	}
}

func TestResolutionLimitDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RTMP_OUTPUT_DIR", tmpDir)

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.MaxResolutionCPU != 1440 {
		t.Errorf("expected MaxResolutionCPU 1440, got %d", cfg.MaxResolutionCPU)
	}
	if cfg.MaxResolutionGPU != 4320 {
		t.Errorf("expected MaxResolutionGPU 4320, got %d", cfg.MaxResolutionGPU)
	}
}

func TestValidateValidConfig(t *testing.T) {
	cfg := &config.Config{
		Port:             1935,
		GRPCPort:         50053,
		SegmentDuration:  6,
		Encoder:          "auto",
		Preset:           "medium",
		MaxResolutionCPU: 1440,
		MaxResolutionGPU: 4320,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidateInvalidEncoder(t *testing.T) {
	cfg := &config.Config{
		Port:             1935,
		GRPCPort:         50053,
		SegmentDuration:  6,
		Encoder:          "invalid-encoder",
		Preset:           "medium",
		MaxResolutionCPU: 1440,
		MaxResolutionGPU: 4320,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid encoder, got nil")
	}
}

func TestValidateInvalidPreset(t *testing.T) {
	cfg := &config.Config{
		Port:             1935,
		GRPCPort:         50053,
		SegmentDuration:  6,
		Encoder:          "x264",
		Preset:           "turbo",
		MaxResolutionCPU: 1440,
		MaxResolutionGPU: 4320,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid preset, got nil")
	}
}

func TestValidateSRTSettingsEnabled(t *testing.T) {
	cfg := &config.Config{
		Port:             1935,
		GRPCPort:         50053,
		SegmentDuration:  6,
		Encoder:          "x264",
		Preset:           "medium",
		MaxResolutionCPU: 1440,
		MaxResolutionGPU: 4320,
		EnableSRT:        true,
		SRTPort:          8890,
		SRTLatency:       120,
		SRTPBKLen:        16,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid SRT config, got: %v", err)
	}
}

func TestValidateSRTInvalidPort(t *testing.T) {
	cfg := &config.Config{
		Port:             1935,
		GRPCPort:         50053,
		SegmentDuration:  6,
		Encoder:          "x264",
		Preset:           "medium",
		MaxResolutionCPU: 1440,
		MaxResolutionGPU: 4320,
		EnableSRT:        true,
		SRTPort:          0,
		SRTLatency:       120,
		SRTPBKLen:        16,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for SRT port 0, got nil")
	}
}

func TestOutputDirCreatedByLoad(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := tmpDir + "/nested/output/dir"

	// Must not exist yet.
	if _, err := os.Stat(subDir); !os.IsNotExist(err) {
		t.Skip("subdir already exists")
	}

	t.Setenv("RTMP_OUTPUT_DIR", subDir)

	_, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("expected Load() to create output directory")
	}
}
