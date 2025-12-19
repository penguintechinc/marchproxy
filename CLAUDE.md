# MarchProxy - Claude Code Context

## Project Overview

MarchProxy is a two-container application suite for managing egress traffic in a data center environment to the internet. Available in two tiers: Community (open source) and Enterprise (licensed). It provides comprehensive proxy management with performance optimization through eBPF and optional hardware acceleration.

**Product Overview:**
- High-performance egress traffic proxy with dual-tier licensing model
- Community tier: Free, maximum 3 proxy servers, single cluster, basic auth
- Enterprise tier: Licensed via license.penguintech.io, unlimited proxies, multi-cluster support, SAML/SCIM/OAuth2

**Architecture Footprint:**
- **Manager Container**: Python/py4web management server with pydal ORM, PostgreSQL database
- **Proxy Container**: High-performance Go/eBPF proxy with optional hardware acceleration (DPDK, XDP, AF_XDP, SR-IOV)
- **WebUI Container**: Node.js/React frontend for management interface
- **Database**: PostgreSQL (default), configurable via `DB_TYPE` to MySQL or SQLite

## Technology Stack

### Primary Technologies

**Manager Backend:**
- Python 3.12+ with py4web framework
- PyDAL ORM for database abstraction
- Native API key system and authentication
- License integration with PenguinTech License Server

**Proxy Engine:**
- Go 1.23.x or 1.24.x
- eBPF (extended Berkeley Packet Filter) for kernel-level packet filtering
- Optional hardware acceleration: DPDK, XDP, AF_XDP, SR-IOV

**Frontend:**
- Node.js 20.x or 22.x LTS
- ReactJS for web interface
- Dark theme with gold (amber) text by default

### Database Architecture

**Hybrid Database Strategy:**
- **Initialization**: SQLAlchemy for schema creation
- **Operations**: PyDAL for queries and ORM
- **Supported**: PostgreSQL, MySQL, SQLite only
- **Configuration**: `DB_TYPE` environment variable

**MariaDB Galera Support (Enterprise):**
- Active-active multi-node replication
- BINLOG with row-based replication required
- InnoDB storage engine mandatory
- Cluster configuration for distributed deployments

### Supported Protocols

- TCP and UDP traffic management
- ICMP processing
- HTTPS with WebSocket support
- HTTP/3 with QUIC protocol
- Multi-cluster API communication

## Core Features

### Service Management
- Service-to-service mapping with cluster isolation
- Multi-cluster support with separate API keys (Enterprise tier)
- Proxy registration and lifecycle management
- Health check endpoints: `/healthz` for both manager and proxy
- Metrics endpoints: Prometheus `/metrics` with comprehensive statistics

### Authentication & Authorization
- Role-based access: Administrator and Service-owner roles
- Cluster-specific assignments
- Authentication methods:
  - Base64 tokens OR JWT (mutually exclusive)
  - 2FA, SAML, SCIM, OAuth2 (Enterprise only)
- License-gated enterprise features via PenguinTech License Server

### Network Configuration
- Port configuration: single ports, ranges, comma-separated lists
- TLS management: Infisical, HashiCorp Vault integration, or direct upload
- Protocol support: TCP, UDP, ICMP, HTTPS (WebSockets), HTTP/3 with QUIC
- Proxy bypass rules and traffic routing policies

### Logging and Observability
- Centralized logging: UDP syslog with per-cluster configuration
- Log types: Authentication, netflow, debug (configurable per cluster)
- Comprehensive metrics for performance monitoring
- Audit logging for compliance tracking

## Performance Architecture

Multi-tier packet processing for maximum performance:

1. **Hardware Acceleration** (Optional)
   - DPDK: Kernel bypass for maximum throughput
   - XDP: Driver-level packet processing
   - AF_XDP: Zero-copy socket for user-space processing
   - SR-IOV: Virtualization-aware packet handling

2. **eBPF Fast-path**
   - Programmable kernel-level packet filtering
   - Rule-based packet forwarding decisions
   - Stateful connection tracking

3. **Go Application Logic**
   - Complex rule processing and application features
   - Service discovery and routing
   - Connection management and load balancing

4. **Standard Networking**
   - Traditional kernel socket processing
   - Fallback for non-optimized traffic paths

## Project Structure

