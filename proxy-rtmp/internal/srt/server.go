package srt

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
	"github.com/sirupsen/logrus"
)

// Server handles SRT connections
type Server struct {
	config     *SRTConfig
	mainConfig *config.Config
	listener   net.Listener
	sessions   map[string]*Session
	mutex      sync.RWMutex
	running    bool
	stats      *ServerStats
	ctx        context.Context
	cancel     context.CancelFunc
	onStream   func(streamKey string, session *Session)
}

// ServerStats holds server statistics
type ServerStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	TotalBytesReceived  int64
	TotalBytesSent      int64
	ConnectionsRejected int64
	mutex               sync.RWMutex
}

// NewServer creates a new SRT server
func NewServer(cfg *config.Config, onStream func(streamKey string, session *Session)) *Server {
	srtConfig := NewSRTConfig(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config:     srtConfig,
		mainConfig: cfg,
		sessions:   make(map[string]*Session),
		stats:      &ServerStats{},
		ctx:        ctx,
		cancel:     cancel,
		onStream:   onStream,
	}
}

// Start starts the SRT server
func (s *Server) Start() error {
	if err := s.config.Validate(); err != nil {
		return fmt.Errorf("invalid SRT config: %w", err)
	}

	// Create UDP listener for SRT
	// Note: This is a simplified implementation
	// In production, use a proper SRT library like github.com/datarhei/gosrt
	addr := fmt.Sprintf(":%d", s.config.Port)

	// For now, we'll use a UDP-based approach
	// Real SRT requires the SRT protocol implementation
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP %s: %w", addr, err)
	}

	// Wrap UDP connection in a custom listener
	s.listener = &srtListener{
		conn:   conn,
		config: s.config,
	}

	s.running = true

	logrus.WithFields(logrus.Fields{
		"port":    s.config.Port,
		"latency": s.config.Latency,
	}).Info("SRT server started")

	go s.acceptLoop()

	return nil
}

// acceptLoop handles incoming SRT connections
func (s *Server) acceptLoop() {
	for s.running {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.running {
					logrus.WithError(err).Error("Failed to accept SRT connection")
				}
				continue
			}

			s.stats.mutex.Lock()
			s.stats.TotalConnections++
			s.stats.ActiveConnections++
			s.stats.mutex.Unlock()

			go s.handleConnection(conn)
		}
	}
}

// handleConnection processes an SRT connection
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.stats.mutex.Lock()
		s.stats.ActiveConnections--
		s.stats.mutex.Unlock()
	}()

	// Extract stream ID from SRT connection
	streamKey := s.extractStreamKey(conn)
	if streamKey == "" {
		logrus.Warn("SRT connection without stream key rejected")
		s.stats.mutex.Lock()
		s.stats.ConnectionsRejected++
		s.stats.mutex.Unlock()
		return
	}

	// Create session
	session := NewSession(streamKey, conn, s.config, s.mainConfig)

	s.mutex.Lock()
	if _, exists := s.sessions[streamKey]; exists {
		s.mutex.Unlock()
		logrus.WithField("stream_key", streamKey).Warn("Duplicate SRT stream rejected")
		s.stats.mutex.Lock()
		s.stats.ConnectionsRejected++
		s.stats.mutex.Unlock()
		return
	}
	s.sessions[streamKey] = session
	s.mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"stream_key":   streamKey,
		"remote_addr":  conn.RemoteAddr(),
	}).Info("SRT stream connected")

	// Notify callback
	if s.onStream != nil {
		s.onStream(streamKey, session)
	}

	// Start session (blocks until complete)
	err := session.Start(s.ctx)
	if err != nil {
		logrus.WithError(err).WithField("stream_key", streamKey).Error("SRT session error")
	}

	// Cleanup
	s.mutex.Lock()
	delete(s.sessions, streamKey)
	s.mutex.Unlock()

	// Update stats
	s.stats.mutex.Lock()
	s.stats.TotalBytesReceived += session.stats.BytesReceived
	s.stats.TotalBytesSent += session.stats.BytesSent
	s.stats.mutex.Unlock()

	logrus.WithField("stream_key", streamKey).Info("SRT stream disconnected")
}

