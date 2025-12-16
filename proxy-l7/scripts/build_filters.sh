#!/bin/bash
# Build script for WASM filters

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
FILTERS_DIR="$PROJECT_ROOT/filters"
OUTPUT_DIR="$PROJECT_ROOT/build"

echo "Building MarchProxy WASM filters..."
echo "Project root: $PROJECT_ROOT"

# Check for Rust and cargo
if ! command -v cargo &> /dev/null; then
    echo "Error: Rust/Cargo not found. Please install Rust from https://rustup.rs/"
    exit 1
fi

# Check for wasm32-unknown-unknown target
if ! rustup target list | grep -q "wasm32-unknown-unknown (installed)"; then
    echo "Installing wasm32-unknown-unknown target..."
    rustup target add wasm32-unknown-unknown
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build each filter
FILTERS=("auth_filter" "license_filter" "metrics_filter")

for filter in "${FILTERS[@]}"; do
    echo ""
    echo "Building $filter..."
    cd "$FILTERS_DIR/$filter"

    # Build in release mode for WASM target
    cargo build --target wasm32-unknown-unknown --release

    # Copy WASM file to output directory
    WASM_FILE="target/wasm32-unknown-unknown/release/marchproxy_${filter}.wasm"
    if [ -f "$WASM_FILE" ]; then
        cp "$WASM_FILE" "$OUTPUT_DIR/${filter}.wasm"
        echo "✓ Built $filter -> $OUTPUT_DIR/${filter}.wasm"

        # Show file size
        SIZE=$(du -h "$OUTPUT_DIR/${filter}.wasm" | cut -f1)
        echo "  Size: $SIZE"
    else
        echo "✗ Failed to build $filter"
        exit 1
    fi
done

echo ""
echo "All WASM filters built successfully!"
echo "Output directory: $OUTPUT_DIR"
ls -lh "$OUTPUT_DIR"/*.wasm
