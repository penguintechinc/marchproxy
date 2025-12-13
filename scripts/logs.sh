#!/bin/bash

set -euo pipefail

# MarchProxy Logs Script
# Displays logs from containers with color coding and filtering options

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"
FOLLOW=false
TAIL_LINES=100
TIMESTAMPS=true

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_usage() {
    echo "Usage: $0 [OPTIONS] [SERVICE]"
    echo ""
    echo "Display logs from MarchProxy services"
    echo ""
    echo "OPTIONS:"
    echo "  -f, --follow          Follow log output (live tail)"
    echo "  -n, --lines NUM       Show last NUM lines (default: 100)"
    echo "  -t, --no-timestamps   Hide timestamps"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "SERVICES:"
    echo "  Infrastructure:"
    echo "    postgres, redis, elasticsearch"
    echo ""
    echo "  Observability:"
    echo "    jaeger, prometheus, grafana, kibana, logstash, loki, alertmanager"
    echo ""
    echo "  Core Services:"
    echo "    api-server, manager, webui"
    echo ""
    echo "  Proxies:"
    echo "    proxy-l7, proxy-l3l4, proxy-egress, proxy-ingress"
    echo ""
    echo "  Other:"
    echo "    config-sync"
    echo ""
    echo "  Special:"
    echo "    all       Show all service logs"
    echo "    critical  Show logs from critical services only"
    echo ""
    echo "EXAMPLES:"
    echo "  $0 api-server                    # Show last 100 lines of api-server logs"
    echo "  $0 -f api-server                 # Follow api-server logs in real-time"
    echo "  $0 -n 50 postgres                # Show last 50 lines of postgres logs"
    echo "  $0 -f critical                   # Follow critical services (postgres, redis, api-server)"
    echo "  $0 -f all                        # Follow all service logs"
    echo ""
}

# Parse arguments
TARGET_SERVICE=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--follow)
            FOLLOW=true
            shift
            ;;
        -n|--lines)
            TAIL_LINES="$2"
            shift 2
            ;;
        -t|--no-timestamps)
            TIMESTAMPS=false
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            TARGET_SERVICE="$1"
            shift
            ;;
    esac
done

# Default to api-server if no service specified
if [ -z "$TARGET_SERVICE" ]; then
    TARGET_SERVICE="api-server"
fi

# Build docker-compose logs command
COMPOSE_CMD="docker-compose -f $COMPOSE_FILE logs"

# Add options
if [ "$FOLLOW" = true ]; then
    COMPOSE_CMD="$COMPOSE_CMD -f"
fi

COMPOSE_CMD="$COMPOSE_CMD --tail $TAIL_LINES"

if [ "$TIMESTAMPS" = false ]; then
    COMPOSE_CMD="$COMPOSE_CMD --no-log-prefix"
fi

# Handle special service names
case "$TARGET_SERVICE" in
    critical)
        log_info "Showing logs for critical services (postgres, redis, api-server, manager)"
        $COMPOSE_CMD postgres redis api-server manager
        ;;
    all)
        log_info "Showing logs for all services"
        $COMPOSE_CMD
        ;;
    *)
        # Check if service exists
        if ! docker-compose -f "$COMPOSE_FILE" ps "$TARGET_SERVICE" &>/dev/null 2>&1; then
            # Try anyway, docker-compose will give us the error
            log_info "Showing logs for: $TARGET_SERVICE"
        fi
        $COMPOSE_CMD "$TARGET_SERVICE"
        ;;
esac

exit 0
