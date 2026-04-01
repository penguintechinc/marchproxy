package billing

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// UsageReport contains data for a single usage report to WaddleAI.
type UsageReport struct {
	UserID       string
	APIKeyID     string
	Model        string
	Provider     string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	LatencyMs    int
	RequestID    string
	Metadata     map[string]string
}

// Reporter sends usage reports to WaddleAI via gRPC.
// It is fire-and-forget: errors are logged but never block the caller.
type Reporter struct {
	address   string
	conn      *grpc.ClientConn
	client    pb.WaddleAIServiceClient
	connected bool
}

// NewReporter creates a usage reporter targeting the given WaddleAI gRPC address.
func NewReporter(address string) *Reporter {
	return &Reporter{address: address}
}

// Connect establishes the gRPC connection. Call this once at startup.
func (r *Reporter) Connect() error {
	conn, err := grpc.NewClient(
		r.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Warn("usage reporter: failed to create gRPC client", "address", r.address, "error", err)
		return nil // fail-open: don't prevent startup
	}
	r.conn = conn
	r.client = pb.NewWaddleAIServiceClient(conn)
	r.connected = true
	slog.Info("usage reporter connected", "address", r.address)
	return nil
}

// Close shuts down the gRPC connection.
func (r *Reporter) Close() {
	if r.conn != nil {
		r.conn.Close()
	}
}

// ReportAsync sends a usage report in a fire-and-forget goroutine.
// If the reporter is not connected or the call fails, it logs a warning
// and drops the report (fail-open).
func (r *Reporter) ReportAsync(report UsageReport) {
	if !r.connected || r.client == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pbReport := &pb.UsageReport{
			UserId:       report.UserID,
			ApiKeyId:     report.APIKeyID,
			Model:        report.Model,
			Provider:     report.Provider,
			InputTokens:  int32(report.InputTokens),
			OutputTokens: int32(report.OutputTokens),
			TotalTokens:  int32(report.TotalTokens),
			LatencyMs:    int32(report.LatencyMs),
			RequestId:    report.RequestID,
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			Metadata:     report.Metadata,
		}

		ack, err := r.client.ReportUsage(ctx, pbReport)
		if err != nil {
			slog.Warn("usage report failed (fail-open)",
				"request_id", report.RequestID,
				"error", err,
			)
			return
		}

		if ack.GetQuotaExceeded() {
			slog.Warn("usage quota exceeded", "request_id", report.RequestID, "message", ack.GetMessage())
		}

		slog.Debug("usage report sent",
			"request_id", report.RequestID,
			"accepted", ack.GetAccepted(),
		)
	}()
}
