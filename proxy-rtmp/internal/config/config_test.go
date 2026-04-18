//go:build ci
// +build ci

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

// Table-driven tests for port validation
func TestValidatePortValidation(t *testing.T) {
	tests := []struct {
		name      string
		port      int
		grpcPort  int
		wantError bool
	}{
		{"valid ports", 1935, 50053, false},
		{"port 1", 1, 50053, false},
		{"port 65535", 65535, 50053, false},
		{"port 0", 0, 50053, true},
		{"port negative", -1, 50053, true},
		{"port 65536", 65536, 50053, true},
		{"grpc port 0", 1935, 0, true},
		{"grpc port 65536", 1935, 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:             tt.port,
				GRPCPort:         tt.grpcPort,
				SegmentDuration:  6,
				Encoder:          "x264",
				Preset:           "medium",
				MaxResolutionCPU: 1440,
				MaxResolutionGPU: 4320,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

// Test all valid encoders
func TestValidateAllValidEncoders(t *testing.T) {
	encoders := []string{"auto", "x264", "x265", "nvenc_h264", "nvenc_h265", "amf_h264", "amf_h265", "libaom_av1", "svt_av1", "nvenc_av1", "amf_av1"}

	for _, encoder := range encoders {
		t.Run(encoder, func(t *testing.T) {
			cfg := &config.Config{
				Port:             1935,
				GRPCPort:         50053,
				SegmentDuration:  6,
				Encoder:          encoder,
				Preset:           "medium",
				MaxResolutionCPU: 1440,
				MaxResolutionGPU: 4320,
			}

			if err := cfg.Validate(); err != nil {
				t.Errorf("expected valid encoder %s, got error: %v", encoder, err)
			}
		})
	}
}

// Test all valid presets
func TestValidateAllValidPresets(t *testing.T) {
	presets := []string{"ultrafast", "superfast", "veryfast", "faster", "fast", "medium", "slow", "slower", "veryslow"}

	for _, preset := range presets {
		t.Run(preset, func(t *testing.T) {
			cfg := &config.Config{
				Port:             1935,
				GRPCPort:         50053,
				SegmentDuration:  6,
				Encoder:          "x264",
				Preset:           preset,
				MaxResolutionCPU: 1440,
				MaxResolutionGPU: 4320,
			}

			if err := cfg.Validate(); err != nil {
				t.Errorf("expected valid preset %s, got error: %v", preset, err)
			}
		})
	}
}

// Test segment duration validation
func TestValidateSegmentDuration(t *testing.T) {
	tests := []struct {
		name      string
		duration  int
		wantError bool
	}{
		{"duration 1", 1, false},
		{"duration 30", 30, false},
		{"duration 60", 60, false},
		{"duration 0", 0, true},
		{"duration -1", -1, true},
		{"duration 61", 61, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:             1935,
				GRPCPort:         50053,
				SegmentDuration:  tt.duration,
				Encoder:          "x264",
				Preset:           "medium",
				MaxResolutionCPU: 1440,
				MaxResolutionGPU: 4320,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

// Test resolution limit validation
func TestValidateResolutionLimits(t *testing.T) {
	tests := []struct {
		name         string
		cpuResolution int
		gpuResolution int
		wantError    bool
	}{
		{"valid 1440/4320", 1440, 4320, false},
		{"valid 360/360", 360, 360, false},
		{"valid 4320/4320", 4320, 4320, false},
		{"cpu too low", 359, 4320, true},
		{"cpu too high", 4321, 4320, true},
		{"gpu too low", 1440, 359, true},
		{"gpu too high", 1440, 4321, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:             1935,
				GRPCPort:         50053,
				SegmentDuration:  6,
				Encoder:          "x264",
				Preset:           "medium",
				MaxResolutionCPU: tt.cpuResolution,
				MaxResolutionGPU: tt.gpuResolution,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

// Test SRT validation with comprehensive cases
func TestValidateSRTComprehensive(t *testing.T) {
	tests := []struct {
		name      string
		enableSRT bool
		port      int
		latency   int
		pbklen    int
		wantError bool
	}{
		{"srt disabled", false, 0, 0, 99, false},
		{"srt valid", true, 8890, 120, 16, false},
		{"srt valid pbklen 24", true, 8890, 120, 24, false},
		{"srt valid pbklen 32", true, 8890, 120, 32, false},
		{"srt valid pbklen 0", true, 8890, 120, 0, false},
		{"srt port 0", true, 0, 120, 16, true},
		{"srt port 65536", true, 65536, 120, 16, true},
		{"srt latency 19", true, 8890, 19, 16, true},
		{"srt latency 8001", true, 8890, 8001, 16, true},
		{"srt pbklen invalid", true, 8890, 120, 17, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:             1935,
				GRPCPort:         50053,
				SegmentDuration:  6,
				Encoder:          "x264",
				Preset:           "medium",
				MaxResolutionCPU: 1440,
				MaxResolutionGPU: 4320,
				EnableSRT:        tt.enableSRT,
				SRTPort:          tt.port,
				SRTLatency:       tt.latency,
				SRTPBKLen:        tt.pbklen,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

// Test WebRTC validation
func TestValidateWebRTCComprehensive(t *testing.T) {
	tests := []struct {
		name      string
		enableWRC bool
		whipPort  int
		whepPort  int
		icePolicy string
		wantError bool
	}{
		{"webrtc disabled", false, 0, 0, "invalid", false},
		{"webrtc valid all", true, 8080, 8081, "all", false},
		{"webrtc valid relay", true, 8080, 8081, "relay", false},
		{"webrtc invalid port", true, 0, 8081, "all", true},
		{"webrtc invalid policy", true, 8080, 8081, "invalid", true},
		{"webrtc port 65536", true, 65536, 8081, "all", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:             1935,
				GRPCPort:         50053,
				SegmentDuration:  6,
				Encoder:          "x264",
				Preset:           "medium",
				MaxResolutionCPU: 1440,
				MaxResolutionGPU: 4320,
				EnableWebRTC:     tt.enableWRC,
				WHIPPort:         tt.whipPort,
				WHEPPort:         tt.whepPort,
				WebRTCICEPolicy:  tt.icePolicy,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

// Test transcode ladder resolution validation
func TestValidateTranscodeLadderResolutions(t *testing.T) {
	tests := []struct {
		name        string
		resolutions []int
		wantError   bool
	}{
		{"valid standard", []int{360, 540, 720, 1080}, false},
		{"valid single", []int{720}, false},
		{"valid all", []int{360, 480, 540, 720, 1080, 1440, 2160, 4320}, false},
		{"invalid resolution", []int{360, 540, 999}, true},
		{"invalid negative", []int{-360}, true},
		{"mixed valid invalid", []int{360, 720, 999}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Port:                      1935,
				GRPCPort:                  50053,
				SegmentDuration:           6,
				Encoder:                   "x264",
				Preset:                    "medium",
				MaxResolutionCPU:          1440,
				MaxResolutionGPU:          4320,
				TranscodeLadderResolutions: tt.resolutions,
			}

			err := cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, want error = %v", err, tt.wantError)
			}
		})
	}
}

// Test env var override for numeric values
func TestEnvVarOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("RTMP_OUTPUT_DIR", tmpDir)
	t.Setenv("RTMP_PORT", "2000")
	t.Setenv("RTMP_GRPC_PORT", "50054")
	t.Setenv("RTMP_LOG_LEVEL", "debug")
	t.Setenv("RTMP_ENCODER", "x265")
	t.Setenv("RTMP_PRESET", "fast")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Port != 2000 {
		t.Errorf("expected Port 2000 from env var, got %d", cfg.Port)
	}
	if cfg.GRPCPort != 50054 {
		t.Errorf("expected GRPCPort 50054 from env var, got %d", cfg.GRPCPort)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel debug, got %q", cfg.LogLevel)
	}
	if cfg.Encoder != "x265" {
		t.Errorf("expected Encoder x265, got %q", cfg.Encoder)
	}
	if cfg.Preset != "fast" {
		t.Errorf("expected Preset fast, got %q", cfg.Preset)
	}
}
