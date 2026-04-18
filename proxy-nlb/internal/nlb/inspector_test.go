//go:build ci

package nlb_test

import (
	"testing"

	"marchproxy-nlb/internal/nlb"
)

func TestNewProtocolInspector(t *testing.T) {
	inspector := nlb.NewProtocolInspector()
	if inspector == nil {
		t.Fatal("expected non-nil inspector")
	}
}

func TestProtocolInspectorStructure(t *testing.T) {
	inspector := nlb.NewProtocolInspector()

	// Verify inspector can be used
	if inspector == nil {
		t.Error("expected inspector to be initialized")
	}
}
