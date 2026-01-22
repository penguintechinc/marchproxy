package srt

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
	"github.com/sirupsen/logrus"
)

// SessionState represents SRT session state
type SessionState string

const (
	SessionStateIdle       SessionState = "idle"
	SessionStateConnecting SessionState = "connecting"
	SessionStateActive     SessionState = "active"
	SessionStateStopping   SessionState = "stopping"
	SessionStateStopped    SessionState = "stopped"
	SessionStateError      SessionState = "error"
)

// Session represents an SRT streaming session
type Session struct {
	ID          string
	StreamKey   string
	conn        net.Conn
	srtConfig   *SRTConfig
	mainConfig  *config.Config
	state       SessionState
	stats       *SessionStats
	startTime   time.Time
	stopTime    time.Time
	error       error
	dataChan    chan []byte
	stopChan    chan struct{}
	mutex       sync.RWMutex
	onData      func(data []byte)
}

// SessionStats holds session statistics
type SessionStats struct {
	BytesReceived     int64
	BytesSent         int64
	PacketsReceived   int64
	PacketsSent       int64
	PacketsDropped    int64
	PacketsRetransmit int64
	RTT               time.Duration
	Bandwidth         int64 // bytes per second
	mutex             sync.RWMutex
}

// NewSession creates a new SRT session
func NewSession(streamKey string, conn net.Conn, srtCfg *SRTConfig, mainCfg *config.Config) *Session {
	return &Session{
		ID:         fmt.Sprintf("srt_%s_%d", streamKey, time.Now().UnixNano()),
		StreamKey:  streamKey,
		conn:       conn,
		srtConfig:  srtCfg,
		mainConfig: mainCfg,
		state:      SessionStateIdle,
		stats:      &SessionStats{},
		dataChan:   make(chan []byte, 1000),
		stopChan:   make(chan struct{}),
	}
}

// Start starts the SRT session
func (s *Session) Start(ctx context.Context) error {
	s.mutex.Lock()
	if s.state != SessionStateIdle {
		s.mutex.Unlock()
		return fmt.Errorf("session already started")
	}
	s.state = SessionStateConnecting
	s.startTime = time.Now()
	s.mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"session_id": s.ID,
		"stream_key": s.StreamKey,
	}).Debug("Starting SRT session")

	// Set to active
	s.mutex.Lock()
	s.state = SessionStateActive
	s.mutex.Unlock()

	// Start reading data
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.readLoop(ctx)
	}()

	// Wait for completion or context cancellation
	select {
	case err := <-errChan:
		s.mutex.Lock()
		s.stopTime = time.Now()
		if err != nil && err != io.EOF {
			s.state = SessionStateError
			s.error = err
		} else {
			s.state = SessionStateStopped
		}
		s.mutex.Unlock()
		return err

	case <-ctx.Done():
		s.Stop()
		return ctx.Err()

	case <-s.stopChan:
		s.mutex.Lock()
		s.state = SessionStateStopped
		s.stopTime = time.Now()
		s.mutex.Unlock()
		return nil
	}
}

// readLoop reads data from SRT connection
func (s *Session) readLoop(ctx context.Context) error {
	buf := make([]byte, 1316*7) // SRT packet size * 7 (TS packets)
	lastStatsTime := time.Now()
	bytesThisSecond := int64(0)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.stopChan:
			return nil
		default:
		}

		// Set read deadline based on latency
		deadline := time.Now().Add(time.Duration(s.srtConfig.Latency*3) * time.Millisecond)
		s.conn.SetReadDeadline(deadline)

		n, err := s.conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Retry on timeout
			}
			return err
		}

		if n > 0 {
			// Update stats
			s.stats.mutex.Lock()
			s.stats.BytesReceived += int64(n)
			s.stats.PacketsReceived++
			bytesThisSecond += int64(n)
			s.stats.mutex.Unlock()

			// Calculate bandwidth every second
			if time.Since(lastStatsTime) >= time.Second {
				s.stats.mutex.Lock()
				s.stats.Bandwidth = bytesThisSecond
				s.stats.mutex.Unlock()
				bytesThisSecond = 0
				lastStatsTime = time.Now()
			}

			// Make a copy of data
			data := make([]byte, n)
			copy(data, buf[:n])

			// Send to data channel (non-blocking)
			select {
			case s.dataChan <- data:
			default:
				// Channel full, drop packet
				s.stats.mutex.Lock()
				s.stats.PacketsDropped++
				s.stats.mutex.Unlock()
			}

			// Call data callback if set
			if s.onData != nil {
				s.onData(data)
			}
		}
	}
}

// Stop stops the SRT session
func (s *Session) Stop() {
	s.mutex.Lock()
	if s.state == SessionStateStopped || s.state == SessionStateStopping {
		s.mutex.Unlock()
		return
	}
	s.state = SessionStateStopping
	s.mutex.Unlock()

	close(s.stopChan)

	if s.conn != nil {
		s.conn.Close()
	}

	s.mutex.Lock()
	s.state = SessionStateStopped
	s.stopTime = time.Now()
	s.mutex.Unlock()

	logrus.WithField("stream_key", s.StreamKey).Debug("SRT session stopped")
}

// SetDataCallback sets callback for received data
func (s *Session) SetDataCallback(callback func(data []byte)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.onData = callback
}

// ReadData reads data from session channel (non-blocking)
func (s *Session) ReadData() ([]byte, bool) {
	select {
	case data := <-s.dataChan:
		return data, true
	default:
		return nil, false
	}
}

// GetState returns session state
func (s *Session) GetState() SessionState {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.state
}

// GetStats returns session statistics
func (s *Session) GetStats() map[string]interface{} {
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	s.mutex.RLock()
	state := s.state
	startTime := s.startTime
	s.mutex.RUnlock()

	duration := time.Since(startTime)
	if state == SessionStateStopped {
		duration = s.stopTime.Sub(startTime)
	}

	return map[string]interface{}{
		"session_id":         s.ID,
		"stream_key":         s.StreamKey,
		"state":              string(state),
		"bytes_received":     s.stats.BytesReceived,
		"bytes_sent":         s.stats.BytesSent,
		"packets_received":   s.stats.PacketsReceived,
		"packets_sent":       s.stats.PacketsSent,
		"packets_dropped":    s.stats.PacketsDropped,
		"packets_retransmit": s.stats.PacketsRetransmit,
		"rtt_ms":             s.stats.RTT.Milliseconds(),
		"bandwidth_bps":      s.stats.Bandwidth * 8,
		"duration_seconds":   duration.Seconds(),
		"start_time":         startTime,
	}
}

// GetRemoteAddr returns remote address
func (s *Session) GetRemoteAddr() string {
	if s.conn != nil {
		return s.conn.RemoteAddr().String()
	}
	return ""
}

// GetDuration returns session duration
func (s *Session) GetDuration() time.Duration {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.startTime.IsZero() {
		return 0
	}

	if s.state == SessionStateStopped {
		return s.stopTime.Sub(s.startTime)
	}

	return time.Since(s.startTime)
}

// GetBitrate returns current bitrate in kbps
func (s *Session) GetBitrate() int64 {
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()
	return (s.stats.Bandwidth * 8) / 1000
}

// IsActive returns true if session is active
func (s *Session) IsActive() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.state == SessionStateActive
}
