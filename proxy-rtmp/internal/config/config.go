package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds the RTMP proxy configuration
type Config struct {
	// Server settings
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	GRPCPort int    `mapstructure:"grpc-port"`

	// Logging
	LogLevel string `mapstructure:"log-level"`

	// Encoder settings
	Encoder   string `mapstructure:"encoder"`    // auto, x264, x265, nvenc_h264, nvenc_h265, amf_h264, amf_h265, libaom_av1, svt_av1, nvenc_av1, amf_av1
	Preset    string `mapstructure:"preset"`     // ultrafast, fast, medium, slow
	PreferAV1 bool   `mapstructure:"prefer-av1"` // Prefer AV1 codec when available

	// Output settings
	OutputDir       string `mapstructure:"output-dir"`
	EnableHLS       bool   `mapstructure:"enable-hls"`
	EnableDASH      bool   `mapstructure:"enable-dash"`
	SegmentDuration int    `mapstructure:"segment-duration"` // seconds

	// FFmpeg settings
	FFmpegPath    string            `mapstructure:"ffmpeg-path"`
	FFprobePath   string            `mapstructure:"ffprobe-path"`
	EncoderParams map[string]string `mapstructure:"encoder-params"`

	// Resolution limits
	MaxResolutionCPU int `mapstructure:"max-resolution-cpu"` // Max height for CPU encoding (default 1440)
	MaxResolutionGPU int `mapstructure:"max-resolution-gpu"` // Max height for GPU encoding (default 4320)

	// Rate limiting (per route)
	MaxBitrate    int `mapstructure:"max-bitrate"`    // Mbps
	MaxStreams    int `mapstructure:"max-streams"`    // concurrent streams
	MaxResolution int `mapstructure:"max-resolution"` // height in pixels (legacy, use resolution limits)

	// Transcode ladder settings
	TranscodeLadderEnabled     bool  `mapstructure:"transcode-ladder-enabled"`
	TranscodeLadderResolutions []int `mapstructure:"transcode-ladder-resolutions"` // e.g., [360, 540, 720, 1080]

	// SRT settings
	EnableSRT     bool   `mapstructure:"enable-srt"`
	SRTPort       int    `mapstructure:"srt-port"`
	SRTLatency    int    `mapstructure:"srt-latency"`    // milliseconds
	SRTPassphrase string `mapstructure:"srt-passphrase"` // encryption key
	SRTPBKLen     int    `mapstructure:"srt-pbklen"`     // 16, 24, or 32

	// WebRTC settings
	EnableWebRTC    bool     `mapstructure:"enable-webrtc"`
	WHIPPort        int      `mapstructure:"whip-port"`
	WHEPPort        int      `mapstructure:"whep-port"`
	STUNServers     []string `mapstructure:"stun-servers"`
	TURNServers     []string `mapstructure:"turn-servers"`
	WebRTCICEPolicy string   `mapstructure:"webrtc-ice-policy"` // "all" or "relay"

	// Health check
	HealthCheckInterval int `mapstructure:"health-check-interval"` // seconds
}

