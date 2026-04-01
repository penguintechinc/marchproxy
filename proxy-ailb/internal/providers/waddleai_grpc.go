// Package providers - WaddleAI gRPC client for routing, security, and memory.
//
// This client wraps the WaddleAI gRPC service for use by the AILB.
// All calls are fail-open with timeouts: errors are logged, never block.
package providers

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultWaddleAITimeout = 2 * time.Second

// WaddleAIGRPCClient is the gRPC client for WaddleAI routing, security,
// memory, and usage RPCs. All methods are fail-open: on error they return
// safe defaults and log a warning.
type WaddleAIGRPCClient struct {
	conn    *grpc.ClientConn
	client  pb.WaddleAIServiceClient
	timeout time.Duration
	enabled bool
}

// NewWaddleAIGRPCClient creates a new gRPC client for WaddleAI.
// If address is empty, the client is disabled (all methods return nil/defaults).
func NewWaddleAIGRPCClient(address string, enabled bool) *WaddleAIGRPCClient {
	c := &WaddleAIGRPCClient{
		timeout: defaultWaddleAITimeout,
		enabled: enabled,
	}
	if !enabled || address == "" {
		return c
	}

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Warn("waddleai grpc: failed to create client, features disabled",
			"address", address, "error", err)
		return c
	}

	c.conn = conn
	c.client = pb.NewWaddleAIServiceClient(conn)
	slog.Info("waddleai grpc client connected", "address", address)
	return c
}

// Close shuts down the gRPC connection.
func (c *WaddleAIGRPCClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Enabled reports whether the WaddleAI gRPC client is active.
func (c *WaddleAIGRPCClient) Enabled() bool {
	return c.enabled && c.client != nil
}

// EvaluateRoute asks WaddleAI for a model recommendation.
// Returns nil on error (fail-open).
func (c *WaddleAIGRPCClient) EvaluateRoute(ctx context.Context, prompt, toolType, sessionID, userID, region string, metadata map[string]string) (*pb.RouteResponse, error) {
	if !c.Enabled() {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.EvaluateRoute(ctx, &pb.RouteRequest{
		Prompt:    prompt,
		ToolType:  toolType,
		SessionId: sessionID,
		UserId:    userID,
		Region:    region,
		Metadata:  metadata,
	})
	if err != nil {
		slog.Warn("waddleai grpc: EvaluateRoute failed (fail-open)", "error", err)
		return nil, nil
	}
	return resp, nil
}

// EvaluateSecurity asks WaddleAI to assess a prompt/command for threats.
// Returns nil on error (fail-open, treat as safe).
func (c *WaddleAIGRPCClient) EvaluateSecurity(ctx context.Context, rawCommand, toolType, userID string, ctxMap map[string]string) (*pb.SecurityResponse, error) {
	if !c.Enabled() {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.EvaluateSecurity(ctx, &pb.SecurityRequest{
		RawCommand: rawCommand,
		ToolType:   toolType,
		UserId:     userID,
		Context:    ctxMap,
	})
	if err != nil {
		slog.Warn("waddleai grpc: EvaluateSecurity failed (fail-open)", "error", err)
		return nil, nil
	}
	return resp, nil
}

// StoreTurn stores a conversation turn in WaddleAI memory.
// Fire-and-forget: logs warning on error, never blocks.
func (c *WaddleAIGRPCClient) StoreTurn(ctx context.Context, sessionID, userID, userMessage, assistantResponse, model, provider string, metadata map[string]string) {
	if !c.Enabled() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.StoreTurn(ctx, &pb.StoreTurnRequest{
		SessionId:         sessionID,
		UserId:            userID,
		UserMessage:       userMessage,
		AssistantResponse: assistantResponse,
		Model:             model,
		Provider:          provider,
		Metadata:          metadata,
	})
	if err != nil {
		slog.Warn("waddleai grpc: StoreTurn failed (fail-open)", "error", err)
	}
}

// GetContext retrieves conversation context from WaddleAI memory.
// Returns nil on error (fail-open).
func (c *WaddleAIGRPCClient) GetContext(ctx context.Context, sessionID, userID string, limit int32) (*pb.GetContextResponse, error) {
	if !c.Enabled() {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetContext(ctx, &pb.GetContextRequest{
		SessionId: sessionID,
		UserId:    userID,
		Limit:     limit,
	})
	if err != nil {
		slog.Warn("waddleai grpc: GetContext failed (fail-open)", "error", err)
		return nil, nil
	}
	return resp, nil
}
