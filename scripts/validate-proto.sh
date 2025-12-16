#!/bin/bash

# validate-proto.sh - Validate protobuf definitions
# This script validates proto files for syntax errors without generating code

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

echo -e "${GREEN}MarchProxy Proto Validation${NC}"
echo "============================================"
echo ""

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo -e "${YELLOW}Warning: protoc is not installed${NC}"
    echo "Validation requires protoc. You can install it with:"
    echo "  - Debian/Ubuntu: sudo apt-get install -y protobuf-compiler"
    echo "  - macOS: brew install protobuf"
    echo ""
    echo "Performing basic syntax validation instead..."
    echo ""

    # Basic validation without protoc
    ERRORS=0
    for proto_file in "${PROTO_DIR}/marchproxy"/*.proto; do
        filename=$(basename "$proto_file")
        echo "  - Checking ${filename}..."

        # Check for syntax = "proto3"
        if ! grep -q 'syntax = "proto3";' "$proto_file"; then
            echo -e "${RED}    ✗ Missing or incorrect syntax declaration${NC}"
            ERRORS=$((ERRORS + 1))
        fi

        # Check for package declaration
        if ! grep -q '^package marchproxy;' "$proto_file"; then
            echo -e "${RED}    ✗ Missing or incorrect package declaration${NC}"
            ERRORS=$((ERRORS + 1))
        fi

        # Check for go_package option
        if ! grep -q 'option go_package' "$proto_file"; then
            echo -e "${RED}    ✗ Missing go_package option${NC}"
            ERRORS=$((ERRORS + 1))
        fi

        # Check for balanced braces
        OPEN_BRACES=$(grep -o '{' "$proto_file" | wc -l)
        CLOSE_BRACES=$(grep -o '}' "$proto_file" | wc -l)
        if [ "$OPEN_BRACES" -ne "$CLOSE_BRACES" ]; then
            echo -e "${RED}    ✗ Unbalanced braces (${OPEN_BRACES} open, ${CLOSE_BRACES} close)${NC}"
            ERRORS=$((ERRORS + 1))
        fi

        if [ "$ERRORS" -eq 0 ]; then
            echo -e "${GREEN}    ✓ Basic validation passed${NC}"
        fi
    done

    if [ "$ERRORS" -gt 0 ]; then
        echo ""
        echo -e "${RED}✗ Validation found ${ERRORS} error(s)${NC}"
        exit 1
    else
        echo ""
        echo -e "${GREEN}✓ Basic validation passed for all files${NC}"
        echo -e "${YELLOW}Note: Install protoc for full validation${NC}"
    fi

    exit 0
fi

# Full validation with protoc
PROTOC_VERSION=$(protoc --version | awk '{print $2}')
echo "Using protoc version: ${PROTOC_VERSION}"
echo ""

ERRORS=0
for proto_file in "${PROTO_DIR}/marchproxy"/*.proto; do
    filename=$(basename "$proto_file")
    echo "  - Validating ${filename}..."

    # Validate proto file
    if protoc \
        --proto_path="${PROTO_DIR}" \
        --descriptor_set_out=/dev/null \
        "${proto_file}" 2>&1; then
        echo -e "${GREEN}    ✓ Validation passed${NC}"
    else
        echo -e "${RED}    ✗ Validation failed${NC}"
        ERRORS=$((ERRORS + 1))
    fi
done

echo ""
if [ "$ERRORS" -gt 0 ]; then
    echo -e "${RED}✗ Validation failed for ${ERRORS} file(s)${NC}"
    exit 1
else
    echo -e "${GREEN}✓ All proto files validated successfully${NC}"
fi
