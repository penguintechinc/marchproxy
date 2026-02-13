package transcode

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

// GPUType represents the detected GPU type
type GPUType string

const (
	GPUTypeNone   GPUType = "none"
	GPUTypeNVIDIA GPUType = "nvidia"
	GPUTypeAMD    GPUType = "amd"
)

// Detector handles GPU detection and encoder selection
type Detector struct {
	gpuType        GPUType
	nvidiaSMIPath  string
	rocmSMIPath    string
	hasNVIDIA      bool
	hasAMD         bool
	vramGB         int
	gpuModel       string
	av1Capable     bool
	detectionError error
}

// NewDetector creates a new GPU detector
func NewDetector() *Detector {
	d := &Detector{
		gpuType:       GPUTypeNone,
		nvidiaSMIPath: "/usr/bin/nvidia-smi",
		rocmSMIPath:   "/opt/rocm/bin/rocm-smi",
	}
	d.detect()
	return d
}

// detect performs GPU detection
func (d *Detector) detect() {
	// Check for NVIDIA GPU
	d.hasNVIDIA = d.detectNVIDIA()
	if d.hasNVIDIA {
		d.gpuType = GPUTypeNVIDIA
		d.vramGB = d.detectNVIDIAVRAM()
		d.gpuModel = d.detectNVIDIAModel()
		d.av1Capable = d.detectNVIDAV1Support()
		logrus.WithFields(logrus.Fields{
			"model":      d.gpuModel,
			"vram_gb":    d.vramGB,
			"av1_capable": d.av1Capable,
		}).Info("NVIDIA GPU detected")
		return
	}

	// Check for AMD GPU
	d.hasAMD = d.detectAMD()
	if d.hasAMD {
		d.gpuType = GPUTypeAMD
		d.vramGB = d.detectAMDVRAM()
		d.gpuModel = d.detectAMDModel()
		d.av1Capable = d.detectAMDAV1Support()
		logrus.WithFields(logrus.Fields{
			"model":      d.gpuModel,
			"vram_gb":    d.vramGB,
			"av1_capable": d.av1Capable,
		}).Info("AMD GPU detected")
		return
	}

	logrus.Info("No GPU detected, using CPU encoding")
}

// detectNVIDIA checks for NVIDIA GPU
func (d *Detector) detectNVIDIA() bool {
	// Check nvidia-smi existence
	if _, err := os.Stat(d.nvidiaSMIPath); err != nil {
		logrus.Debug("nvidia-smi not found")
		return false
	}

	// Run nvidia-smi
	cmd := exec.Command(d.nvidiaSMIPath, "-L")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).Debug("nvidia-smi execution failed")
		return false
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "GPU") {
		logrus.WithField("gpus", outputStr).Debug("NVIDIA GPUs found")
		return true
	}

	// Check environment variable (Docker runtime)
	if os.Getenv("NVIDIA_VISIBLE_DEVICES") != "" {
		logrus.Debug("NVIDIA_VISIBLE_DEVICES set")
		return true
	}

	return false
}

// detectAMD checks for AMD GPU
func (d *Detector) detectAMD() bool {
	// Check rocm-smi existence
	if _, err := os.Stat(d.rocmSMIPath); err != nil {
		logrus.Debug("rocm-smi not found")
		return false
	}

	// Run rocm-smi
	cmd := exec.Command(d.rocmSMIPath, "--showproductname")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).Debug("rocm-smi execution failed")
		return false
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "GPU") || strings.Contains(outputStr, "Radeon") {
		logrus.WithField("gpus", outputStr).Debug("AMD GPUs found")
		return true
	}

	// Check /dev/kfd (AMD kernel device)
	if _, err := os.Stat("/dev/kfd"); err == nil {
		logrus.Debug("/dev/kfd exists (AMD GPU)")
		return true
	}

	return false
}

