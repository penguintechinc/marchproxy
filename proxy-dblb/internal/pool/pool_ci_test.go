//go:build ci

package pool

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"marchproxy-dblb/internal/logging"
)

// TestNewPool_Initialization tests basic pool initialization
func TestNewPool_Initialization(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	if p == nil {
		t.Fatal("NewPool returned nil")
	}

	stats := p.GetStats()
	if stats == nil {
		t.Error("stats should not be nil")
	}
}

// TestCreatePool_Single tests creating a single protocol pool
func TestCreatePool_Single(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	err := p.CreatePool("mysql", 50)
	if err != nil {
		t.Fatalf("CreatePool failed: %v", err)
	}

	stats := p.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}
}

// TestCreatePool_Multiple tests creating multiple protocol pools
func TestCreatePool_Multiple(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	protocols := []struct {
		name     string
		maxConns int
	}{
		{"mysql", 50},
		{"postgresql", 75},
		{"redis", 25},
		{"mongodb", 100},
	}

	for _, proto := range protocols {
		err := p.CreatePool(proto.name, proto.maxConns)
		if err != nil {
			t.Fatalf("CreatePool(%s) failed: %v", proto.name, err)
		}
	}

	stats := p.GetStats()
	if len(stats) != len(protocols) {
		t.Errorf("Expected %d protocol stats, got %d", len(protocols), len(stats))
	}

	for _, proto := range protocols {
		if _, ok := stats[proto.name]; !ok {
			t.Errorf("Protocol %s missing from stats", proto.name)
		}
	}
}

// TestCreatePool_Duplicate tests creating duplicate protocol pool
func TestCreatePool_Duplicate(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	err := p.CreatePool("mysql", 50)
	if err != nil {
		t.Fatalf("First CreatePool failed: %v", err)
	}

	err = p.CreatePool("mysql", 75)
	if err == nil {
		t.Error("Duplicate CreatePool should return error")
	}
}

// TestGetStats_Empty tests stats for empty pool
func TestGetStats_Empty(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	stats := p.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil for empty pool")
	}

	if len(stats) != 0 {
		t.Errorf("Empty pool should have no stats, got %d", len(stats))
	}
}

// TestGetStats_Content tests stats content structure
func TestGetStats_Content(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	p.CreatePool("mysql", 50)
	p.CreatePool("postgresql", 75)

	stats := p.GetStats()
	if len(stats) != 2 {
		t.Errorf("Expected 2 stats, got %d", len(stats))
	}

	for proto := range stats {
		if proto != "mysql" && proto != "postgresql" {
			t.Errorf("Unexpected protocol in stats: %s", proto)
		}
	}
}

// TestConcurrentCreatePool tests concurrent pool creation
func TestConcurrentCreatePool(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	errors := make(chan error, 5)
	protocols := []string{"mysql", "postgresql", "redis", "mongodb", "elasticsearch"}

	for i, proto := range protocols {
		go func(name string, idx int) {
			err := p.CreatePool(name, 10+idx*10)
			errors <- err
		}(proto, i)
	}

	successCount := 0
	for i := 0; i < 5; i++ {
		err := <-errors
		if err == nil {
			successCount++
		}
	}

	if successCount != 5 {
		t.Errorf("Expected all 5 pools to be created, got %d", successCount)
	}

	stats := p.GetStats()
	if len(stats) != 5 {
		t.Errorf("Expected 5 protocols in stats, got %d", len(stats))
	}
}

// TestConcurrentGetStats tests concurrent stats retrieval
func TestConcurrentGetStats(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	p.CreatePool("mysql", 50)
	p.CreatePool("postgresql", 75)

	results := make(chan int, 10)
	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats := p.GetStats()
			results <- len(stats)
		}()
	}

	wg.Wait()
	close(results)

	for count := range results {
		if count != 2 {
			t.Errorf("Expected 2 protocols in stats, got %d", count)
		}
	}
}

// TestConcurrentMixed tests mixed concurrent operations
func TestConcurrentMixed(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	var createErrors int64
	var readCount int64
	var wg sync.WaitGroup

	// Concurrent creators
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				name := fmt.Sprintf("proto-%d-%d", idx, j)
				err := p.CreatePool(name, 20)
				if err != nil {
					atomic.AddInt64(&createErrors, 1)
				}
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				stats := p.GetStats()
				if stats != nil {
					atomic.AddInt64(&readCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	if readCount != 20 {
		t.Errorf("Expected 20 reads, got %d", readCount)
	}
}

// TestCreatePool_VariousSizes tests pool creation with various max connection sizes
func TestCreatePool_VariousSizes(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(1000, logger)

	sizes := []int{1, 5, 10, 50, 100, 500, 1000}

	for i, size := range sizes {
		proto := fmt.Sprintf("protocol-%d", i)
		err := p.CreatePool(proto, size)
		if err != nil {
			t.Errorf("CreatePool with size %d failed: %v", size, err)
		}
	}

	stats := p.GetStats()
	if len(stats) != len(sizes) {
		t.Errorf("Expected %d protocols, got %d", len(sizes), len(stats))
	}
}

// TestPoolStats_Consistency tests that stats remain consistent
func TestPoolStats_Consistency(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(100, logger)

	p.CreatePool("mysql", 50)

	stats1 := p.GetStats()
	stats2 := p.GetStats()

	// Both should have mysql
	if _, ok1 := stats1["mysql"]; !ok1 {
		t.Error("mysql missing from first stats call")
	}
	if _, ok2 := stats2["mysql"]; !ok2 {
		t.Error("mysql missing from second stats call")
	}
}

// TestCreatePool_Stress tests pool creation under stress
func TestCreatePool_Stress(t *testing.T) {
	logger, _ := logging.NewLogrusAdapter("test")
	p := NewPool(1000, logger)

	for i := 0; i < 100; i++ {
		proto := fmt.Sprintf("stress-proto-%d", i)
		err := p.CreatePool(proto, 10)
		if err != nil {
			t.Fatalf("CreatePool iteration %d failed: %v", i, err)
		}
	}

	stats := p.GetStats()
	if len(stats) != 100 {
		t.Errorf("Expected 100 protocols in stats, got %d", len(stats))
	}
}
