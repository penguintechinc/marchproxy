//go:build ci
// +build ci

package numa

import (
	"testing"
)

func TestSetCPUAffinityLinux(t *testing.T) {
	// Test with empty list
	err := SetCPUAffinity([]int{})
	if err != nil {
		t.Errorf("SetCPUAffinity([]): unexpected error %v", err)
	}

	// Test with single CPU
	err = SetCPUAffinity([]int{0})
	if err != nil {
		t.Errorf("SetCPUAffinity([0]): unexpected error %v", err)
	}

	// Test with multiple CPUs
	err = SetCPUAffinity([]int{0, 1, 2})
	if err != nil {
		t.Errorf("SetCPUAffinity([0,1,2]): unexpected error %v", err)
	}
}

func TestSetCPUAffinityNegativeCPU(t *testing.T) {
	// Test with negative CPU (should not error, platform-specific handling)
	err := SetCPUAffinity([]int{-1})
	// Platform-specific behavior; test only that function doesn't panic
	_ = err
}

func TestSetCPUAffinityLargeCPUNumber(t *testing.T) {
	// Test with large CPU number
	err := SetCPUAffinity([]int{999})
	// Platform-specific behavior; test only that function doesn't panic
	_ = err
}

func TestSetCPUAffinityMultipleCalls(t *testing.T) {
	// Multiple calls should be idempotent
	cpus := []int{0, 1}

	err1 := SetCPUAffinity(cpus)
	err2 := SetCPUAffinity(cpus)

	if (err1 == nil) != (err2 == nil) {
		t.Errorf("SetCPUAffinity calls not consistent: first %v, second %v", err1, err2)
	}
}

func TestSetCPUAffinityDifferentLists(t *testing.T) {
	// Test with different affinity lists
	lists := [][]int{
		{0},
		{0, 1},
		{1, 2, 3},
		{},
	}

	for _, cpus := range lists {
		err := SetCPUAffinity(cpus)
		_ = err // Platform-specific; just verify no panic
	}
}
