#!/bin/bash

# MarchProxy Database Downgrade Runner
# Downgrades migrations to a previous revision

set -e

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

# Print header
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}MarchProxy Database Downgrade Runner${NC}"
echo -e "${BLUE}========================================${NC}"

# Verify Alembic config exists
if [ ! -f "$ALEMBIC_INI" ]; then
    echo -e "${RED}Error: alembic.ini not found at $ALEMBIC_INI${NC}"
    exit 1
fi

# Parse command line arguments
if [ $# -eq 0 ]; then
    echo -e "${RED}Error: No arguments provided${NC}"
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 <REVISION> [--verbose] [--force]"
    echo ""
    echo -e "${YELLOW}Arguments:${NC}"
    echo "  REVISION    Target migration revision or number of steps"
    echo "  --verbose   Enable verbose output"
    echo "  --force     Skip confirmation prompt"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0 -1                # Downgrade one step"
    echo "  $0 -2                # Downgrade two steps"
    echo "  $0 base              # Downgrade to initial state"
    echo "  $0 001               # Downgrade to migration 001"
    echo "  $0 -1 --force        # Downgrade without confirmation"
    echo "  $0 base --verbose    # Downgrade to base with verbose output"
    echo ""
    exit 1
fi

# Check for help flag
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 <REVISION> [--verbose] [--force]"
    echo ""
    echo -e "${YELLOW}Arguments:${NC}"
    echo "  REVISION    Target migration revision or number of steps"
    echo "            - Use 'base' to downgrade all migrations"
    echo "            - Use '-N' (e.g., -1, -2) to go back N steps"
    echo "            - Use revision ID (e.g., '001') to go to specific revision"
    echo "  --verbose   Enable verbose output"
    echo "  --force     Skip confirmation prompt"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0 -1                # Downgrade one step"
    echo "  $0 -2                # Downgrade two steps"
    echo "  $0 base              # Downgrade to initial state"
    echo "  $0 001               # Downgrade to migration 001"
    echo ""
    echo -e "${YELLOW}Warning:${NC}"
    echo "  Downgrading migrations will DELETE DATA. This action is irreversible."
    echo "  Always backup your database before downgrading."
    echo ""
    exit 0
fi

# Parse arguments
REVISION="$1"
VERBOSE=""
FORCE=""

shift || true
while [ $# -gt 0 ]; do
    case "$1" in
        --verbose)
            VERBOSE="yes"
            ;;
        --force)
            FORCE="yes"
            ;;
        *)
            echo -e "${RED}Error: Unknown option '$1'${NC}"
            exit 1
            ;;
    esac
    shift
done

# Change to project directory
cd "$PROJECT_ROOT"

# Display configuration
echo -e "${YELLOW}Configuration:${NC}"
echo "  Project Root: $PROJECT_ROOT"
echo "  Alembic Config: $ALEMBIC_INI"
echo "  Target Revision: $REVISION"
if [ "$VERBOSE" = "yes" ]; then
    echo "  Verbose Mode: ON"
fi
echo ""

# Extract database URL from alembic.ini
DB_URL=$(grep "sqlalchemy.url" "$ALEMBIC_INI" | cut -d '=' -f2 | xargs)

if [ -z "$DB_URL" ]; then
    echo -e "${RED}Error: Could not find sqlalchemy.url in alembic.ini${NC}"
    exit 1
fi

# Show database info (masked for security)
DB_USER=$(echo "$DB_URL" | sed -E 's/.*:\/\/([^:]+):.*/\1/')
DB_HOST=$(echo "$DB_URL" | sed -E 's/.*@([^:\/]+).*/\1/')
DB_NAME=$(echo "$DB_URL" | sed -E 's/.*\/([^?]+).*/\1/')
echo -e "${YELLOW}Database Information:${NC}"
echo "  Database: $DB_NAME@$DB_HOST (user: $DB_USER)"
echo ""

# Show current revision
echo -e "${YELLOW}Current State:${NC}"
CURRENT=$(alembic -c "$ALEMBIC_INI" current || echo "No migrations applied")
echo "  Current: $CURRENT"
echo ""

# Show migration history
echo -e "${YELLOW}Migration History:${NC}"
alembic -c "$ALEMBIC_INI" history -r head || true
echo ""

# Show confirmation warning
echo -e "${RED}WARNING - DATA LOSS RISK!${NC}"
echo -e "${RED}==============================${NC}"
echo "Downgrading will DELETE all data created after the target revision."
echo "This action is IRREVERSIBLE."
echo "You must have a recent backup before proceeding."
echo ""

# Request confirmation unless --force is used
if [ "$FORCE" != "yes" ]; then
    echo -e "${YELLOW}Type 'yes' to confirm downgrade to: $REVISION${NC}"
    read -r -p "Confirm (yes/no): " CONFIRM
    if [ "$CONFIRM" != "yes" ]; then
        echo -e "${YELLOW}Downgrade cancelled.${NC}"
        exit 0
    fi
fi

echo ""
echo -e "${YELLOW}Applying downgrade...${NC}"

# Apply downgrade
if [ "$VERBOSE" = "yes" ]; then
    alembic -c "$ALEMBIC_INI" downgrade "$REVISION"
else
    alembic -c "$ALEMBIC_INI" downgrade "$REVISION" 2>&1 | tail -20
fi

# Check downgrade result
if [ $? -eq 0 ]; then
    echo ""
    echo -e "${YELLOW}Verifying downgraded state:${NC}"
    NEW_CURRENT=$(alembic -c "$ALEMBIC_INI" current || echo "Unknown")
    echo "  Current: $NEW_CURRENT"
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Downgrade completed successfully!${NC}"
    echo -e "${GREEN}========================================${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}Downgrade failed!${NC}"
    echo -e "${RED}========================================${NC}"
    exit 1
fi
