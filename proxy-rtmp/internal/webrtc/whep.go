package webrtc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
	"github.com/sirupsen/logrus"
)

// WHEPServer handles WebRTC-HTTP egress protocol (playback)
type WHEPServer struct {
	config     *WebRTCConfig
	mainConfig *config.Config
	server     *http.Server
	sessions   map[string]*WHEPSession
	streams    map[string]*StreamInfo // Available streams for playback
	mutex      sync.RWMutex
	running    bool
	stats      *WHEPStats
}

// WHEPStats holds WHEP server statistics
type WHEPStats struct {
	TotalRequests     int64
	ActiveViewers     int64
	SuccessfulOffers  int64
	FailedOffers      int64
	TotalBytesOut     int64
	mutex             sync.RWMutex
}

// WHEPSession represents an active WHEP viewer session
type WHEPSession struct {
	ID           string
	StreamKey    string
	ViewerID     string
	Offer        SessionDescription
	Answer       SessionDescription
	State        string
	CreatedAt    time.Time
	LastActivity time.Time
	RemoteAddr   string
	UserAgent    string
	BytesSent    int64
	mutex        sync.RWMutex
}

// StreamInfo represents an available stream for playback
type StreamInfo struct {
	StreamKey   string
	Active      bool
	Codec       string // h264, h265, av1
	Resolution  string // 1080p, 720p, etc
	Bitrate     int    // kbps
	ViewerCount int
	StartedAt   time.Time
}

// NewWHEPServer creates a new WHEP server
func NewWHEPServer(cfg *config.Config) *WHEPServer {
	webrtcConfig := NewWebRTCConfig(cfg)

	return &WHEPServer{
		config:     webrtcConfig,
		mainConfig: cfg,
		sessions:   make(map[string]*WHEPSession),
		streams:    make(map[string]*StreamInfo),
		stats:      &WHEPStats{},
	}
}

