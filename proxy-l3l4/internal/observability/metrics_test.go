package observability_test

import (
	"testing"

	"marchproxy-l3l4/internal/observability"
)

// newMetrics wraps NewMetrics with a unique namespace so tests can call it
// more than once without triggering promauto's duplicate-registration panic.
var metricCounter int

func newMetricsOnce(t *testing.T) *observability.Metrics {
	t.Helper()
	metricCounter++
	// Each call uses a unique namespace so promauto does not panic on
	// duplicate metric registration when tests are run together.
	ns := "test_metrics_ns_" + string(rune('a'+metricCounter))
	return observability.NewMetrics(ns)
}

func TestNewMetrics_ReturnsNonNil(t *testing.T) {
	m := observability.NewMetrics("test_nonnil")
	if m == nil {
		t.Fatal("NewMetrics() returned nil")
	}
}

func TestNewMetrics_ActiveConnectionsNonNil(t *testing.T) {
	m := observability.NewMetrics("test_ac")
	if m.ActiveConnections == nil {
		t.Error("ActiveConnections gauge is nil")
	}
}

func TestNewMetrics_TotalConnectionsNonNil(t *testing.T) {
	m := observability.NewMetrics("test_tc")
	if m.TotalConnections == nil {
		t.Error("TotalConnections counter is nil")
	}
}

func TestNewMetrics_ConnectionDurationNonNil(t *testing.T) {
	m := observability.NewMetrics("test_cd")
	if m.ConnectionDuration == nil {
		t.Error("ConnectionDuration histogram is nil")
	}
}

func TestNewMetrics_ConnectionErrorsNonNil(t *testing.T) {
	m := observability.NewMetrics("test_ce")
	if m.ConnectionErrors == nil {
		t.Error("ConnectionErrors counter is nil")
	}
}

func TestNewMetrics_BytesSentNonNil(t *testing.T) {
	m := observability.NewMetrics("test_bs")
	if m.BytesSent == nil {
		t.Error("BytesSent counter is nil")
	}
}

func TestNewMetrics_BytesReceivedNonNil(t *testing.T) {
	m := observability.NewMetrics("test_br")
	if m.BytesReceived == nil {
		t.Error("BytesReceived counter is nil")
	}
}

func TestNewMetrics_PacketsSentNonNil(t *testing.T) {
	m := observability.NewMetrics("test_ps")
	if m.PacketsSent == nil {
		t.Error("PacketsSent counter is nil")
	}
}

func TestNewMetrics_PacketsReceivedNonNil(t *testing.T) {
	m := observability.NewMetrics("test_pr")
	if m.PacketsReceived == nil {
		t.Error("PacketsReceived counter is nil")
	}
}

func TestNewMetrics_NumaWorkersNonNil(t *testing.T) {
	m := observability.NewMetrics("test_nw")
	if m.NumaWorkers == nil {
		t.Error("NumaWorkers gauge is nil")
	}
}

func TestNewMetrics_NumaNodesActiveNonNil(t *testing.T) {
	m := observability.NewMetrics("test_nna")
	if m.NumaNodesActive == nil {
		t.Error("NumaNodesActive gauge is nil")
	}
}

func TestNewMetrics_XDPPacketsProcessedNonNil(t *testing.T) {
	m := observability.NewMetrics("test_xdp")
	if m.XDPPacketsProcessed == nil {
		t.Error("XDPPacketsProcessed counter is nil")
	}
}

func TestNewMetrics_AFXDPPacketsProcessedNonNil(t *testing.T) {
	m := observability.NewMetrics("test_afxdp")
	if m.AFXDPPacketsProcessed == nil {
		t.Error("AFXDPPacketsProcessed counter is nil")
	}
}

// Smoke-test: metric operations must not panic.

func TestMetrics_GaugeOperations(t *testing.T) {
	m := observability.NewMetrics("test_gauge_ops")
	m.ActiveConnections.Set(42)
	m.ActiveConnections.Inc()
	m.ActiveConnections.Dec()
	m.NumaWorkers.Set(4)
	m.NumaNodesActive.Set(2)
}

func TestMetrics_CounterOperations(t *testing.T) {
	m := observability.NewMetrics("test_counter_ops")
	m.TotalConnections.Inc()
	m.ConnectionErrors.Inc()
	m.BytesSent.Add(1024)
	m.BytesReceived.Add(512)
	m.PacketsSent.Inc()
	m.PacketsReceived.Inc()
	m.XDPPacketsProcessed.Add(100)
	m.AFXDPPacketsProcessed.Add(50)
}

func TestMetrics_HistogramObserve(t *testing.T) {
	m := observability.NewMetrics("test_hist_ops")
	m.ConnectionDuration.Observe(0.05)
	m.ConnectionDuration.Observe(1.5)
}

func TestMetrics_CounterVecWithLabels(t *testing.T) {
	m := observability.NewMetrics("test_cvec_ops")
	m.RoutingDecisions.WithLabelValues("latency", "backend-1").Inc()
	m.RoutingDecisions.WithLabelValues("cost", "backend-2").Add(5)
	m.QoSBytesProcessed.WithLabelValues("P0").Add(4096)
	m.QoSPacketsDropped.WithLabelValues("P1", "queue_full").Inc()
}

func TestMetrics_GaugeVecWithLabels(t *testing.T) {
	m := observability.NewMetrics("test_gvec_ops")
	m.QoSQueueDepth.WithLabelValues("P0").Set(10)
	m.QoSQueueDepth.WithLabelValues("P3").Set(0)
}

func TestMetrics_HistogramVecWithLabels(t *testing.T) {
	m := observability.NewMetrics("test_hvec_ops")
	m.BackendLatency.WithLabelValues("backend-1", "aws", "us-east-1").Observe(0.002)
	m.BackendLatency.WithLabelValues("backend-2", "gcp", "eu-west1").Observe(0.010)
}
