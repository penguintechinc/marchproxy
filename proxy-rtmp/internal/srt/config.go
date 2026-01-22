package srt

import (
	"fmt"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
)

// SRTConfig holds SRT-specific configuration
type SRTConfig struct {
	Port         int    // UDP port for SRT (default 8890)
	Latency      int    // Latency in milliseconds (default 120)
	Passphrase   string // Encryption passphrase (empty = no encryption)
	PBKeyLen     int    // Passphrase-based key length: 0, 16, 24, or 32
	MaxBandwidth int64  // Maximum bandwidth in bytes/sec (0 = unlimited)
	RcvBufSize   int    // Receive buffer size in packets
	SndBufSize   int    // Send buffer size in packets
	StreamID     string // Stream identifier format
}

// NewSRTConfig creates SRT config from main config
func NewSRTConfig(cfg *config.Config) *SRTConfig {
	return &SRTConfig{
		Port:         cfg.SRTPort,
		Latency:      cfg.SRTLatency,
		Passphrase:   cfg.SRTPassphrase,
		PBKeyLen:     cfg.SRTPBKLen,
		MaxBandwidth: 0, // Unlimited by default
		RcvBufSize:   8192,
		SndBufSize:   8192,
		StreamID:     "",
	}
}

// Validate validates SRT configuration
func (c *SRTConfig) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid SRT port: %d", c.Port)
	}

	if c.Latency < 20 || c.Latency > 8000 {
		return fmt.Errorf("SRT latency must be between 20 and 8000 ms: %d", c.Latency)
	}

	if c.PBKeyLen != 0 && c.PBKeyLen != 16 && c.PBKeyLen != 24 && c.PBKeyLen != 32 {
		return fmt.Errorf("invalid SRT PBKLEN: %d (must be 0, 16, 24, or 32)", c.PBKeyLen)
	}

	if c.Passphrase != "" && c.PBKeyLen == 0 {
		return fmt.Errorf("SRT PBKLEN must be set when passphrase is provided")
	}

	return nil
}

// BuildConnectOptions returns SRT connection options string
func (c *SRTConfig) BuildConnectOptions() string {
	opts := fmt.Sprintf("latency=%d", c.Latency)

	if c.Passphrase != "" && c.PBKeyLen > 0 {
		opts += fmt.Sprintf("&passphrase=%s&pbkeylen=%d", c.Passphrase, c.PBKeyLen)
	}

	if c.MaxBandwidth > 0 {
		opts += fmt.Sprintf("&maxbw=%d", c.MaxBandwidth)
	}

	return opts
}

// SRTMode represents SRT connection mode
type SRTMode string

const (
	SRTModeListener SRTMode = "listener"
	SRTModeCaller   SRTMode = "caller"
	SRTModeRendezvous SRTMode = "rendezvous"
)

// SRTSocketOption represents an SRT socket option
type SRTSocketOption struct {
	Name  string
	Value interface{}
}

// DefaultSocketOptions returns default SRT socket options
func DefaultSocketOptions(cfg *SRTConfig) []SRTSocketOption {
	opts := []SRTSocketOption{
		{Name: "SRTO_LATENCY", Value: cfg.Latency},
		{Name: "SRTO_RCVBUF", Value: cfg.RcvBufSize * 1316}, // In bytes
		{Name: "SRTO_SNDBUF", Value: cfg.SndBufSize * 1316},
		{Name: "SRTO_PEERLATENCY", Value: cfg.Latency},
		{Name: "SRTO_TSBPDMODE", Value: true},
		{Name: "SRTO_TLPKTDROP", Value: true},
		{Name: "SRTO_NAKREPORT", Value: true},
	}

	if cfg.Passphrase != "" {
		opts = append(opts,
			SRTSocketOption{Name: "SRTO_PASSPHRASE", Value: cfg.Passphrase},
			SRTSocketOption{Name: "SRTO_PBKEYLEN", Value: cfg.PBKeyLen},
		)
	}

	if cfg.MaxBandwidth > 0 {
		opts = append(opts, SRTSocketOption{Name: "SRTO_MAXBW", Value: cfg.MaxBandwidth})
	}

	return opts
}
