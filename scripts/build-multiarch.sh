#!/bin/bash
#
# MarchProxy Multi-Architecture Build Script
# Builds Docker images for multiple architectures using Docker Buildx
#
# Usage:
#   ./scripts/build-multiarch.sh [component] [tag] [push]
#   
# Examples:
#   ./scripts/build-multiarch.sh manager latest        # Build manager for all platforms
#   ./scripts/build-multiarch.sh proxy v1.0.0 push    # Build and push proxy
#   ./scripts/build-multiarch.sh all latest push      # Build and push both components
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[BUILD]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[BUILD]${NC} $1"
}

error() {
    echo -e "${RED}[BUILD]${NC} $1"
    exit 1
}

info() {
    echo -e "${BLUE}[BUILD]${NC} $1"
}

# Default values
COMPONENT="${1:-all}"
TAG="${2:-latest}"
PUSH="${3:-}"
PLATFORMS="linux/amd64,linux/arm64,linux/arm/v7"
REGISTRY="ghcr.io"
NAMESPACE="$(echo "$GITHUB_REPOSITORY" | tr '[:upper:]' '[:lower:]' | sed 's#^#ghcr.io/#' || echo 'marchproxy')"

# Parse component
case "$COMPONENT" in
    "manager"|"proxy"|"all")
        ;;
    *)
        error "Invalid component: $COMPONENT. Use: manager, proxy, or all"
        ;;
esac

# Validate Docker Buildx
if ! docker buildx version &> /dev/null; then
    error "Docker Buildx is required for multi-architecture builds"
fi

# Create/use buildx builder
BUILDER_NAME="marchproxy-multiarch"
if ! docker buildx inspect $BUILDER_NAME &> /dev/null; then
    log "Creating buildx builder: $BUILDER_NAME"
    docker buildx create --name $BUILDER_NAME --platform $PLATFORMS --use
    docker buildx bootstrap
else
    log "Using existing buildx builder: $BUILDER_NAME"
    docker buildx use $BUILDER_NAME
fi

# Get current version
VERSION_FILE="$PROJECT_ROOT/.version"
if [[ -f "$VERSION_FILE" ]]; then
    CURRENT_VERSION=$(cat "$VERSION_FILE")
    log "Current version: $CURRENT_VERSION"
else
    CURRENT_VERSION="unknown"
    warn "No .version file found, using: $CURRENT_VERSION"
fi

# Build function
build_component() {
    local component="$1"
    local tag="$2"
    local should_push="$3"
    
    local image_name="$NAMESPACE/$component"
    local dockerfile="$PROJECT_ROOT/docker/$component/Dockerfile"
    
    log "Building $component for platforms: $PLATFORMS"
    info "Image: $image_name:$tag"
    info "Dockerfile: $dockerfile"
    
    # Build arguments
    local build_args=(
        --platform "$PLATFORMS"
        --file "$dockerfile"
        --tag "$image_name:$tag"
        --build-arg "VERSION=$CURRENT_VERSION"
        --build-arg "BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
        --build-arg "VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
        --target production
    )
    
    # Add push if requested
    if [[ "$should_push" == "push" ]]; then
        build_args+=(--push)
        log "Will push to registry"
    else
        build_args+=(--load)
        warn "Building without push (use 'push' as third argument to push)"
    fi
    
    # Add additional tags
    if [[ "$tag" != "latest" ]]; then
        build_args+=(--tag "$image_name:latest")
    fi
    
    # Add version tags if not latest
    if [[ "$CURRENT_VERSION" != "unknown" && "$tag" == "latest" ]]; then
        # Extract version without epoch for clean tag
        VERSION_CLEAN=$(echo "$CURRENT_VERSION" | sed 's/\.[0-9]\{10\}$//')
        if [[ "$VERSION_CLEAN" != "$CURRENT_VERSION" ]]; then
            build_args+=(--tag "$image_name:$VERSION_CLEAN")
            log "Adding clean version tag: $VERSION_CLEAN"
        fi
    fi
    
    # Execute build
    log "Executing docker buildx build..."
    docker buildx build "${build_args[@]}" "$PROJECT_ROOT"
    
    if [[ $? -eq 0 ]]; then
        log "✅ Successfully built $component"
    else
        error "❌ Failed to build $component"
    fi
}

# Main build logic
cd "$PROJECT_ROOT"

case "$COMPONENT" in
    "manager")
        build_component "manager" "$TAG" "$PUSH"
        ;;
    "proxy")
        build_component "proxy" "$TAG" "$PUSH"
        ;;
    "all")
        build_component "manager" "$TAG" "$PUSH"
        build_component "proxy" "$TAG" "$PUSH"
        ;;
esac

log "🎉 Multi-architecture build complete!"

# Show platform information
info "Built for platforms:"
for platform in $(echo $PLATFORMS | tr ',' '\n'); do
    info "  ✓ $platform"
done

# Show image information
if [[ "$PUSH" == "push" ]]; then
    info "Images pushed to registry:"
else
    info "Images available locally:"
fi

case "$COMPONENT" in
    "manager"|"all")
        info "  📦 $NAMESPACE/manager:$TAG"
        ;;
esac

case "$COMPONENT" in
    "proxy"|"all")
        info "  📦 $NAMESPACE/proxy:$TAG"
        ;;
esac

info ""
info "Usage examples:"
info "  # Pull and run (any architecture)"
info "  docker run --rm $NAMESPACE/manager:$TAG"
info "  docker run --rm $NAMESPACE/proxy:$TAG"
info ""
info "  # Use in docker-compose.yml"
info "  services:"
info "    manager:"
info "      image: $NAMESPACE/manager:$TAG"
info "    proxy:"
info "      image: $NAMESPACE/proxy:$TAG"

# Platform-specific examples
info ""
info "Platform-specific examples:"
info "  # Force specific architecture"
info "  docker run --platform linux/amd64 $NAMESPACE/manager:$TAG"
info "  docker run --platform linux/arm64 $NAMESPACE/proxy:$TAG"