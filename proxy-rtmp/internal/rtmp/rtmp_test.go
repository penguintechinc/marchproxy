//go:build ci

package rtmp

import (
	"testing"
)

func TestSessionStatusValues(t *testing.T) {
	tests := []struct {
		status SessionStatus
		name   string
	}{
		{SessionIdle, "idle"},
		{SessionConnecting, "connecting"},
		{SessionActive, "active"},
		{SessionStopping, "stopping"},
		{SessionStopped, "stopped"},
		{SessionError, "error"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.name {
			t.Errorf("status %v does not equal %s", tt.status, tt.name)
		}
	}
}
