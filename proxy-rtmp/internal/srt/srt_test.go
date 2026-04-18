//go:build ci

package srt

import (
	"testing"
)

func TestSRTConfig(t *testing.T) {
	config := &SRTConfig{
		Port:         8000,
		Latency:      100,
		Passphrase:   "test_pass",
		PBKeyLen:     16,
		MaxBandwidth: 10000000,
		RcvBufSize:   8192,
		SndBufSize:   8192,
		StreamID:     "stream1",
	}

	if config.Port != 8000 {
		t.Error("Port not set correctly")
	}
	if config.Latency != 100 {
		t.Error("Latency not set correctly")
	}
	if config.Passphrase != "test_pass" {
		t.Error("Passphrase not set correctly")
	}
	if config.PBKeyLen != 16 {
		t.Error("PBKeyLen not set correctly")
	}
	if config.MaxBandwidth != 10000000 {
		t.Error("MaxBandwidth not set correctly")
	}
	if config.StreamID != "stream1" {
		t.Error("StreamID not set correctly")
	}
}

func TestSRTConfigMinimal(t *testing.T) {
	config := &SRTConfig{
		Port: 8890,
	}

	if config.Port != 8890 {
		t.Error("Port not set correctly")
	}
	if config.Passphrase != "" {
		t.Error("Passphrase should be empty by default")
	}
}
