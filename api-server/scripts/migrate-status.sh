#!/bin/bash

# MarchProxy Database Migration Status Checker
# Displays current migration status and history

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
echo -e "${BLUE}MarchProxy Migration Status${NC}"
echo -e "${BLUE}========================================${NC}"

# Verify Alembic config exists
if [ ! -f "$ALEMBIC_INI" ]; then
    echo -e "${RED}Error: alembic.ini not found at $ALEMBIC_INI${NC}"
    exit 1
fi

# Change to project directory
cd "$PROJECT_ROOT" || exit 1

# Extract database URL from alembic.ini
DB_URL=$(grep "sqlalchemy.url" "$ALEMBIC_INI" | cut -d '=' -f2 | xargs)

if [ -z "$DB_URL" ]; then
    echo -e "${RED}Error: Could not find sqlalchemy.url in alembic.ini${NC}"
    exit 1
fi

# Show database info
DB_USER=$(echo "$DB_URL" | sed -E 's/.*:\/\/([^:]+):.*/\1/')
DB_HOST=$(echo "$DB_URL" | sed -E 's/.*@([^:\/]+).*/\1/')
DB_NAME=$(echo "$DB_URL" | sed -E 's/.*\/([^?]+).*/\1/')

echo -e "${YELLOW}Database Information:${NC}"
echo "  Database: $DB_NAME"
echo "  Host: $DB_HOST"
echo "  User: $DB_USER"
echo ""

# Get current revision
echo -e "${YELLOW}Current Revision:${NC}"
CURRENT=$(alembic -c "$ALEMBIC_INI" current 2>&1 || echo "Error: Unable to determine current revision")
echo "  $CURRENT"
echo ""

# Get available branches
echo -e "${YELLOW}Available Branches:${NC}"
if alembic -c "$ALEMBIC_INI" branches > /dev/null 2>&1; then
    alembic -c "$ALEMBIC_INI" branches
else
    echo "  No branches found"
fi
echo ""

# Get migration history
echo -e "${YELLOW}Migration History (all revisions):${NC}"
alembic -c "$ALEMBIC_INI" history -i -r head

echo ""
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${YELLOW}Common Migration Commands:${NC}"
echo "  Apply latest:         $SCRIPT_DIR/migrate.sh"
echo "  Apply specific:       $SCRIPT_DIR/migrate.sh 001"
echo "  Downgrade one step:   $SCRIPT_DIR/migrate-down.sh -1"
echo "  Downgrade to base:    $SCRIPT_DIR/migrate-down.sh base"
echo "  Create new migration: alembic -c $ALEMBIC_INI revision --autogenerate -m 'description'"
echo ""
