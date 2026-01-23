// Package extauth implements an Envoy external authorization gRPC server
// This server receives authorization requests from Envoy and checks them
// against the threat intelligence engine and access control rules
package extauth

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"marchproxy-egress/internal/auth"
	"marchproxy-egress/internal/threat"
)

// Server implements the Envoy external authorization service
type Server struct {
	authv3.UnimplementedAuthorizationServer

	threatManager    *threat.Manager
	accessController *threat.AccessController
	authenticator    *auth.Authenticator

	grpcServer *grpc.Server
	listener   net.Listener
	port       int

	logger *logrus.Logger

	// Statistics
	stats struct {
		TotalRequests  int64
		AllowedRequests int64
		DeniedRequests  int64
		Errors         int64
	}
}

// ServerConfig holds configuration for the external authorization server
type ServerConfig struct {
	Port             int
	ThreatManager    *threat.Manager
	AccessController *threat.AccessController
	Authenticator    *auth.Authenticator
}

// NewServer creates a new external authorization server
func NewServer(cfg ServerConfig, logger *logrus.Logger) *Server {
	if logger == nil {
		logger = logrus.New()
	}

	return &Server{
		threatManager:    cfg.ThreatManager,
		accessController: cfg.AccessController,
		authenticator:    cfg.Authenticator,
		port:             cfg.Port,
		logger:           logger,
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	s.grpcServer = grpc.NewServer()
	authv3.RegisterAuthorizationServer(s.grpcServer, s)

	s.logger.WithField("port", s.port).Info("Starting external authorization gRPC server")

	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil {
			s.logger.WithError(err).Error("gRPC server error")
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	if s.listener != nil {
		s.listener.Close()
	}
	s.logger.Info("Stopped external authorization server")
}

// Check implements the Envoy ext_authz Check RPC
func (s *Server) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	s.stats.TotalRequests++

	// Extract request information
	httpReq := req.GetAttributes().GetRequest().GetHttp()
	if httpReq == nil {
		s.stats.Errors++
		return s.createDeniedResponse("missing HTTP request attributes", codes.InvalidArgument), nil
	}

	// Build request context for threat manager
	threatCtx := &threat.RequestContext{
		Host:     httpReq.GetHost(),
		Path:     httpReq.GetPath(),
		Method:   httpReq.GetMethod(),
		Protocol: httpReq.GetProtocol(),
		Headers:  make(map[string]string),
		TLS:      httpReq.GetScheme() == "https",
	}

	// Extract headers
	for key, value := range httpReq.GetHeaders() {
		threatCtx.Headers[key] = value
	}

	// Extract source IP
	source := req.GetAttributes().GetSource()
	if source != nil && source.GetAddress() != nil {
		if sockAddr := source.GetAddress().GetSocketAddress(); sockAddr != nil {
			threatCtx.SourceIP = sockAddr.GetAddress()
		}
	}

	// Extract destination IP
	destination := req.GetAttributes().GetDestination()
	if destination != nil && destination.GetAddress() != nil {
		if sockAddr := destination.GetAddress().GetSocketAddress(); sockAddr != nil {
			threatCtx.DestinationIP = sockAddr.GetAddress()
		}
	}

	// Log the request
	s.logger.WithFields(logrus.Fields{
		"host":       threatCtx.Host,
		"path":       threatCtx.Path,
		"method":     threatCtx.Method,
		"source_ip":  threatCtx.SourceIP,
		"dest_ip":    threatCtx.DestinationIP,
	}).Debug("Received authorization request")

	// Step 1: Check threat intelligence (IP, domain, URL blocking)
	if s.threatManager != nil {
		decision := s.threatManager.Check(ctx, threatCtx)
		if decision.Blocked {
			s.stats.DeniedRequests++
			s.logger.WithFields(logrus.Fields{
				"host":     threatCtx.Host,
				"path":     threatCtx.Path,
				"reason":   decision.Reason,
				"category": decision.Category,
				"rule_id":  decision.MatchedRule,
			}).Warn("Request blocked by threat intelligence")
			return s.createDeniedResponse(decision.Reason, codes.PermissionDenied), nil
		}
	}

	// Step 2: Check access control (authentication-based restrictions)
	if s.accessController != nil {
		// Extract service context from authentication
		svcCtx := s.extractServiceContext(httpReq)

		// Check access control for the destination
		acDecision := s.accessController.Check(threatCtx.Host, "domain", svcCtx)
		if !acDecision.Allowed {
			s.stats.DeniedRequests++

			// If authentication is required, return 401
			if acDecision.RequiresAuth {
				s.logger.WithFields(logrus.Fields{
					"host":   threatCtx.Host,
					"reason": acDecision.Reason,
				}).Warn("Request denied - authentication required")
				return s.createDeniedResponse(acDecision.Reason, codes.Unauthenticated), nil
			}

			// Otherwise return 403
			s.logger.WithFields(logrus.Fields{
				"host":     threatCtx.Host,
				"reason":   acDecision.Reason,
				"rule_id":  acDecision.MatchedRule,
			}).Warn("Request denied by access control")
			return s.createDeniedResponse(acDecision.Reason, codes.PermissionDenied), nil
		}
	}

	// All checks passed
	s.stats.AllowedRequests++
	return s.createAllowedResponse(threatCtx), nil
}

// extractServiceContext extracts service authentication context from the request
func (s *Server) extractServiceContext(httpReq *authv3.AttributeContext_HttpRequest) *threat.ServiceContext {
	if s.authenticator == nil {
		return nil
	}

	headers := httpReq.GetHeaders()

	// Check for Authorization header (Bearer token)
	authHeader := ""
	if auth, ok := headers["authorization"]; ok {
		authHeader = auth
	} else if auth, ok := headers["Authorization"]; ok {
		authHeader = auth
	}

	if authHeader == "" {
		return &threat.ServiceContext{
			Authenticated: false,
		}
	}

	// Parse Bearer token
	token := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	} else if strings.HasPrefix(authHeader, "bearer ") {
		token = strings.TrimPrefix(authHeader, "bearer ")
	}

	if token == "" {
		return &threat.ServiceContext{
			Authenticated: false,
		}
	}

	// Validate the token
	// Note: In production, this would call the authenticator to validate JWT/Base64 tokens
	// For now, we'll assume any Bearer token is valid and extract service info from it
	// The actual validation logic depends on your token format

	// Try to get service info from X-Service-ID header (if provided)
	serviceID := headers["x-service-id"]
	serviceName := headers["x-service-name"]

	return &threat.ServiceContext{
		ServiceID:     serviceID,
		ServiceName:   serviceName,
		TokenID:       token[:min(8, len(token))] + "...", // Truncated for logging
		Authenticated: true,
	}
}

