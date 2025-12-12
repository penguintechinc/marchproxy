#!/bin/bash
# Entrypoint script for Envoy L7 proxy

set -e

echo "Starting MarchProxy Envoy L7 Proxy..."
echo "XDS Server: ${XDS_SERVER:-api-server:18000}"
echo "Cluster API Key: ${CLUSTER_API_KEY:0:10}..." # Show only first 10 chars

# Wait for xDS server to be ready
XDS_HOST="${XDS_SERVER%%:*}"
XDS_PORT="${XDS_SERVER##*:}"

if [ -n "$XDS_HOST" ] && [ -n "$XDS_PORT" ]; then
    echo "Waiting for xDS server at $XDS_HOST:$XDS_PORT..."
    timeout=30
    while ! nc -z "$XDS_HOST" "$XDS_PORT" 2>/dev/null; do
        timeout=$((timeout - 1))
        if [ $timeout -le 0 ]; then
            echo "Warning: xDS server not reachable after 30 seconds, proceeding anyway..."
            break
        fi
        sleep 1
    done
    echo "xDS server is reachable"
fi

# Optional: Load XDP program if interface is specified
if [ -n "$XDP_INTERFACE" ]; then
    echo "XDP interface specified: $XDP_INTERFACE"
    XDP_MODE="${XDP_MODE:-native}"

    # Check if we have CAP_NET_ADMIN capability
    if capsh --print | grep -q "cap_net_admin"; then
        echo "Loading XDP program on $XDP_INTERFACE in $XDP_MODE mode..."
        if [ -f "/var/lib/envoy/xdp/envoy_xdp.o" ]; then
            # Attempt to load XDP program
            ip link set dev "$XDP_INTERFACE" xdp off 2>/dev/null || true

            case "$XDP_MODE" in
                native)
                    ip link set dev "$XDP_INTERFACE" xdp object /var/lib/envoy/xdp/envoy_xdp.o section xdp || \
                        echo "Warning: Failed to load XDP in native mode"
                    ;;
                skb)
                    ip link set dev "$XDP_INTERFACE" xdpgeneric object /var/lib/envoy/xdp/envoy_xdp.o section xdp || \
                        echo "Warning: Failed to load XDP in SKB mode"
                    ;;
                *)
                    echo "Warning: Invalid XDP mode: $XDP_MODE"
                    ;;
            esac

            echo "XDP program loaded successfully"
        else
            echo "Warning: XDP program not found at /var/lib/envoy/xdp/envoy_xdp.o"
        fi
    else
        echo "Warning: CAP_NET_ADMIN not available, skipping XDP load"
        echo "Run container with --cap-add=NET_ADMIN to enable XDP"
    fi
fi

# Display WASM filters
echo "Available WASM filters:"
ls -lh /var/lib/envoy/wasm/*.wasm

# Display configuration file
echo "Envoy configuration:"
cat /etc/envoy/envoy.yaml

echo ""
echo "Starting Envoy proxy..."

# Execute Envoy with provided arguments
exec /usr/local/bin/envoy "$@"
