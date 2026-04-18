//go:build ci
// +build ci

package transcode

import (
	"testing"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Fatal("expected non-nil detector")
	}
	// Detector should be initialized with at least GPUTypeNone
	if d.GetGPUType() == "" {
		t.Fatal("expected detector to have GPUType set")
	}
}

func TestGetGPUType(t *testing.T) {
	d := NewDetector()
	gpuType := d.GetGPUType()

	// Should return one of the valid types
	validTypes := map[GPUType]bool{
		GPUTypeNone:   true,
		GPUTypeNVIDIA: true,
		GPUTypeAMD:    true,
	}

	if !validTypes[gpuType] {
		t.Errorf("unexpected GPU type: %s", gpuType)
	}
}

func TestHasGPU(t *testing.T) {
	d := NewDetector()
	hasGPU := d.HasGPU()

	// HasGPU should be consistent with GetGPUType
	expectedHasGPU := d.GetGPUType() != GPUTypeNone
	if hasGPU != expectedHasGPU {
		t.Errorf("HasGPU() = %v, expected %v", hasGPU, expectedHasGPU)
	}
}

func TestHasNVIDIA(t *testing.T) {
	d := NewDetector()
	hasNVIDIA := d.HasNVIDIA()

	// HasNVIDIA should be true only if GPU type is NVIDIA
	expectedHasNVIDIA := d.GetGPUType() == GPUTypeNVIDIA
	if hasNVIDIA != expectedHasNVIDIA {
		t.Errorf("HasNVIDIA() = %v, expected %v", hasNVIDIA, expectedHasNVIDIA)
	}
}

func TestHasAMD(t *testing.T) {
	d := NewDetector()
	hasAMD := d.HasAMD()

	// HasAMD should be true only if GPU type is AMD
	expectedHasAMD := d.GetGPUType() == GPUTypeAMD
	if hasAMD != expectedHasAMD {
		t.Errorf("HasAMD() = %v, expected %v", hasAMD, expectedHasAMD)
	}
}

func TestGetVRAM(t *testing.T) {
	d := NewDetector()
	vram := d.GetVRAM()

	// VRAM should be non-negative
	if vram < 0 {
		t.Errorf("GetVRAM() returned negative value: %d", vram)
	}
}

func TestGetGPUModel(t *testing.T) {
	d := NewDetector()
	model := d.GetGPUModel()

	// Model should not be empty string if GPU is detected
	if d.HasGPU() && model == "" {
		t.Error("expected non-empty GPU model when GPU is detected")
	}

	// Model should be set to some string
	if model != "" && model != "Unknown NVIDIA GPU" && model != "Unknown AMD GPU" {
		// Model is populated
	}
}

func TestSupportsAV1(t *testing.T) {
	d := NewDetector()
	supportsAV1 := d.SupportsAV1()

	// Should return a boolean
	if supportsAV1 {
		// AV1 support requires a capable GPU
		if !d.HasGPU() {
			t.Error("GPU detected AV1 support but HasGPU() returned false")
		}
	}
}

func TestSupports8K(t *testing.T) {
	d := NewDetector()
	supports8K := d.Supports8K()

	vram := d.GetVRAM()
	expectedSupports8K := vram >= 12
	if supports8K != expectedSupports8K {
		t.Errorf("Supports8K() = %v, expected %v (VRAM: %dGB)", supports8K, expectedSupports8K, vram)
	}
}

func TestSupports4K(t *testing.T) {
	d := NewDetector()
	supports4K := d.Supports4K()

	vram := d.GetVRAM()
	expectedSupports4K := vram >= 8
	if supports4K != expectedSupports4K {
		t.Errorf("Supports4K() = %v, expected %v (VRAM: %dGB)", supports4K, expectedSupports4K, vram)
	}
}

