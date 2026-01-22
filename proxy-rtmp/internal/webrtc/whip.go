package webrtc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
	"github.com/sirupsen/logrus"
)

// WHIPServer handles WebRTC-HTTP ingestion protocol
type WHIPServer struct {
	config     *WebRTCConfig
	mainConfig *config.Config
	server     *http.Server
	sessions   map[string]*WHIPSession
	mutex      sync.RWMutex
	running    bool
	stats      *WHIPStats
	onStream   func(streamKey string, session *WHIPSession)
}

// WHIPStats holds WHIP server statistics
type WHIPStats struct {
	TotalRequests     int64
	ActiveSessions    int64
	SuccessfulOffers  int64
	FailedOffers      int64
	TotalBytesIn      int64
	mutex             sync.RWMutex
}

// WHIPSession represents an active WHIP session
type WHIPSession struct {
	ID           string
	StreamKey    string
	Offer        SessionDescription
	Answer       SessionDescription
	State        string
	CreatedAt    time.Time
	LastActivity time.Time
	RemoteAddr   string
	UserAgent    string
	mutex        sync.RWMutex
}

// NewWHIPServer creates a new WHIP server
func NewWHIPServer(cfg *config.Config, onStream func(streamKey string, session *WHIPSession)) *WHIPServer {
	webrtcConfig := NewWebRTCConfig(cfg)

	return &WHIPServer{
		config:     webrtcConfig,
		mainConfig: cfg,
		sessions:   make(map[string]*WHIPSession),
		stats:      &WHIPStats{},
		onStream:   onStream,
	}
}

// Start starts the WHIP server
func (s *WHIPServer) Start() error {
	if err := s.config.Validate(); err != nil {
		return fmt.Errorf("invalid WebRTC config: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/whip/", s.handleWHIP)
	mux.HandleFunc("/whip", s.handleWHIP)

	addr := fmt.Sprintf(":%d", s.config.WHIPPort)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.running = true

	logrus.WithField("port", s.config.WHIPPort).Info("WHIP server started")

	go func() {
		var err error
		if s.config.EnableTLS {
			err = s.server.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
		} else {
			err = s.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("WHIP server error")
		}
	}()

	return nil
}

// handleWHIP handles WHIP requests
func (s *WHIPServer) handleWHIP(w http.ResponseWriter, r *http.Request) {
	s.stats.mutex.Lock()
	s.stats.TotalRequests++
	s.stats.mutex.Unlock()

	// Extract stream key from URL path
	// Format: /whip/{stream_key}
	path := strings.TrimPrefix(r.URL.Path, "/whip/")
	path = strings.TrimPrefix(path, "/whip")
	streamKey := strings.TrimPrefix(path, "/")

	if streamKey == "" {
		http.Error(w, "Stream key required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleOffer(w, r, streamKey)
	case http.MethodPatch:
		s.handleICECandidate(w, r, streamKey)
	case http.MethodDelete:
		s.handleDelete(w, r, streamKey)
	case http.MethodOptions:
		s.handleOptions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOffer processes WHIP offer (POST)
func (s *WHIPServer) handleOffer(w http.ResponseWriter, r *http.Request, streamKey string) {
	// Check content type
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/sdp") {
		http.Error(w, "Content-Type must be application/sdp", http.StatusUnsupportedMediaType)
		return
	}

	// Read SDP offer
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		s.stats.mutex.Lock()
		s.stats.FailedOffers++
		s.stats.mutex.Unlock()
		return
	}

	offer := SessionDescription{
		Type: SDPTypeOffer,
		SDP:  string(body),
	}

	// Check for existing session
	s.mutex.Lock()
	if _, exists := s.sessions[streamKey]; exists {
		s.mutex.Unlock()
		http.Error(w, "Stream already active", http.StatusConflict)
		s.stats.mutex.Lock()
		s.stats.FailedOffers++
		s.stats.mutex.Unlock()
		return
	}

	// Create session
	sessionID := fmt.Sprintf("whip_%s_%d", streamKey, time.Now().UnixNano())
	session := &WHIPSession{
		ID:           sessionID,
		StreamKey:    streamKey,
		Offer:        offer,
		State:        "connecting",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		RemoteAddr:   r.RemoteAddr,
		UserAgent:    r.UserAgent(),
	}

	// Generate SDP answer (simplified - real implementation uses pion/webrtc)
	answer := s.generateAnswer(offer)
	session.Answer = answer
	session.State = "connected"

	s.sessions[streamKey] = session
	s.mutex.Unlock()

	s.stats.mutex.Lock()
	s.stats.SuccessfulOffers++
	s.stats.ActiveSessions++
	s.stats.mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"stream_key":  streamKey,
		"session_id":  sessionID,
		"remote_addr": r.RemoteAddr,
	}).Info("WHIP stream connected")

	// Notify callback
	if s.onStream != nil {
		s.onStream(streamKey, session)
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/sdp")
	w.Header().Set("Location", fmt.Sprintf("/whip/%s", streamKey))
	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", sessionID))
	w.Header().Set("Accept-Patch", "application/trickle-ice-sdpfrag")

	// Add Link headers for ICE servers
	for _, server := range s.config.GetICEServers() {
		for _, url := range server.URLs {
			w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"ice-server\"", url))
		}
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(answer.SDP))
}

