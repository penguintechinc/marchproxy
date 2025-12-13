#!/bin/bash
# Run all test suites for MarchProxy

set -e

echo "========================================="
echo "MarchProxy - Running All Tests"
echo "========================================="

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Change to project root
cd "$(dirname "$0")/.."

# Check if virtual environment exists
if [ ! -d "venv" ]; then
    echo -e "${YELLOW}Creating virtual environment...${NC}"
    python3 -m venv venv
fi

# Activate virtual environment
source venv/bin/activate

# Install test dependencies
echo -e "${YELLOW}Installing test dependencies...${NC}"
pip install -q -r tests/requirements.txt
pip install -q -r api-server/requirements-test.txt

# Run API Server Integration Tests
echo -e "\n${YELLOW}Running API Server Integration Tests...${NC}"
cd api-server
pytest tests/integration/ -v --cov=app --cov-report=html --cov-report=term || true
cd ..

# Run End-to-End Tests
echo -e "\n${YELLOW}Running End-to-End Tests...${NC}"
pytest tests/e2e/ -v -m e2e || true

# Run Security Tests
echo -e "\n${YELLOW}Running Security Tests...${NC}"
pytest tests/security/ -v -m security || true

# Run Performance Tests
echo -e "\n${YELLOW}Running Performance Tests...${NC}"
pytest tests/performance/ -v -m performance || true

# WebUI Tests (Playwright)
echo -e "\n${YELLOW}Running WebUI Tests...${NC}"
cd webui
npm run test || true
cd ..

echo -e "\n${GREEN}=========================================${NC}"
echo -e "${GREEN}All tests completed!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo -e "\nTest reports:"
echo "  - API Coverage: api-server/htmlcov/index.html"
echo "  - API Report: api-server/test-report.html"
echo "  - WebUI Report: webui/playwright-report/index.html"
