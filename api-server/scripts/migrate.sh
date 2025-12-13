#!/bin/bash

# MarchProxy Database Migration Runner
# Applies pending migrations to the database

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

# Check if running in Docker
if [ -f "/.dockerenv" ]; then
    IN_DOCKER=true
else
    IN_DOCKER=false
fi

# Print header
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}MarchProxy Database Migration Runner${NC}"
echo -e "${BLUE}========================================${NC}"

# Verify Alembic config exists
if [ ! -f "$ALEMBIC_INI" ]; then
    echo -e "${RED}Error: alembic.ini not found at $ALEMBIC_INI${NC}"
    exit 1
fi

# Parse command line arguments
REVISION="${1:-head}"
VERBOSE="${2:-}"

# Check for help flag
if [ "$REVISION" = "-h" ] || [ "$REVISION" = "--help" ]; then
    echo ""
    echo -e "${YELLOW}Usage:${NC}"
    echo "  $0 [REVISION] [--verbose]"
    echo ""
    echo -e "${YELLOW}Arguments:${NC}"
    echo "  REVISION    Target migration revision (default: head)"
    echo "  --verbose   Enable verbose output"
    echo ""
    echo -e "${YELLOW}Examples:${NC}"
    echo "  $0                    # Upgrade to latest migration"
    echo "  $0 head               # Same as above"
    echo "  $0 001                # Upgrade to migration 001"
    echo "  $0 head --verbose     # Upgrade with verbose output"
    echo ""
    exit 0
fi

# Handle verbose flag
if [ "$VERBOSE" = "--verbose" ]; then
    ALEMBIC_VERBOSE=""
else
    ALEMBIC_VERBOSE="--ndjson"
fi

# Display configuration
echo -e "${YELLOW}Configuration:${NC}"
echo "  Project Root: $PROJECT_ROOT"
echo "  Alembic Config: $ALEMBIC_INI"
echo "  Target Revision: $REVISION"
if [ "$VERBOSE" = "--verbose" ]; then
    echo "  Verbose Mode: ON"
fi
echo ""

# Verify database connection
echo -e "${YELLOW}Verifying database connection...${NC}"
cd "$PROJECT_ROOT"

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
echo "  Database: $DB_NAME@$DB_HOST (user: $DB_USER)"
echo ""

# Show current migration history
echo -e "${YELLOW}Current migration history:${NC}"
alembic -c "$ALEMBIC_INI" history -r head || true
echo ""

# Get current head revision
echo -e "${YELLOW}Checking current revision:${NC}"
CURRENT=$(alembic -c "$ALEMBIC_INI" current || echo "No migrations applied")
echo "  Current: $CURRENT"
echo ""

# Apply migrations
echo -e "${YELLOW}Applying migrations...${NC}"
if [ "$VERBOSE" = "--verbose" ]; then
    alembic -c "$ALEMBIC_INI" upgrade "$REVISION"
else
    alembic -c "$ALEMBIC_INI" upgrade "$REVISION" 2>&1 | tail -20
fi

# Check migration result
if [ $? -eq 0 ]; then
    echo ""
    echo -e "${YELLOW}Verifying applied migrations:${NC}"
    NEW_CURRENT=$(alembic -c "$ALEMBIC_INI" current || echo "Unknown")
    echo "  Current: $NEW_CURRENT"
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Migration completed successfully!${NC}"
    echo -e "${GREEN}========================================${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}Migration failed!${NC}"
    echo -e "${RED}========================================${NC}"
    exit 1
fi
