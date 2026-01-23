package transcode

// Bitrate profiles for various resolutions from 360p to 8K
// These are standard profiles used for ABR (Adaptive Bitrate) streaming

// StandardProfiles contains recommended bitrate settings for each resolution
var StandardProfiles = map[int]BitrateConfig{
	360: {
		Name:       "360p",
		Width:      640,
		Height:     360,
		Bitrate:    800,  // 0.8 Mbps
		AudioRate:  64,   // 64 kbps
		Framerate:  30,
		BufferSize: 2,
	},
	480: {
		Name:       "480p",
		Width:      854,
		Height:     480,
		Bitrate:    1500, // 1.5 Mbps
		AudioRate:  96,   // 96 kbps
		Framerate:  30,
		BufferSize: 2,
	},
	540: {
		Name:       "540p",
		Width:      960,
		Height:     540,
		Bitrate:    2000, // 2 Mbps
		AudioRate:  96,   // 96 kbps
		Framerate:  30,
		BufferSize: 2,
	},
	720: {
		Name:       "720p",
		Width:      1280,
		Height:     720,
		Bitrate:    3000, // 3 Mbps
		AudioRate:  128,  // 128 kbps
		Framerate:  30,
		BufferSize: 2,
	},
	1080: {
		Name:       "1080p",
		Width:      1920,
		Height:     1080,
		Bitrate:    5000, // 5 Mbps
		AudioRate:  192,  // 192 kbps
		Framerate:  30,
		BufferSize: 2,
	},
	1440: {
		Name:       "1440p",
		Width:      2560,
		Height:     1440,
		Bitrate:    8000, // 8 Mbps
		AudioRate:  192,  // 192 kbps
		Framerate:  30,
		BufferSize: 2,
	},
	2160: {
		Name:       "2160p",
		Width:      3840,
		Height:     2160,
		Bitrate:    20000, // 20 Mbps
		AudioRate:  256,   // 256 kbps
		Framerate:  30,
		BufferSize: 2,
	},
	4320: {
		Name:       "4320p",
		Width:      7680,
		Height:     4320,
		Bitrate:    60000, // 60 Mbps
		AudioRate:  256,   // 256 kbps
		Framerate:  30,
		BufferSize: 2,
	},
}

// AV1Profiles are optimized for AV1 codec (typically 30-40% lower bitrate for same quality)
var AV1Profiles = map[int]BitrateConfig{
	360: {
		Name:       "360p",
		Width:      640,
		Height:     360,
		Bitrate:    500,  // 0.5 Mbps (AV1 efficiency)
		AudioRate:  64,
		Framerate:  30,
		BufferSize: 2,
	},
	480: {
		Name:       "480p",
		Width:      854,
		Height:     480,
		Bitrate:    900,  // 0.9 Mbps
		AudioRate:  96,
		Framerate:  30,
		BufferSize: 2,
	},
	540: {
		Name:       "540p",
		Width:      960,
		Height:     540,
		Bitrate:    1200, // 1.2 Mbps
		AudioRate:  96,
		Framerate:  30,
		BufferSize: 2,
	},
	720: {
		Name:       "720p",
		Width:      1280,
		Height:     720,
		Bitrate:    2000, // 2 Mbps
		AudioRate:  128,
		Framerate:  30,
		BufferSize: 2,
	},
	1080: {
		Name:       "1080p",
		Width:      1920,
		Height:     1080,
		Bitrate:    3500, // 3.5 Mbps
		AudioRate:  192,
		Framerate:  30,
		BufferSize: 2,
	},
	1440: {
		Name:       "1440p",
		Width:      2560,
		Height:     1440,
		Bitrate:    5500, // 5.5 Mbps
		AudioRate:  192,
		Framerate:  30,
		BufferSize: 2,
	},
	2160: {
		Name:       "2160p",
		Width:      3840,
		Height:     2160,
		Bitrate:    14000, // 14 Mbps
		AudioRate:  256,
		Framerate:  30,
		BufferSize: 2,
	},
	4320: {
		Name:       "4320p",
		Width:      7680,
		Height:     4320,
		Bitrate:    40000, // 40 Mbps (AV1 efficiency at 8K)
		AudioRate:  256,
		Framerate:  30,
		BufferSize: 2,
	},
}

// HighFramerateProfiles for 60fps content
var HighFramerateProfiles = map[int]BitrateConfig{
	720: {
		Name:       "720p60",
		Width:      1280,
		Height:     720,
		Bitrate:    4500, // 4.5 Mbps
		AudioRate:  128,
		Framerate:  60,
		BufferSize: 2,
	},
	1080: {
		Name:       "1080p60",
		Width:      1920,
		Height:     1080,
		Bitrate:    7500, // 7.5 Mbps
		AudioRate:  192,
		Framerate:  60,
		BufferSize: 2,
	},
	1440: {
		Name:       "1440p60",
		Width:      2560,
		Height:     1440,
		Bitrate:    12000, // 12 Mbps
		AudioRate:  192,
		Framerate:  60,
		BufferSize: 2,
	},
	2160: {
		Name:       "2160p60",
		Width:      3840,
		Height:     2160,
		Bitrate:    30000, // 30 Mbps
		AudioRate:  256,
		Framerate:  60,
		BufferSize: 2,
	},
}

// GetStandardProfile returns the standard profile for given height
func GetStandardProfile(height int) (BitrateConfig, bool) {
	profile, ok := StandardProfiles[height]
	return profile, ok
}

// GetAV1Profile returns the AV1-optimized profile for given height
func GetAV1Profile(height int) (BitrateConfig, bool) {
	profile, ok := AV1Profiles[height]
	return profile, ok
}

// GetHighFramerateProfile returns the 60fps profile for given height
func GetHighFramerateProfile(height int) (BitrateConfig, bool) {
	profile, ok := HighFramerateProfiles[height]
	return profile, ok
}

// GetProfileForCodec returns appropriate profile based on codec
func GetProfileForCodec(height int, codec string) (BitrateConfig, bool) {
	switch codec {
	case "av1":
		return GetAV1Profile(height)
	default:
		return GetStandardProfile(height)
	}
}

// GetTranscodeLadder returns profiles for all heights up to maxHeight
func GetTranscodeLadder(maxHeight int, codec string) []BitrateConfig {
	heights := []int{360, 480, 540, 720, 1080, 1440, 2160, 4320}
	var ladder []BitrateConfig

	for _, h := range heights {
		if h > maxHeight {
			break
		}
		profile, ok := GetProfileForCodec(h, codec)
		if ok {
			ladder = append(ladder, profile)
		}
	}

	return ladder
}

// DefaultTranscodeLadderHeights returns commonly used ladder heights
func DefaultTranscodeLadderHeights() []int {
	return []int{360, 540, 720, 1080}
}

// PlatformRecommendedLadder returns platform-specific ladder
func PlatformRecommendedLadder(platform string) []int {
	switch platform {
	case "twitch":
		return []int{360, 480, 720, 1080}
	case "youtube":
		return []int{360, 480, 720, 1080, 1440, 2160}
	case "facebook":
		return []int{360, 720, 1080}
	default:
		return DefaultTranscodeLadderHeights()
	}
}
