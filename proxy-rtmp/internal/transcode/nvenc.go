package transcode

import "fmt"

// NVIDIA NVENC hardware encoder configuration

// NewNVENCH264Config creates NVENC H.264 encoder configuration
func NewNVENCH264Config(preset string) *EncoderConfig {
	return &EncoderConfig{
		Name:    "nvenc_h264",
		Codec:   "h264",
		Encoder: "h264_nvenc",
		HWAccel: "cuda",
		Preset:  mapPresetToNVENC(preset),
		Params: map[string]string{
			"profile":     "high",
			"level":       "4.1",
			"rc":          "vbr", // Variable bitrate
			"cq":          "23",  // Constant quality
			"gpu":         "0",   // GPU index
			"delay":       "0",   // Low latency
			"zerolatency": "1",
			"spatial-aq":  "1", // Spatial AQ
			"temporal-aq": "1", // Temporal AQ
			"b_adapt":     "0",
			"bf":          "0",
		},
		Bitrates: DefaultBitrateLadder(),
	}
}

// NewNVENCH265Config creates NVENC H.265 encoder configuration
func NewNVENCH265Config(preset string) *EncoderConfig {
	return &EncoderConfig{
		Name:    "nvenc_h265",
		Codec:   "h265",
		Encoder: "hevc_nvenc",
		HWAccel: "cuda",
		Preset:  mapPresetToNVENC(preset),
		Params: map[string]string{
			"profile":     "main",
			"level":       "4.1",
			"rc":          "vbr",
			"cq":          "23",
			"gpu":         "0",
			"delay":       "0",
			"zerolatency": "1",
			"spatial-aq":  "1",
			"temporal-aq": "1",
			"b_adapt":     "0",
			"bf":          "0",
		},
		Bitrates: DefaultBitrateLadder(),
	}
}

// mapPresetToNVENC maps CPU preset names to NVENC presets
func mapPresetToNVENC(preset string) string {
	mapping := map[string]string{
		"ultrafast": "p1",
		"superfast": "p2",
		"veryfast":  "p3",
		"faster":    "p4",
		"fast":      "p5",
		"medium":    "p6",
		"slow":      "p7",
		"slower":    "p7",
		"veryslow":  "p7",
	}

	if nvencPreset, ok := mapping[preset]; ok {
		return nvencPreset
	}
	return "p6" // Default to medium (p6)
}

// GetFFmpegArgsNVENC returns FFmpeg arguments for NVENC encoding
func (e *EncoderConfig) GetFFmpegArgsNVENC(input string, output string, bitrate BitrateConfig) []string {
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
		"-c:a", "aac",
		"-b:a", fmt.Sprintf("%dk", bitrate.AudioRate),
		"-ar", "48000",
		"-ac", "2",
	}

	// Add encoder-specific params
	for key, value := range e.Params {
		args = append(args, "-"+key, value)
	}

	// Output format
	args = append(args, "-f", "flv", output)

	return args
}
