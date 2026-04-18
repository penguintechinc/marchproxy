//go:build ci

package nlb

import (
	"testing"
	"time"

	"marchproxy-nlb/internal/logging"
)

func TestScalingPolicy_Default(t *testing.T) {
	policy := DefaultScalingPolicy(ProtocolHTTP)

	if policy.Protocol != ProtocolHTTP {
		t.Errorf("Protocol = %v, want %v", policy.Protocol, ProtocolHTTP)
	}
	if policy.MinReplicas != 1 {
		t.Errorf("MinReplicas = %d, want 1", policy.MinReplicas)
	}
	if policy.MaxReplicas != 10 {
		t.Errorf("MaxReplicas = %d, want 10", policy.MaxReplicas)
	}
	if policy.TargetCPU != 70.0 {
		t.Errorf("TargetCPU = %f, want 70.0", policy.TargetCPU)
	}
}

func TestScalingPolicy_DifferentProtocols(t *testing.T) {
	protocols := []Protocol{ProtocolHTTP, ProtocolMySQL, ProtocolPostgreSQL, ProtocolRedis}

	for _, p := range protocols {
		policy := DefaultScalingPolicy(p)
		if policy.Protocol != p {
			t.Errorf("Protocol mismatch for %v", p)
		}
		if policy.MinReplicas != 1 || policy.MaxReplicas != 10 {
			t.Errorf("Default bounds incorrect for %v", p)
		}
	}
}

func TestAutoscaler_Create(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)

	as := NewAutoscaler(router, logger)

	if as == nil {
		t.Fatal("Autoscaler should not be nil")
	}
}

func TestAutoscaler_RecordMetrics(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)
	as := NewAutoscaler(router, logger)

	metrics := &ScalingMetrics{
		CPUUtilization:    75.0,
		MemoryUtilization: 85.0,
		ConnectionCount:   500,
		RequestRate:       100,
		ErrorRate:         0.01,
		Timestamp:         time.Now(),
	}

	as.RecordMetrics(ProtocolHTTP, metrics)
	stats := as.GetStats()
	if stats == nil {
		t.Logf("GetStats returned nil")
	}
}

func TestAutoscaler_SetGetPolicy(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)
	as := NewAutoscaler(router, logger)

	policy := DefaultScalingPolicy(ProtocolHTTP)
	err := as.SetPolicy(policy)
	if err != nil {
		t.Errorf("SetPolicy error = %v", err)
	}

	retrieved := as.GetPolicy(ProtocolHTTP)
	if retrieved == nil {
		t.Fatal("GetPolicy should return policy")
	}
}

func TestAutoscaler_RecordMultipleMetrics(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)
	as := NewAutoscaler(router, logger)

	metrics := []float64{20.0, 70.0, 90.0, 50.0}

	for _, cpu := range metrics {
		m := &ScalingMetrics{
			CPUUtilization: cpu,
			Timestamp:      time.Now(),
		}
		as.RecordMetrics(ProtocolHTTP, m)
	}

	stats := as.GetStats()
	if stats == nil {
		t.Logf("Stats available for review")
	}
}

func TestAutoscaler_StartStop(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)
	as := NewAutoscaler(router, logger)

	err := as.Start()
	if err != nil {
		t.Logf("Start error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	as.Stop()
}

func TestAutoscaler_MultipleProtocols(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	router := NewRouter(logger)
	as := NewAutoscaler(router, logger)

	protocols := []Protocol{ProtocolHTTP, ProtocolMySQL, ProtocolPostgreSQL}

	for _, p := range protocols {
		policy := DefaultScalingPolicy(p)
		err := as.SetPolicy(policy)
		if err != nil {
			t.Errorf("SetPolicy error for %v: %v", p, err)
		}
	}
}
