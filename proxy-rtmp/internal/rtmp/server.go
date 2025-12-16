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

// Server handles RTMP connections and stream routing
type Server struct {
	config        *config.Config
	ffmpegManager *transcode.Manager
	listener      net.Listener
	sessions      map[string]*Session
	sessionsMutex sync.RWMutex
	running       bool
	runningMutex  sync.RWMutex
}

// NewServer creates a new RTMP server
func NewServer(cfg *config.Config, ffmpegMgr *transcode.Manager) (*Server, error) {
	return &Server{
		config:        cfg,
		ffmpegManager: ffmpegMgr,
		sessions:      make(map[string]*Session),
		running:       false,
	}, nil
}

// Start starts the RTMP server
func (s *Server) Start(ctx context.Context) error {
	s.runningMutex.Lock()
	if s.running {
		s.runningMutex.Unlock()
		return fmt.Errorf("RTMP server already running")
	}
	s.running = true
	s.runningMutex.Unlock()

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.runningMutex.Lock()
		s.running = false
		s.runningMutex.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener
	logrus.WithField("address", addr).Info("RTMP server started")

	// Accept connections
	go s.acceptLoop(ctx)

	return nil
}

// Stop stops the RTMP server
func (s *Server) Stop(ctx context.Context) error {
	s.runningMutex.Lock()
	if !s.running {
		s.runningMutex.Unlock()
		return nil
	}
	s.running = false
	s.runningMutex.Unlock()

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Stop all sessions
	s.sessionsMutex.Lock()
	for _, session := range s.sessions {
		session.Stop()
	}
	s.sessionsMutex.Unlock()

	logrus.Info("RTMP server stopped")
	return nil
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop(ctx context.Context) {
	for {
		s.runningMutex.RLock()
		running := s.running
		s.runningMutex.RUnlock()

		if !running {
			break
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if !running {
				break
			}
			logrus.WithError(err).Error("Failed to accept connection")
			continue
		}

		go s.handleConnection(ctx, conn)
	}
}

// handleConnection handles a new RTMP connection
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	logrus.WithField("client", clientAddr).Debug("New RTMP connection")

	// Perform RTMP handshake
	streamKey, err := s.performHandshake(conn)
	if err != nil {
		logrus.WithError(err).WithField("client", clientAddr).Warn("Handshake failed")
		return
	}

	if streamKey == "" {
		logrus.WithField("client", clientAddr).Warn("No stream key provided")
		return
	}

	logrus.WithFields(logrus.Fields{
		"client":     clientAddr,
		"stream_key": streamKey,
	}).Info("RTMP handshake successful")

	// Create session
	session := NewSession(streamKey, conn, s.config, s.ffmpegManager)

	// Register session
	s.sessionsMutex.Lock()
	s.sessions[streamKey] = session
	s.sessionsMutex.Unlock()

	// Handle stream
	if err := session.Start(ctx); err != nil {
		logrus.WithError(err).WithField("stream_key", streamKey).Error("Session failed")
	}

	// Unregister session
	s.sessionsMutex.Lock()
	delete(s.sessions, streamKey)
	s.sessionsMutex.Unlock()

	logrus.WithField("stream_key", streamKey).Info("Session ended")
}

// performHandshake performs RTMP handshake and extracts stream key
func (s *Server) performHandshake(conn net.Conn) (string, error) {
	// Set handshake timeout
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	// C0 + C1 (1537 bytes)
	c0c1 := make([]byte, 1537)
	if _, err := io.ReadFull(conn, c0c1); err != nil {
		return "", fmt.Errorf("failed to read C0+C1: %w", err)
	}

	// Validate version (C0)
	if c0c1[0] != 3 {
		return "", fmt.Errorf("unsupported RTMP version: %d", c0c1[0])
	}

	// Send S0 + S1 + S2
	s0 := []byte{3} // Version 3
	s1 := make([]byte, 1536)
	copy(s1, c0c1[1:1537]) // Echo C1 as S1

	s2 := make([]byte, 1536)
	// S2 is echo of C1 with timestamp

	response := append(s0, s1...)
	response = append(response, s2...)

	if _, err := conn.Write(response); err != nil {
		return "", fmt.Errorf("failed to write S0+S1+S2: %w", err)
	}

	// Read C2 (1536 bytes)
	c2 := make([]byte, 1536)
	if _, err := io.ReadFull(conn, c2); err != nil {
		return "", fmt.Errorf("failed to read C2: %w", err)
	}

	// Parse RTMP connect command to extract stream key
	// Simplified: read some bytes and extract stream key
	// In production, use a full RTMP protocol parser
	streamKey := s.extractStreamKey(conn)

	return streamKey, nil
}

// extractStreamKey extracts stream key from RTMP connect/publish commands
func (s *Server) extractStreamKey(conn net.Conn) string {
	// Simplified extraction
	// In production, implement full RTMP command parsing
	// For now, return a default key or parse from first few chunks

	// Read RTMP chunks to find connect/publish commands
	// This is a placeholder implementation
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "default"
	}

	// Simple parsing: look for stream key patterns
	// Real implementation should parse AMF0/AMF3 data
	_ = string(buf[:n]) // TODO: Parse RTMP commands to extract stream key

	// Look for common patterns in RTMP publish command
	// Stream key often appears after "publish" command
	// This is highly simplified

	return "stream_" + fmt.Sprintf("%d", time.Now().Unix())
}

// GetSession returns a session by stream key
func (s *Server) GetSession(streamKey string) (*Session, bool) {
	s.sessionsMutex.RLock()
	defer s.sessionsMutex.RUnlock()
	session, exists := s.sessions[streamKey]
	return session, exists
}

// GetAllSessions returns all active sessions
func (s *Server) GetAllSessions() []*Session {
	s.sessionsMutex.RLock()
	defer s.sessionsMutex.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetStats returns server statistics
func (s *Server) GetStats() map[string]interface{} {
	s.sessionsMutex.RLock()
	defer s.sessionsMutex.RUnlock()

	stats := map[string]interface{}{
		"total_sessions": len(s.sessions),
		"running":        s.running,
	}

	var totalBytesIn, totalBytesOut int64
	for _, session := range s.sessions {
		session.mutex.RLock()
		totalBytesIn += session.BytesIn
		totalBytesOut += session.BytesOut
		session.mutex.RUnlock()
	}

	stats["total_bytes_in"] = totalBytesIn
	stats["total_bytes_out"] = totalBytesOut

	return stats
}
