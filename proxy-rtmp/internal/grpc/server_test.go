//go:build ci

package grpc

import (
	"testing"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
)

func TestNewServerCreatesInstance(t *testing.T) {
	cfg := &config.Config{
		Host:            "localhost",
		Port:            1935,
		GRPCPort:        50051,
		Encoder:         "h264",
		EnableHLS:       true,
		EnableDASH:      false,
		SegmentDuration: 10,
	}

	srv := NewServer(cfg, nil, nil)

	if srv == nil {
		t.Fatal("NewServer should not return nil")
	}
	if srv.config != cfg {
		t.Error("config not set correctly")
	}
	if srv.rtmpServer != nil {
		t.Error("rtmpServer should be nil when passed nil")
	}
	if srv.ffmpegManager != nil {
		t.Error("ffmpegManager should be nil when passed nil")
	}
	if srv.grpcServer != nil {
		t.Error("grpcServer should be nil initially")
	}
	if srv.listener != nil {
		t.Error("listener should be nil initially")
	}
}

func TestStopWithoutStart(t *testing.T) {
	cfg := &config.Config{
		Host:     "localhost",
		GRPCPort: 50051,
	}

	srv := NewServer(cfg, nil, nil)
	// Should not panic even though grpcServer is nil
	srv.Stop()
}

func TestServerConfigFields(t *testing.T) {
	cfg := &config.Config{
		Host:            "0.0.0.0",
		Port:            8888,
		GRPCPort:        9999,
		Encoder:         "h265",
		EnableHLS:       false,
		EnableDASH:      true,
		SegmentDuration: 20,
	}

	srv := NewServer(cfg, nil, nil)

	if srv.config.Host != "0.0.0.0" {
		t.Error("Host not preserved")
	}
	if srv.config.Port != 8888 {
		t.Error("Port not preserved")
	}
	if srv.config.GRPCPort != 9999 {
		t.Error("GRPCPort not preserved")
	}
	if srv.config.Encoder != "h265" {
		t.Error("Encoder not preserved")
	}
	if srv.config.EnableHLS != false {
		t.Error("EnableHLS not preserved")
	}
	if srv.config.EnableDASH != true {
		t.Error("EnableDASH not preserved")
	}
	if srv.config.SegmentDuration != 20 {
		t.Error("SegmentDuration not preserved")
	}
}

func TestServerMultipleInstances(t *testing.T) {
	cfg1 := &config.Config{Host: "localhost", GRPCPort: 50051}
	cfg2 := &config.Config{Host: "127.0.0.1", GRPCPort: 50052}

	srv1 := NewServer(cfg1, nil, nil)
	srv2 := NewServer(cfg2, nil, nil)

	if srv1 == srv2 {
		t.Error("different NewServer calls should create different instances")
	}
	if srv1.config == srv2.config {
		t.Error("different servers should have different configs")
	}
	if srv1.config.Host != "localhost" {
		t.Error("srv1 config not isolated")
	}
	if srv2.config.Host != "127.0.0.1" {
		t.Error("srv2 config not isolated")
	}
}