```
MarchProxy/
├── .github/                  # CI/CD pipelines
│   └── workflows/            # GitHub Actions for each service
├── services/                 # Microservices (separate containers)
│   ├── manager/              # py4web management server
│   ├── proxy-egress/         # Go high-performance egress proxy
│   ├── proxy-ingress/        # Go ingress proxy (optional)
│   └── webui/                # Node.js/React frontend
├── shared/                   # Shared libraries
│   ├── py_libs/              # Python shared library
│   ├── go_libs/              # Go shared library
│   └── node_libs/            # TypeScript shared library
├── k8s/                      # Kubernetes deployment templates
├── docs/                     # Comprehensive documentation
├── tests/                    # Test suites
├── docker-compose.yml        # Production environment
├── docker-compose.dev.yml    # Local development
├── .version                  # Version tracking (vX.Y.Z format)
├── CLAUDE.md                 # This file
└── README.md                 # Project overview
```

## Development Standards

Comprehensive development standards are documented separately to keep this file concise.

📚 **Complete Standards Documentation**: [Development Standards](docs/STANDARDS.md)

### Key Standards Quick Reference

**API Versioning:**
- ALL REST APIs use versioning: `/api/v{major}/endpoint` format
- Support current and previous versions (N-1) minimum
- Add deprecation headers to old versions

**Database Standards:**
- Hybrid approach: SQLAlchemy for init, PyDAL for day-to-day operations
- DB_TYPE environment variable: `postgres`, `mysql`, or `sqlite` only
- Thread-safe usage with thread-local connections
- Connection pooling and retry logic required

**Protocol Support:**
- REST API, gRPC, HTTP/1.1, HTTP/2, HTTP/3 support
- Environment variables for protocol configuration
- Multi-protocol implementation required

**Performance Optimization (Python):**
- Dataclasses with slots mandatory (30-50% memory reduction)
- Type hints required for all Python code
- asyncio for I/O-bound operations
- threading for blocking I/O
- multiprocessing for CPU-bound operations

**Docker Standards:**
- Multi-arch builds (amd64/arm64)
- Debian-slim base images (NO Alpine)
- Docker Compose for local development
- Minimal host port exposure

**Testing:**
- Unit tests: Network isolated, mocked dependencies (80%+ coverage)
- Integration tests: Component interactions
- E2E tests: Critical workflows
- Security scanning: gosec, bandit, Trivy, CodeQL

## MarchProxy-Specific Implementation Notes

### Manager Container (py4web)
- Uses py4web native authentication and API key system
- SQLAlchemy for database initialization/migrations
- PyDAL ORM for query operations
- `/healthz` health check endpoint required
- `/metrics` Prometheus metrics endpoint required
- Role-based access control (Administrator, Service-owner)
- License integration with PenguinTech License Server

### Proxy Containers (Go)
- eBPF-based kernel-level packet filtering
- Optional hardware acceleration (DPDK, XDP, AF_XDP, SR-IOV)
- Registration and heartbeat with manager
- Health check and metrics endpoints
- Stateful connection tracking
- Multi-protocol support (TCP, UDP, ICMP, HTTPS, QUIC)

### WebUI Container (React)
- Dark theme with gold (amber) text default
- Cluster management interface
- Service configuration dashboard
- Proxy monitoring and analytics
- Role-based navigation

### Key Environment Variables
- `CLUSTER_API_KEY`: Cluster-specific API key for proxy registration
- `DATABASE_URL`: Database connection string
- `DB_TYPE`: Database type (postgres, mysql, sqlite)
- `LICENSE_KEY`: Enterprise license key (format: PENG-XXXX-XXXX-XXXX-XXXX-ABCD)
- `RELEASE_MODE`: License enforcement mode (false=dev, true=prod)

### Docker Image Standards

**Base Images:**
- ALWAYS use Debian variants (no Alpine)
- Go: `golang:1.24-bookworm` or `golang:1.23-bookworm`
- Python: `python:3.12-slim` or `python:3.12-bookworm`
- Node: `node:20-bookworm-slim` or `node:22-bookworm-slim`
- Runtime: `debian:bookworm-slim`

**Health Checks:**
- Use native language health checks (not curl/wget)
- Go: Implement `--healthcheck` flag in application
- Python: Use urllib.request for HTTP checks
- Node: Use native http module for checks

