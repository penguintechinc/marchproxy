# MarchProxy Testing Guide

This guide covers testing the dual proxy architecture (proxy-egress and proxy-ingress) with mTLS authentication.

## Prerequisites

- Docker and Docker Compose installed
- OpenSSL for certificate generation and testing
- curl for HTTP/HTTPS testing
- Optional: Apache Bench (ab) for load testing

## Quick Start

1. **Generate certificates:**
   ```bash
   docker-compose --profile tools run --rm cert-generator
   ```

2. **Start services:**
   ```bash
   docker-compose up -d
   ```

3. **Run comprehensive tests:**
   ```bash
   ./scripts/test-proxies.sh
   ```

4. **Run mTLS-specific tests:**
   ```bash
   ./scripts/test-mtls.sh
   ```

## Test Scripts

### 1. Certificate Generation (`scripts/generate-certs.sh`)

Generates a complete mTLS certificate chain:
- CA certificate and private key
- Server certificate for both proxies
- Client certificates for testing
- Test client certificates for validation scenarios

**Usage:**
```bash
# Via Docker (recommended)
docker-compose --profile tools run --rm cert-generator

# Direct execution (requires OpenSSL)
./scripts/generate-certs.sh
```

**Generated files:**
- `certs/ca.pem` - Certificate Authority
- `certs/server-cert.pem` - Server certificate
- `certs/server-key.pem` - Server private key
- `certs/client-cert.pem` - Client certificate
- `certs/client-key.pem` - Client private key
- `certs/test-client-*` - Additional test certificates

### 2. Comprehensive Testing (`scripts/test-proxies.sh`)

Tests the complete MarchProxy dual proxy system:
- Certificate validation
- Docker service startup
- Manager API functionality
- Proxy-egress functionality
- Proxy-ingress functionality
- mTLS communication
- Inter-service integration
- Monitoring stack
- Basic performance

**Usage:**
```bash
# Full test suite
./scripts/test-proxies.sh

# Quick test (skip monitoring and performance)
./scripts/test-proxies.sh --quick

# Custom certificate directory
./scripts/test-proxies.sh --cert-dir /path/to/certs

# Cleanup volumes after testing
./scripts/test-proxies.sh --cleanup-volumes
```

### 3. mTLS Testing (`scripts/test-mtls.sh`)

Focused testing of mTLS functionality:
- Certificate chain validation
- TLS connection testing
- mTLS authentication
- Client certificate validation
- Certificate rejection scenarios
- Cipher suite verification

**Usage:**
```bash
# Test with default settings
./scripts/test-mtls.sh

# Custom endpoints
./scripts/test-mtls.sh --ingress-host proxy.example.com --ingress-port 8443
```

## Service Endpoints

### Manager
- Main API: http://localhost:8000
- Health: http://localhost:8000/healthz
- Metrics: http://localhost:8000/metrics
- License Status: http://localhost:8000/license-status

### Proxy Egress
- Admin/Health: http://localhost:8081/healthz
- Metrics: http://localhost:8081/metrics
- Stats: http://localhost:8081/stats

### Proxy Ingress
- HTTP: http://localhost:80
- HTTPS: https://localhost:443
- Admin/Health: http://localhost:8082/healthz
- Metrics: http://localhost:8082/metrics

### Monitoring
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin123)
- Kibana: http://localhost:5601

## Manual Testing

### 1. Basic Connectivity

```bash
# Manager health
curl http://localhost:8000/healthz

# Egress proxy health
curl http://localhost:8081/healthz

# Ingress proxy health
curl http://localhost:8082/healthz
```

### 2. mTLS Testing

```bash
# Test ingress HTTPS with client certificate
curl --cert certs/client-cert.pem \
     --key certs/client-key.pem \
     --cacert certs/ca.pem \
     -k https://localhost:443/

# Test without client certificate (should fail if mTLS required)
curl -k https://localhost:443/
```

### 3. Metrics Collection

```bash
# View egress proxy metrics
curl http://localhost:8081/metrics | grep marchproxy

# View ingress proxy metrics
curl http://localhost:8082/metrics | grep marchproxy_ingress

# View manager metrics
curl http://localhost:8000/metrics | grep marchproxy
```

