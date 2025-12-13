#!/bin/bash

# gen-proto.sh - Generate Go and Python code from protobuf definitions
# This script generates gRPC code for the MarchProxy Unified NLB Architecture

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="${PROJECT_ROOT}/proto"

echo -e "${GREEN}MarchProxy Proto Generation${NC}"
echo "============================================"
echo ""

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo -e "${RED}Error: protoc is not installed${NC}"
    echo "Please install protoc:"
    echo "  - Debian/Ubuntu: sudo apt-get install -y protobuf-compiler"
    echo "  - macOS: brew install protobuf"
    echo "  - Or download from: https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

# Check protoc version
PROTOC_VERSION=$(protoc --version | awk '{print $2}')
echo "Using protoc version: ${PROTOC_VERSION}"

# Check for Go protoc plugin
if ! command -v protoc-gen-go &> /dev/null; then
    echo -e "${YELLOW}Warning: protoc-gen-go not found. Installing...${NC}"
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo -e "${YELLOW}Warning: protoc-gen-go-grpc not found. Installing...${NC}"
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Check for Python protoc plugin
if ! command -v grpc_tools.protoc &> /dev/null && ! python3 -c "import grpc_tools.protoc" 2>/dev/null; then
    echo -e "${YELLOW}Warning: grpcio-tools not found for Python. Installing...${NC}"
    pip3 install grpcio-tools
fi

echo ""
echo "Generating Go code..."
echo "============================================"

# Create output directories
GO_OUT_DIR="${PROJECT_ROOT}/pkg/proto/marchproxy"
mkdir -p "${GO_OUT_DIR}"

# Generate Go code for all proto files
for proto_file in "${PROTO_DIR}/marchproxy"/*.proto; do
    filename=$(basename "$proto_file")
    echo "  - Generating Go code for ${filename}..."

    protoc \
        --proto_path="${PROTO_DIR}" \
        --go_out="${PROJECT_ROOT}/pkg/proto" \
        --go_opt=paths=source_relative \
        --go-grpc_out="${PROJECT_ROOT}/pkg/proto" \
        --go-grpc_opt=paths=source_relative \
        "${proto_file}"
done

echo -e "${GREEN}✓ Go code generated successfully${NC}"
echo ""

echo "Generating Python code..."
echo "============================================"

# Create output directories for Python
PYTHON_OUT_DIR="${PROJECT_ROOT}/manager/proto/marchproxy"
mkdir -p "${PYTHON_OUT_DIR}"

# Create __init__.py for Python package
touch "${PROJECT_ROOT}/manager/proto/__init__.py"
touch "${PYTHON_OUT_DIR}/__init__.py"

# Generate Python code for all proto files
for proto_file in "${PROTO_DIR}/marchproxy"/*.proto; do
    filename=$(basename "$proto_file")
    echo "  - Generating Python code for ${filename}..."

    python3 -m grpc_tools.protoc \
        --proto_path="${PROTO_DIR}" \
        --python_out="${PROJECT_ROOT}/manager/proto" \
        --grpc_python_out="${PROJECT_ROOT}/manager/proto" \
        "${proto_file}"
done

echo -e "${GREEN}✓ Python code generated successfully${NC}"
echo ""

# Fix Python imports (grpc_tools generates relative imports that may need fixing)
echo "Fixing Python import paths..."
find "${PROJECT_ROOT}/manager/proto" -name "*_pb2*.py" -type f -exec sed -i \
    's/^import marchproxy\./from . import /g' {} \;
echo -e "${GREEN}✓ Python imports fixed${NC}"
echo ""

# Generate summary
echo "Summary:"
echo "============================================"
echo "Proto files processed:"
find "${PROTO_DIR}/marchproxy" -name "*.proto" -type f | while read -r proto; do
    echo "  - $(basename "$proto")"
done
echo ""
echo "Generated files:"
echo "  Go output:     ${GO_OUT_DIR}/"
echo "  Python output: ${PYTHON_OUT_DIR}/"
echo ""

# Count generated files
GO_FILES=$(find "${GO_OUT_DIR}" -name "*.pb.go" -o -name "*_grpc.pb.go" 2>/dev/null | wc -l)
PYTHON_FILES=$(find "${PYTHON_OUT_DIR}" -name "*_pb2*.py" 2>/dev/null | wc -l)

echo "  Go files:     ${GO_FILES}"
echo "  Python files: ${PYTHON_FILES}"
echo ""

# Verify generation
if [ "$GO_FILES" -eq 0 ]; then
    echo -e "${RED}Error: No Go files were generated${NC}"
    exit 1
fi

if [ "$PYTHON_FILES" -eq 0 ]; then
    echo -e "${YELLOW}Warning: No Python files were generated${NC}"
fi

echo -e "${GREEN}✓ Proto generation complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. Import the generated packages in your Go code:"
echo "     import \"github.com/penguintech/marchproxy/pkg/proto/marchproxy\""
echo ""
echo "  2. Import the generated packages in your Python code:"
echo "     from proto.marchproxy import types_pb2, module_pb2, nlb_pb2"
echo ""