### py4web Framework Integration
- Official docs: https://py4web.com/_documentation
- Use py4web native features for:
  - Authentication and user management
  - API key generation and validation
  - Form validation and CSRF protection
  - Built-in utilities and decorators
- Priority: py4web native > PyDAL features > custom code

### Input Validation Standards
- ALL fields and inputs MUST have appropriate validators
- Use PyDAL's built-in validators (IS_EMAIL, IS_STRONG, IS_IN_SET, etc.)
- Implement input sanitization for XSS prevention
- Validate data types, lengths, and formats at database and API levels
- Use py4web's native form validation
- Never trust client-side input - always validate server-side
- Implement CSRF protection using py4web's native features

### Development Philosophy: Safe, Stable, and Feature-Complete

**NEVER take shortcuts or the "easy route" - ALWAYS prioritize safety, stability, and feature completeness**

**Core Principles:**
- No Quick Fixes: Resist quick workarounds or partial solutions
- Complete Features: Fully implemented with proper error handling and validation
- Safety First: Security, data integrity, and fault tolerance are non-negotiable
- Stable Foundations: Build on solid, tested components
- Future-Proof Design: Consider long-term maintainability and scalability
- No Technical Debt: Address issues properly the first time

**Red Flags (Never Do These):**
- Skipping input validation "just this once"
- Hardcoding credentials or configuration
- Ignoring error returns or exceptions
- Commenting out failing tests to make CI pass
- Deploying without proper testing
- Using deprecated or unmaintained dependencies
- Implementing partial features with "TODO" placeholders
- Bypassing security checks for convenience
- Assuming data is valid without verification
- Leaving debug code or backdoors in production

**Quality Checklist Before Completion:**
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

### Git & CI/CD Workflow
- **NEVER commit automatically** unless explicitly requested by the user
- **NEVER push to remote repositories** under any circumstances
- **ONLY commit when explicitly asked** - never assume commit permission
- Always use feature branches for development
- Require pull request reviews for main branch
- Automated testing must pass before merge

**Before Every Commit - Security Scanning:**
- Run security audits on all modified packages:
  - **Go packages**: Run `gosec ./...` on modified services
  - **Node.js packages**: Run `npm audit` on modified services
  - **Python packages**: Run `bandit -r .` and `safety check` on modified services
- Do NOT commit if security vulnerabilities are found
- Document vulnerability fixes in commit message

**Before Every Commit - API Testing:**
- Create and run API testing scripts for each modified container service
- Test all new endpoints and modified functionality
- Test files location: `tests/api/` directory with service-specific subdirectories
- Run before commit: Each test script should pass completely
- Test coverage: Health checks, authentication, CRUD operations, error cases

### Linting & Code Quality Requirements
- **ALL code must pass linting** before commit - no exceptions
- **Python**: flake8, black, isort, mypy (type checking), bandit (security)
- **Go**: golangci-lint (includes staticcheck, gosec, etc.)
- **JavaScript/TypeScript**: ESLint, Prettier, TypeScript
- **Docker**: hadolint, trivy
- **YAML**: yamllint
- **Shell**: shellcheck
- **CodeQL**: All code must pass CodeQL security analysis
- **PEP Compliance**: Python code must follow PEP 8, PEP 257, PEP 484

### Build & Deployment Requirements
- **NEVER mark tasks as completed until successful build verification**
- All Go and Python builds MUST be executed within Docker containers
- Use containerized builds for local development and CI/CD pipelines
- Build failures must be resolved before task completion
- Critical rule: **NOTHING IS CONSIDERED COMPLETE UNTIL IT HAS A SUCCESSFUL TEST BUILD**

### Documentation Standards
- **README.md**: Project overview and pointer to comprehensive docs/ folder
- **docs/ folder**: Create comprehensive documentation for all aspects
- **CLAUDE.md**: High-level context (max 39,000 characters)
- **docs/STANDARDS.md**: Development standards and best practices
- **docs/WORKFLOWS.md**: CI/CD workflow documentation
- **Build status badges**: Always include in README.md
- **ASCII art**: Include catchy project-appropriate ASCII art in README
- **Company homepage**: Point to www.penguintech.io
- **License**: All projects use Limited AGPL3 with fair use preamble

## Licensing Strategy

**IMPORTANT: License enforcement is ONLY enabled when project is marked as release-ready**
- Development phase: All features available, no license checks
- Release phase: License validation required, feature gating active

