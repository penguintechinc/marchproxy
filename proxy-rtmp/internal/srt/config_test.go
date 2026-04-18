//go:build ci
// +build ci

package srt

import (
	"testing"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
)

// TestNewSRTConfig tests SRT config creation
func TestNewSRTConfig(t *testing.T) {
	cfg := &config.Config{
		SRTPort:       8890,
		SRTLatency:    120,
		SRTPassphrase: "",
		SRTPBKLen:     16,
	}

	srtCfg := NewSRTConfig(cfg)
	if srtCfg == nil {
		t.Fatal("expected non-nil SRT config")
	}
	if srtCfg.Port != 8890 {
		t.Errorf("port mismatch: expected 8890, got %d", srtCfg.Port)
	}
	if srtCfg.Latency != 120 {
		t.Errorf("latency mismatch: expected 120, got %d", srtCfg.Latency)
	}
}

// TestSRTConfig_Validate_ValidPort tests port validation
func TestSRTConfig_Validate_ValidPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"minimum port", 1, false},
		{"standard SRT port", 8890, false},
		{"maximum port", 65535, false},
		{"port zero", 0, true},
		{"port negative", -1, true},
		{"port too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srtCfg := &SRTConfig{
				Port:      tt.port,
				Latency:   120,
				PBKeyLen:  0,
				RcvBufSize: 8192,
				SndBufSize: 8192,
			}

			err := srtCfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate error: expected %v, got %v", tt.wantErr, err != nil)
			}
		})
	}
}

// TestSRTConfig_Validate_ValidLatency tests latency validation
func TestSRTConfig_Validate_ValidLatency(t *testing.T) {
	tests := []struct {
		name      string
		latency   int
		wantErr   bool
	}{
		{"minimum latency", 20, false},
		{"standard latency", 120, false},
		{"maximum latency", 8000, false},
		{"latency too low", 19, true},
		{"latency too high", 8001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srtCfg := &SRTConfig{
				Port:      8890,
				Latency:   tt.latency,
				PBKeyLen:  0,
				RcvBufSize: 8192,
				SndBufSize: 8192,
			}

			err := srtCfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate error: expected %v, got %v", tt.wantErr, err != nil)
			}
		})
	}
}

// TestSRTConfig_Validate_PBKeyLen tests PBKLEN validation
func TestSRTConfig_Validate_PBKeyLen(t *testing.T) {
	tests := []struct {
		name    string
		pbklen  int
		wantErr bool
	}{
		{"no encryption", 0, false},
		{"16 byte key", 16, false},
		{"24 byte key", 24, false},
		{"32 byte key", 32, false},
		{"invalid length", 8, true},
		{"invalid length", 31, true},
		{"invalid length", 64, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srtCfg := &SRTConfig{
				Port:       8890,
				Latency:    120,
				PBKeyLen:   tt.pbklen,
				RcvBufSize: 8192,
				SndBufSize: 8192,
			}

			err := srtCfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate error: expected %v, got %v", tt.wantErr, err != nil)
			}
		})
	}
}

// TestSRTConfig_Validate_PassphraseWithoutPBKeyLen tests passphrase validation
func TestSRTConfig_Validate_PassphraseWithoutPBKeyLen(t *testing.T) {
	srtCfg := &SRTConfig{
		Port:        8890,
		Latency:     120,
		Passphrase:  "mypassword",
		PBKeyLen:    0, // Invalid: passphrase without key length
		RcvBufSize:  8192,
		SndBufSize:  8192,
	}

	err := srtCfg.Validate()
	if err == nil {
		t.Error("expected error when passphrase provided without PBKLEN")
	}
}

