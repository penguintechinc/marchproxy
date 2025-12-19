# MarchProxy Development Standards

This document consolidates all development standards, patterns, and requirements for MarchProxy following the Penguin Tech Inc gold standard template while incorporating MarchProxy-specific requirements.

**Aligned with**: `/home/penguin/code/project-template/docs/STANDARDS.md`

## Table of Contents

1. [Language Selection](#language-selection)
2. [py4web Framework Integration](#py4web-framework-integration)
3. [Go Development Standards](#go-development-standards)
4. [ReactJS Frontend Standards](#reactjs-frontend-standards)
5. [Database Standards](#database-standards)
6. [API Versioning](#api-versioning)
7. [Performance Optimization](#performance-optimization)
8. [eBPF and Hardware Acceleration](#ebpf-and-hardware-acceleration)
9. [Microservices Architecture](#microservices-architecture)
10. [Docker Standards](#docker-standards)
11. [Testing Requirements](#testing-requirements)
12. [Security Standards](#security-standards)
13. [CI/CD Standards](#cicd-standards)

---

## Language Selection

### Python 3.12+ (Manager Service)

**Use Python for the management server:**
- py4web framework for rapid development
- PyDAL ORM for database abstraction
- Business logic and service management
- Configuration and licensing handling
- REST API endpoints
- Authentication and authorization

**Advantages:**
- Rapid development and iteration
- Rich ecosystem of libraries
- Excellent for data processing
- Strong py4web and PyDAL integration
- Easy maintenance and debugging

### Go 1.23.x or 1.24.x (Proxy Services)

**Use Go for high-performance proxy services:**
- eBPF kernel-level packet filtering
- High-throughput traffic processing (<10ms latency)
- Network-intensive operations (>10K packets/sec)
- Low-latency packet forwarding
- Optional hardware acceleration (DPDK, XDP, AF_XDP, SR-IOV)
- Performance-critical operations

**Advantages:**
- Exceptional performance for networking
- eBPF integration for kernel operations
- Minimal memory footprint
- Excellent concurrency with goroutines
- Direct system-level control

---

## py4web Framework Integration

**MANDATORY for ALL Manager (Python) services**

### Core Features

- User authentication and session management
- Role-based access control (RBAC)
- Native API key system
- Built-in form validation and CSRF protection
- ORM integration with PyDAL
- RESTful API support

### py4web Documentation Research

- **Official Documentation**: https://py4web.com/_documentation
- **Always research py4web native features before implementing custom solutions**
- **Leverage py4web's built-in authentication, user management, and API systems**
- **Use py4web's native decorators, validators, and utilities wherever possible**
- **Priority order: py4web native → PyDAL features → custom implementation**

### Input Validation Standards

- **ALL fields and inputs MUST have appropriate validators**
- **Use PyDAL's built-in validators (IS_EMAIL, IS_STRONG, IS_IN_SET, etc.)**
- **Implement input sanitization for XSS prevention**
- **Validate data types, lengths, and formats at database and API levels**
- **Use py4web's native form validation**
- **Never trust client-side input - always validate server-side**
- **Implement CSRF protection using py4web's native features**

### Linting & Code Quality Requirements

- **Python**: flake8, black, isort, mypy (type checking), bandit (security)
- **Package Manager**: pip with requirements.txt
- **Type Checking**: mypy with type hints
- **PEP Compliance**: Python code must follow PEP 8, PEP 257, PEP 484

---

## Go Development Standards

### Project Structure

```
services/proxy-egress/
├── cmd/
│   └── proxy/
│       └── main.go          # Application entry point
├── pkg/
│   ├── ebpf/               # eBPF programs
│   ├── packet/             # Packet processing
│   ├── registry/           # Manager registration
│   └── metrics/            # Prometheus metrics
├── internal/
│   ├── config/             # Configuration handling
│   ├── proxy/              # Proxy logic
│   ├── health/             # Health checks
│   └── logger/             # Logging
├── go.mod
├── go.sum
├── Dockerfile
└── tests/
```

### Linting & Code Quality Requirements

- **Go**: golangci-lint (includes staticcheck, gosec, etc.)
- **gosec**: Security scanning (mandatory before commits)
- **go fmt**: Standard Go formatter
- **go vet**: Built-in Go analysis

### Concurrency Patterns

**Use goroutines for high-performance operations:**
- Process multiple packets concurrently
- Use sync.Pool for buffer reuse
- Implement connection pooling
- Proper error handling and cancellation

---

## ReactJS Frontend Standards

### Project Structure

```
services/webui/
├── src/
│   ├── components/         # Reusable components
│   ├── pages/              # Page components
│   ├── services/           # API client services
│   ├── hooks/              # Custom React hooks
│   ├── context/            # React context providers
│   ├── utils/              # Utility functions
│   └── App.jsx
├── package.json
├── Dockerfile
└── .env
```

### Color Theme (Dark Mode with Gold)

**CSS Variables (Required):**

```css
:root {
  /* Background colors */
  --bg-primary: #0f172a;      /* slate-900 */
  --bg-secondary: #1e293b;    /* slate-800 */
  --bg-tertiary: #334155;     /* slate-700 */

  /* Text colors - Gold default */
  --text-primary: #fbbf24;    /* amber-400 */
  --text-secondary: #f59e0b;  /* amber-500 */

  /* Interactive elements - Sky blue */
  --primary-500: #0ea5e9;
  --primary-600: #0284c7;
}
```

### Linting & Code Quality Requirements

- **JavaScript/TypeScript**: ESLint, Prettier, TypeScript
- **ESLint Configuration**: All projects MUST include `.eslintrc` configuration
- **TypeScript**: Strict mode enabled
- **80%+ test coverage** required

---

## Database Standards

### Hybrid Approach: SQLAlchemy + PyDAL

**SQLAlchemy for initialization:**
- Schema creation and migrations
- Initial table setup

**PyDAL for day-to-day operations:**
- All CRUD operations
- Queries and data retrieval
- Migrations and schema changes
- Multi-database support

### Supported Databases

- **PostgreSQL** (default and recommended)
- **MySQL/MariaDB** (including Galera clusters for Enterprise)
- **SQLite** (development and testing only)

### Environment Variables

```bash
DB_TYPE=postgres            # postgres, mysql, sqlite
DB_HOST=localhost
DB_PORT=5432
DB_NAME=marchproxy
DB_USER=app_user
DB_PASS=app_pass
DB_POOL_SIZE=10
```

### Thread Safety

**Use thread-local connections:**
- Each thread gets its own DAL instance
- Connection pooling handles multi-threaded access
- Flask/WSGI applications use request context

---

## API Versioning

**ALL REST APIs MUST use versioning in URL path:**

### Required Format

```
/api/v1/clusters
/api/v2/analytics
```

**Key Rules:**
1. Always include version prefix in URL path
2. Semantic versioning for API versions: v1, v2, v3, etc.
3. Major version only in URL - minor/patch versions NOT in URL
4. Support current and previous versions (N-1) minimum
5. Add deprecation headers to old versions

### Version Lifecycle

- **Current Version**: Active development and fully supported
- **Previous Version (N-1)**: Supported with bug fixes and security patches
- **Older Versions (N-2+)**: Deprecated with deprecation warning headers

---

## Performance Optimization

### Python Performance (Manager)

**Dataclasses with slots:**
- 30-50% memory reduction per instance
- Faster attribute access

**Type hints (mandatory):**
- Required for all Python code
- Enables static type checking

**Async/await for I/O operations:**
- Database queries and connections
- HTTP/REST API calls
- File I/O operations

### Go Performance (Proxy)

**Goroutines for packet processing:**
- Concurrent processing of multiple packets
- sync.Pool for buffer reuse
- Connection pooling

**Memory optimization:**
- Reuse buffers to reduce GC pressure
- Pre-allocate slices with known capacity

---

## eBPF and Hardware Acceleration

### eBPF Fast Path

eBPF programs execute at kernel level for ultra-low latency packet processing:
- Programmable kernel-level packet filtering
- Rule-based packet forwarding decisions
- Stateful connection tracking
- Ultra-low latency (<1ms for lookups)

### Hardware Acceleration Options

**Optional acceleration methods (in priority order):**

1. **DPDK**: Kernel bypass for maximum throughput
   - For extreme traffic (>1M packets/sec)
   - Requires dedicated CPU cores
   - Significant complexity

2. **XDP**: Driver-level packet processing
   - Easier than DPDK
   - Good performance improvement

3. **AF_XDP**: Zero-copy user-space sockets
   - Balance between performance and flexibility

4. **SR-IOV**: Virtualization-aware packet handling
   - For virtualized deployments

---

## Microservices Architecture

### Three-Container Model

| Container | Technology | Purpose |
|-----------|-----------|---------|
| **manager** | Python/py4web | Configuration, licensing, clustering |
| **proxy-egress** | Go/eBPF | Egress traffic proxy |
| **webui** | Node.js/React | Management UI |

### Container Communication

**Internal Docker networking:**
- Services communicate via internal hostnames
- manager: `http://manager:8000`
- proxy: `http://manager:8000/api/v1/register`
- webui: `http://manager:8000/api/v1`

---

## Docker Standards

### Base Images

**ALWAYS use Debian variants (NO Alpine):**

```dockerfile
# Go service
FROM golang:1.24-bookworm AS builder
FROM debian:bookworm-slim

# Python service
FROM python:3.12-slim

# Node service
FROM node:20-bookworm-slim
```

**Why not Alpine:**
- musl libc compatibility issues
- Missing dependencies for many packages
- Performance inconsistency

### Multi-Stage Builds

Use multi-stage Dockerfile to:
- Reduce final image size
- Separate build and runtime environments
- Exclude development dependencies

### Health Checks

**Use native language health checks (NOT curl/wget):**

```dockerfile
# Go service
HEALTHCHECK CMD ["proxy", "--healthcheck"]

# Python service
HEALTHCHECK CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/healthz')"

# Node service
HEALTHCHECK CMD node -e "const http = require('http'); http.get('http://localhost:3000/', (res) => process.exit(res.statusCode === 200 ? 0 : 1))"
```

---

## Testing Requirements

### Unit Tests

**Mandatory:**
- 80%+ code coverage
- No external dependencies
- Mocked I/O operations
- Fast execution (<100ms per test)

### Integration Tests

- Component interactions
- API endpoint testing
- Database operations
- External service mocking

### E2E Tests

- Critical workflows
- User journeys
- Production-like scenarios

### Performance Tests

- Benchmark critical operations
- Load testing for scalability
- Regression testing

---

## Security Standards

### Input Validation

**MANDATORY for all APIs:**
- Validate all external inputs
- Use framework-native validation
- Implement XSS and SQL injection prevention
- Server-side validation for all client input

### Authentication & Authorization

**Role-based access control:**
- Administrator role
- Service-owner role
- Cluster-specific assignments

**Authentication methods:**
- py4web native authentication
- API key management via py4web native system
- JWT for stateless API calls
- Base64 tokens as alternative

### Encryption

**Use AES-256-GCM for sensitive data:**
- API keys encryption
- TLS 1.2 minimum (prefer TLS 1.3)
- Certificate validation required
- HSTS headers enabled

### Dependency Security

- **ALWAYS check for Dependabot alerts** before commits
- **Monitor vulnerabilities** via Socket.dev
- **Mandatory security scanning** before dependency changes
- **Fix all security alerts immediately**
- **Regular security audits**: `npm audit`, `go mod audit`, `safety check`

---

## CI/CD Standards

### Build Pipeline Stages

1. **Lint Stage** (fail fast)
   - Python: flake8, black, mypy
   - Go: golangci-lint, gosec
   - Node: ESLint, Prettier

2. **Test Stage** (unit + integration)
   - 80%+ coverage required
   - Must pass before proceeding

3. **Build Stage** (multi-arch)
   - AMD64 and ARM64
   - Multi-stage Docker builds

4. **Security Scan Stage**
   - gosec, bandit, Trivy
   - CodeQL analysis
   - Fail on HIGH/CRITICAL

### Image Tagging

```
# Development builds
manager:beta-1702000000      (main branch)
manager:alpha-1702000000     (feature branch)

# Version releases
manager:v1.2.3-beta          (main branch)
manager:v1.2.3-alpha         (feature branch)

# Production
manager:v1.2.3               (git tag)
manager:latest               (latest release)
```

### Version Management

Update `.version` for releases:

```bash
./scripts/update-version.sh patch    # Patch release
./scripts/update-version.sh minor    # Minor release
./scripts/update-version.sh major    # Major release
```

**Path filters in workflows:**
- ALL workflows must include `.version` in path filters
- Ensures all services version-lock together
- Prevents partial version releases

---

## Quality Checklist Before Task Completion

- All error cases handled properly
- Unit tests cover all code paths (80%+ coverage)
- Integration tests verify component interactions
- Security requirements fully implemented
- Performance meets acceptable standards
- Documentation complete and accurate
- Code review standards met
- No hardcoded secrets or credentials
- Logging and monitoring in place
- Build passes in containerized environment
- No security vulnerabilities in dependencies
- Edge cases and boundary conditions tested

---

**Standards Version**: 1.0 for MarchProxy
**Last Updated**: 2025-12-18
**Maintained by**: Penguin Tech Inc
**Reference Template**: `/home/penguin/code/project-template/docs/STANDARDS.md`
