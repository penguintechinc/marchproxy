//go:build ci
// +build ci

package transcode_test

import (
	"testing"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/transcode"
)

// TestGetStandardProfileKnownHeights verifies known heights return profiles.
func TestGetStandardProfileKnownHeights(t *testing.T) {
	knownHeights := []int{360, 480, 540, 720, 1080, 1440, 2160, 4320}

	for _, h := range knownHeights {
		t.Run(transcode.StandardProfiles[h].Name, func(t *testing.T) {
			profile, ok := transcode.GetStandardProfile(h)
			if !ok {
				t.Fatalf("expected profile for height %d, not found", h)
			}
			if profile.Height != h {
				t.Errorf("expected Height %d, got %d", h, profile.Height)
			}
			if profile.Name == "" {
				t.Error("expected non-empty profile Name")
			}
		})
	}
}

// TestGetStandardProfileUnknownHeight verifies unknown height returns false.
func TestGetStandardProfileUnknownHeight(t *testing.T) {
	_, ok := transcode.GetStandardProfile(999)
	if ok {
		t.Error("expected ok=false for unknown height 999")
	}
}

// TestStandardProfileBitratesReasonable verifies all bitrates are positive.
func TestStandardProfileBitratesReasonable(t *testing.T) {
	for height, profile := range transcode.StandardProfiles {
		t.Run(profile.Name, func(t *testing.T) {
			if profile.Bitrate <= 0 {
				t.Errorf("height %d: expected Bitrate > 0, got %d", height, profile.Bitrate)
			}
			if profile.AudioRate <= 0 {
				t.Errorf("height %d: expected AudioRate > 0, got %d", height, profile.AudioRate)
			}
			if profile.Width <= 0 {
				t.Errorf("height %d: expected Width > 0, got %d", height, profile.Width)
			}
			if profile.Framerate <= 0 {
				t.Errorf("height %d: expected Framerate > 0, got %d", height, profile.Framerate)
			}
		})
	}
}

// TestAV1ProfileBitratesLowerThanStandard verifies AV1 is more efficient.
func TestAV1ProfileBitratesLowerThanStandard(t *testing.T) {
	for height, av1Profile := range transcode.AV1Profiles {
		stdProfile, ok := transcode.StandardProfiles[height]
		if !ok {
			continue
		}
		if av1Profile.Bitrate >= stdProfile.Bitrate {
			t.Errorf("height %d: AV1 bitrate (%d) should be lower than standard (%d)",
				height, av1Profile.Bitrate, stdProfile.Bitrate)
		}
	}
}

// TestGetAV1Profile verifies AV1 profile lookup works.
func TestGetAV1Profile(t *testing.T) {
	profile, ok := transcode.GetAV1Profile(1080)
	if !ok {
		t.Fatal("expected AV1 profile for 1080p")
	}
	if profile.Bitrate <= 0 {
		t.Error("expected Bitrate > 0 for AV1 1080p profile")
	}
}

// TestGetHighFramerateProfile verifies 60fps profile lookup works.
func TestGetHighFramerateProfile(t *testing.T) {
	profile, ok := transcode.GetHighFramerateProfile(1080)
	if !ok {
		t.Fatal("expected high framerate profile for 1080p")
	}
	if profile.Framerate != 60 {
		t.Errorf("expected Framerate 60, got %d", profile.Framerate)
	}
}

// TestHighFramerateProfileBitratesHigherThanStandard verifies 60fps needs more bits.
func TestHighFramerateProfileBitratesHigherThanStandard(t *testing.T) {
	for height, hfrProfile := range transcode.HighFramerateProfiles {
		stdProfile, ok := transcode.StandardProfiles[height]
		if !ok {
			continue
		}
		if hfrProfile.Bitrate <= stdProfile.Bitrate {
			t.Errorf("height %d: HFR bitrate (%d) should exceed standard (%d)",
				height, hfrProfile.Bitrate, stdProfile.Bitrate)
		}
	}
}

// TestGetProfileForCodecAV1 verifies codec routing to AV1 profiles.
func TestGetProfileForCodecAV1(t *testing.T) {
	av1Profile, _ := transcode.GetAV1Profile(720)
	codecProfile, ok := transcode.GetProfileForCodec(720, "av1")
	if !ok {
		t.Fatal("expected profile for codec 'av1' at 720p")
	}
	if codecProfile.Bitrate != av1Profile.Bitrate {
		t.Errorf("expected AV1 bitrate %d, got %d", av1Profile.Bitrate, codecProfile.Bitrate)
	}
}

