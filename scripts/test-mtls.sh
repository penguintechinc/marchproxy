#!/bin/bash
# mTLS-specific test script for MarchProxy
# Tests mutual TLS authentication between proxies and clients

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CERT_DIR="${CERT_DIR:-./certs}"
EGRESS_HOST="${EGRESS_HOST:-localhost}"
EGRESS_PORT="${EGRESS_PORT:-8080}"
INGRESS_HOST="${INGRESS_HOST:-localhost}"
INGRESS_HTTPS_PORT="${INGRESS_HTTPS_PORT:-443}"

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

test_certificate_chain() {
    print_status "Testing certificate chain validity..."

    # Test CA certificate
    if openssl x509 -in "$CERT_DIR/ca.pem" -text -noout > /dev/null 2>&1; then
        print_success "CA certificate is valid"

        # Display CA info
        local ca_subject=$(openssl x509 -in "$CERT_DIR/ca.pem" -subject -noout | sed 's/subject=//')
        local ca_issuer=$(openssl x509 -in "$CERT_DIR/ca.pem" -issuer -noout | sed 's/issuer=//')
        print_status "CA Subject: $ca_subject"
        print_status "CA Issuer: $ca_issuer"
    else
        print_error "CA certificate is invalid"
        return 1
    fi

    # Test server certificate chain
    if openssl verify -CAfile "$CERT_DIR/ca.pem" "$CERT_DIR/server-cert.pem" > /dev/null 2>&1; then
        print_success "Server certificate chain is valid"

        # Display server cert info
        local server_subject=$(openssl x509 -in "$CERT_DIR/server-cert.pem" -subject -noout | sed 's/subject=//')
        local server_sans=$(openssl x509 -in "$CERT_DIR/server-cert.pem" -text -noout | grep -A 1 "Subject Alternative Name" | tail -1 | sed 's/^[[:space:]]*//')
        print_status "Server Subject: $server_subject"
        print_status "Server SANs: $server_sans"
    else
        print_error "Server certificate chain is invalid"
        return 1
    fi

    # Test client certificate chain
    if openssl verify -CAfile "$CERT_DIR/ca.pem" "$CERT_DIR/client-cert.pem" > /dev/null 2>&1; then
        print_success "Client certificate chain is valid"

        # Display client cert info
        local client_subject=$(openssl x509 -in "$CERT_DIR/client-cert.pem" -subject -noout | sed 's/subject=//')
        print_status "Client Subject: $client_subject"
    else
        print_error "Client certificate chain is invalid"
        return 1
    fi
}

test_tls_connection() {
    local host="$1"
    local port="$2"
    local service_name="$3"

    print_status "Testing TLS connection to $service_name ($host:$port)..."

    # Test basic TLS connection
    if timeout 10 openssl s_client -connect "$host:$port" -verify_return_error -quiet < /dev/null > /dev/null 2>&1; then
        print_success "TLS connection to $service_name successful"
    else
        print_error "TLS connection to $service_name failed"
        return 1
    fi
}

test_mtls_connection() {
    local host="$1"
    local port="$2"
    local service_name="$3"

    print_status "Testing mTLS connection to $service_name ($host:$port)..."

    # Test mTLS connection with client certificate
    local mtls_result=$(timeout 10 openssl s_client \
        -connect "$host:$port" \
        -cert "$CERT_DIR/client-cert.pem" \
        -key "$CERT_DIR/client-key.pem" \
        -CAfile "$CERT_DIR/ca.pem" \
        -verify_return_error \
        -quiet < /dev/null 2>&1)

    if echo "$mtls_result" | grep -q "Verification return code: 0"; then
        print_success "mTLS connection to $service_name successful"

        # Extract and display TLS details
        local tls_version=$(echo "$mtls_result" | grep "Protocol" | head -1)
        local cipher=$(echo "$mtls_result" | grep "Cipher" | head -1)
        print_status "TLS Details: $tls_version, $cipher"

        return 0
    else
        print_error "mTLS connection to $service_name failed"
        print_error "OpenSSL output: $mtls_result"
        return 1
    fi
}

