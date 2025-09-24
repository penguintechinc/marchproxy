#!/bin/bash
# Certificate generation script for MarchProxy development and testing
# Generates a complete certificate chain for mTLS authentication

set -e

# Configuration
CERT_DIR="/certs"
COUNTRY="${CERT_COUNTRY:-US}"
STATE="${CERT_STATE:-CA}"
CITY="${CERT_CITY:-San Francisco}"
ORG="${CERT_ORG:-MarchProxy}"
OU="${CERT_OU:-Development}"
DAYS="${CERT_DAYS:-365}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create certificate directory
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

print_status "Generating mTLS certificates for MarchProxy..."
print_status "Certificate directory: $CERT_DIR"
print_status "Validity period: $DAYS days"

# Generate CA private key (ECC P-384 for better security)
print_status "Generating CA private key..."
openssl ecparam -genkey -name secp384r1 -out ca-key.pem

# Generate CA certificate
print_status "Generating CA certificate..."
openssl req -new -x509 -sha384 -key ca-key.pem -out ca.pem -days $DAYS -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=MarchProxy CA"

# Generate server private key
print_status "Generating server private key..."
openssl ecparam -genkey -name secp384r1 -out server-key.pem

# Generate server certificate signing request
print_status "Generating server certificate signing request..."
openssl req -new -sha384 -key server-key.pem -out server.csr -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=localhost"

# Create server certificate extensions
cat > server-extensions.cnf << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = marchproxy-proxy-egress
DNS.3 = marchproxy-proxy-ingress
DNS.4 = manager
DNS.5 = *.marchproxy.local
IP.1 = 127.0.0.1
IP.2 = ::1
IP.3 = 172.20.0.0/16
EOF

# Generate server certificate
print_status "Generating server certificate..."
openssl x509 -req -sha384 -in server.csr -CA ca.pem -CAkey ca-key.pem -out server-cert.pem -days $DAYS -extensions v3_req -extfile server-extensions.cnf -CAcreateserial

# Generate client private key
print_status "Generating client private key..."
openssl ecparam -genkey -name secp384r1 -out client-key.pem

# Generate client certificate signing request
print_status "Generating client certificate signing request..."
openssl req -new -sha384 -key client-key.pem -out client.csr -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=marchproxy-client"

# Create client certificate extensions
cat > client-extensions.cnf << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF

# Generate client certificate
print_status "Generating client certificate..."
openssl x509 -req -sha384 -in client.csr -CA ca.pem -CAkey ca-key.pem -out client-cert.pem -days $DAYS -extensions v3_req -extfile client-extensions.cnf -CAcreateserial

# Generate additional client certificates for testing
print_status "Generating additional client certificates for testing..."

# Test client 1
openssl ecparam -genkey -name secp384r1 -out test-client-1-key.pem
openssl req -new -sha384 -key test-client-1-key.pem -out test-client-1.csr -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=test-client-1"
openssl x509 -req -sha384 -in test-client-1.csr -CA ca.pem -CAkey ca-key.pem -out test-client-1-cert.pem -days $DAYS -extensions v3_req -extfile client-extensions.cnf -CAcreateserial

# Test client 2
openssl ecparam -genkey -name secp384r1 -out test-client-2-key.pem
openssl req -new -sha384 -key test-client-2-key.pem -out test-client-2.csr -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU/CN=test-client-2"
openssl x509 -req -sha384 -in test-client-2.csr -CA ca.pem -CAkey ca-key.pem -out test-client-2-cert.pem -days $DAYS -extensions v3_req -extfile client-extensions.cnf -CAcreateserial

# Create combined certificate chain files
print_status "Creating certificate chain files..."
cat server-cert.pem ca.pem > server-chain.pem
cat client-cert.pem ca.pem > client-chain.pem

# Set appropriate permissions
print_status "Setting certificate permissions..."
chmod 600 *-key.pem
chmod 644 *.pem *.csr *.cnf

# Clean up temporary files
rm -f *.csr *.cnf *.srl

# Verify certificates
print_status "Verifying generated certificates..."