// detectNVIDIAVRAM queries NVIDIA GPU VRAM in GB
func (d *Detector) detectNVIDIAVRAM() int {
	cmd := exec.Command(d.nvidiaSMIPath, "--query-gpu=memory.total", "--format=csv,noheader,nounits")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).Debug("Failed to query NVIDIA VRAM")
		return 0
	}

	// Parse VRAM in MiB, convert to GB
	var vramMiB int
	_, err = fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &vramMiB)
	if err != nil {
		logrus.WithError(err).Debug("Failed to parse NVIDIA VRAM")
		return 0
	}

	return vramMiB / 1024
}

// detectNVIDIAModel queries NVIDIA GPU model name
func (d *Detector) detectNVIDIAModel() string {
	cmd := exec.Command(d.nvidiaSMIPath, "--query-gpu=name", "--format=csv,noheader")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).Debug("Failed to query NVIDIA model")
		return "Unknown NVIDIA GPU"
	}
	return strings.TrimSpace(string(output))
}

// detectNVIDAV1Support checks if NVIDIA GPU supports AV1 encoding
// AV1 encoding requires RTX 40xx series (Ada Lovelace) or newer
func (d *Detector) detectNVIDAV1Support() bool {
	model := d.gpuModel
	// RTX 40xx series supports AV1 encoding
	if strings.Contains(model, "RTX 40") ||
		strings.Contains(model, "RTX 50") ||
		strings.Contains(model, "Ada") ||
		strings.Contains(model, "L40") ||
		strings.Contains(model, "A100") {
		return true
	}
	return false
}

// detectAMDVRAM queries AMD GPU VRAM in GB
func (d *Detector) detectAMDVRAM() int {
	cmd := exec.Command(d.rocmSMIPath, "--showmeminfo", "vram", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try alternative approach
		cmd = exec.Command(d.rocmSMIPath, "--showmeminfo", "vram")
		output, err = cmd.CombinedOutput()
		if err != nil {
			logrus.WithError(err).Debug("Failed to query AMD VRAM")
			return 0
		}
	}

	outputStr := string(output)
	// Parse "Total Memory (B): XXXXXXXXX" or similar patterns
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Total") && strings.Contains(line, "Memory") {
			var bytes int64
			_, err := fmt.Sscanf(line, "Total Memory (B): %d", &bytes)
			if err == nil {
				return int(bytes / (1024 * 1024 * 1024))
			}
		}
	}

	// Default to 8GB if detection fails but GPU exists
	return 8
}

// detectAMDModel queries AMD GPU model name
func (d *Detector) detectAMDModel() string {
	cmd := exec.Command(d.rocmSMIPath, "--showproductname")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithError(err).Debug("Failed to query AMD model")
		return "Unknown AMD GPU"
	}

	outputStr := strings.TrimSpace(string(output))
	// Parse model from rocm-smi output
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Radeon") || strings.Contains(line, "RX") {
			return strings.TrimSpace(line)
		}
	}

	return "AMD GPU"
}

// detectAMDAV1Support checks if AMD GPU supports AV1 encoding
// AV1 encoding requires RX 7000 series (RDNA 3) or newer
func (d *Detector) detectAMDAV1Support() bool {
	model := d.gpuModel
	// RX 7xxx series supports AV1 encoding
	if strings.Contains(model, "RX 7") ||
		strings.Contains(model, "RX 8") ||
		strings.Contains(model, "RDNA 3") ||
		strings.Contains(model, "RDNA3") {
		return true
	}
	return false
}

// GetGPUType returns the detected GPU type
func (d *Detector) GetGPUType() GPUType {
	return d.gpuType
}

// HasGPU returns true if any GPU was detected
func (d *Detector) HasGPU() bool {
	return d.gpuType != GPUTypeNone
}

// HasNVIDIA returns true if NVIDIA GPU was detected
func (d *Detector) HasNVIDIA() bool {
	return d.hasNVIDIA
}

// HasAMD returns true if AMD GPU was detected
func (d *Detector) HasAMD() bool {
	return d.hasAMD
}

// GetVRAM returns detected VRAM in GB
func (d *Detector) GetVRAM() int {
	return d.vramGB
}

// GetGPUModel returns the GPU model name
func (d *Detector) GetGPUModel() string {
	return d.gpuModel
}

// SupportsAV1 returns true if GPU supports AV1 encoding
func (d *Detector) SupportsAV1() bool {
	return d.av1Capable
}

