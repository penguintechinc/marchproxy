package transcode

import "fmt"

// NVIDIA AV1 encoder configuration (RTX 40xx+ series)

// NewNVENCAV1Config creates NVIDIA av1_nvenc GPU encoder configuration
// Requires NVIDIA RTX 40xx series or newer (Ada Lovelace architecture)
func NewNVENCAV1Config(preset string) *EncoderConfig {
	return &EncoderConfig{
		Name:    "nvenc_av1",
		Codec:   "av1",
		Encoder: "av1_nvenc",
		HWAccel: "cuda",
		Preset:  mapPresetToNVENCAV1(preset),
		Params: map[string]string{
			"tier":          "main",
			"level":         "auto",
			"rc":            "vbr",    // Variable bitrate
			"cq":            "25",     // Constant quality
			"gpu":           "0",      // GPU index
			"delay":         "0",      // Low latency
			"spatial-aq":    "1",      // Spatial AQ
			"temporal-aq":   "1",      // Temporal AQ
			"lookahead":     "0",      // Disable for low latency
			"multipass":     "disabled",
			"b_adapt":       "0",
			"bf":            "0",      // No B-frames for low latency
			"weighted_pred": "0",
		},
		Bitrates: DefaultBitrateLadder(),
	}
}

// mapPresetToNVENCAV1 maps CPU preset names to NVENC AV1 presets
// NVENC AV1 uses p1-p7 (similar to H.264/H.265 NVENC)
func mapPresetToNVENCAV1(preset string) string {
	mapping := map[string]string{
		"ultrafast": "p1",
		"superfast": "p2",
		"veryfast":  "p3",
		"faster":    "p4",
		"fast":      "p5",
		"medium":    "p5",
		"slow":      "p6",
		"slower":    "p7",
		"veryslow":  "p7",
	}

	if nvencPreset, ok := mapping[preset]; ok {
		return nvencPreset
	}
	return "p5" // Default to medium
}

// GetFFmpegArgsNVENCAV1 returns FFmpeg arguments for NVENC AV1 encoding
func (e *EncoderConfig) GetFFmpegArgsNVENCAV1(input string, output string, bitrate BitrateConfig) []string {
	args := []string{
		"-hwaccel", e.HWAccel,
		"-hwaccel_output_format", "cuda",
		"-i", input,
		"-c:v", e.Encoder,
		"-preset", e.Preset,
		"-b:v", fmt.Sprintf("%dk", bitrate.Bitrate),
		"-maxrate", fmt.Sprintf("%dk", bitrate.Bitrate),
		"-bufsize", fmt.Sprintf("%dk", bitrate.Bitrate*bitrate.BufferSize),
		"-vf", fmt.Sprintf("scale_cuda=%d:%d", bitrate.Width, bitrate.Height),
		"-r", fmt.Sprintf("%d", bitrate.Framerate),
	}

	// Add encoder-specific params
	for key, value := range e.Params {
		args = append(args, "-"+key, value)
	}

	// Audio encoding
	args = append(args,
		"-c:a", "aac",
		"-b:a", fmt.Sprintf("%dk", bitrate.AudioRate),
		"-ar", "48000",
		"-ac", "2",
	)

	// Output format
	args = append(args, "-f", "flv", output)

	return args
}

// RequiresNVIDIARTX40 returns true - av1_nvenc requires RTX 40xx+
func RequiresNVIDIARTX40() bool {
	return true
}