// Start starts the WHEP server
func (s *WHEPServer) Start() error {
	if err := s.config.Validate(); err != nil {
		return fmt.Errorf("invalid WebRTC config: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/whep/", s.handleWHEP)
	mux.HandleFunc("/whep", s.handleWHEP)
	mux.HandleFunc("/streams", s.handleListStreams)

	addr := fmt.Sprintf(":%d", s.config.WHEPPort)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.running = true

	logrus.WithField("port", s.config.WHEPPort).Info("WHEP server started")

	go func() {
		var err error
		if s.config.EnableTLS {
			err = s.server.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
		} else {
			err = s.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("WHEP server error")
		}
	}()

	return nil
}

// handleWHEP handles WHEP requests
func (s *WHEPServer) handleWHEP(w http.ResponseWriter, r *http.Request) {
	s.stats.mutex.Lock()
	s.stats.TotalRequests++
	s.stats.mutex.Unlock()

	// Extract stream key from URL path
	// Format: /whep/{stream_key}
	path := strings.TrimPrefix(r.URL.Path, "/whep/")
	path = strings.TrimPrefix(path, "/whep")
	streamKey := strings.TrimPrefix(path, "/")

	if streamKey == "" {
		http.Error(w, "Stream key required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		s.handleViewerOffer(w, r, streamKey)
	case http.MethodPatch:
		s.handleViewerICE(w, r, streamKey)
	case http.MethodDelete:
		s.handleViewerLeave(w, r, streamKey)
	case http.MethodOptions:
		s.handleOptions(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleViewerOffer processes viewer offer (POST)
func (s *WHEPServer) handleViewerOffer(w http.ResponseWriter, r *http.Request, streamKey string) {
	// Check if stream exists
	s.mutex.RLock()
	stream, exists := s.streams[streamKey]
	s.mutex.RUnlock()

	if !exists || !stream.Active {
		http.Error(w, "Stream not found or not active", http.StatusNotFound)
		s.stats.mutex.Lock()
		s.stats.FailedOffers++
		s.stats.mutex.Unlock()
		return
	}

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

	// Create viewer session
	viewerID := fmt.Sprintf("viewer_%d", time.Now().UnixNano())
	sessionID := fmt.Sprintf("whep_%s_%s", streamKey, viewerID)

	session := &WHEPSession{
		ID:           sessionID,
		StreamKey:    streamKey,
		ViewerID:     viewerID,
		Offer:        offer,
		State:        "connecting",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		RemoteAddr:   r.RemoteAddr,
		UserAgent:    r.UserAgent(),
	}

	// Generate answer
	answer := s.generateViewerAnswer(offer, stream)
	session.Answer = answer
	session.State = "connected"

	s.mutex.Lock()
	s.sessions[sessionID] = session
	if streamInfo, ok := s.streams[streamKey]; ok {
		streamInfo.ViewerCount++
	}
	s.mutex.Unlock()

	s.stats.mutex.Lock()
	s.stats.SuccessfulOffers++
	s.stats.ActiveViewers++
	s.stats.mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"stream_key":  streamKey,
		"viewer_id":   viewerID,
		"remote_addr": r.RemoteAddr,
	}).Info("WHEP viewer connected")

	// Set response headers
	w.Header().Set("Content-Type", "application/sdp")
	w.Header().Set("Location", fmt.Sprintf("/whep/%s/%s", streamKey, viewerID))
	w.Header().Set("ETag", fmt.Sprintf("\"%s\"", sessionID))

	// Add Link headers for ICE servers
	for _, server := range s.config.GetICEServers() {
		for _, url := range server.URLs {
			w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"ice-server\"", url))
		}
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(answer.SDP))
}

// handleViewerICE processes viewer ICE candidate (PATCH)
func (s *WHEPServer) handleViewerICE(w http.ResponseWriter, r *http.Request, streamKey string) {
	// Find session for this stream key
	s.mutex.RLock()
	var session *WHEPSession
	for _, sess := range s.sessions {
		if sess.StreamKey == streamKey {
			session = sess
			break
		}
	}
	s.mutex.RUnlock()

	if session == nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session.mutex.Lock()
	session.LastActivity = time.Now()
	session.mutex.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// handleViewerLeave handles viewer disconnect (DELETE)
func (s *WHEPServer) handleViewerLeave(w http.ResponseWriter, r *http.Request, streamKey string) {
	// Find and remove session
	s.mutex.Lock()
	var sessionID string
	for id, sess := range s.sessions {
		if sess.StreamKey == streamKey {
			sessionID = id
			break
		}
	}

	if sessionID == "" {
		s.mutex.Unlock()
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	delete(s.sessions, sessionID)
	if streamInfo, ok := s.streams[streamKey]; ok {
		streamInfo.ViewerCount--
	}
	s.mutex.Unlock()

	s.stats.mutex.Lock()
	s.stats.ActiveViewers--
	s.stats.mutex.Unlock()

	logrus.WithField("stream_key", streamKey).Info("WHEP viewer disconnected")

	w.WriteHeader(http.StatusOK)
}

// handleOptions handles CORS preflight
func (s *WHEPServer) handleOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "Location, ETag, Link")
	w.WriteHeader(http.StatusNoContent)
}

// handleListStreams returns available streams
func (s *WHEPServer) handleListStreams(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	streams := make([]map[string]interface{}, 0)
	for key, stream := range s.streams {
		if stream.Active {
			streams = append(streams, map[string]interface{}{
				"stream_key":   key,
				"codec":        stream.Codec,
				"resolution":   stream.Resolution,
				"bitrate_kbps": stream.Bitrate,
				"viewer_count": stream.ViewerCount,
				"started_at":   stream.StartedAt,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"streams": %v}`, streams)
}

// generateViewerAnswer generates SDP answer for viewer (simplified)
func (s *WHEPServer) generateViewerAnswer(offer SessionDescription, stream *StreamInfo) SessionDescription {
	// In a real implementation, this would use pion/webrtc
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
a=sendonly
a=rtcp-mux
a=rtpmap:96 H264/90000
m=audio 9 UDP/TLS/RTP/SAVPF 111
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=mid:1
a=sendonly
a=rtcp-mux
a=rtpmap:111 opus/48000/2
`

	return SessionDescription{
		Type: SDPTypeAnswer,
		SDP:  answerSDP,
	}
}

// RegisterStream registers a stream for playback
func (s *WHEPServer) RegisterStream(streamKey string, codec string, resolution string, bitrate int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.streams[streamKey] = &StreamInfo{
		StreamKey:   streamKey,
		Active:      true,
		Codec:       codec,
		Resolution:  resolution,
		Bitrate:     bitrate,
		ViewerCount: 0,
		StartedAt:   time.Now(),
	}

	logrus.WithField("stream_key", streamKey).Info("Stream registered for WHEP playback")
}

// UnregisterStream removes a stream from playback
func (s *WHEPServer) UnregisterStream(streamKey string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if stream, ok := s.streams[streamKey]; ok {
		stream.Active = false
	}
	delete(s.streams, streamKey)

	// Disconnect all viewers for this stream
	for id, session := range s.sessions {
		if session.StreamKey == streamKey {
			delete(s.sessions, id)
			s.stats.mutex.Lock()
			s.stats.ActiveViewers--
			s.stats.mutex.Unlock()
		}
	}

	logrus.WithField("stream_key", streamKey).Info("Stream unregistered from WHEP")
}

// Stop stops the WHEP server
func (s *WHEPServer) Stop() error {
	s.running = false

	// Clear all sessions
	s.mutex.Lock()
	s.sessions = make(map[string]*WHEPSession)
	s.streams = make(map[string]*StreamInfo)
	s.mutex.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}

	logrus.Info("WHEP server stopped")
	return nil
}

// GetStats returns server statistics
func (s *WHEPServer) GetStats() map[string]interface{} {
	s.stats.mutex.RLock()
	defer s.stats.mutex.RUnlock()

	s.mutex.RLock()
	streamCount := len(s.streams)
	s.mutex.RUnlock()

	return map[string]interface{}{
		"total_requests":    s.stats.TotalRequests,
		"active_viewers":    s.stats.ActiveViewers,
		"successful_offers": s.stats.SuccessfulOffers,
		"failed_offers":     s.stats.FailedOffers,
		"total_bytes_out":   s.stats.TotalBytesOut,
		"active_streams":    streamCount,
		"port":              s.config.WHEPPort,
	}
}

// IsRunning returns true if server is running
func (s *WHEPServer) IsRunning() bool {
	return s.running
}