// Supports8K returns true if GPU has enough VRAM for 8K encoding
func (d *Detector) Supports8K() bool {
	return d.vramGB >= 12
}

// Supports4K returns true if GPU has enough VRAM for 4K encoding
func (d *Detector) Supports4K() bool {
	return d.vramGB >= 8
}

// SelectEncoder selects the best encoder based on GPU availability
func (d *Detector) SelectEncoder(preference string) (*EncoderConfig, error) {
	// If specific encoder requested, try to use it
	if preference != "auto" {
		return d.getEncoderConfig(preference)
	}

	// Auto-select based on GPU type
	switch d.gpuType {
	case GPUTypeNVIDIA:
		logrus.Info("Auto-selecting NVIDIA NVENC H.264 encoder")
		return d.getEncoderConfig("nvenc_h264")
	case GPUTypeAMD:
		logrus.Info("Auto-selecting AMD AMF H.264 encoder")
		return d.getEncoderConfig("amf_h264")
	default:
		logrus.Info("Auto-selecting CPU x264 encoder")
		return d.getEncoderConfig("x264")
	}
}

// SelectEncoderWithAV1Preference selects encoder with AV1 preference if available
func (d *Detector) SelectEncoderWithAV1Preference(preference string, preferAV1 bool) (*EncoderConfig, error) {
	// If specific encoder requested, try to use it
	if preference != "auto" {
		return d.getEncoderConfig(preference)
	}

	// If AV1 preferred and available
	if preferAV1 && d.av1Capable {
		switch d.gpuType {
		case GPUTypeNVIDIA:
			logrus.Info("Auto-selecting NVIDIA NVENC AV1 encoder")
			return d.getEncoderConfig("nvenc_av1")
		case GPUTypeAMD:
			logrus.Info("Auto-selecting AMD AMF AV1 encoder")
			return d.getEncoderConfig("amf_av1")
		}
	}

	// Fall back to standard auto-selection
	return d.SelectEncoder(preference)
}

// getEncoderConfig returns encoder configuration
func (d *Detector) getEncoderConfig(encoder string) (*EncoderConfig, error) {
	switch encoder {
	case "x264":
		return NewX264Config("medium"), nil
	case "x265":
		return NewX265Config("medium"), nil
	case "nvenc_h264":
		if !d.hasNVIDIA {
			return nil, fmt.Errorf("NVIDIA GPU not available, cannot use nvenc_h264")
		}
		return NewNVENCH264Config("medium"), nil
	case "nvenc_h265":
		if !d.hasNVIDIA {
			return nil, fmt.Errorf("NVIDIA GPU not available, cannot use nvenc_h265")
		}
		return NewNVENCH265Config("medium"), nil
	case "amf_h264":
		if !d.hasAMD {
			return nil, fmt.Errorf("AMD GPU not available, cannot use amf_h264")
		}
		return NewAMFH264Config("medium"), nil
	case "amf_h265":
		if !d.hasAMD {
			return nil, fmt.Errorf("AMD GPU not available, cannot use amf_h265")
		}
		return NewAMFH265Config("medium"), nil
	// AV1 CPU encoders
	case "libaom_av1":
		return NewLibaomAV1Config("medium"), nil
	case "svt_av1":
		return NewSVTAV1Config("medium"), nil
	// AV1 GPU encoders
	case "nvenc_av1":
		if !d.hasNVIDIA {
			return nil, fmt.Errorf("NVIDIA GPU not available, cannot use nvenc_av1")
		}
		if !d.av1Capable {
			return nil, fmt.Errorf("NVIDIA GPU does not support AV1 encoding (requires RTX 40xx+)")
		}
		return NewNVENCAV1Config("medium"), nil
	case "amf_av1":
		if !d.hasAMD {
			return nil, fmt.Errorf("AMD GPU not available, cannot use amf_av1")
		}
		if !d.av1Capable {
			return nil, fmt.Errorf("AMD GPU does not support AV1 encoding (requires RX 7000+)")
		}
		return NewAMFAV1Config("medium"), nil
	default:
		return nil, fmt.Errorf("unknown encoder: %s", encoder)
	}
}