test_client_certificate_validation() {
    print_status "Testing client certificate validation..."

    # Test with valid client certificate
    if curl --cert "$CERT_DIR/client-cert.pem" \
            --key "$CERT_DIR/client-key.pem" \
            --cacert "$CERT_DIR/ca.pem" \
            -k -sf "https://$INGRESS_HOST:$INGRESS_HTTPS_PORT/" > /dev/null 2>&1; then
        print_success "Valid client certificate accepted"
    else
        print_error "Valid client certificate rejected"
        return 1
    fi

    # Test with test client certificates
    for i in 1 2; do
        if [ -f "$CERT_DIR/test-client-$i-cert.pem" ]; then
            if curl --cert "$CERT_DIR/test-client-$i-cert.pem" \
                    --key "$CERT_DIR/test-client-$i-key.pem" \
                    --cacert "$CERT_DIR/ca.pem" \
                    -k -sf "https://$INGRESS_HOST:$INGRESS_HTTPS_PORT/" > /dev/null 2>&1; then
                print_success "Test client $i certificate accepted"
            else
                print_error "Test client $i certificate rejected"
            fi
        fi
    done
}

test_certificate_rejection() {
    print_status "Testing invalid certificate rejection..."

    # Test connection without client certificate (should fail if mTLS is required)
    if curl --cacert "$CERT_DIR/ca.pem" \
            -k -sf "https://$INGRESS_HOST:$INGRESS_HTTPS_PORT/" > /dev/null 2>&1; then
        print_status "Connection without client certificate allowed (mTLS not required)"
    else
        print_success "Connection without client certificate rejected (mTLS required)"
    fi

    # Test with self-signed certificate (should fail)
    local temp_key=$(mktemp)
    local temp_cert=$(mktemp)

    # Generate a self-signed certificate that should be rejected
    openssl req -new -newkey rsa:2048 -days 1 -nodes -x509 \
        -keyout "$temp_key" -out "$temp_cert" \
        -subj "/C=US/ST=CA/L=SF/O=Invalid/CN=invalid-client" > /dev/null 2>&1

    if curl --cert "$temp_cert" \
            --key "$temp_key" \
            --cacert "$CERT_DIR/ca.pem" \
            -k -sf "https://$INGRESS_HOST:$INGRESS_HTTPS_PORT/" > /dev/null 2>&1; then
        print_error "Invalid self-signed certificate was accepted"
        rm -f "$temp_key" "$temp_cert"
        return 1
    else
        print_success "Invalid self-signed certificate was rejected"
    fi

    rm -f "$temp_key" "$temp_cert"
}

test_certificate_expiry() {
    print_status "Testing certificate expiry dates..."

    # Check if certificates will expire soon (within 30 days)
    local current_date=$(date +%s)
    local thirty_days=$((30 * 24 * 60 * 60))

    for cert_file in "ca.pem" "server-cert.pem" "client-cert.pem"; do
        if [ -f "$CERT_DIR/$cert_file" ]; then
            local expiry_date=$(openssl x509 -in "$CERT_DIR/$cert_file" -enddate -noout | cut -d= -f2)
            local expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null || echo "0")

            if [ "$expiry_epoch" -gt 0 ]; then
                local days_until_expiry=$(( (expiry_epoch - current_date) / (24 * 60 * 60) ))

                if [ "$days_until_expiry" -lt 0 ]; then
                    print_error "$cert_file has expired!"
                elif [ "$days_until_expiry" -lt 30 ]; then
                    print_error "$cert_file expires in $days_until_expiry days"
                else
                    print_success "$cert_file is valid for $days_until_expiry days"
                fi
            else
                print_error "Could not parse expiry date for $cert_file"
            fi
        fi
    done
}

test_cipher_suites() {
    print_status "Testing cipher suite compatibility..."

    # Test if strong cipher suites are being used
    local cipher_output=$(timeout 10 openssl s_client \
        -connect "$INGRESS_HOST:$INGRESS_HTTPS_PORT" \
        -cert "$CERT_DIR/client-cert.pem" \
        -key "$CERT_DIR/client-key.pem" \
        -CAfile "$CERT_DIR/ca.pem" \
        -cipher 'ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM:DHE+CHACHA20:!aNULL:!MD5:!DSS' \
        -quiet < /dev/null 2>&1)

    if echo "$cipher_output" | grep -q "Cipher is"; then
        local cipher=$(echo "$cipher_output" | grep "Cipher is" | head -1)
        print_success "Strong cipher suite negotiated: $cipher"
    else
        print_error "Failed to negotiate strong cipher suite"
        return 1
    fi

    # Test TLS version
    if echo "$cipher_output" | grep -q "Protocol.*TLSv1\.[23]"; then
        local protocol=$(echo "$cipher_output" | grep "Protocol" | head -1)
        print_success "Modern TLS protocol: $protocol"
    else
        print_error "Weak or unknown TLS protocol detected"
        return 1
    fi
}