// extractStreamKey extracts stream key from SRT connection
// SRT uses streamid socket option for stream identification
func (s *Server) extractStreamKey(conn net.Conn) string {
	// In a real implementation, this would read the SRT streamid option
	// For now, we'll use the connection info
	// The format is typically: #!::r=<resource>,m=<mode>
	// or simply the stream key as the streamid

	// Simplified: use remote address hash as placeholder
	// Real implementation would use gosrt library to get streamid
	remoteAddr := conn.RemoteAddr().String()

	// Check if we have a custom SRT connection type
	if srtConn, ok := conn.(*srtConnection); ok {
		return srtConn.streamID
	}

	// Fallback: generate from address (not ideal for production)
	return fmt.Sprintf("srt_%s_%d", remoteAddr, time.Now().UnixNano())
}

// Stop stops the SRT server
func (s *Server) Stop() error {
	s.running = false
	s.cancel()

	if s.listener != nil {
		s.listener.Close()
	}

	// Stop all active sessions
	s.mutex.Lock()
	for _, session := range s.sessions {
		session.Stop()
	}
	s.mutex.Unlock()

	logrus.Info("SRT server stopped")
	return nil
}

// GetSession returns a session by stream key
func (s *Server) GetSession(streamKey string) (*Session, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	session, ok := s.sessions[streamKey]
	return session, ok
}

// GetAllSessions returns all active sessions
func (s *Server) GetAllSessions() []*Session {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sessions := make([]*Session, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetStats returns server statistics
func (s *Server) GetStats() map[string]interface{} {
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	return map[string]interface{}{
		"total_connections":     s.stats.TotalConnections,
		"active_connections":    s.stats.ActiveConnections,
		"total_bytes_received":  s.stats.TotalBytesReceived,
		"total_bytes_sent":      s.stats.TotalBytesSent,
		"connections_rejected":  s.stats.ConnectionsRejected,
		"port":                  s.config.Port,
		"latency_ms":            s.config.Latency,
	}
}

// IsRunning returns true if server is running
func (s *Server) IsRunning() bool {
	return s.running
}

// srtListener wraps UDP connection as a net.Listener
type srtListener struct {
	conn   *net.UDPConn
	config *SRTConfig
}

func (l *srtListener) Accept() (net.Conn, error) {
	// This is a simplified implementation
	// Real SRT requires proper handshake handling
	buf := make([]byte, 65536)

	l.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	n, addr, err := l.conn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}

	// Create a virtual connection for this client
	return &srtConnection{
		udpConn:    l.conn,
		remoteAddr: addr,
		data:       buf[:n],
		config:     l.config,
	}, nil
}

func (l *srtListener) Close() error {
	return l.conn.Close()
}

func (l *srtListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

// srtConnection represents an SRT connection (simplified)
type srtConnection struct {
	udpConn    *net.UDPConn
	remoteAddr *net.UDPAddr
	data       []byte
	config     *SRTConfig
	streamID   string
	readBuf    []byte
	writeBuf   []byte
	mutex      sync.Mutex
}

func (c *srtConnection) Read(b []byte) (n int, err error) {
	// Simplified: read from UDP
	c.udpConn.SetReadDeadline(time.Now().Add(time.Duration(c.config.Latency*2) * time.Millisecond))
	return c.udpConn.Read(b)
}

func (c *srtConnection) Write(b []byte) (n int, err error) {
	return c.udpConn.WriteToUDP(b, c.remoteAddr)
}

func (c *srtConnection) Close() error {
	return nil // UDP is connectionless
}

func (c *srtConnection) LocalAddr() net.Addr {
	return c.udpConn.LocalAddr()
}

func (c *srtConnection) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *srtConnection) SetDeadline(t time.Time) error {
	return c.udpConn.SetDeadline(t)
}

func (c *srtConnection) SetReadDeadline(t time.Time) error {
	return c.udpConn.SetReadDeadline(t)
}

func (c *srtConnection) SetWriteDeadline(t time.Time) error {
	return c.udpConn.SetWriteDeadline(t)
}
