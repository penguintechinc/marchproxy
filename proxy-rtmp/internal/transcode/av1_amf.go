package transcode

import "fmt"

// AMD AV1 encoder configuration (RX 7000+ series)

// NewAMFAV1Config creates AMD av1_amf GPU encoder configuration
// Requires AMD RX 7000 series or newer (RDNA 3 architecture)
func NewAMFAV1Config(preset string) *EncoderConfig {
	return &EncoderConfig{
		Name:    "amf_av1",
		Codec:   "av1",
		Encoder: "av1_amf",
		HWAccel: "amf",
		Preset:  mapPresetToAMFAV1(preset),
		Params: map[string]string{
			"profile":          "main",
			"level":            "auto",
			"rc":               "vbr_latency",       // VBR with latency constraint
			"quality":          mapPresetToAMFAV1(preset),
			"pa_lookahead_buffer_depth": "0",        // Low latency
			"pa_activity_type":           "y",       // Luma-based activity
			"pa_scene_change_detection": "1",
			"pa_initial_qp_after_scene_change": "25",
			"enable_av1_low_overhead_mode": "1",
			"latency_mode":      "low_latency",
		},
		Bitrates: DefaultBitrateLadder(),
	}
}

// mapPresetToAMFAV1 maps CPU preset names to AMF AV1 quality presets
// AMF uses quality: speed, balanced, quality
func mapPresetToAMFAV1(preset string) string {
	mapping := map[string]string{
		"ultrafast": "speed",
		"superfast": "speed",
		"veryfast":  "speed",
		"faster":    "speed",
		"fast":      "balanced",
		"medium":    "balanced",
		"slow":      "quality",
		"slower":    "quality",
		"veryslow":  "quality",
	}

	if amfQuality, ok := mapping[preset]; ok {
		return amfQuality
	}
	return "balanced" // Default to balanced
}

// GetFFmpegArgsAMFAV1 returns FFmpeg arguments for AMF AV1 encoding
func (e *EncoderConfig) GetFFmpegArgsAMFAV1(input string, output string, bitrate BitrateConfig) []string {
	args := []string{
		"-hwaccel", "auto",
		"-i", input,
		"-c:v", e.Encoder,
		"-quality", e.Preset,
		"-b:v", fmt.Sprintf("%dk", bitrate.Bitrate),
		"-maxrate", fmt.Sprintf("%dk", bitrate.Bitrate),
		"-bufsize", fmt.Sprintf("%dk", bitrate.Bitrate*bitrate.BufferSize),
		"-vf", fmt.Sprintf("scale=%d:%d", bitrate.Width, bitrate.Height),
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

// RequiresAMDRDNA3 returns true - av1_amf requires RX 7000+
func RequiresAMDRDNA3() bool {
	return true
}