### 4. Certificate Validation

```bash
# Verify certificate chain
openssl verify -CAfile certs/ca.pem certs/server-cert.pem
openssl verify -CAfile certs/ca.pem certs/client-cert.pem

# View certificate details
openssl x509 -in certs/server-cert.pem -text -noout
openssl x509 -in certs/client-cert.pem -text -noout
```

## Troubleshooting

### Common Issues

1. **Certificate errors:**
   - Regenerate certificates with `docker-compose --profile tools run --rm cert-generator`
   - Check certificate permissions (keys should be 600)
   - Verify certificate paths in docker-compose.yml

2. **Service startup failures:**
   - Check logs: `docker-compose logs [service-name]`
   - Verify environment variables in docker-compose.yml
   - Ensure required dependencies are running

3. **mTLS connection failures:**
   - Verify certificate chain with `openssl verify`
   - Check TLS configuration in proxy settings
   - Test with `openssl s_client` for detailed TLS debugging

4. **Proxy registration failures:**
   - Check manager connectivity
   - Verify API keys match
   - Check license limits

### Debugging Commands

```bash
# View service logs
docker-compose logs manager
docker-compose logs proxy-egress
docker-compose logs proxy-ingress

# Check service status
docker-compose ps

# Debug TLS connection
openssl s_client -connect localhost:443 \
  -cert certs/client-cert.pem \
  -key certs/client-key.pem \
  -CAfile certs/ca.pem \
  -verify_return_error

# Test certificate loading
openssl s_server -accept 9999 \
  -cert certs/server-cert.pem \
  -key certs/server-key.pem \
  -CAfile certs/ca.pem \
  -verify_return_error
```

## Performance Testing

### Load Testing with Apache Bench

```bash
# Test ingress proxy HTTP
ab -n 1000 -c 10 http://localhost:80/

# Test ingress proxy admin endpoint
ab -n 1000 -c 10 http://localhost:8082/healthz

# Test egress proxy admin endpoint
ab -n 1000 -c 10 http://localhost:8081/healthz
```

### Load Testing with curl

```bash
# Concurrent requests to ingress
for i in {1..100}; do
  curl -s http://localhost:80/ >/dev/null &
done
wait

# Monitor metrics during load
watch -n 1 'curl -s http://localhost:8082/metrics | grep active_connections'
```

## Environment Variables

### Development Override

The `docker-compose.override.yml` file sets development-friendly defaults:

```yaml
environment:
  - MTLS_ENABLED=true
  - LOG_LEVEL=DEBUG
  - ENABLE_PROFILING=true
```

### Production Configuration

For production, override these variables:

```bash
export MTLS_ENABLED=true
export MTLS_REQUIRE_CLIENT_CERT=true
export MTLS_VERIFY_CLIENT_CERT=true
export LOG_LEVEL=INFO
export RATE_LIMIT_ENABLED=true
```

## Continuous Integration

For CI/CD pipelines:

```bash
# Quick validation
./scripts/test-proxies.sh --quick

# Full test with cleanup
./scripts/test-proxies.sh --cleanup-volumes

# mTLS-only testing
./scripts/test-mtls.sh
```

## Security Considerations

1. **Certificate Management:**
   - Use strong ECC P-384 keys in production
   - Implement certificate rotation
   - Monitor certificate expiry

2. **mTLS Configuration:**
   - Require client certificates for sensitive operations
   - Implement certificate revocation checking
   - Use strong cipher suites only

3. **Network Security:**
   - Use TLS 1.2+ only
   - Disable weak cipher suites
   - Implement proper firewall rules

## Success Criteria

A successful test run should show:
- ✅ All certificates valid and properly chained
- ✅ Both proxies start and register with manager
- ✅ Health endpoints respond correctly
- ✅ Metrics are being collected
- ✅ mTLS authentication works
- ✅ Client certificate validation works
- ✅ Invalid certificates are rejected
- ✅ Strong cipher suites are used
- ✅ Integration between services works

## Support

If tests fail, please:
1. Check the troubleshooting section above
2. Review service logs with `docker-compose logs`
3. Verify your environment meets the prerequisites
4. Ensure certificates are properly generated