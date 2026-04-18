//go:build ci
// +build ci

package transcode

import (
	"testing"
)

func TestResolutionConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"360p", Resolution360p, 360},
		{"480p", Resolution480p, 480},
		{"540p", Resolution540p, 540},
		{"720p", Resolution720p, 720},
		{"1080p", Resolution1080p, 1080},
		{"1440p", Resolution1440p, 1440},
		{"2160p", Resolution2160p, 2160},
		{"4320p", Resolution4320p, 4320},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.value)
			}
		})
	}
}

func TestNewResolutionPolicy_CPUOnly(t *testing.T) {
	// Create detector with no GPU
	detector := &Detector{
		gpuType: GPUTypeNone,
	}

	policy := NewResolutionPolicy(detector)
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}

	if policy.GPUType != GPUTypeNone {
		t.Errorf("expected GPUTypeNone, got %s", policy.GPUType)
	}

	if policy.HardwareMax != DefaultMaxResolutionCPU {
		t.Errorf("expected hardware max %d, got %d", DefaultMaxResolutionCPU, policy.HardwareMax)
	}

	if policy.VRAMGB != 0 {
		t.Errorf("expected VRAM 0, got %d", policy.VRAMGB)
	}

	if policy.AV1Capable {
		t.Error("expected AV1Capable to be false")
	}
}

func TestNewResolutionPolicy_WithGPU(t *testing.T) {
	detector := &Detector{
		gpuType:    GPUTypeNVIDIA,
		hasNVIDIA:  true,
		vramGB:     8,
		av1Capable: true,
	}

	policy := NewResolutionPolicy(detector)
	if policy == nil {
		t.Fatal("expected non-nil policy")
	}

	if policy.GPUType != GPUTypeNVIDIA {
		t.Errorf("expected GPUTypeNVIDIA, got %s", policy.GPUType)
	}

	if policy.VRAMGB != 8 {
		t.Errorf("expected VRAM 8, got %d", policy.VRAMGB)
	}

	if !policy.AV1Capable {
		t.Error("expected AV1Capable to be true")
	}
}

func TestSetAdminMax_AndClear(t *testing.T) {
	detector := &Detector{gpuType: GPUTypeNone}
	policy := NewResolutionPolicy(detector)

	if policy.AdminMax != nil {
		t.Error("expected AdminMax to be nil initially")
	}

	maxRes := 720
	policy.SetAdminMax(&maxRes)

	if policy.AdminMax == nil {
		t.Fatal("expected AdminMax to be set")
	}
	if *policy.AdminMax != 720 {
		t.Errorf("expected AdminMax 720, got %d", *policy.AdminMax)
	}

	policy.ClearAdminMax()
	if policy.AdminMax != nil {
		t.Error("expected AdminMax to be nil after clear")
	}
}

func TestEffectiveMax(t *testing.T) {
	tests := []struct {
		name        string
		hardwareMax int
		adminMax    *int
		expected    int
	}{
		{
			name:        "no admin limit",
			hardwareMax: 1080,
			adminMax:    nil,
			expected:    1080,
		},
		{
			name:        "admin limit lower than hardware",
			hardwareMax: 1080,
			adminMax:    intPtr(720),
			expected:    720,
		},
		{
			name:        "admin limit higher than hardware",
			hardwareMax: 1080,
			adminMax:    intPtr(2160),
			expected:    1080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &ResolutionPolicy{
				HardwareMax: tt.hardwareMax,
				AdminMax:    tt.adminMax,
			}

			max := policy.EffectiveMax()
			if max != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, max)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		policy    *ResolutionPolicy
		inputRes  int
		wantErr   bool
	}{
		{
			name: "valid resolution",
			policy: &ResolutionPolicy{
				HardwareMax: 1080,
				GPUType:     GPUTypeNone,
				VRAMGB:      0,
			},
			inputRes: 720,
			wantErr:  false,
		},
		{
			name: "resolution at limit",
			policy: &ResolutionPolicy{
				HardwareMax: 1080,
				GPUType:     GPUTypeNone,
				VRAMGB:      0,
			},
			inputRes: 1080,
			wantErr:  false,
		},
		{
			name: "resolution exceeds limit",
			policy: &ResolutionPolicy{
				HardwareMax: 1080,
				GPUType:     GPUTypeNone,
				VRAMGB:      0,
			},
			inputRes: 2160,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate(tt.inputRes)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !IsResolutionExceededError(err) {
				t.Errorf("expected ResolutionExceededError, got %T", err)
			}
		})
	}
}

func TestIsResolutionSupported(t *testing.T) {
	policy := &ResolutionPolicy{
		HardwareMax: 1080,
	}

	tests := []struct {
		res      int
		expected bool
	}{
		{360, true},
		{720, true},
		{1080, true},
		{1440, false},
		{2160, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := policy.IsResolutionSupported(tt.res)
			if result != tt.expected {
				t.Errorf("IsResolutionSupported(%d) = %v, expected %v", tt.res, result, tt.expected)
			}
		})
	}
}

