package transcode

import (
	"fmt"
	"sync"
)

// Resolution constants
const (
	Resolution360p  = 360
	Resolution480p  = 480
	Resolution540p  = 540
	Resolution720p  = 720
	Resolution1080p = 1080
	Resolution1440p = 1440  // 2K
	Resolution2160p = 2160  // 4K
	Resolution4320p = 4320  // 8K

	// Default limits
	DefaultMaxResolutionCPU = Resolution1440p // 2K max for CPU-only
	DefaultMaxResolutionGPU = Resolution4320p // 8K max for GPU

	// VRAM requirements (in GB) for different resolutions
	VRAMRequirement4K = 8   // 8GB minimum for 4K
	VRAMRequirement8K = 12  // 12GB minimum for 8K
)

// ResolutionPolicy enforces resolution limits based on hardware and admin settings
type ResolutionPolicy struct {
	HardwareMax int  // Maximum supported by hardware
	AdminMax    *int // Admin override (nil = no override)
	GPUType     GPUType
	VRAMGB      int  // Available GPU VRAM in GB
	AV1Capable  bool
	mutex       sync.RWMutex
}

// NewResolutionPolicy creates a new resolution policy
func NewResolutionPolicy(detector *Detector) *ResolutionPolicy {
	policy := &ResolutionPolicy{
		GPUType: detector.GetGPUType(),
	}

	// Detect hardware capabilities
	if detector.HasGPU() {
		policy.VRAMGB = detector.GetVRAM()
		policy.AV1Capable = detector.SupportsAV1()
		policy.HardwareMax = policy.calculateHardwareMax()
	} else {
		policy.VRAMGB = 0
		policy.AV1Capable = false
		policy.HardwareMax = DefaultMaxResolutionCPU
	}

	return policy
}

// calculateHardwareMax determines max resolution based on GPU capabilities
func (p *ResolutionPolicy) calculateHardwareMax() int {
	if p.GPUType == GPUTypeNone {
		return DefaultMaxResolutionCPU
	}

	// Check VRAM-based limits
	switch {
	case p.VRAMGB >= VRAMRequirement8K:
		return Resolution4320p // 8K
	case p.VRAMGB >= VRAMRequirement4K:
		return Resolution2160p // 4K
	default:
		return Resolution1440p // 2K
	}
}

// SetAdminMax sets administrator override for max resolution
func (p *ResolutionPolicy) SetAdminMax(maxHeight *int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.AdminMax = maxHeight
}

// ClearAdminMax removes administrator override
func (p *ResolutionPolicy) ClearAdminMax() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.AdminMax = nil
}

// EffectiveMax returns the effective maximum resolution
func (p *ResolutionPolicy) EffectiveMax() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.AdminMax != nil && *p.AdminMax < p.HardwareMax {
		return *p.AdminMax
	}
	return p.HardwareMax
}

// Validate checks if the requested resolution is allowed
func (p *ResolutionPolicy) Validate(inputHeight int) error {
	max := p.EffectiveMax()
	if inputHeight > max {
		return &ResolutionExceededError{
			Requested: inputHeight,
			Maximum:   max,
			Reason:    p.getReason(max),
		}
	}
	return nil
}

// getReason returns human-readable reason for the limit
func (p *ResolutionPolicy) getReason(effectiveMax int) string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.AdminMax != nil && *p.AdminMax < p.HardwareMax {
		return "Administrator limit"
	}

	if p.GPUType == GPUTypeNone {
		return "Hardware acceleration not available (CPU-only mode)"
	}

	if effectiveMax <= Resolution1440p && p.GPUType != GPUTypeNone {
		return fmt.Sprintf("Insufficient GPU VRAM (%dGB)", p.VRAMGB)
	}

	return "Hardware capability limit"
}

// GetCapabilities returns current hardware capabilities
func (p *ResolutionPolicy) GetCapabilities() *HardwareCapabilities {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	caps := &HardwareCapabilities{
		GPUType:          string(p.GPUType),
		VRAMGB:           p.VRAMGB,
		HardwareMax:      p.HardwareMax,
		AV1Supported:     p.AV1Capable,
		Supports8K:       p.HardwareMax >= Resolution4320p,
		Supports4K:       p.HardwareMax >= Resolution2160p,
		EffectiveMax:     p.EffectiveMax(),
	}

	if p.AdminMax != nil {
		adminMax := *p.AdminMax
		caps.AdminMax = &adminMax
	}

	return caps
}

// IsResolutionSupported checks if a specific resolution is supported
func (p *ResolutionPolicy) IsResolutionSupported(height int) bool {
	return height <= p.EffectiveMax()
}

// GetSupportedResolutions returns list of supported resolutions
func (p *ResolutionPolicy) GetSupportedResolutions() []int {
	max := p.EffectiveMax()
	supported := []int{}

	resolutions := []int{
		Resolution360p, Resolution480p, Resolution540p,
		Resolution720p, Resolution1080p, Resolution1440p,
		Resolution2160p, Resolution4320p,
	}

	for _, res := range resolutions {
		if res <= max {
			supported = append(supported, res)
		}
	}

	return supported
}

// GetDisabledReason returns reason why a resolution is disabled (or empty if enabled)
func (p *ResolutionPolicy) GetDisabledReason(height int) string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if height <= p.EffectiveMax() {
		return "" // Not disabled
	}

	// Check if admin limit is the bottleneck
	if p.AdminMax != nil && height > *p.AdminMax {
		return fmt.Sprintf("Administrator limit: %dp maximum", *p.AdminMax)
	}

	// Hardware is the bottleneck
	if p.GPUType == GPUTypeNone {
		return "Requires GPU hardware acceleration"
	}

	if height > p.HardwareMax {
		return fmt.Sprintf("GPU does not support %dp (requires more VRAM)", height)
	}

	return "Not available"
}

// HardwareCapabilities represents detected hardware capabilities
type HardwareCapabilities struct {
	GPUType      string `json:"gpu_type"`
	GPUModel     string `json:"gpu_model,omitempty"`
	VRAMGB       int    `json:"vram_gb"`
	HardwareMax  int    `json:"hardware_max_resolution"`
	AdminMax     *int   `json:"admin_max_resolution,omitempty"`
	EffectiveMax int    `json:"effective_max_resolution"`
	AV1Supported bool   `json:"av1_supported"`
	Supports8K   bool   `json:"supports_8k"`
	Supports4K   bool   `json:"supports_4k"`
}

// ResolutionExceededError is returned when requested resolution exceeds limit
type ResolutionExceededError struct {
	Requested int
	Maximum   int
	Reason    string
}

func (e *ResolutionExceededError) Error() string {
	return fmt.Sprintf("resolution %dp exceeds maximum %dp: %s", e.Requested, e.Maximum, e.Reason)
}

// IsResolutionExceededError checks if error is ResolutionExceededError
func IsResolutionExceededError(err error) bool {
	_, ok := err.(*ResolutionExceededError)
	return ok
}

// ResolutionLabel returns human-readable label for resolution
func ResolutionLabel(height int) string {
	labels := map[int]string{
		360:  "360p",
		480:  "480p (SD)",
		540:  "540p",
		720:  "720p (HD)",
		1080: "1080p (Full HD)",
		1440: "1440p (2K)",
		2160: "2160p (4K)",
		4320: "4320p (8K)",
	}

	if label, ok := labels[height]; ok {
		return label
	}
	return fmt.Sprintf("%dp", height)
}
