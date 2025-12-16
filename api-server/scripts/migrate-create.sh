#!/bin/bash

# MarchProxy Migration Creator
# Creates a new migration file with autogenerate

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ALEMBIC_INI="$PROJECT_ROOT/alembic.ini"

# Check for help flag or no arguments
if [ $# -eq 0 ] || [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}MarchProxy Migration Creator${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 <MESSAGE> [--autogenerate] [--manual]"
    echo ""
    echo -e "${YELLOW}Arguments:${NC}"
    echo "  MESSAGE        Description of the migration"
    echo "  --autogenerate Generate from model changes (default)"
    echo "  --manual       Create empty migration for manual SQL"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0 \"Add certificate table\""
    echo "  $0 \"Add certificate table\" --autogenerate"
    echo "  $0 \"Fix user table constraints\" --manual"
    echo ""
    echo -e "${YELLOW}Tips:${NC}"
    echo "  - Always review generated migrations before applying"
    echo "  - Test migrations on a copy of production data"
    echo "  - Write descriptive messages that explain the change"
    echo ""
    exit 0
fi

# Verify Alembic config exists
if [ ! -f "$ALEMBIC_INI" ]; then
    echo -e "${RED}Error: alembic.ini not found at $ALEMBIC_INI${NC}"
    exit 1
fi

# Parse arguments
MESSAGE="$1"
MODE="--autogenerate"

shift || true
while [ $# -gt 0 ]; do
    case "$1" in
        --autogenerate)
            MODE="--autogenerate"
            ;;
        --manual)
            MODE=""
            ;;
        *)
            echo -e "${RED}Error: Unknown option '$1'${NC}"
            exit 1
            ;;
    esac
    shift
done

# Change to project directory
cd "$PROJECT_ROOT" || exit 1

# Print header
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}MarchProxy Migration Creator${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Display configuration
echo -e "${YELLOW}Configuration:${NC}"
echo "  Project Root: $PROJECT_ROOT"
echo "  Message: $MESSAGE"
if [ "$MODE" = "--autogenerate" ]; then
    echo "  Mode: Autogenerate from model changes"
else
    echo "  Mode: Manual (empty migration)"
fi
echo ""

# Create the migration
echo -e "${YELLOW}Creating migration...${NC}"
if [ "$MODE" = "--autogenerate" ]; then
    alembic -c "$ALEMBIC_INI" revision --autogenerate -m "$MESSAGE"
else
    alembic -c "$ALEMBIC_INI" revision -m "$MESSAGE"
fi

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Migration created successfully!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "  1. Review the generated migration file in alembic/versions/"
    echo "  2. Test the migration on a development database"
    echo "  3. Run: $SCRIPT_DIR/migrate.sh [revision]"
    echo ""

    # Show the latest migration file
    echo -e "${YELLOW}Latest migration file:${NC}"
    LATEST=$(ls -t "$PROJECT_ROOT/alembic/versions/"*.py 2>/dev/null | head -1)
    if [ -n "$LATEST" ]; then
        echo "  $LATEST"
        echo ""
        echo -e "${YELLOW}Preview (first 30 lines):${NC}"
        head -30 "$LATEST" | sed 's/^/  /'
    fi
    exit 0
else
    echo ""
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}Failed to create migration!${NC}"
    echo -e "${RED}========================================${NC}"
    exit 1
fi