// createAllowedResponse creates an OK response that allows the request
func (s *Server) createAllowedResponse(ctx *threat.RequestContext) *authv3.CheckResponse {
	return &authv3.CheckResponse{
		Status: &status.Status{
			Code: int32(codes.OK),
		},
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   "x-marchproxy-checked",
							Value: "true",
						},
					},
					{
						Header: &corev3.HeaderValue{
							Key:   "x-marchproxy-check-time",
							Value: time.Now().Format(time.RFC3339),
						},
					},
				},
			},
		},
	}
}

// createDeniedResponse creates a denied response
func (s *Server) createDeniedResponse(reason string, code codes.Code) *authv3.CheckResponse {
	httpCode := typev3.StatusCode_Forbidden
	if code == codes.Unauthenticated {
		httpCode = typev3.StatusCode_Unauthorized
	} else if code == codes.InvalidArgument {
		httpCode = typev3.StatusCode_BadRequest
	}

	return &authv3.CheckResponse{
		Status: &status.Status{
			Code:    int32(code),
			Message: reason,
		},
		HttpResponse: &authv3.CheckResponse_DeniedResponse{
			DeniedResponse: &authv3.DeniedHttpResponse{
				Status: &typev3.HttpStatus{
					Code: httpCode,
				},
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   "x-marchproxy-blocked",
							Value: "true",
						},
					},
					{
						Header: &corev3.HeaderValue{
							Key:   "x-marchproxy-block-reason",
							Value: reason,
						},
					},
				},
				Body: fmt.Sprintf(`{"error": "blocked", "reason": "%s"}`, reason),
			},
		},
	}
}

// GetStats returns server statistics
func (s *Server) GetStats() map[string]int64 {
	return map[string]int64{
		"total_requests":   s.stats.TotalRequests,
		"allowed_requests": s.stats.AllowedRequests,
		"denied_requests":  s.stats.DeniedRequests,
		"errors":          s.stats.Errors,
	}
}

// ResetStats resets all statistics
func (s *Server) ResetStats() {
	s.stats.TotalRequests = 0
	s.stats.AllowedRequests = 0
	s.stats.DeniedRequests = 0
	s.stats.Errors = 0
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
