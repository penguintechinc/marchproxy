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
		logrus.Info("NVIDIA GPU detected")
		return
	}

	// Check for AMD GPU
	d.hasAMD = d.detectAMD()
	if d.hasAMD {
		d.gpuType = GPUTypeAMD
		logrus.Info("AMD GPU detected")
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
	default:
		return nil, fmt.Errorf("unknown encoder: %s", encoder)
	}
}
