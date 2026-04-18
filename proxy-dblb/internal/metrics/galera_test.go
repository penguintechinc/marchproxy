package metrics_test

import (
	"testing"

	"marchproxy-dblb/internal/metrics"
)

func TestGaleraMetricsNodeState(t *testing.T) {
	nodeID := "node1"
	state := 2

	metrics.SetGaleraNodeState(nodeID, state)
	// Metric was recorded, no error expected
}

func TestGaleraMetricsNodeReady(t *testing.T) {
	nodeID := "node1"

	metrics.SetGaleraNodeReady(nodeID, true)
	metrics.SetGaleraNodeReady(nodeID, false)
	// Metrics were recorded, no error expected
}

func TestGaleraMetricsClusterSize(t *testing.T) {
	nodeID := "node1"
	size := float64(3)

	metrics.SetGaleraClusterSize(nodeID, size)
	// Metric was recorded, no error expected
}

func TestGaleraMetricsFlowControl(t *testing.T) {
	nodeID := "node1"

	metrics.SetGaleraFlowControl(nodeID, true)
	metrics.SetGaleraFlowControl(nodeID, false)
	// Metrics were recorded, no error expected
}

func TestGaleraMetricsNodeError(t *testing.T) {
	nodeID := "node1"

	metrics.IncGaleraNodeErrors(nodeID)
	metrics.IncGaleraNodeErrors(nodeID)
	// Errors were recorded, no error expected
}

func TestGaleraMetricsIncConnection(t *testing.T) {
	protocol := "mysql"

	metrics.IncConnection(protocol)
	metrics.IncConnection(protocol)
	// Metric was recorded, no error expected
}

func TestGaleraMetricsDecConnection(t *testing.T) {
	protocol := "mysql"

	metrics.DecConnection(protocol)
	metrics.DecConnection(protocol)
	// Metric was recorded, no error expected
}

func TestGaleraMetricsIncQuery(t *testing.T) {
	protocol := "mysql"

	metrics.IncQuery(protocol, false) // read query
	metrics.IncQuery(protocol, true)  // write query
	metrics.IncQuery(protocol, false) // read query
	// Queries were recorded, no error expected
}

func TestGaleraMetricsIncSQLInjection(t *testing.T) {
	protocol := "mysql"

	metrics.IncSQLInjection(protocol)
	metrics.IncSQLInjection(protocol)
	// Injection attempts were recorded, no error expected
}

func TestAuthMetrics(t *testing.T) {
	protocol := "mysql"
	user := "testuser"

	metrics.IncAuthFailure(protocol, user)
	metrics.IncAuthSuccess(protocol, user)
	// Auth metrics were recorded, no error expected
}

func TestBytesTransferred(t *testing.T) {
	protocol := "mysql"

	metrics.AddBytesTransferred(protocol, "in", 1024)
	metrics.AddBytesTransferred(protocol, "out", 2048)
	// Bytes transferred were recorded, no error expected
}

func TestRecordBytesTransferred(t *testing.T) {
	protocol := "postgresql"

	metrics.RecordBytesTransferred(protocol, "in", 512)
	metrics.RecordBytesTransferred(protocol, "out", 1024)
	// Bytes transferred were recorded, no error expected
}

func TestGaleraMetricsMultipleNodes(t *testing.T) {
	nodes := []string{"node1", "node2", "node3"}

	for _, node := range nodes {
		metrics.SetGaleraNodeState(node, 2)
		metrics.SetGaleraNodeReady(node, true)
		metrics.SetGaleraClusterSize(node, float64(len(nodes)))
	}
	// All metrics recorded successfully
}