// handleICECandidate processes ICE candidate (PATCH)
func (s *WHIPServer) handleICECandidate(w http.ResponseWriter, r *http.Request, streamKey string) {
	s.mutex.RLock()
	session, exists := s.sessions[streamKey]
	s.mutex.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Read ICE candidate
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse ICE candidate (simplified)
	var candidate ICECandidate
	if err := json.Unmarshal(body, &candidate); err != nil {
		// Might be SDP fragment format
		logrus.WithField("body", string(body)).Debug("Received ICE candidate")
	}

	session.mutex.Lock()
	session.LastActivity = time.Now()
	session.mutex.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// handleDelete terminates WHIP session (DELETE)
func (s *WHIPServer) handleDelete(w http.ResponseWriter, r *http.Request, streamKey string) {
	s.mutex.Lock()
	session, exists := s.sessions[streamKey]
	if !exists {
		s.mutex.Unlock()
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	delete(s.sessions, streamKey)
	s.mutex.Unlock()

	s.stats.mutex.Lock()
	s.stats.ActiveSessions--
	s.stats.mutex.Unlock()

	logrus.WithField("stream_key", streamKey).Info("WHIP stream disconnected")

	_ = session // Could cleanup resources here

	w.WriteHeader(http.StatusOK)
}

// handleOptions handles CORS preflight
func (s *WHIPServer) handleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "Location, ETag, Link")
	w.WriteHeader(http.StatusNoContent)
}

// generateAnswer generates SDP answer (simplified placeholder)
func (s *WHIPServer) generateAnswer(offer SessionDescription) SessionDescription {
	// In a real implementation, this would use pion/webrtc to:
	// 1. Parse the offer SDP
	// 2. Create a peer connection
	// 3. Generate an answer SDP
	// This is a placeholder that returns a basic answer structure

	// For now, return a minimal SDP answer
	answerSDP := `v=0
o=- 0 0 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE 0 1
m=video 9 UDP/TLS/RTP/SAVPF 96
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:marchproxy
a=ice-pwd:marchproxysecret
a=fingerprint:sha-256 00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00
a=setup:active
a=mid:0
a=recvonly
a=rtcp-mux
a=rtpmap:96 H264/90000
m=audio 9 UDP/TLS/RTP/SAVPF 111
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=mid:1
a=recvonly
a=rtcp-mux
a=rtpmap:111 opus/48000/2
`

	return SessionDescription{
		Type: SDPTypeAnswer,
		SDP:  answerSDP,
	}
}

// Stop stops the WHIP server
func (s *WHIPServer) Stop() error {
	s.running = false

	// Stop all sessions
	s.mutex.Lock()
	for streamKey := range s.sessions {
		delete(s.sessions, streamKey)
	}
	s.mutex.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}

	logrus.Info("WHIP server stopped")
	return nil
}

// GetSession returns session by stream key
func (s *WHIPServer) GetSession(streamKey string) (*WHIPSession, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	session, ok := s.sessions[streamKey]
	return session, ok
}

// GetAllSessions returns all active sessions
func (s *WHIPServer) GetAllSessions() []*WHIPSession {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	sessions := make([]*WHIPSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetStats returns server statistics
func (s *WHIPServer) GetStats() map[string]interface{} {
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	return map[string]interface{}{
		"total_requests":     s.stats.TotalRequests,
		"active_sessions":    s.stats.ActiveSessions,
		"successful_offers":  s.stats.SuccessfulOffers,
		"failed_offers":      s.stats.FailedOffers,
		"total_bytes_in":     s.stats.TotalBytesIn,
		"port":               s.config.WHIPPort,
	}
}

// IsRunning returns true if server is running
func (s *WHIPServer) IsRunning() bool {
	return s.running
}
