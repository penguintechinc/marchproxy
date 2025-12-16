package rtmp

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
	"github.com/penguintech/marchproxy/proxy-rtmp/internal/transcode"
	"github.com/sirupsen/logrus"
)

// SessionStatus represents session status
type SessionStatus string

const (
	SessionIdle       SessionStatus = "idle"
	SessionConnecting SessionStatus = "connecting"
	SessionActive     SessionStatus = "active"
	SessionStopping   SessionStatus = "stopping"
	SessionStopped    SessionStatus = "stopped"
	SessionError      SessionStatus = "error"
)

// Session represents an active RTMP streaming session
type Session struct {
	ID            string
	StreamKey     string
	ClientAddr    string
	Conn          net.Conn
	Status        SessionStatus
	StartTime     time.Time
	StopTime      time.Time
	BytesIn       int64
	BytesOut      int64
	Error         error
	config        *config.Config
	ffmpegManager *transcode.Manager
	ffmpegProc    *transcode.Process
	mutex         sync.RWMutex
	stopChan      chan struct{}
	stopped       bool
}

// NewSession creates a new RTMP session
func NewSession(streamKey string, conn net.Conn, cfg *config.Config, ffmpegMgr *transcode.Manager) *Session {
	return &Session{
		ID:            fmt.Sprintf("sess_%s_%d", streamKey, time.Now().UnixNano()),
		StreamKey:     streamKey,
		ClientAddr:    conn.RemoteAddr().String(),
		Conn:          conn,
		Status:        SessionIdle,
		StartTime:     time.Now(),
		config:        cfg,
		ffmpegManager: ffmpegMgr,
		stopChan:      make(chan struct{}),
		stopped:       false,
	}
}

// Start starts the session
func (s *Session) Start(ctx context.Context) error {
	s.mutex.Lock()
	s.Status = SessionConnecting
	s.mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"session_id": s.ID,
		"stream_key": s.StreamKey,
		"client":     s.ClientAddr,
	}).Info("Starting RTMP session")

	// Create input URL for FFmpeg (RTMP pipe)
	inputURL := fmt.Sprintf("rtmp://127.0.0.1:%d/live/%s", s.config.Port, s.StreamKey)

	// Start FFmpeg transcoding
	// Use default 1080p bitrate config
	bitrate := transcode.DefaultBitrateLadder()[0]
	proc, err := s.ffmpegManager.StartTranscode(ctx, s.StreamKey, inputURL, bitrate)
	if err != nil {
		s.mutex.Lock()
		s.Status = SessionError
		s.Error = err
		s.mutex.Unlock()
		return fmt.Errorf("failed to start transcoding: %w", err)
	}

	s.mutex.Lock()
	s.ffmpegProc = proc
	s.Status = SessionActive
	s.mutex.Unlock()

	logrus.WithField("session_id", s.ID).Info("Session active, streaming started")

	// Handle stream data
	if err := s.handleStream(ctx); err != nil {
		s.mutex.Lock()
		s.Status = SessionError
		s.Error = err
		s.mutex.Unlock()
		return err
	}

	return nil
}

// handleStream handles the RTMP stream data
func (s *Session) handleStream(ctx context.Context) error {
	// Read RTMP chunks and forward to FFmpeg
	// Simplified implementation: just track bytes
	// Real implementation would parse RTMP chunks and handle commands

	buf := make([]byte, 4096)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.stopChan:
			return nil
		case <-ticker.C:
			// Update stats periodically
			s.updateStats()
		default:
			// Set read timeout
			s.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			n, err := s.Conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					logrus.WithField("session_id", s.ID).Info("Client disconnected")
					return nil
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return fmt.Errorf("read error: %w", err)
			}

			// Track bytes received
			s.mutex.Lock()
			s.BytesIn += int64(n)
			s.mutex.Unlock()

			// In real implementation, parse RTMP chunks here
			// and feed video/audio data to FFmpeg stdin
			// For now, we're using FFmpeg's RTMP input directly
		}
	}
}

// updateStats updates session statistics
func (s *Session) updateStats() {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	duration := time.Since(s.StartTime)
	bitrateIn := float64(s.BytesIn*8) / duration.Seconds() / 1000000 // Mbps

	logrus.WithFields(logrus.Fields{
		"session_id": s.ID,
		"stream_key": s.StreamKey,
		"bytes_in":   s.BytesIn,
		"bytes_out":  s.BytesOut,
		"bitrate_in": fmt.Sprintf("%.2f Mbps", bitrateIn),
		"duration":   duration.String(),
	}).Debug("Session stats")
}

// Stop stops the session
func (s *Session) Stop() error {
	s.mutex.Lock()
	if s.stopped {
		s.mutex.Unlock()
		return nil
	}
	s.stopped = true
	s.Status = SessionStopping
	s.mutex.Unlock()

	logrus.WithField("session_id", s.ID).Info("Stopping session")

	// Signal stop
	close(s.stopChan)

	// Stop FFmpeg process
	if s.ffmpegProc != nil {
		if err := s.ffmpegManager.StopTranscode(s.StreamKey); err != nil {
			logrus.WithError(err).Warn("Failed to stop FFmpeg process")
		}
	}

	// Close connection
	if s.Conn != nil {
		s.Conn.Close()
	}

	s.mutex.Lock()
	s.Status = SessionStopped
	s.StopTime = time.Now()
	s.mutex.Unlock()

	return nil
}

// GetInfo returns session information
func (s *Session) GetInfo() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	duration := time.Since(s.StartTime)
	info := map[string]interface{}{
		"id":          s.ID,
		"stream_key":  s.StreamKey,
		"client_addr": s.ClientAddr,
		"status":      s.Status,
		"start_time":  s.StartTime,
		"duration":    duration.String(),
		"bytes_in":    s.BytesIn,
		"bytes_out":   s.BytesOut,
	}

	if !s.StopTime.IsZero() {
		info["stop_time"] = s.StopTime
		info["total_duration"] = s.StopTime.Sub(s.StartTime).String()
	}

	if s.Error != nil {
		info["error"] = s.Error.Error()
	}

	if s.ffmpegProc != nil {
		info["encoder"] = s.ffmpegProc.Encoder.Name
		info["codec"] = s.ffmpegProc.Encoder.Codec
		info["bitrate_profile"] = s.ffmpegProc.Bitrate.Name
	}

	return info
}
