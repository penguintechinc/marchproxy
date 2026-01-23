package transcode

import "fmt"

// AV1 CPU encoder configurations (libaom-av1, SVT-AV1)

// NewLibaomAV1Config creates libaom-av1 CPU encoder configuration
// libaom is the reference AV1 encoder - high quality but slow
func NewLibaomAV1Config(preset string) *EncoderConfig {
	cpuUsed := mapPresetToLibaomCPUUsed(preset)
	return &EncoderConfig{
		Name:    "libaom_av1",
		Codec:   "av1",
		Encoder: "libaom-av1",
		HWAccel: "",
		Preset:  preset,
		Params: map[string]string{
			"cpu-used":    cpuUsed,      // Speed preset (0-8, higher = faster)
			"crf":         "30",         // Quality (0-63, lower = better)
			"aq-mode":     "1",          // Adaptive quantization
			"row-mt":      "1",          // Row-based multi-threading
			"tiles":       "2x2",        // Tile parallelism
			"tile-columns": "1",         // Tile columns (log2)
			"tile-rows":   "1",          // Tile rows (log2)
			"g":           "240",        // GOP size
			"keyint_min":  "60",         // Min keyframe interval
		},
		Bitrates: DefaultBitrateLadder(),
	}
}

// NewSVTAV1Config creates SVT-AV1 CPU encoder configuration
// SVT-AV1 is faster than libaom while maintaining good quality
func NewSVTAV1Config(preset string) *EncoderConfig {
	svtPreset := mapPresetToSVTAV1(preset)
	return &EncoderConfig{
		Name:    "svt_av1",
		Codec:   "av1",
		Encoder: "libsvtav1",
		HWAccel: "",
		Preset:  svtPreset,
		Params: map[string]string{
			"crf":             "30",     // Quality (0-63, lower = better)
			"svtav1-params":   "tune=0", // PSNR tuning
			"g":               "240",    // GOP size
			"preset":          svtPreset,
			"tier":            "main",
		},
		Bitrates: DefaultBitrateLadder(),
	}
}

// mapPresetToLibaomCPUUsed maps x264-style presets to libaom cpu-used values
func mapPresetToLibaomCPUUsed(preset string) string {
	// cpu-used: 0 (slowest/best quality) to 8 (fastest/lowest quality)
	mapping := map[string]string{
		"ultrafast": "8",
		"superfast": "7",
		"veryfast":  "6",
		"faster":    "5",
		"fast":      "4",
		"medium":    "4",
		"slow":      "3",
		"slower":    "2",
		"veryslow":  "1",
	}

	if cpuUsed, ok := mapping[preset]; ok {
		return cpuUsed
	}
	return "4" // Default to medium
}

// mapPresetToSVTAV1 maps x264-style presets to SVT-AV1 presets
func mapPresetToSVTAV1(preset string) string {
	// SVT-AV1 presets: 0 (slowest/best) to 13 (fastest)
	mapping := map[string]string{
		"ultrafast": "12",
		"superfast": "11",
		"veryfast":  "10",
		"faster":    "9",
		"fast":      "8",
		"medium":    "6",
		"slow":      "4",
		"slower":    "2",
		"veryslow":  "0",
	}

	if svtPreset, ok := mapping[preset]; ok {
		return svtPreset
	}
	return "6" // Default to medium
}

// GetFFmpegArgsAV1CPU returns FFmpeg arguments for AV1 CPU encoding
func (e *EncoderConfig) GetFFmpegArgsAV1CPU(input string, output string, bitrate BitrateConfig) []string {
	args := []string{
		"-i", input,
		"-c:v", e.Encoder,
		"-b:v", fmt.Sprintf("%dk", bitrate.Bitrate),
		"-maxrate", fmt.Sprintf("%dk", int(float64(bitrate.Bitrate)*1.5)), // AV1 uses higher maxrate
		"-bufsize", fmt.Sprintf("%dk", bitrate.Bitrate*bitrate.BufferSize),
		"-vf", fmt.Sprintf("scale=%d:%d", bitrate.Width, bitrate.Height),
		"-r", fmt.Sprintf("%d", bitrate.Framerate),
		"-pix_fmt", "yuv420p",
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
