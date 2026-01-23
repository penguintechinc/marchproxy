package webrtc

import (
	"fmt"

	"github.com/penguintech/marchproxy/proxy-rtmp/internal/config"
)

// WebRTCConfig holds WebRTC-specific configuration
type WebRTCConfig struct {
	WHIPPort        int      // HTTP port for WHIP ingress
	WHEPPort        int      // HTTP port for WHEP egress
	STUNServers     []string // STUN server URLs
	TURNServers     []string // TURN server URLs (optional)
	ICEPolicy       string   // "all" or "relay"
	MaxBitrate      int      // Max video bitrate in kbps
	MinPort         int      // Min port for ICE candidates
	MaxPort         int      // Max port for ICE candidates
	EnableTLS       bool     // Enable TLS for WHIP/WHEP HTTP servers
	TLSCertFile     string   // TLS certificate file
	TLSKeyFile      string   // TLS key file
}

// NewWebRTCConfig creates WebRTC config from main config
func NewWebRTCConfig(cfg *config.Config) *WebRTCConfig {
	return &WebRTCConfig{
		WHIPPort:    cfg.WHIPPort,
		WHEPPort:    cfg.WHEPPort,
		STUNServers: cfg.STUNServers,
		TURNServers: cfg.TURNServers,
		ICEPolicy:   cfg.WebRTCICEPolicy,
		MaxBitrate:  cfg.MaxBitrate * 1000, // Convert Mbps to kbps
		MinPort:     10000,
		MaxPort:     20000,
		EnableTLS:   false,
	}
}

// Validate validates WebRTC configuration
func (c *WebRTCConfig) Validate() error {
	if c.WHIPPort < 1 || c.WHIPPort > 65535 {
		return fmt.Errorf("invalid WHIP port: %d", c.WHIPPort)
	}

	if c.WHEPPort < 1 || c.WHEPPort > 65535 {
		return fmt.Errorf("invalid WHEP port: %d", c.WHEPPort)
	}

	if c.WHIPPort == c.WHEPPort {
		return fmt.Errorf("WHIP and WHEP ports must be different")
	}

	if c.ICEPolicy != "all" && c.ICEPolicy != "relay" {
		return fmt.Errorf("invalid ICE policy: %s (must be 'all' or 'relay')", c.ICEPolicy)
	}

	if len(c.STUNServers) == 0 && c.ICEPolicy != "relay" {
		return fmt.Errorf("at least one STUN server required when ICE policy is not 'relay'")
	}

	if c.ICEPolicy == "relay" && len(c.TURNServers) == 0 {
		return fmt.Errorf("at least one TURN server required when ICE policy is 'relay'")
	}

	if c.MinPort >= c.MaxPort {
		return fmt.Errorf("min port must be less than max port")
	}

	return nil
}

// ICEServer represents an ICE server configuration
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// GetICEServers returns configured ICE servers
func (c *WebRTCConfig) GetICEServers() []ICEServer {
	servers := make([]ICEServer, 0)

	// Add STUN servers
	for _, url := range c.STUNServers {
		servers = append(servers, ICEServer{
			URLs: []string{url},
		})
	}

	// Add TURN servers
	for _, url := range c.TURNServers {
		servers = append(servers, ICEServer{
			URLs: []string{url},
			// TURN credentials would be added here in production
		})
	}

	return servers
}

// SDPType represents SDP message type
type SDPType string

const (
	SDPTypeOffer  SDPType = "offer"
	SDPTypeAnswer SDPType = "answer"
)

// SessionDescription represents an SDP message
type SessionDescription struct {
	Type SDPType `json:"type"`
	SDP  string  `json:"sdp"`
}

// ICECandidate represents an ICE candidate
type ICECandidate struct {
	Candidate        string `json:"candidate"`
	SDPMid           string `json:"sdpMid"`
	SDPMLineIndex    int    `json:"sdpMLineIndex"`
	UsernameFragment string `json:"usernameFragment,omitempty"`
}

// WHIPRequest represents a WHIP ingress request
type WHIPRequest struct {
	StreamKey string
	Offer     SessionDescription
}

// WHIPResponse represents a WHIP ingress response
type WHIPResponse struct {
	Answer    SessionDescription
	Location  string // Resource URL for DELETE
	ETag      string // Version identifier
}

// WHEPRequest represents a WHEP playback request
type WHEPRequest struct {
	StreamKey string
	Offer     SessionDescription
}

// WHEPResponse represents a WHEP playback response
type WHEPResponse struct {
	Answer   SessionDescription
	Location string
}
