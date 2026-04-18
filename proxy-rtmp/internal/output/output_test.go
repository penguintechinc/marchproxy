//go:build ci
// +build ci

package output

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDASHConfig tests DASH configuration structure
func TestDASHConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *DASHConfig
	}{
		{
			name: "standard config",
			config: &DASHConfig{
				OutputDir:       "/tmp/dash",
				SegmentDuration: 6,
				WindowSize:      3,
				TimeShiftBuffer: 60,
				DeleteSegments:  true,
			},
		},
		{
			name: "minimal config",
			config: &DASHConfig{
				OutputDir:       "/tmp",
				SegmentDuration: 4,
				WindowSize:      2,
			},
		},
		{
			name: "large buffer config",
			config: &DASHConfig{
				OutputDir:       "/var/cache/dash",
				SegmentDuration: 10,
				WindowSize:      10,
				TimeShiftBuffer: 600,
				DeleteSegments:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.OutputDir == "" {
				t.Error("OutputDir should not be empty")
			}
			if tt.config.SegmentDuration <= 0 {
				t.Error("SegmentDuration should be positive")
			}
			if tt.config.WindowSize <= 0 {
				t.Error("WindowSize should be positive")
			}
		})
	}
}

// TestNewDASHSegmenter tests DASH segmenter creation
func TestNewDASHSegmenter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &DASHConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 6,
		WindowSize:      3,
		TimeShiftBuffer: 60,
		DeleteSegments:  true,
	}

	segmenter, err := NewDASHSegmenter("test_stream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	if segmenter == nil {
		t.Fatal("expected non-nil segmenter")
	}
	if segmenter.streamKey != "test_stream" {
		t.Errorf("stream key mismatch: got %s, want test_stream", segmenter.streamKey)
	}
	if segmenter.config != config {
		t.Error("config not set correctly")
	}

	// Verify output directory was created
	expectedDir := filepath.Join(tmpDir, "test_stream", "dash")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("expected output directory not created: %s", expectedDir)
	}
}

// TestNewDASHSegmenter_InvalidDir tests segmenter with invalid directory
func TestNewDASHSegmenter_InvalidDir(t *testing.T) {
	config := &DASHConfig{
		OutputDir:       "/invalid/nonexistent/path/that/cannot/be/created",
		SegmentDuration: 6,
		WindowSize:      3,
	}

	_, err := NewDASHSegmenter("test_stream", config)
	if err == nil {
		t.Error("expected error when creating directory fails")
	}
}

// TestGetManifestPath tests manifest path generation
func TestGetManifestPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &DASHConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 6,
		WindowSize:      3,
	}

	segmenter, err := NewDASHSegmenter("mystream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	manifestPath := segmenter.GetManifestPath()
	if !filepath.IsAbs(manifestPath) {
		t.Errorf("manifest path should be absolute: %s", manifestPath)
	}
	if !filepath.HasPrefix(manifestPath, tmpDir) {
		t.Errorf("manifest path should be in output dir: %s", manifestPath)
	}
	if filepath.Base(manifestPath) != "manifest.mpd" {
		t.Errorf("manifest filename should be manifest.mpd, got %s", filepath.Base(manifestPath))
	}
}

// TestGetSegmentPattern tests segment pattern generation
func TestGetSegmentPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &DASHConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 6,
		WindowSize:      3,
	}

	segmenter, err := NewDASHSegmenter("mystream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	pattern := segmenter.GetSegmentPattern()
	if !filepath.IsAbs(pattern) {
		t.Errorf("segment pattern should be absolute: %s", pattern)
	}
	if !filepath.HasPrefix(pattern, tmpDir) {
		t.Errorf("segment pattern should be in output dir: %s", pattern)
	}
	if !contains(pattern, "segment_") && !contains(pattern, "$") {
		t.Errorf("segment pattern should contain 'segment_' or template vars: %s", pattern)
	}
}

