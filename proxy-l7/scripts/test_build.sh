#!/bin/bash
# Test script for proxy-l7 build verification

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_DIR="$PROJECT_ROOT/build"

echo "═══════════════════════════════════════════════════════════"
echo " MarchProxy L7 Proxy - Build Verification"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Test 1: Check build directory
echo "Test 1: Build directory structure"
if [ -d "$BUILD_DIR" ]; then
    success "Build directory exists: $BUILD_DIR"
else
    error "Build directory not found: $BUILD_DIR"
    exit 1
fi

# Test 2: Check XDP program
echo ""
echo "Test 2: XDP program"
if [ -f "$BUILD_DIR/envoy_xdp.o" ]; then
    success "XDP program built: envoy_xdp.o"

    # Check file type
    if command -v file &> /dev/null; then
        FILE_TYPE=$(file "$BUILD_DIR/envoy_xdp.o")
        if [[ $FILE_TYPE == *"eBPF"* ]] || [[ $FILE_TYPE == *"ELF"* ]]; then
            success "XDP program is valid BPF object"
        else
            warning "XDP program file type: $FILE_TYPE"
        fi
    fi

    # Show size
    SIZE=$(du -h "$BUILD_DIR/envoy_xdp.o" | cut -f1)
    echo "  Size: $SIZE"
else
    error "XDP program not found: envoy_xdp.o"
    echo "  Run: ./scripts/build_xdp.sh"
fi

# Test 3: Check WASM filters
echo ""
echo "Test 3: WASM filters"
FILTERS=("auth_filter" "license_filter" "metrics_filter")
WASM_OK=0

for filter in "${FILTERS[@]}"; do
    WASM_FILE="$BUILD_DIR/${filter}.wasm"
    if [ -f "$WASM_FILE" ]; then
        success "WASM filter built: ${filter}.wasm"

        # Show size
        SIZE=$(du -h "$WASM_FILE" | cut -f1)
        echo "  Size: $SIZE"

        # Check if it's a valid WASM file
        if command -v file &> /dev/null; then
            FILE_TYPE=$(file "$WASM_FILE")
            if [[ $FILE_TYPE == *"WebAssembly"* ]]; then
                success "  Valid WebAssembly binary"
            fi
        fi

        WASM_OK=$((WASM_OK + 1))
    else
        error "WASM filter not found: ${filter}.wasm"
    fi
done

if [ $WASM_OK -eq 3 ]; then
    success "All WASM filters built successfully"
else
    error "Missing WASM filters (found $WASM_OK/3)"
    echo "  Run: ./scripts/build_filters.sh"
fi

# Test 4: Check Rust toolchain
echo ""
echo "Test 4: Rust toolchain"
if command -v cargo &> /dev/null; then
    RUST_VERSION=$(cargo --version)
    success "Rust installed: $RUST_VERSION"

    # Check wasm32 target
    if rustup target list | grep -q "wasm32-unknown-unknown (installed)"; then
        success "wasm32-unknown-unknown target installed"
    else
        warning "wasm32-unknown-unknown target not installed"
        echo "  Run: rustup target add wasm32-unknown-unknown"
    fi
else
    warning "Rust/Cargo not found (not required for Docker build)"
fi

# Test 5: Check LLVM/Clang
echo ""
echo "Test 5: LLVM/Clang toolchain"
if command -v clang &> /dev/null; then
    CLANG_VERSION=$(clang --version | head -n1)
    success "Clang installed: $CLANG_VERSION"
else
    warning "Clang not found (not required for Docker build)"
fi

if command -v llc &> /dev/null; then
    LLC_VERSION=$(llc --version | head -n1)
    success "LLC installed: $LLC_VERSION"
else
    warning "LLC not found (not required for Docker build)"
fi

# Test 6: Check Docker
echo ""
echo "Test 6: Docker environment"
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    success "Docker installed: $DOCKER_VERSION"

    # Check if Docker daemon is running
    if docker info &> /dev/null; then
        success "Docker daemon is running"
    else
        error "Docker daemon is not running"
    fi
else
    error "Docker not found"
fi

# Test 7: Check Envoy configuration
echo ""
echo "Test 7: Envoy configuration"
BOOTSTRAP_FILE="$PROJECT_ROOT/envoy/bootstrap.yaml"
if [ -f "$BOOTSTRAP_FILE" ]; then
    success "Bootstrap configuration exists"

    # Check for required fields
    if grep -q "xds_cluster" "$BOOTSTRAP_FILE"; then
        success "xDS cluster configured"
    else
        error "xDS cluster not found in bootstrap"
    fi

    if grep -q "api-server" "$BOOTSTRAP_FILE"; then
        success "API server reference found"
    else
        warning "API server reference not found"
    fi
else
    error "Bootstrap configuration not found"
fi

# Test 8: Check scripts are executable
echo ""
echo "Test 8: Build scripts"
SCRIPTS=("build_xdp.sh" "build_filters.sh" "load_xdp.sh" "entrypoint.sh")
for script in "${SCRIPTS[@]}"; do
    SCRIPT_FILE="$SCRIPT_DIR/$script"
    if [ -f "$SCRIPT_FILE" ]; then
        if [ -x "$SCRIPT_FILE" ]; then
            success "Script executable: $script"
        else
            warning "Script not executable: $script"
            echo "  Run: chmod +x $SCRIPT_FILE"
        fi
    else
        error "Script not found: $script"
    fi
done

# Test 9: Summary
echo ""
echo "═══════════════════════════════════════════════════════════"
echo " Build Verification Summary"
echo "═══════════════════════════════════════════════════════════"

# Count components
TOTAL_COMPONENTS=4
BUILT_COMPONENTS=0

if [ -f "$BUILD_DIR/envoy_xdp.o" ]; then
    BUILT_COMPONENTS=$((BUILT_COMPONENTS + 1))
fi

BUILT_COMPONENTS=$((BUILT_COMPONENTS + WASM_OK))

echo ""
echo "Components built: $BUILT_COMPONENTS/$TOTAL_COMPONENTS"
echo "  - XDP program: $([ -f "$BUILD_DIR/envoy_xdp.o" ] && echo "✓" || echo "✗")"
echo "  - Auth filter: $([ -f "$BUILD_DIR/auth_filter.wasm" ] && echo "✓" || echo "✗")"
echo "  - License filter: $([ -f "$BUILD_DIR/license_filter.wasm" ] && echo "✓" || echo "✗")"
echo "  - Metrics filter: $([ -f "$BUILD_DIR/metrics_filter.wasm" ] && echo "✓" || echo "✗")"

echo ""
if [ $BUILT_COMPONENTS -eq $TOTAL_COMPONENTS ]; then
    success "All components built successfully!"
    echo ""
    echo "Next steps:"
    echo "  1. Build Docker image: make build-docker"
    echo "  2. Run container: docker run -d -p 10000:10000 -p 9901:9901 marchproxy/proxy-l7"
    echo "  3. Test connectivity: curl http://localhost:9901/ready"
    exit 0
else
    error "Build incomplete ($BUILT_COMPONENTS/$TOTAL_COMPONENTS components)"
    echo ""
    echo "To build missing components:"
    echo "  - XDP: ./scripts/build_xdp.sh"
    echo "  - WASM: ./scripts/build_filters.sh"
    echo "  - All: make build"
    exit 1
fi
