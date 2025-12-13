#!/bin/bash
# Generate Python gRPC code from proto files

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Generating Python gRPC code from proto files...${NC}"

# Check if grpcio-tools is installed
if ! python -c "import grpc_tools.protoc" 2>/dev/null; then
    echo -e "${RED}Error: grpcio-tools not installed${NC}"
    echo "Install with: pip install grpcio-tools"
    exit 1
fi

# Create proto output directory
mkdir -p proto/marchproxy

# Generate Python code
python -m grpc_tools.protoc \
    -I../proto \
    --python_out=./proto \
    --grpc_python_out=./proto \
    ../proto/marchproxy/module_service.proto

# Check if generation was successful
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Successfully generated proto files:${NC}"
    ls -la proto/marchproxy/*_pb2*.py 2>/dev/null || echo -e "${YELLOW}Warning: Generated files not found${NC}"
else
    echo -e "${RED}Failed to generate proto files${NC}"
    exit 1
fi

# Create __init__.py files
touch proto/__init__.py
touch proto/marchproxy/__init__.py

echo -e "${GREEN}Proto code generation complete!${NC}"
echo -e "${YELLOW}Note: You may need to fix imports in the generated files${NC}"
