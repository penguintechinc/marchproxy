#!/bin/bash
# Build script for XDP program

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
XDP_DIR="$PROJECT_ROOT/xdp"
OUTPUT_DIR="$PROJECT_ROOT/build"

echo "Building MarchProxy XDP program..."
echo "Project root: $PROJECT_ROOT"

# Check for clang and llc
if ! command -v clang &> /dev/null; then
    echo "Error: clang not found. Please install LLVM/Clang."
    exit 1
fi

if ! command -v llc &> /dev/null; then
    echo "Error: llc not found. Please install LLVM."
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build XDP program
echo "Building XDP program..."
cd "$XDP_DIR"
make clean
make

# Copy XDP object file to output directory
if [ -f "envoy_xdp.o" ]; then
    cp envoy_xdp.o "$OUTPUT_DIR/envoy_xdp.o"
    echo "✓ Built XDP program -> $OUTPUT_DIR/envoy_xdp.o"

    # Show file size
    SIZE=$(du -h "$OUTPUT_DIR/envoy_xdp.o" | cut -f1)
    echo "  Size: $SIZE"

    # Verify it's a valid BPF object
    if command -v file &> /dev/null; then
        file "$OUTPUT_DIR/envoy_xdp.o"
    fi
else
    echo "✗ Failed to build XDP program"
    exit 1
fi

echo ""
echo "XDP program built successfully!"
