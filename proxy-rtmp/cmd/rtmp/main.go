package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
	"github.com/penguintech/marchproxy/proxy-rtmp/internal/grpc"
	"github.com/penguintech/marchproxy/proxy-rtmp/internal/rtmp"
	"github.com/penguintech/marchproxy/proxy-rtmp/internal/transcode"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version = "1.0.0"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "rtmp-proxy",
		Short: "MarchProxy RTMP Container with FFmpeg Transcoding",
		Long: `RTMP proxy container for MarchProxy with FFmpeg transcoding support.
Supports CPU (x264/x265) and GPU (NVENC/AMF) hardware acceleration.
Outputs HLS and DASH adaptive streams.`,
		Version: version,
		Run:     run,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/marchproxy/rtmp.yaml)")
	rootCmd.PersistentFlags().String("host", "0.0.0.0", "RTMP server host")
	rootCmd.PersistentFlags().Int("port", 1935, "RTMP server port")
	rootCmd.PersistentFlags().Int("grpc-port", 50053, "gRPC server port")
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("encoder", "auto", "Video encoder (auto, x264, x265, nvenc_h264, nvenc_h265, amf_h264, amf_h265)")
	rootCmd.PersistentFlags().String("output-dir", "/var/lib/marchproxy/streams", "Output directory for HLS/DASH segments")
	rootCmd.PersistentFlags().Bool("enable-hls", true, "Enable HLS output")
	rootCmd.PersistentFlags().Bool("enable-dash", true, "Enable DASH output")
	rootCmd.PersistentFlags().Int("segment-duration", 6, "Segment duration in seconds")
	rootCmd.PersistentFlags().String("preset", "medium", "Encoding preset (ultrafast, fast, medium, slow)")

	viper.BindPFlags(rootCmd.PersistentFlags())

	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to execute command")
	}
}

func run(cmd *cobra.Command, args []string) {
	// Initialize configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	// Configure logging
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.WithError(err).Warn("Invalid log level, using info")
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	logrus.WithFields(logrus.Fields{
		"version":    version,
		"host":       cfg.Host,
		"port":       cfg.Port,
		"grpc_port":  cfg.GRPCPort,
		"encoder":    cfg.Encoder,
		"output_dir": cfg.OutputDir,
	}).Info("Starting MarchProxy RTMP Container")

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize GPU detector and select encoder
	detector := transcode.NewDetector()
	encoderConfig, err := detector.SelectEncoder(cfg.Encoder)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to select encoder")
	}

	logrus.WithFields(logrus.Fields{
		"encoder":      encoderConfig.Name,
		"codec":        encoderConfig.Codec,
		"hw_accel":     encoderConfig.HWAccel,
		"gpu_detected": detector.HasGPU(),
	}).Info("Encoder selected")

	// Initialize FFmpeg manager
	ffmpegManager := transcode.NewManager(encoderConfig, cfg)

	// Initialize RTMP server
	rtmpServer, err := rtmp.NewServer(cfg, ffmpegManager)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create RTMP server")
	}

	// Initialize gRPC server (ModuleService)
	grpcServer := grpc.NewServer(cfg, rtmpServer, ffmpegManager)

	// Start servers
	errChan := make(chan error, 2)

	// Start RTMP server
	go func() {
		if err := rtmpServer.Start(ctx); err != nil {
			errChan <- fmt.Errorf("RTMP server error: %w", err)
		}
	}()

	// Start gRPC server
	go func() {
		if err := grpcServer.Start(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Wait for ready
	time.Sleep(100 * time.Millisecond)
	logrus.Info("All servers started successfully")

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logrus.WithError(err).Error("Server error")
	case sig := <-sigChan:
		logrus.WithField("signal", sig).Info("Received shutdown signal")
	}

	// Graceful shutdown
	logrus.Info("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop gRPC server
	grpcServer.Stop()

	// Stop RTMP server
	if err := rtmpServer.Stop(shutdownCtx); err != nil {
		logrus.WithError(err).Error("Error stopping RTMP server")
	}

	logrus.Info("Shutdown complete")
}