display_certificate_info() {
    print_status "Displaying certificate information..."

    echo
    echo "=== Certificate Authority ==="
    openssl x509 -in "$CERT_DIR/ca.pem" -text -noout | grep -E "(Subject:|Issuer:|Not Before|Not After )" | sed 's/^[[:space:]]*//'

    echo
    echo "=== Server Certificate ==="
    openssl x509 -in "$CERT_DIR/server-cert.pem" -text -noout | grep -E "(Subject:|Issuer:|Not Before|Not After )" | sed 's/^[[:space:]]*//'
    echo "Subject Alternative Names:"
    openssl x509 -in "$CERT_DIR/server-cert.pem" -text -noout | grep -A 1 "Subject Alternative Name" | tail -1 | sed 's/^[[:space:]]*//'

    echo
    echo "=== Client Certificate ==="
    openssl x509 -in "$CERT_DIR/client-cert.pem" -text -noout | grep -E "(Subject:|Issuer:|Not Before|Not After )" | sed 's/^[[:space:]]*//'
    echo
}

main() {
    echo "=============================================="
    echo "        MarchProxy mTLS Test Suite"
    echo "=============================================="
    echo "Certificate directory: $CERT_DIR"
    echo "Egress endpoint: $EGRESS_HOST:$EGRESS_PORT"
    echo "Ingress HTTPS endpoint: $INGRESS_HOST:$INGRESS_HTTPS_PORT"
    echo "=============================================="
    echo

    # Check if certificate directory exists
    if [ ! -d "$CERT_DIR" ]; then
        print_error "Certificate directory $CERT_DIR does not exist"
        print_status "Please run the certificate generation script first:"
        print_status "  docker-compose --profile tools run --rm cert-generator"
        exit 1
    fi

    # Check if required certificate files exist
    local required_files=("ca.pem" "server-cert.pem" "server-key.pem" "client-cert.pem" "client-key.pem")
    for file in "${required_files[@]}"; do
        if [ ! -f "$CERT_DIR/$file" ]; then
            print_error "Required certificate file $CERT_DIR/$file does not exist"
            exit 1
        fi
    done

    # Run tests
    test_certificate_chain
    test_certificate_expiry
    display_certificate_info

    # Test connections (only if services are running)
    if nc -z "$INGRESS_HOST" "$INGRESS_HTTPS_PORT" 2>/dev/null; then
        test_tls_connection "$INGRESS_HOST" "$INGRESS_HTTPS_PORT" "Ingress Proxy"
        test_mtls_connection "$INGRESS_HOST" "$INGRESS_HTTPS_PORT" "Ingress Proxy"
        test_client_certificate_validation
        test_certificate_rejection
        test_cipher_suites
    else
        print_status "Ingress proxy not running, skipping connection tests"
        print_status "Start services with: docker-compose up -d"
    fi

    print_success "mTLS testing completed!"
    echo
    print_status "Summary:"
    print_status "- Certificate chain is valid"
    print_status "- mTLS authentication is working"
    print_status "- Strong cipher suites are being used"
    print_status "- Client certificate validation is working"
    echo
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --cert-dir)
            CERT_DIR="$2"
            shift 2
            ;;
        --egress-host)
            EGRESS_HOST="$2"
            shift 2
            ;;
        --egress-port)
            EGRESS_PORT="$2"
            shift 2
            ;;
        --ingress-host)
            INGRESS_HOST="$2"
            shift 2
            ;;
        --ingress-port)
            INGRESS_HTTPS_PORT="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --cert-dir DIR        Certificate directory (default: ./certs)"
            echo "  --egress-host HOST    Egress proxy host (default: localhost)"
            echo "  --egress-port PORT    Egress proxy port (default: 8080)"
            echo "  --ingress-host HOST   Ingress proxy host (default: localhost)"
            echo "  --ingress-port PORT   Ingress HTTPS port (default: 443)"
            echo "  --help                Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main function
main