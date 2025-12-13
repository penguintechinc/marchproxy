package output

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// HLSConfig holds HLS output configuration
type HLSConfig struct {
	OutputDir       string
	SegmentDuration int    // seconds
	PlaylistSize    int    // number of segments in playlist
	SegmentType     string // mpegts or fmp4
	DeleteSegments  bool   // delete old segments
}

// HLSSegmenter handles HLS output generation
type HLSSegmenter struct {
	config    *HLSConfig
	streamKey string
	outputDir string
}

// NewHLSSegmenter creates a new HLS segmenter
func NewHLSSegmenter(streamKey string, config *HLSConfig) (*HLSSegmenter, error) {
	outputDir := filepath.Join(config.OutputDir, streamKey, "hls")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create HLS output directory: %w", err)
	}

	return &HLSSegmenter{
		config:    config,
		streamKey: streamKey,
		outputDir: outputDir,
	}, nil
}

// GetPlaylistPath returns the path to the HLS playlist
func (h *HLSSegmenter) GetPlaylistPath() string {
	return filepath.Join(h.outputDir, "index.m3u8")
}

// GetSegmentPattern returns the segment filename pattern
func (h *HLSSegmenter) GetSegmentPattern() string {
	return filepath.Join(h.outputDir, "segment_%03d.ts")
}

// GetFFmpegArgs returns FFmpeg arguments for HLS output
func (h *HLSSegmenter) GetFFmpegArgs() []string {
	return []string{
		"-f", "hls",
		"-hls_time", fmt.Sprintf("%d", h.config.SegmentDuration),
		"-hls_list_size", fmt.Sprintf("%d", h.config.PlaylistSize),
		"-hls_segment_type", h.config.SegmentType,
		"-hls_segment_filename", h.GetSegmentPattern(),
	}
}

// GenerateMasterPlaylist generates a master playlist for adaptive streaming
func (h *HLSSegmenter) GenerateMasterPlaylist(variants []VariantStream) error {
	masterPath := filepath.Join(h.config.OutputDir, h.streamKey, "master.m3u8")

	content := "#EXTM3U\n"
	content += "#EXT-X-VERSION:3\n\n"

	for _, variant := range variants {
		content += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,NAME=\"%s\"\n",
			variant.Bandwidth,
			variant.Width,
			variant.Height,
			variant.Name,
		)
		content += fmt.Sprintf("hls/%s/index.m3u8\n\n", variant.Name)
	}

	if err := os.WriteFile(masterPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write master playlist: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"stream_key": h.streamKey,
		"path":       masterPath,
		"variants":   len(variants),
	}).Info("Generated HLS master playlist")

	return nil
}

// Cleanup removes old segments
func (h *HLSSegmenter) Cleanup() error {
	if !h.config.DeleteSegments {
		return nil
	}

	// Remove all .ts files older than playlist duration
	maxAge := time.Duration(h.config.SegmentDuration*h.config.PlaylistSize) * time.Second

	files, err := filepath.Glob(filepath.Join(h.outputDir, "*.ts"))
	if err != nil {
		return err
	}

	now := time.Now()
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > maxAge {
			if err := os.Remove(file); err != nil {
				logrus.WithError(err).WithField("file", file).Warn("Failed to delete old segment")
			}
		}
	}

	return nil
}

// VariantStream represents a variant stream configuration
type VariantStream struct {
	Name      string // e.g., "1080p", "720p"
	Width     int
	Height    int
	Bandwidth int // bits per second
	Codecs    string
}