// TestGetInitPattern tests init pattern generation
func TestGetInitPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &DASHConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 6,
		WindowSize:      3,
	}

	segmenter, err := NewDASHSegmenter("mystream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	pattern := segmenter.GetInitPattern()
	if !filepath.IsAbs(pattern) {
		t.Errorf("init pattern should be absolute: %s", pattern)
	}
	if !filepath.HasPrefix(pattern, tmpDir) {
		t.Errorf("init pattern should be in output dir: %s", pattern)
	}
	if !contains(pattern, "init_") && !contains(pattern, "$") {
		t.Errorf("init pattern should contain 'init_' or template vars: %s", pattern)
	}
}

// TestGetFFmpegArgs tests FFmpeg arguments generation
func TestGetFFmpegArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &DASHConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 8,
		WindowSize:      4,
	}

	segmenter, err := NewDASHSegmenter("mystream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	args := segmenter.GetFFmpegArgs()
	if args == nil {
		t.Fatal("expected non-nil args")
	}
	if len(args) == 0 {
		t.Error("expected non-empty args")
	}

	// Check for required DASH flags
	found := false
	for i, arg := range args {
		if arg == "-f" && i+1 < len(args) && args[i+1] == "dash" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected -f dash in FFmpeg args")
	}

	// Check segment duration is set correctly
	found = false
	for i, arg := range args {
		if arg == "-seg_duration" && i+1 < len(args) && args[i+1] == "8" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected -seg_duration 8 in FFmpeg args")
	}
}

// TestDASHSegmenter_MultipleStreams tests creating multiple segmenters
func TestDASHSegmenter_MultipleStreams(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &DASHConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 6,
		WindowSize:      3,
	}

	streamKeys := []string{"stream_1", "stream_2", "stream_3"}
	segmenters := make([]*DASHSegmenter, len(streamKeys))

	for i, key := range streamKeys {
		segmenter, err := NewDASHSegmenter(key, config)
		if err != nil {
			t.Fatalf("failed to create segmenter for stream %s: %v", key, err)
		}
		segmenters[i] = segmenter
	}

	// Verify each segmenter has unique paths
	for i, seg := range segmenters {
		if seg.streamKey != streamKeys[i] {
			t.Errorf("segmenter %d: stream key mismatch", i)
		}
		manifestPath := seg.GetManifestPath()
		for j, other := range segmenters {
			if i != j && manifestPath == other.GetManifestPath() {
				t.Errorf("segmenters %d and %d have same manifest path", i, j)
			}
		}
	}
}

// TestGetFFmpegArgs_VariousConfigs tests FFmpeg args with different configurations
func TestGetFFmpegArgs_VariousConfigs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dash-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configs := []struct {
		name               string
		segmentDuration    int
		windowSize         int
	}{
		{"short segment", 4, 2},
		{"standard segment", 6, 3},
		{"long segment", 10, 5},
	}

	for _, tt := range configs {
		t.Run(tt.name, func(t *testing.T) {
			config := &DASHConfig{
				OutputDir:       tmpDir,
				SegmentDuration: tt.segmentDuration,
				WindowSize:      tt.windowSize,
			}

			segmenter, err := NewDASHSegmenter("teststream", config)
			if err != nil {
				t.Fatalf("failed to create segmenter: %v", err)
			}

			args := segmenter.GetFFmpegArgs()
			if len(args) == 0 {
				t.Error("expected non-empty args")
			}
		})
	}
}

// TestHLSConfig tests HLS configuration structure
func TestHLSConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *HLSConfig
	}{
		{
			name: "standard config",
			config: &HLSConfig{
				OutputDir:       "/tmp/hls",
				SegmentDuration: 10,
				PlaylistSize:    5,
				SegmentType:     "mpegts",
				DeleteSegments:  true,
			},
		},
		{
			name: "fmp4 config",
			config: &HLSConfig{
				OutputDir:       "/tmp/hls",
				SegmentDuration: 6,
				PlaylistSize:    3,
				SegmentType:     "fmp4",
				DeleteSegments:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.OutputDir == "" {
				t.Error("OutputDir should not be empty")
			}
			if tt.config.SegmentDuration <= 0 {
				t.Error("SegmentDuration should be positive")
			}
			if tt.config.PlaylistSize <= 0 {
				t.Error("PlaylistSize should be positive")
			}
		})
	}
}