// TestSRTConfig_Validate_PassphraseWithPBKeyLen tests valid passphrase config
func TestSRTConfig_Validate_PassphraseWithPBKeyLen(t *testing.T) {
	srtCfg := &SRTConfig{
		Port:        8890,
		Latency:     120,
		Passphrase:  "mypassword",
		PBKeyLen:    16,
		RcvBufSize:  8192,
		SndBufSize:  8192,
	}

	err := srtCfg.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestBuildConnectOptions tests connection options string generation
func TestBuildConnectOptions(t *testing.T) {
	srtCfg := &SRTConfig{
		Port:       8890,
		Latency:    120,
		Passphrase: "",
		PBKeyLen:   0,
	}

	opts := srtCfg.BuildConnectOptions()
	if opts == "" {
		t.Error("expected non-empty options string")
	}
	if opts != "latency=120" {
		t.Errorf("unexpected options: %s", opts)
	}
}

// TestBuildConnectOptions_WithPassphrase tests options with encryption
func TestBuildConnectOptions_WithPassphrase(t *testing.T) {
	srtCfg := &SRTConfig{
		Port:        8890,
		Latency:     120,
		Passphrase:  "secret",
		PBKeyLen:    16,
		MaxBandwidth: 0,
	}

	opts := srtCfg.BuildConnectOptions()
	if !contains(opts, "latency=120") {
		t.Errorf("missing latency in options: %s", opts)
	}
	if !contains(opts, "passphrase=secret") {
		t.Errorf("missing passphrase in options: %s", opts)
	}
	if !contains(opts, "pbkeylen=16") {
		t.Errorf("missing pbkeylen in options: %s", opts)
	}
}

// TestBuildConnectOptions_WithMaxBandwidth tests options with bandwidth limit
func TestBuildConnectOptions_WithMaxBandwidth(t *testing.T) {
	srtCfg := &SRTConfig{
		Port:         8890,
		Latency:      120,
		Passphrase:   "",
		PBKeyLen:     0,
		MaxBandwidth: 10000000, // 10 Mbps
	}

	opts := srtCfg.BuildConnectOptions()
	if !contains(opts, "latency=120") {
		t.Errorf("missing latency in options: %s", opts)
	}
	if !contains(opts, "maxbw=10000000") {
		t.Errorf("missing maxbw in options: %s", opts)
	}
}

// TestDefaultSocketOptions tests default socket options generation
func TestDefaultSocketOptions(t *testing.T) {
	srtCfg := &SRTConfig{
		Port:       8890,
		Latency:    120,
		Passphrase: "",
		PBKeyLen:   0,
		RcvBufSize: 8192,
		SndBufSize: 8192,
	}

	opts := DefaultSocketOptions(srtCfg)
	if opts == nil {
		t.Fatal("expected non-nil socket options")
	}
	if len(opts) == 0 {
		t.Error("expected non-empty socket options")
	}

	// Check for required options
	optNames := make(map[string]bool)
	for _, opt := range opts {
		optNames[opt.Name] = true
	}

	required := []string{"SRTO_LATENCY", "SRTO_RCVBUF", "SRTO_SNDBUF"}
	for _, req := range required {
		if !optNames[req] {
			t.Errorf("missing required option: %s", req)
		}
	}
}

// TestDefaultSocketOptions_WithPassphrase tests socket options with encryption
func TestDefaultSocketOptions_WithPassphrase(t *testing.T) {
	srtCfg := &SRTConfig{
		Port:       8890,
		Latency:    120,
		Passphrase: "secret",
		PBKeyLen:   16,
		RcvBufSize: 8192,
		SndBufSize: 8192,
	}

	opts := DefaultSocketOptions(srtCfg)
	if opts == nil {
		t.Fatal("expected non-nil socket options")
	}

	// Check for passphrase option
	hasPassphrase := false
	for _, opt := range opts {
		if opt.Name == "SRTO_PASSPHRASE" {
			hasPassphrase = true
			break
		}
	}
	if !hasPassphrase {
		t.Error("expected SRTO_PASSPHRASE option when passphrase is set")
	}
}

// TestSRTMode constants
func TestSRTMode_Constants(t *testing.T) {
	tests := []struct {
		mode  SRTMode
		value string
	}{
		{SRTModeListener, "listener"},
		{SRTModeCaller, "caller"},
		{SRTModeRendezvous, "rendezvous"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.value {
			t.Errorf("mode mismatch: expected %s, got %s", tt.value, tt.mode)
		}
	}
}

// TestNewSRTConfig_FromDifferentConfigs tests creating SRT configs from various configs
func TestNewSRTConfig_FromDifferentConfigs(t *testing.T) {
	configs := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "default",
			cfg: &config.Config{
				SRTPort:       8890,
				SRTLatency:    120,
				SRTPassphrase: "",
				SRTPBKLen:     16,
			},
		},
		{
			name: "custom port and latency",
			cfg: &config.Config{
				SRTPort:       9000,
				SRTLatency:    200,
				SRTPassphrase: "",
				SRTPBKLen:     0,
			},
		},
		{
			name: "with encryption",
			cfg: &config.Config{
				SRTPort:       8890,
				SRTLatency:    120,
				SRTPassphrase: "mykey",
				SRTPBKLen:     32,
			},
		},
	}

	for _, tt := range configs {
		t.Run(tt.name, func(t *testing.T) {
			srtCfg := NewSRTConfig(tt.cfg)
			if srtCfg == nil {
				t.Fatal("expected non-nil SRT config")
			}
			if srtCfg.Port != tt.cfg.SRTPort {
				t.Errorf("port mismatch: expected %d, got %d", tt.cfg.SRTPort, srtCfg.Port)
			}
			if srtCfg.Latency != tt.cfg.SRTLatency {
				t.Errorf("latency mismatch: expected %d, got %d", tt.cfg.SRTLatency, srtCfg.Latency)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
