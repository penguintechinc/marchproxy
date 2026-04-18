//go:build ci
// +build ci

package qos

import (
	"testing"
)

func TestNewDSCPMarker(t *testing.T) {
	marker := NewDSCPMarker(nil)
	if marker == nil {
		t.Fatal("NewDSCPMarker returned nil")
	}
}

func TestDSCPMarkerMark(t *testing.T) {
	marker := NewDSCPMarker(nil)

	tests := []struct {
		name    string
		packet  *Packet
		wantErr bool
	}{
		{
			name:    "valid priority 0",
			packet:  &Packet{Priority: PriorityP0},
			wantErr: false,
		},
		{
			name:    "valid priority 1",
			packet:  &Packet{Priority: PriorityP1},
			wantErr: false,
		},
		{
			name:    "valid priority 2",
			packet:  &Packet{Priority: PriorityP2},
			wantErr: false,
		},
		{
			name:    "valid priority 3",
			packet:  &Packet{Priority: PriorityP3},
			wantErr: false,
		},
		{
			name:    "invalid priority",
			packet:  &Packet{Priority: 99},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := marker.Mark(tt.packet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Mark(): got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDSCPMarkerGetMapping(t *testing.T) {
	marker := NewDSCPMarker(nil)

	mapping := marker.GetMapping()
	if mapping == nil {
		t.Fatal("GetMapping returned nil")
	}

	// Check default mappings exist
	if val, ok := mapping[PriorityP0]; !ok || val != DSCP_EF {
		t.Errorf("Expected P0 to map to DSCP_EF (%d), got %d", DSCP_EF, val)
	}

	if val, ok := mapping[PriorityP1]; !ok || val != DSCP_AF41 {
		t.Errorf("Expected P1 to map to DSCP_AF41 (%d), got %d", DSCP_AF41, val)
	}
}

func TestDSCPMarkerUpdateMapping(t *testing.T) {
	marker := NewDSCPMarker(nil)

	// Update P0 to use different DSCP value
	err := marker.UpdateMapping(PriorityP0, DSCP_AF42)
	if err != nil {
		t.Errorf("UpdateMapping failed: %v", err)
	}

	mapping := marker.GetMapping()
	if val, ok := mapping[PriorityP0]; !ok || val != DSCP_AF42 {
		t.Errorf("Expected updated P0 to map to DSCP_AF42 (%d), got %d", DSCP_AF42, val)
	}
}

func TestDSCPMarkerUpdateMappingInvalidPriority(t *testing.T) {
	marker := NewDSCPMarker(nil)

	err := marker.UpdateMapping(99, DSCP_EF)
	if err == nil {
		t.Error("UpdateMapping should fail for invalid priority")
	}
}

func TestDSCPMarkerUpdateMappingInvalidDSCP(t *testing.T) {
	marker := NewDSCPMarker(nil)

	err := marker.UpdateMapping(PriorityP0, 100)
	if err == nil {
		t.Error("UpdateMapping should fail for invalid DSCP value")
	}
}

func TestDSCPToString(t *testing.T) {
	tests := []struct {
		dscp    uint8
		contains string
	}{
		{DSCP_CS0, "CS0"},
		{DSCP_EF, "EF"},
		{DSCP_AF11, "AF11"},
		{255, "Unknown"},
	}

	for _, tt := range tests {
		result := DSCPToString(tt.dscp)
		if result == "" {
			t.Errorf("DSCPToString(%d) returned empty string", tt.dscp)
		}
	}
}