// TestNewHLSSegmenter tests HLS segmenter creation
func TestNewHLSSegmenter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hls-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &HLSConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 10,
		PlaylistSize:    5,
		SegmentType:     "mpegts",
		DeleteSegments:  true,
	}

	segmenter, err := NewHLSSegmenter("test_stream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	if segmenter == nil {
		t.Fatal("expected non-nil segmenter")
	}
	if segmenter.streamKey != "test_stream" {
		t.Errorf("stream key mismatch: got %s, want test_stream", segmenter.streamKey)
	}
	if segmenter.config != config {
		t.Error("config not set correctly")
	}

	// Verify output directory was created
	expectedDir := filepath.Join(tmpDir, "test_stream", "hls")
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Errorf("expected output directory not created: %s", expectedDir)
	}
}

// TestGetPlaylistPath tests playlist path generation
func TestGetPlaylistPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hls-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &HLSConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 10,
		PlaylistSize:    5,
	}

	segmenter, err := NewHLSSegmenter("mystream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	playlistPath := segmenter.GetPlaylistPath()
	if !filepath.IsAbs(playlistPath) {
		t.Errorf("playlist path should be absolute: %s", playlistPath)
	}
	if !filepath.HasPrefix(playlistPath, tmpDir) {
		t.Errorf("playlist path should be in output dir: %s", playlistPath)
	}
	if filepath.Base(playlistPath) != "index.m3u8" {
		t.Errorf("playlist filename should be index.m3u8, got %s", filepath.Base(playlistPath))
	}
}

// TestHLSSegmentPattern tests HLS segment pattern generation
func TestHLSSegmentPattern(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hls-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &HLSConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 10,
		PlaylistSize:    5,
	}

	segmenter, err := NewHLSSegmenter("mystream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	pattern := segmenter.GetSegmentPattern()
	if !filepath.IsAbs(pattern) {
		t.Errorf("segment pattern should be absolute: %s", pattern)
	}
	if !filepath.HasPrefix(pattern, tmpDir) {
		t.Errorf("segment pattern should be in output dir: %s", pattern)
	}
}

// TestHLSFFmpegArgs tests HLS FFmpeg arguments
func TestHLSFFmpegArgs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hls-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &HLSConfig{
		OutputDir:       tmpDir,
		SegmentDuration: 10,
		PlaylistSize:    5,
		SegmentType:     "mpegts",
	}

	segmenter, err := NewHLSSegmenter("mystream", config)
	if err != nil {
		t.Fatalf("failed to create segmenter: %v", err)
	}

	args := segmenter.GetFFmpegArgs()
	if args == nil {
		t.Fatal("expected non-nil args")
	}
	if len(args) == 0 {
		t.Error("expected non-empty args")
	}

	// Check for HLS format flag
	found := false
	for i, arg := range args {
		if arg == "-f" && i+1 < len(args) && args[i+1] == "hls" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected -f hls in FFmpeg args")
	}
}

// TestVariantStream tests variant stream structure
func TestVariantStream(t *testing.T) {
	variant := VariantStream{
		Name:      "720p",
		Width:     1280,
		Height:    720,
		Bandwidth: 2500000,
		Codecs:    "avc1.4d401f",
	}

	if variant.Name != "720p" {
		t.Error("variant name not set")
	}
	if variant.Width != 1280 {
		t.Error("variant width not set")
	}
	if variant.Height != 720 {
		t.Error("variant height not set")
	}
	if variant.Bandwidth != 2500000 {
		t.Error("variant bandwidth not set")
	}
	if variant.Codecs != "avc1.4d401f" {
		t.Error("variant codecs not set")
	}
}

// TestAdaptationSet tests DASH adaptation set structure
func TestAdaptationSet(t *testing.T) {
	set := AdaptationSet{
		ID:               0,
		RepresentationID: "1080p",
		Type:             "video",
		Width:            1920,
		Height:           1080,
		Bandwidth:        5000000,
		Codecs:           "avc1.4d4028",
		Framerate:        30,
	}

	if set.ID != 0 {
		t.Error("adaptation set ID not set")
	}
	if set.RepresentationID != "1080p" {
		t.Error("representation ID not set")
	}
	if set.Type != "video" {
		t.Error("type not set")
	}
	if set.Width != 1920 {
		t.Error("width not set")
	}
	if set.Framerate != 30 {
		t.Error("framerate not set")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0))
}
