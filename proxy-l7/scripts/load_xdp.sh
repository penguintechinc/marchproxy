#!/bin/bash
# Script to load XDP program onto network interface

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
XDP_PROGRAM="$PROJECT_ROOT/build/envoy_xdp.o"

# Default interface (can be overridden)
INTERFACE="${1:-eth0}"
MODE="${2:-native}" # native, skb, or hw

echo "Loading XDP program for MarchProxy..."
echo "Interface: $INTERFACE"
echo "Mode: $MODE"
echo "XDP program: $XDP_PROGRAM"

# Check if XDP program exists
if [ ! -f "$XDP_PROGRAM" ]; then
    echo "Error: XDP program not found at $XDP_PROGRAM"
    echo "Please run build_xdp.sh first"
    exit 1
fi

# Check for ip command
if ! command -v ip &> /dev/null; then
    echo "Error: 'ip' command not found. Please install iproute2."
    exit 1
fi

# Check if interface exists
if ! ip link show "$INTERFACE" &> /dev/null; then
    echo "Error: Network interface $INTERFACE not found"
    echo "Available interfaces:"
    ip link show | grep -E '^[0-9]+:' | cut -d: -f2 | tr -d ' '
    exit 1
fi

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Unload existing XDP program if any
echo "Removing existing XDP program from $INTERFACE (if any)..."
ip link set dev "$INTERFACE" xdp off 2>/dev/null || true

# Load XDP program based on mode
echo "Loading XDP program in $MODE mode..."
case "$MODE" in
    native)
        ip link set dev "$INTERFACE" xdpgeneric off 2>/dev/null || true
        ip link set dev "$INTERFACE" xdp object "$XDP_PROGRAM" section xdp
        ;;
    skb)
        ip link set dev "$INTERFACE" xdpgeneric object "$XDP_PROGRAM" section xdp
        ;;
    hw)
        ip link set dev "$INTERFACE" xdpoffload object "$XDP_PROGRAM" section xdp
        ;;
    *)
        echo "Error: Invalid mode '$MODE'. Use: native, skb, or hw"
        exit 1
        ;;
esac

echo "✓ XDP program loaded successfully!"
echo ""

# Show XDP program info
echo "XDP program info:"
ip link show dev "$INTERFACE" | grep -i xdp || echo "  No XDP info available"

# Show BPF programs
if command -v bpftool &> /dev/null; then
    echo ""
    echo "BPF programs attached to $INTERFACE:"
    bpftool net show dev "$INTERFACE"
fi

echo ""
echo "To unload: sudo ip link set dev $INTERFACE xdp off"
