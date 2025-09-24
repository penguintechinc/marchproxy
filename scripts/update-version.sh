#!/bin/bash
#
# MarchProxy Version Update Script
# Updates the build number with current epoch timestamp
#
# Usage:
#   ./scripts/update-version.sh [major] [minor] [patch]
#   ./scripts/update-version.sh            # Update build number only
#   ./scripts/update-version.sh 0 2 0     # Set version to v0.2.0.EPOCH
#   ./scripts/update-version.sh patch     # Increment patch version
#   ./scripts/update-version.sh minor     # Increment minor version
#   ./scripts/update-version.sh major     # Increment major version
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VERSION_FILE="$PROJECT_ROOT/.version"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[VERSION]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[VERSION]${NC} $1"
}

error() {
    echo -e "${RED}[VERSION]${NC} $1"
    exit 1
}

# Read current version
if [[ ! -f "$VERSION_FILE" ]]; then
    error "Version file not found: $VERSION_FILE"
fi

CURRENT_VERSION=$(cat "$VERSION_FILE")
log "Current version: $CURRENT_VERSION"

# Parse version components
if [[ $CURRENT_VERSION =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    MAJOR="${BASH_REMATCH[1]}"
    MINOR="${BASH_REMATCH[2]}"
    PATCH="${BASH_REMATCH[3]}"
    BUILD="${BASH_REMATCH[4]}"
else
    error "Invalid version format: $CURRENT_VERSION"
fi

# Get new epoch timestamp
NEW_BUILD=$(date +%s)

# Handle command line arguments
case "${1:-}" in
    "major")
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        log "Incrementing major version"
        ;;
    "minor")
        MINOR=$((MINOR + 1))
        PATCH=0
        log "Incrementing minor version"
        ;;
    "patch")
        PATCH=$((PATCH + 1))
        log "Incrementing patch version"
        ;;
    "")
        log "Updating build number only"
        ;;
    [0-9]*)
        if [[ $# -eq 3 ]]; then
            MAJOR="$1"
            MINOR="$2"
            PATCH="$3"
            log "Setting version to v$MAJOR.$MINOR.$PATCH"
        else
            error "Invalid arguments. Use: major minor patch (e.g., 0 2 0)"
        fi
        ;;
    *)
        error "Invalid argument: $1. Use: major, minor, patch, or specific version numbers"
        ;;
esac

# Create new version
NEW_VERSION="v$MAJOR.$MINOR.$PATCH.$NEW_BUILD"

# Update version file
echo "$NEW_VERSION" > "$VERSION_FILE"
log "Updated version: $CURRENT_VERSION → $NEW_VERSION"

# Update Go application
GO_MAIN="$PROJECT_ROOT/proxy/cmd/proxy/main.go"
if [[ -f "$GO_MAIN" ]]; then
    sed -i.bak "s/version   = \"[^\"]*\"/version   = \"$NEW_VERSION\"/" "$GO_MAIN"
    rm "$GO_MAIN.bak" 2>/dev/null || true
    log "Updated Go version in $GO_MAIN"
fi

# Verify Python can load the version
PYTHON_INIT="$PROJECT_ROOT/manager/apps/marchproxy/__init__.py"
if [[ -f "$PYTHON_INIT" ]]; then
    log "Python version will be loaded dynamically from .version file"
fi

# Show epoch timestamp information
READABLE_TIME=$(date -d "@$NEW_BUILD" 2>/dev/null || date -r "$NEW_BUILD" 2>/dev/null || echo "Unable to convert")
log "Build timestamp: $NEW_BUILD ($READABLE_TIME)"

# Show git status if available
if command -v git &> /dev/null && git rev-parse --git-dir > /dev/null 2>&1; then
    GIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    GIT_STATUS=$(git status --porcelain 2>/dev/null || echo "")
    
    if [[ -n "$GIT_STATUS" ]]; then
        warn "Working directory has uncommitted changes"
    fi
    
    log "Git commit: $GIT_HASH"
fi

log "Version update complete!"

# Display usage examples
echo ""
echo -e "${BLUE}Examples:${NC}"
echo "  ./scripts/update-version.sh              # Update build timestamp only"
echo "  ./scripts/update-version.sh patch        # Increment patch: v0.1.0.EPOCH → v0.1.1.EPOCH"
echo "  ./scripts/update-version.sh minor        # Increment minor: v0.1.0.EPOCH → v0.2.0.EPOCH"  
echo "  ./scripts/update-version.sh major        # Increment major: v0.1.0.EPOCH → v1.0.0.EPOCH"
echo "  ./scripts/update-version.sh 1 0 0        # Set specific: v1.0.0.EPOCH"
echo ""
echo -e "${BLUE}Current components:${NC}"
echo "  Major: $MAJOR (breaking changes)"
echo "  Minor: $MINOR (new features)"
echo "  Patch: $PATCH (bug fixes)"
echo "  Build: $NEW_BUILD (epoch timestamp)"