**License-Gated Enterprise Features:**
- Unlimited proxy servers (Community limited to 3)
- Multi-cluster support with isolated API keys per cluster
- SAML/SCIM/OAuth2 authentication (Community uses basic auth)
- Advanced analytics and reporting
- Priority support and SLA

**Environment Variables:**
```bash
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
LICENSE_SERVER_URL=https://license.penguintech.io
PRODUCT_NAME=marchproxy
RELEASE_MODE=false  # Development (default)
RELEASE_MODE=true   # Production (explicitly set)
```

## Version Management

**Format**: `vMajor.Minor.Patch.build`
- **Major**: Breaking API changes, removed features
- **Minor**: Significant new features and functionality
- **Patch**: Bug fixes, security patches, minor updates
- **Build**: Epoch64 timestamp of build time

**Update Commands**:
```bash
./scripts/update-version.sh          # Update build timestamp only
./scripts/update-version.sh patch    # Increment patch version
./scripts/update-version.sh minor    # Increment minor version
./scripts/update-version.sh major    # Increment major version
```

## Shared Security Libraries

**ALL applications MUST use shared libraries** for input validation, security, and cryptographic operations.

### Library Overview

| Library | Package | Install Command |
|---------|---------|-----------------|
| **Python** | `py_libs` | `pip install -e "shared/py_libs[all]"` |
| **Go** | `go_libs` | `go get github.com/penguintechinc/go_libs` |
| **TypeScript** | `@penguin/node_libs` | `npm install file:shared/node_libs` |

### Required Usage

**Input Validation - MANDATORY for all user input:**
```python
from py_libs.validation import chain, NotEmpty, Email, Length

email_validator = chain(NotEmpty(), Length(3, 255), Email())
result = email_validator(user_input)
if not result.is_valid:
    return {"error": result.error}, 400
```

**Security Middleware - MANDATORY for all HTTP endpoints:**
- Rate limiting (in-memory + Redis backends)
- Secure headers (CSP, HSTS, X-Frame-Options)
- CSRF protection
- Audit logging

**Cryptographic Operations - MANDATORY for sensitive data:**
- Password hashing: Argon2id (Python/Node.js), bcrypt (Go)
- Encryption: AES-256-GCM
- Token generation: Cryptographically secure random

📚 **Detailed Documentation**: See [Development Standards](docs/STANDARDS.md)

## Development Workflow

### Project State Management
- **`.PLAN` file**: Detailed development plan and architectural guidance (git ignored)
- **`.TODO` file**: Comprehensive task list and completion tracking (git ignored)
- **`.version` file**: Current semantic version for all services

### Phases and Tasks
- Track progress systematically in `.TODO` file
- Reference `.PLAN` for architectural decisions
- Update CLAUDE.md when adding significant context
- Maintain crash recovery files locally

### Notes for Claude
- Always refer to `.PLAN` and `.TODO` for current project state
- Focus on performance, security, and scalability
- Maintain stateless proxy design for horizontal scaling
- Prioritize multi-tier performance: Hardware acceleration → eBPF → Go → Standard
- Consider Community (3 proxies) vs Enterprise (unlimited) licensing
- Use py4web native functions where possible
- Implement proper cluster isolation
- Follow comprehensive documentation standards

## CI/CD Pipeline

Complete CI/CD documentation available in:
- **docs/WORKFLOWS.md** - Workflow reference and troubleshooting
- **docs/STANDARDS.md** - Development standards and CI/CD section

**Three Main Services:**
1. **Manager** (Python/py4web)
2. **Proxy Egress** (Go with eBPF)
3. **Proxy Ingress** (Go optional)

**Build Pipeline:**
1. Lint Stage (fail fast on quality issues)
2. Test Stage (unit tests, 80%+ coverage)
3. Build Stage (multi-arch Docker builds)
4. Security Scan Stage (gosec, bandit, Trivy, CodeQL)

**Automatic Versioning:**
- Development: `alpha-<epoch64>` (feature), `beta-<epoch64>` (main)
- Version Release: `v<X.Y.Z>-alpha/beta` when .version changes
- Production: `v<X.Y.Z>` and `latest` on git tags

---

**Document Version**: 1.0 for MarchProxy
**Last Updated**: 2025-12-18
**Maintained by**: Penguin Tech Inc
**License**: Limited AGPL3 with fair use preamble
**License Server**: https://license.penguintech.io