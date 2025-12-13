package output

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// DASHConfig holds DASH output configuration
type DASHConfig struct {
	OutputDir       string
	SegmentDuration int // seconds
	WindowSize      int // number of segments in window
	TimeShiftBuffer int // seconds of DVR buffer
	DeleteSegments  bool
}

// DASHSegmenter handles DASH output generation
type DASHSegmenter struct {
	config    *DASHConfig
	streamKey string
	outputDir string
}

// NewDASHSegmenter creates a new DASH segmenter
func NewDASHSegmenter(streamKey string, config *DASHConfig) (*DASHSegmenter, error) {
	outputDir := filepath.Join(config.OutputDir, streamKey, "dash")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create DASH output directory: %w", err)
	}

	return &DASHSegmenter{
		config:    config,
		streamKey: streamKey,
		outputDir: outputDir,
	}, nil
}

// GetManifestPath returns the path to the DASH manifest
func (d *DASHSegmenter) GetManifestPath() string {
	return filepath.Join(d.outputDir, "manifest.mpd")
}

// GetSegmentPattern returns the segment filename pattern
func (d *DASHSegmenter) GetSegmentPattern() string {
	return filepath.Join(d.outputDir, "segment_$RepresentationID$_$Number%05d$.m4s")
}

// GetInitPattern returns the init filename pattern
func (d *DASHSegmenter) GetInitPattern() string {
	return filepath.Join(d.outputDir, "init_$RepresentationID$.mp4")
}

// GetFFmpegArgs returns FFmpeg arguments for DASH output
func (d *DASHSegmenter) GetFFmpegArgs() []string {
	return []string{
		"-f", "dash",
		"-seg_duration", fmt.Sprintf("%d", d.config.SegmentDuration),
		"-window_size", fmt.Sprintf("%d", d.config.WindowSize),
		"-extra_window_size", fmt.Sprintf("%d", d.config.WindowSize/2),
		"-use_timeline", "1",
		"-use_template", "1",
		"-adaptation_sets", "id=0,streams=v id=1,streams=a",
		"-init_seg_name", d.GetInitPattern(),
		"-media_seg_name", d.GetSegmentPattern(),
	}
}

// GenerateManifest generates a DASH MPD manifest
func (d *DASHSegmenter) GenerateManifest(variants []AdaptationSet) error {
	manifestPath := d.GetManifestPath()

	// Basic MPD structure
	content := `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011"
     type="dynamic"
     minimumUpdatePeriod="PT%dS"
     timeShiftBufferDepth="PT%dS"
     availabilityStartTime="%s"
     publishTime="%s"
     minBufferTime="PT%dS"
     profiles="urn:mpeg:dash:profile:isoff-live:2011">
  <Period start="PT0S">
`

	now := time.Now().UTC().Format(time.RFC3339)
	content = fmt.Sprintf(content,
		d.config.SegmentDuration,
		d.config.TimeShiftBuffer,
		now,
		now,
		d.config.SegmentDuration,
	)

	// Add video adaptation sets
	for _, variant := range variants {
		if variant.Type == "video" {
			content += fmt.Sprintf(`    <AdaptationSet id="%d" contentType="video" mimeType="video/mp4">
      <Representation id="%s" bandwidth="%d" width="%d" height="%d" codecs="%s">
        <SegmentTemplate timescale="1000" duration="%d000" initialization="%s" media="%s"/>
      </Representation>
    </AdaptationSet>
`,
				variant.ID,
				variant.RepresentationID,
				variant.Bandwidth,
				variant.Width,
				variant.Height,
				variant.Codecs,
				d.config.SegmentDuration,
				d.GetInitPattern(),
				d.GetSegmentPattern(),
			)
		}
	}

	// Add audio adaptation set
	content += `    <AdaptationSet id="1" contentType="audio" mimeType="audio/mp4">
      <Representation id="audio" bandwidth="192000" codecs="mp4a.40.2">
        <SegmentTemplate timescale="1000" duration="` + fmt.Sprintf("%d", d.config.SegmentDuration*1000) + `" initialization="` + d.GetInitPattern() + `" media="` + d.GetSegmentPattern() + `"/>
      </Representation>
    </AdaptationSet>
`

	content += `  </Period>
</MPD>
`

	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write DASH manifest: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"stream_key": d.streamKey,
		"path":       manifestPath,
		"variants":   len(variants),
	}).Info("Generated DASH manifest")

	return nil
}

// Cleanup removes old segments
func (d *DASHSegmenter) Cleanup() error {
	if !d.config.DeleteSegments {
		return nil
	}

	// Remove segments older than time shift buffer
	maxAge := time.Duration(d.config.TimeShiftBuffer) * time.Second

	// Clean up .m4s files
	files, err := filepath.Glob(filepath.Join(d.outputDir, "*.m4s"))
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

// AdaptationSet represents a DASH adaptation set
type AdaptationSet struct {
	ID               int
	RepresentationID string
	Type             string // video or audio
	Width            int
	Height           int
	Bandwidth        int
	Codecs           string
	Framerate        int
}