// TestGetProfileForCodecDefault verifies non-AV1 codec falls back to standard.
func TestGetProfileForCodecDefault(t *testing.T) {
	stdProfile, _ := transcode.GetStandardProfile(720)
	codecProfile, ok := transcode.GetProfileForCodec(720, "h264")
	if !ok {
		t.Fatal("expected profile for codec 'h264' at 720p")
	}
	if codecProfile.Bitrate != stdProfile.Bitrate {
		t.Errorf("expected standard bitrate %d, got %d", stdProfile.Bitrate, codecProfile.Bitrate)
	}
}

// TestGetTranscodeLadderStopsAtMaxHeight verifies ladder respects maxHeight.
func TestGetTranscodeLadderStopsAtMaxHeight(t *testing.T) {
	ladder := transcode.GetTranscodeLadder(720, "h264")

	for _, profile := range ladder {
		if profile.Height > 720 {
			t.Errorf("expected all profiles <= 720p, got %d", profile.Height)
		}
	}

	if len(ladder) == 0 {
		t.Error("expected at least one profile in ladder for maxHeight 720")
	}
}

// TestGetTranscodeLadderAV1 verifies AV1 ladder uses AV1 bitrates.
func TestGetTranscodeLadderAV1(t *testing.T) {
	ladder := transcode.GetTranscodeLadder(1080, "av1")

	if len(ladder) == 0 {
		t.Fatal("expected non-empty AV1 ladder")
	}

	for _, profile := range ladder {
		stdProfile, ok := transcode.StandardProfiles[profile.Height]
		if !ok {
			continue
		}
		if profile.Bitrate >= stdProfile.Bitrate {
			t.Errorf("AV1 profile at %dp: bitrate %d should be less than standard %d",
				profile.Height, profile.Bitrate, stdProfile.Bitrate)
		}
	}
}

// TestDefaultTranscodeLadderHeights verifies the default ladder heights.
func TestDefaultTranscodeLadderHeights(t *testing.T) {
	heights := transcode.DefaultTranscodeLadderHeights()

	expected := map[int]bool{360: true, 540: true, 720: true, 1080: true}
	if len(heights) != len(expected) {
		t.Errorf("expected %d default heights, got %d", len(expected), len(heights))
	}
	for _, h := range heights {
		if !expected[h] {
			t.Errorf("unexpected height %d in default ladder", h)
		}
	}
}

// TestPlatformRecommendedLadder verifies platform-specific ladders.
func TestPlatformRecommendedLadder(t *testing.T) {
	tests := []struct {
		platform        string
		expectedHeights []int
	}{
		{"twitch", []int{360, 480, 720, 1080}},
		{"youtube", []int{360, 480, 720, 1080, 1440, 2160}},
		{"facebook", []int{360, 720, 1080}},
		{"unknown", transcode.DefaultTranscodeLadderHeights()},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			heights := transcode.PlatformRecommendedLadder(tt.platform)
			if len(heights) != len(tt.expectedHeights) {
				t.Errorf("platform %q: expected %d heights, got %d",
					tt.platform, len(tt.expectedHeights), len(heights))
				return
			}
			for i, h := range heights {
				if h != tt.expectedHeights[i] {
					t.Errorf("platform %q index %d: expected %d, got %d",
						tt.platform, i, tt.expectedHeights[i], h)
				}
			}
		})
	}
}

// TestStandardProfileNames verifies profile Name fields match expected patterns.
func TestStandardProfileNames(t *testing.T) {
	expectedNames := map[int]string{
		360:  "360p",
		480:  "480p",
		540:  "540p",
		720:  "720p",
		1080: "1080p",
		1440: "1440p",
		2160: "2160p",
		4320: "4320p",
	}

	for height, expectedName := range expectedNames {
		profile, ok := transcode.GetStandardProfile(height)
		if !ok {
			t.Errorf("expected profile for height %d", height)
			continue
		}
		if profile.Name != expectedName {
			t.Errorf("height %d: expected Name %q, got %q", height, expectedName, profile.Name)
		}
	}
}