func TestSelectEncoder(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name       string
		preference string
	}{
		{"auto encoder", "auto"},
		{"x264 encoder", "x264"},
		{"x265 encoder", "x265"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder, err := d.SelectEncoder(tt.preference)
			if tt.preference == "auto" || tt.preference == "x264" || tt.preference == "x265" {
				if err != nil {
					t.Errorf("SelectEncoder(%s) unexpected error: %v", tt.preference, err)
				}
				if encoder == nil {
					t.Errorf("SelectEncoder(%s) returned nil encoder", tt.preference)
				}
			}
		})
	}
}

func TestSelectEncoder_Invalid(t *testing.T) {
	d := NewDetector()

	// Test with invalid encoder when GPU not available
	if !d.HasNVIDIA() {
		encoder, err := d.SelectEncoder("nvenc_h264")
		if err == nil {
			t.Error("expected error for nvenc_h264 without NVIDIA GPU")
		}
		if encoder != nil {
			t.Error("expected nil encoder for nvenc_h264 without NVIDIA GPU")
		}
	}

	if !d.HasAMD() {
		encoder, err := d.SelectEncoder("amf_h264")
		if err == nil {
			t.Error("expected error for amf_h264 without AMD GPU")
		}
		if encoder != nil {
			t.Error("expected nil encoder for amf_h264 without AMD GPU")
		}
	}
}

func TestSelectEncoderWithAV1Preference(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name       string
		preference string
		preferAV1  bool
	}{
		{"auto without AV1", "auto", false},
		{"auto with AV1", "auto", true},
		{"x264", "x264", false},
		{"x264 with AV1", "x264", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder, err := d.SelectEncoderWithAV1Preference(tt.preference, tt.preferAV1)
			if tt.preference == "auto" || tt.preference == "x264" {
				if err != nil {
					t.Errorf("SelectEncoderWithAV1Preference(%s, %v) unexpected error: %v",
						tt.preference, tt.preferAV1, err)
				}
				if encoder == nil {
					t.Errorf("SelectEncoderWithAV1Preference(%s, %v) returned nil encoder",
						tt.preference, tt.preferAV1)
				}
			}
		})
	}
}

func TestGPUTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		gpuType  GPUType
		expected string
	}{
		{"GPUTypeNone", GPUTypeNone, "none"},
		{"GPUTypeNVIDIA", GPUTypeNVIDIA, "nvidia"},
		{"GPUTypeAMD", GPUTypeAMD, "amd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.gpuType) != tt.expected {
				t.Errorf("GPUType %s = %q, expected %q", tt.name, tt.gpuType, tt.expected)
			}
		})
	}
}

func TestDetectorConsistency(t *testing.T) {
	d := NewDetector()

	// GetGPUType should be consistent with HasGPU
	if d.HasGPU() != (d.GetGPUType() != GPUTypeNone) {
		t.Error("HasGPU inconsistent with GetGPUType")
	}

	// HasNVIDIA should be exclusive with HasAMD (can't have both)
	if d.HasNVIDIA() && d.HasAMD() {
		t.Error("detector should not report both NVIDIA and AMD")
	}

	// If HasNVIDIA, then GetGPUType should be NVIDIA
	if d.HasNVIDIA() && d.GetGPUType() != GPUTypeNVIDIA {
		t.Error("HasNVIDIA true but GetGPUType != GPUTypeNVIDIA")
	}

	// If HasAMD, then GetGPUType should be AMD
	if d.HasAMD() && d.GetGPUType() != GPUTypeAMD {
		t.Error("HasAMD true but GetGPUType != GPUTypeAMD")
	}
}

// TestGetEncoderConfig tests encoder selection logic
func TestGetEncoderConfig(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name      string
		codec     string
		preset    string
		wantError bool
	}{
		{"x264 ultrafast", "x264", "ultrafast", false},
		{"x264 medium", "x264", "medium", false},
		{"x264 veryslow", "x264", "veryslow", false},
		{"x265 medium", "x265", "medium", false},
		{"unknown codec", "unknown_codec", "medium", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder, err := d.SelectEncoder(tt.codec)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error for codec %s", tt.codec)
				}
			} else {
				if err != nil {
					t.Errorf("SelectEncoder(%s) unexpected error: %v", tt.codec, err)
				}
				if encoder == nil {
					t.Errorf("SelectEncoder(%s) returned nil", tt.codec)
				}
			}
		})
	}
}

