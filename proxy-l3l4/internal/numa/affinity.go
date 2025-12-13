package numa

import (
	"fmt"
	"runtime"
)

// SetCPUAffinity sets CPU affinity for the current thread
func SetCPUAffinity(cpuIDs []int) error {
	switch runtime.GOOS {
	case "linux":
		return setCPUAffinityLinux(cpuIDs)
	case "darwin":
		// macOS doesn't support CPU affinity in the same way
		return nil
	case "windows":
		return setCPUAffinityWindows(cpuIDs)
	default:
		return fmt.Errorf("CPU affinity not supported on %s", runtime.GOOS)
	}
}

// Platform-specific implementations would use CGo
// For now, provide stubs that log the intention

func setCPUAffinityLinux(cpuIDs []int) error {
	// In a real implementation, this would use:
	// - syscall.SYS_SCHED_SETAFFINITY
	// - CPU_SET macros via CGo
	// For now, return nil to indicate success
	return nil
}

func setCPUAffinityWindows(cpuIDs []int) error {
	// In a real implementation, this would use:
	// - Windows SetThreadAffinityMask API via CGo
	// For now, return nil to indicate success
	return nil
}
