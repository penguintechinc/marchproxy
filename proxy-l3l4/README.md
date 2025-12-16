# MarchProxy L3/L4 Proxy with Zero-Trust Security

Enterprise-grade L3/L4 proxy with comprehensive zero-trust security features.

## Features

### Zero-Trust Security (Enterprise)

- **OPA Policy Enforcement**: Integrate with Open Policy Agent for flexible, policy-based access control
- **RBAC Evaluation**: Per-request role-based access control with caching
- **mTLS Verification**: Enhanced certificate validation with CRL and OCSP support
- **Certificate Rotation**: Automated certificate rotation with zero-downtime
- **Immutable Audit Logging**: SHA-256 chained audit logs with tamper detection
- **Compliance Reporting**: Generate SOC2, HIPAA, and PCI-DSS compliance reports

## Directory Structure

```
proxy-l3l4/
├── cmd/
│   └── proxy/
│       └── main.go              # Main entry point
├── internal/
│   └── zerotrust/               # Zero-trust security implementation
│       ├── policy_enforcer.go   # OPA integration
│       ├── opa_client.go        # OPA HTTP client
│       ├── rbac_evaluator.go    # RBAC evaluation
│       ├── mtls_verifier.go     # mTLS certificate verification
│       ├── cert_rotator.go      # Automated certificate rotation
│       ├── audit_logger.go      # Immutable audit logging
│       └── compliance_reporter.go # Compliance report generation
├── policies/                    # OPA Rego policies
│   ├── rbac.rego               # Role-based access control
│   ├── rate_limit.rego         # Rate limiting policies
│   └── compliance.rego         # Compliance validation
├── Dockerfile                  # Multi-stage Docker build
├── go.mod                     # Go module definition
└── README.md                  # This file
```

## Configuration

### Environment Variables

- `MANAGER_URL`: Manager API URL (default: `http://api-server:8000`)
- `CLUSTER_API_KEY`: Cluster API key for authentication
- `OPA_URL`: OPA server URL (default: `http://opa:8181`)
- `AUDIT_LOG_PATH`: Audit log file path (default: `/var/log/marchproxy/audit/audit.log`)
- `CERT_PATH`: Server certificate path (default: `/etc/marchproxy/certs/server.crt`)
- `KEY_PATH`: Server key path (default: `/etc/marchproxy/certs/server.key`)
- `ENABLE_ZERO_TRUST`: Enable zero-trust features (default: `true`)
- `BIND_ADDR`: Proxy bind address (default: `:8081`)
- `METRICS_ADDR`: Metrics/health bind address (default: `:8082`)

## Building

### Docker Build

```bash
# Production build
docker build --target production -t marchproxy/proxy-l3l4:latest .

# Development build
docker build --target development -t marchproxy/proxy-l3l4:dev .

# Testing
docker build --target testing -t marchproxy/proxy-l3l4:test .
```

### Local Build

```bash
go build -o proxy-l3l4 ./cmd/proxy/main.go
```

## Running

### Docker

```bash
docker run -d \
  --name proxy-l3l4 \
  -e CLUSTER_API_KEY=your-api-key \
  -e OPA_URL=http://opa:8181 \
  -v /path/to/certs:/etc/marchproxy/certs \
  -v /path/to/logs:/var/log/marchproxy/audit \
  -p 8081:8081 \
  -p 8082:8082 \
  marchproxy/proxy-l3l4:latest
```

### Local

```bash
./proxy-l3l4 \
  --manager-url http://localhost:8000 \
  --cluster-api-key your-api-key \
  --opa-url http://localhost:8181 \
  --enable-zero-trust true
```

## OPA Policies

### RBAC Policy

The RBAC policy (`policies/rbac.rego`) implements role-based access control:

```rego
package marchproxy.rbac

import rego.v1

default allow := false

allow if {
    input.user != ""
    user_roles := data.users[input.user].roles
    some role in user_roles
    role_permissions := data.roles[role].permissions
    required_permission := concat(":", [input.action, input.resource])
    required_permission in role_permissions
}
```