func TestGetSupportedResolutions(t *testing.T) {
	tests := []struct {
		name         string
		hardwareMax  int
		expectedList []int
	}{
		{
			name:        "CPU only (1440p max)",
			hardwareMax: 1440,
			expectedList: []int{360, 480, 540, 720, 1080, 1440},
		},
		{
			name:        "4K capable (2160p max)",
			hardwareMax: 2160,
			expectedList: []int{360, 480, 540, 720, 1080, 1440, 2160},
		},
		{
			name:        "8K capable (4320p max)",
			hardwareMax: 4320,
			expectedList: []int{360, 480, 540, 720, 1080, 1440, 2160, 4320},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &ResolutionPolicy{
				HardwareMax: tt.hardwareMax,
			}

			supported := policy.GetSupportedResolutions()
			if len(supported) != len(tt.expectedList) {
				t.Errorf("expected %d resolutions, got %d", len(tt.expectedList), len(supported))
				return
			}

			for i, res := range supported {
				if res != tt.expectedList[i] {
					t.Errorf("resolution[%d]: expected %d, got %d", i, tt.expectedList[i], res)
				}
			}
		})
	}
}

func TestGetDisabledReason(t *testing.T) {
	tests := []struct {
		name              string
		policy            *ResolutionPolicy
		inputRes          int
		expectedReasonLen int // > 0 means disabled, == 0 means enabled
	}{
		{
			name: "resolution enabled",
			policy: &ResolutionPolicy{
				HardwareMax: 1080,
				GPUType:     GPUTypeNone,
				VRAMGB:      0,
				AdminMax:    nil,
			},
			inputRes:          720,
			expectedReasonLen: 0, // Empty reason
		},
		{
			name: "resolution disabled by admin",
			policy: &ResolutionPolicy{
				HardwareMax: 1080,
				GPUType:     GPUTypeNone,
				VRAMGB:      0,
				AdminMax:    intPtr(720),
			},
			inputRes:          1080,
			expectedReasonLen: 1, // Has reason
		},
		{
			name: "resolution disabled by hardware",
			policy: &ResolutionPolicy{
				HardwareMax: 1080,
				GPUType:     GPUTypeNone,
				VRAMGB:      0,
				AdminMax:    nil,
			},
			inputRes:          2160,
			expectedReasonLen: 1, // Has reason
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := tt.policy.GetDisabledReason(tt.inputRes)
			hasReason := len(reason) > 0

			expectedHasReason := tt.expectedReasonLen > 0
			if hasReason != expectedHasReason {
				t.Errorf("expected disabled=%v, got disabled=%v (reason: %s)", expectedHasReason, hasReason, reason)
			}
		})
	}
}

func TestResolutionLabel(t *testing.T) {
	tests := []struct {
		height   int
		expected string
	}{
		{360, "360p"},
		{480, "480p (SD)"},
		{540, "540p"},
		{720, "720p (HD)"},
		{1080, "1080p (Full HD)"},
		{1440, "1440p (2K)"},
		{2160, "2160p (4K)"},
		{4320, "4320p (8K)"},
		{999, "999p"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			label := ResolutionLabel(tt.height)
			if label != tt.expected {
				t.Errorf("ResolutionLabel(%d) = %q, expected %q", tt.height, label, tt.expected)
			}
		})
	}
}

func TestGetCapabilities(t *testing.T) {
	maxRes := 1080
	policy := &ResolutionPolicy{
		GPUType:     GPUTypeNVIDIA,
		VRAMGB:      8,
		HardwareMax: 2160,
		AdminMax:    &maxRes,
		AV1Capable:  true,
	}

	caps := policy.GetCapabilities()

	if caps.GPUType != "nvidia" {
		t.Errorf("expected GPUType 'nvidia', got %q", caps.GPUType)
	}
	if caps.VRAMGB != 8 {
		t.Errorf("expected VRAMGB 8, got %d", caps.VRAMGB)
	}
	if caps.HardwareMax != 2160 {
		t.Errorf("expected HardwareMax 2160, got %d", caps.HardwareMax)
	}
	if !caps.AV1Supported {
		t.Error("expected AV1Supported to be true")
	}
	if caps.EffectiveMax != 1080 {
		t.Errorf("expected EffectiveMax 1080, got %d", caps.EffectiveMax)
	}
	if caps.AdminMax == nil || *caps.AdminMax != 1080 {
		t.Error("expected AdminMax 1080")
	}
}

func TestCalculateHardwareMax(t *testing.T) {
	tests := []struct {
		name      string
		gpuType   GPUType
		vramGB    int
		expected  int
	}{
		{"CPU only", GPUTypeNone, 0, DefaultMaxResolutionCPU},
		{"GPU 4GB", GPUTypeNVIDIA, 4, Resolution1440p},
		{"GPU 8GB", GPUTypeNVIDIA, 8, Resolution2160p},
		{"GPU 12GB", GPUTypeNVIDIA, 12, Resolution4320p},
		{"GPU 16GB", GPUTypeAMD, 16, Resolution4320p},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &ResolutionPolicy{
				GPUType: tt.gpuType,
				VRAMGB:  tt.vramGB,
			}

			result := policy.calculateHardwareMax()
			if result != tt.expected {
				t.Errorf("calculateHardwareMax() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
