#!/bin/bash
# verify-phase5-docs.sh
# Verifies that all Phase 5 documentation and scripts are in place

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

BASE_DIR="/home/penguin/code/MarchProxy"

echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}Phase 5 Documentation Verification${NC}"
echo -e "${GREEN}=====================================${NC}\n"

# Check for required files
REQUIRED_FILES=(
    "docs/PHASE5_L3L4_IMPLEMENTATION.md"
    "docs/PHASE5_QUICKSTART.md"
    "docs/README_PHASE5.md"
    "PHASE5_SUMMARY.md"
    "scripts/implement-phase5-l3l4.sh"
)

PASSED=0
FAILED=0

for file in "${REQUIRED_FILES[@]}"; do
    filepath="${BASE_DIR}/${file}"
    if [ -f "$filepath" ]; then
        size=$(stat -f%z "$filepath" 2>/dev/null || stat -c%s "$filepath" 2>/dev/null)
        echo -e "${GREEN}✓${NC} Found: ${file} (${size} bytes)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗${NC} Missing: ${file}"
        FAILED=$((FAILED + 1))
    fi
done

# Check if script is executable
script="${BASE_DIR}/scripts/implement-phase5-l3l4.sh"
if [ -x "$script" ]; then
    echo -e "${GREEN}✓${NC} Script is executable: implement-phase5-l3l4.sh"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} Script is not executable: implement-phase5-l3l4.sh"
    FAILED=$((FAILED + 1))
fi

# Verify file sizes (ensure they're not empty)
echo -e "\n${YELLOW}File Size Verification:${NC}"

impl_size=$(stat -f%z "${BASE_DIR}/docs/PHASE5_L3L4_IMPLEMENTATION.md" 2>/dev/null || stat -c%s "${BASE_DIR}/docs/PHASE5_L3L4_IMPLEMENTATION.md" 2>/dev/null)
quick_size=$(stat -f%z "${BASE_DIR}/docs/PHASE5_QUICKSTART.md" 2>/dev/null || stat -c%s "${BASE_DIR}/docs/PHASE5_QUICKSTART.md" 2>/dev/null)
readme_size=$(stat -f%z "${BASE_DIR}/docs/README_PHASE5.md" 2>/dev/null || stat -c%s "${BASE_DIR}/docs/README_PHASE5.md" 2>/dev/null)
summary_size=$(stat -f%z "${BASE_DIR}/PHASE5_SUMMARY.md" 2>/dev/null || stat -c%s "${BASE_DIR}/PHASE5_SUMMARY.md" 2>/dev/null)
script_size=$(stat -f%z "${BASE_DIR}/scripts/implement-phase5-l3l4.sh" 2>/dev/null || stat -c%s "${BASE_DIR}/scripts/implement-phase5-l3l4.sh" 2>/dev/null)

echo "  - PHASE5_L3L4_IMPLEMENTATION.md: ${impl_size} bytes"
echo "  - PHASE5_QUICKSTART.md: ${quick_size} bytes"
echo "  - README_PHASE5.md: ${readme_size} bytes"
echo "  - PHASE5_SUMMARY.md: ${summary_size} bytes"
echo "  - implement-phase5-l3l4.sh: ${script_size} bytes"

# Check if proxy-egress exists (prerequisite)
echo -e "\n${YELLOW}Prerequisite Check:${NC}"
if [ -d "${BASE_DIR}/proxy-egress" ]; then
    echo -e "${GREEN}✓${NC} proxy-egress directory exists (baseline for proxy-l3l4)"
    PASSED=$((PASSED + 1))
else
    echo -e "${RED}✗${NC} proxy-egress directory not found (required baseline)"
    FAILED=$((FAILED + 1))
fi

# Check if proxy-l3l4 already exists
if [ -d "${BASE_DIR}/proxy-l3l4" ]; then
    echo -e "${YELLOW}⚠${NC} proxy-l3l4 directory already exists"
    echo "  (Script will prompt to remove and recreate)"
else
    echo -e "${GREEN}✓${NC} proxy-l3l4 does not exist (ready for creation)"
    PASSED=$((PASSED + 1))
fi

# Summary
echo -e "\n${GREEN}=====================================${NC}"
echo -e "${GREEN}Verification Summary${NC}"
echo -e "${GREEN}=====================================${NC}"
echo -e "Passed: ${GREEN}${PASSED}${NC}"
echo -e "Failed: ${RED}${FAILED}${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "\n${GREEN}✓ All Phase 5 documentation is in place!${NC}"
    echo -e "\n${YELLOW}Next Step:${NC}"
    echo -e "Run the implementation script:"
    echo -e "  ${GREEN}./scripts/implement-phase5-l3l4.sh${NC}"
    exit 0
else
    echo -e "\n${RED}✗ Some files are missing or incorrect${NC}"
    echo -e "Please ensure all Phase 5 files have been created."
    exit 1
fi