### Rate Limiting Policy

The rate limiting policy (`policies/rate_limit.rego`) defines rate limits per service:

```rego
package marchproxy.rate_limit

import rego.v1

default_rate_limit := {
    "requests_per_second": 100,
    "requests_per_minute": 1000,
    "burst_size": 50,
}

rate_limit contains result if {
    input.service != ""
    service_config := data.rate_limits[input.service]
    service_config != null
    result := service_config
}
```

### Compliance Policy

The compliance policy (`policies/compliance.rego`) validates SOC2, HIPAA, and PCI-DSS requirements:

```rego
package marchproxy.compliance

import rego.v1

soc2_compliant if {
    authentication_required
    audit_trail_intact
    encryption_enabled
}
```

## API Endpoints

### Health Check

```
GET /healthz
```

Returns `200 OK` if the proxy is healthy.

### Metrics

```
GET /metrics
```

Returns Prometheus-formatted metrics.

### Zero-Trust Status

```
GET /zerotrust/status
```

Returns zero-trust feature status:

```json
{
  "enabled": true,
  "opa_connected": true,
  "audit_chain_valid": true,
  "cert_rotation_active": true
}
```

## Audit Logging

The audit logger creates immutable, SHA-256 chained logs:

```json
{
  "event": {
    "timestamp": "2025-12-12T15:30:00Z",
    "event_id": 1234,
    "event_type": "policy_evaluation",
    "service": "api-gateway",
    "user": "john.doe",
    "action": "read",
    "resource": "/api/users",
    "source_ip": "192.168.1.100",
    "allowed": true,
    "reason": "access granted",
    "policy_name": "rbac",
    "prev_hash": "0000...0000",
    "current_hash": "abcd...1234"
  },
  "hash": "abcd...1234"
}
```

## Compliance Reporting

Generate compliance reports for SOC2, HIPAA, or PCI-DSS:

```go
reporter := zerotrust.NewComplianceReporter(auditLogger, logger)

report, err := reporter.GenerateSOC2Report(startTime, endTime)
if err != nil {
    log.Fatal(err)
}

// Export to JSON
reporter.ExportReportJSON(report, "soc2-report.json")

// Export to HTML
reporter.ExportReportHTML(report, "soc2-report.html")
```

## License Gating

Zero-trust features require an Enterprise license. The proxy validates licenses with the license server:

```go
// Set license status
policyEnforcer.SetLicenseStatus(licenseValid)

// Check if enabled
if policyEnforcer.IsEnabled() {
    // Zero-trust features available
}
```

## Testing

### Unit Tests

```bash
go test -v -race ./...
```

### Integration Tests

```bash
docker-compose -f docker-compose.test.yml up --build
```

## Troubleshooting

### OPA Connection Issues

```bash
# Check OPA server is running
curl http://opa:8181/health

# Check OPA policies
curl http://opa:8181/v1/policies
```

### Audit Chain Integrity

```bash
# Verify audit chain
curl http://localhost:8082/api/v1/zerotrust/audit-logs/verify
```

### Certificate Rotation

```bash
# Check certificate expiry
curl http://localhost:8082/zerotrust/status | jq '.cert_rotation_active'
```

## Performance

- **OPA Policy Evaluation**: < 5ms p99
- **Audit Log Write**: < 1ms p99
- **RBAC Evaluation**: < 2ms p99 (with caching)
- **Certificate Verification**: < 10ms p99

## Security

- All audit logs are immutable and cryptographically chained
- Certificates are verified with CRL and OCSP
- Automatic certificate rotation prevents expiry
- Policy enforcement is fail-secure (deny by default)

## Contributing

See the main MarchProxy CONTRIBUTING.md for guidelines.

## License

See the main MarchProxy LICENSE file.