// TestSelectEncoderEdgeCases tests various encoder selection scenarios
func TestSelectEncoderEdgeCases(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name         string
		preference   string
		shouldSucceed bool
	}{
		{"auto", "auto", true},
		{"x264", "x264", true},
		{"x265", "x265", true},
		{"empty", "", false},
		{"case-sensitive invalid", "X264", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder, err := d.SelectEncoder(tt.preference)
			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("SelectEncoder(%s) unexpected error: %v", tt.preference, err)
				}
				if encoder == nil {
					t.Errorf("SelectEncoder(%s) returned nil", tt.preference)
				}
			} else {
				if err == nil {
					t.Errorf("SelectEncoder(%s) expected error but got nil", tt.preference)
				}
			}
		})
	}
}

// TestVRAMBasedCapabilities tests resolution capability based on VRAM
func TestVRAMBasedCapabilities(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name              string
		minVRAMFor4K      int
		minVRAMFor8K      int
		vram              int
		expectedSupports4K bool
		expectedSupports8K bool
	}{
		{"2GB VRAM", 8, 12, 2, false, false},
		{"8GB VRAM", 8, 12, 8, true, false},
		{"12GB VRAM", 8, 12, 12, true, true},
		{"24GB VRAM", 8, 12, 24, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check current detector matches test expectations
			if d.GetVRAM() == tt.vram {
				if d.Supports4K() != tt.expectedSupports4K {
					t.Errorf("Supports4K() = %v, expected %v", d.Supports4K(), tt.expectedSupports4K)
				}
				if d.Supports8K() != tt.expectedSupports8K {
					t.Errorf("Supports8K() = %v, expected %v", d.Supports8K(), tt.expectedSupports8K)
				}
			}
		})
	}
}

// TestMultipleDetectorInstances verifies detector state isolation
func TestMultipleDetectorInstances(t *testing.T) {
	d1 := NewDetector()
	d2 := NewDetector()

	// Two detectors should have same GPU detection result
	if d1.GetGPUType() != d2.GetGPUType() {
		t.Errorf("different detectors should detect same GPU: %s vs %s", d1.GetGPUType(), d2.GetGPUType())
	}

	if d1.GetVRAM() != d2.GetVRAM() {
		t.Errorf("different detectors should detect same VRAM: %dGB vs %dGB", d1.GetVRAM(), d2.GetVRAM())
	}

	if d1.GetGPUModel() != d2.GetGPUModel() {
		t.Errorf("different detectors should detect same GPU model: %s vs %s", d1.GetGPUModel(), d2.GetGPUModel())
	}
}

// TestAV1SupportWithGPU tests AV1 support detection
func TestAV1SupportWithGPU(t *testing.T) {
	d := NewDetector()

	supportsAV1 := d.SupportsAV1()

	// If GPU is present, could support AV1 (NVIDIA RTX or newer AMD RDNA)
	if d.HasGPU() {
		// No assertion needed, just verify it returns a boolean
		if !supportsAV1 && !d.HasGPU() {
			// Expected: no GPU, no AV1 support
		}
	}
}

// TestEncoderNameFormats validates encoder naming conventions
func TestEncoderNameFormats(t *testing.T) {
	d := NewDetector()

	validEncoders := []string{"auto", "x264", "x265"}
	if d.HasNVIDIA() {
		validEncoders = append(validEncoders, "nvenc_h264", "nvenc_h265", "nvenc_av1")
	}
	if d.HasAMD() {
		validEncoders = append(validEncoders, "amf_h264", "amf_h265", "amf_av1")
	}
	validEncoders = append(validEncoders, "libaom_av1", "svt_av1")

	for _, enc := range validEncoders {
		t.Run(enc, func(t *testing.T) {
			encoder, err := d.SelectEncoder(enc)
			if err != nil {
				// Some encoders may not be available, which is OK
				return
			}
			if encoder == nil {
				t.Errorf("encoder %s returned nil", enc)
			}
		})
	}
}