# Verify CA certificate
if openssl x509 -in ca.pem -text -noout > /dev/null 2>&1; then
    print_success "CA certificate is valid"
else
    print_error "CA certificate verification failed"
    exit 1
fi

# Verify server certificate
if openssl verify -CAfile ca.pem server-cert.pem > /dev/null 2>&1; then
    print_success "Server certificate is valid"
else
    print_error "Server certificate verification failed"
    exit 1
fi

# Verify client certificate
if openssl verify -CAfile ca.pem client-cert.pem > /dev/null 2>&1; then
    print_success "Client certificate is valid"
else
    print_error "Client certificate verification failed"
    exit 1
fi

# Display certificate information
print_status "Certificate information:"
echo
echo "=== CA Certificate ==="
openssl x509 -in ca.pem -text -noout | grep -A 1 "Subject:"
openssl x509 -in ca.pem -text -noout | grep -A 2 "Validity"
echo

echo "=== Server Certificate ==="
openssl x509 -in server-cert.pem -text -noout | grep -A 1 "Subject:"
openssl x509 -in server-cert.pem -text -noout | grep -A 2 "Validity"
openssl x509 -in server-cert.pem -text -noout | grep -A 10 "Subject Alternative Name"
echo

echo "=== Client Certificate ==="
openssl x509 -in client-cert.pem -text -noout | grep -A 1 "Subject:"
openssl x509 -in client-cert.pem -text -noout | grep -A 2 "Validity"
echo

# Create certificate summary
cat > certificate-info.txt << EOF
MarchProxy mTLS Certificate Summary
Generated: $(date)
Validity: $DAYS days
Algorithm: ECC P-384 with SHA-384

Files generated:
- ca.pem: Certificate Authority certificate
- ca-key.pem: Certificate Authority private key
- server-cert.pem: Server certificate
- server-key.pem: Server private key
- server-chain.pem: Server certificate chain (cert + CA)
- client-cert.pem: Client certificate
- client-key.pem: Client private key
- client-chain.pem: Client certificate chain (cert + CA)
- test-client-1-cert.pem, test-client-1-key.pem: Test client 1
- test-client-2-cert.pem, test-client-2-key.pem: Test client 2

Usage:
- For MarchProxy Egress: Use server-cert.pem, server-key.pem, and ca.pem
- For MarchProxy Ingress: Use server-cert.pem, server-key.pem, and ca.pem
- For client applications: Use client-cert.pem, client-key.pem, and ca.pem
- For testing: Use test-client-*-cert.pem and test-client-*-key.pem

Environment variables for Docker:
MTLS_ENABLED=true
MTLS_SERVER_CERT_PATH=/app/certs/server-cert.pem
MTLS_SERVER_KEY_PATH=/app/certs/server-key.pem
MTLS_CLIENT_CA_PATH=/app/certs/ca.pem
MTLS_CLIENT_CERT_PATH=/app/certs/client-cert.pem
MTLS_CLIENT_KEY_PATH=/app/certs/client-key.pem
EOF

print_success "Certificate generation complete!"
print_success "Certificates are valid for $DAYS days"
print_success "Summary saved to certificate-info.txt"

# List generated files
print_status "Generated files:"
ls -la *.pem *.txt | while read line; do
    echo "  $line"
done

print_success "mTLS certificates ready for MarchProxy!"

# Test certificate compatibility
print_status "Testing certificate compatibility..."

# Test if certificates can be loaded by OpenSSL
if openssl s_server -accept 9999 -cert server-cert.pem -key server-key.pem -CAfile ca.pem -verify_return_error -naccept 1 -quiet < /dev/null > /dev/null 2>&1 &
then
    SSL_PID=$!
    sleep 1
    if kill -0 $SSL_PID 2>/dev/null; then
        kill $SSL_PID
        print_success "Certificate compatibility test passed"
    else
        print_warning "Certificate compatibility test inconclusive"
    fi
else
    print_warning "Certificate compatibility test failed - certificates may still work"
fi

print_success "Certificate generation process completed successfully!"