// Load loads configuration from file and environment
func Load(cfgFile string) (*Config, error) {
	// Set defaults
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("port", 1935)
	viper.SetDefault("grpc-port", 50053)
	viper.SetDefault("log-level", "info")
	viper.SetDefault("encoder", "auto")
	viper.SetDefault("preset", "medium")
	viper.SetDefault("prefer-av1", false)
	viper.SetDefault("output-dir", "/var/lib/marchproxy/streams")
	viper.SetDefault("enable-hls", true)
	viper.SetDefault("enable-dash", true)
	viper.SetDefault("segment-duration", 6)
	viper.SetDefault("ffmpeg-path", "ffmpeg")
	viper.SetDefault("ffprobe-path", "ffprobe")
	viper.SetDefault("max-resolution-cpu", 1440)  // 2K max for CPU
	viper.SetDefault("max-resolution-gpu", 4320)  // 8K max for GPU
	viper.SetDefault("max-bitrate", 10)           // 10 Mbps default
	viper.SetDefault("max-streams", 100)          // 100 concurrent streams
	viper.SetDefault("max-resolution", 1080)      // 1080p max (legacy)
	viper.SetDefault("transcode-ladder-enabled", true)
	viper.SetDefault("transcode-ladder-resolutions", []int{360, 540, 720, 1080})
	// SRT defaults
	viper.SetDefault("enable-srt", false)
	viper.SetDefault("srt-port", 8890)
	viper.SetDefault("srt-latency", 120)    // 120ms default
	viper.SetDefault("srt-passphrase", "")
	viper.SetDefault("srt-pbklen", 16)
	// WebRTC defaults
	viper.SetDefault("enable-webrtc", false)
	viper.SetDefault("whip-port", 8080)
	viper.SetDefault("whep-port", 8081)
	viper.SetDefault("stun-servers", []string{"stun:stun.l.google.com:19302"})
	viper.SetDefault("turn-servers", []string{})
	viper.SetDefault("webrtc-ice-policy", "all")
	viper.SetDefault("health-check-interval", 30)

	// Load config file if specified
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("rtmp")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/marchproxy/")
		viper.AddConfigPath(".")
	}

	// Environment variables
	viper.SetEnvPrefix("RTMP")
	viper.AutomaticEnv()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	// Unmarshal config
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate and create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.GRPCPort < 1 || c.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.GRPCPort)
	}

	if c.SegmentDuration < 1 || c.SegmentDuration > 60 {
		return fmt.Errorf("segment duration must be between 1 and 60 seconds")
	}

	validEncoders := map[string]bool{
		"auto": true, "x264": true, "x265": true,
		"nvenc_h264": true, "nvenc_h265": true,
		"amf_h264": true, "amf_h265": true,
		// AV1 encoders
		"libaom_av1": true, "svt_av1": true,
		"nvenc_av1": true, "amf_av1": true,
	}
	if !validEncoders[c.Encoder] {
		return fmt.Errorf("invalid encoder: %s", c.Encoder)
	}

	validPresets := map[string]bool{
		"ultrafast": true, "superfast": true, "veryfast": true,
		"faster": true, "fast": true, "medium": true,
		"slow": true, "slower": true, "veryslow": true,
	}
	if !validPresets[c.Preset] {
		return fmt.Errorf("invalid preset: %s", c.Preset)
	}

	// Validate resolution limits
	if c.MaxResolutionCPU < 360 || c.MaxResolutionCPU > 4320 {
		return fmt.Errorf("max-resolution-cpu must be between 360 and 4320")
	}
	if c.MaxResolutionGPU < 360 || c.MaxResolutionGPU > 4320 {
		return fmt.Errorf("max-resolution-gpu must be between 360 and 4320")
	}

	// Validate SRT settings if enabled
	if c.EnableSRT {
		if c.SRTPort < 1 || c.SRTPort > 65535 {
			return fmt.Errorf("invalid SRT port: %d", c.SRTPort)
		}
		if c.SRTLatency < 20 || c.SRTLatency > 8000 {
			return fmt.Errorf("SRT latency must be between 20 and 8000 ms")
		}
		if c.SRTPBKLen != 0 && c.SRTPBKLen != 16 && c.SRTPBKLen != 24 && c.SRTPBKLen != 32 {
			return fmt.Errorf("SRT PBKLEN must be 0 (disabled), 16, 24, or 32")
		}
	}

	// Validate WebRTC settings if enabled
	if c.EnableWebRTC {
		if c.WHIPPort < 1 || c.WHIPPort > 65535 {
			return fmt.Errorf("invalid WHIP port: %d", c.WHIPPort)
		}
		if c.WHEPPort < 1 || c.WHEPPort > 65535 {
			return fmt.Errorf("invalid WHEP port: %d", c.WHEPPort)
		}
		if c.WebRTCICEPolicy != "all" && c.WebRTCICEPolicy != "relay" {
			return fmt.Errorf("invalid WebRTC ICE policy: %s (must be 'all' or 'relay')", c.WebRTCICEPolicy)
		}
	}

	// Validate transcode ladder resolutions
	for _, res := range c.TranscodeLadderResolutions {
		validResolutions := map[int]bool{
			360: true, 480: true, 540: true, 720: true,
			1080: true, 1440: true, 2160: true, 4320: true,
		}
		if !validResolutions[res] {
			return fmt.Errorf("invalid transcode ladder resolution: %d", res)
		}
	}

	return nil
}
