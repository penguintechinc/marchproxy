# AILB Release Notes

## v0.1.0 - Initial Release (2024-01-20)

### Features

#### Core Functionality
- OpenAI-compatible REST API (`/v1/chat/completions`)
- Multi-provider LLM support:
  - OpenAI (GPT-4, GPT-3.5-turbo)
  - Anthropic (Claude 3 series)
  - Ollama (local LLM hosting)
- gRPC ModuleService for MarchProxy integration

#### Intelligent Routing
- Round-robin load balancing
- Cost-optimized routing
- Latency-optimized routing
- Load-balanced routing (default)
- Failover to alternative providers
- Random provider selection

#### Conversation Memory
- Session-based conversation context
- Vector similarity search
- ChromaDB-backed persistent storage
- Session isolation
- Automatic context window management

#### RAG (Retrieval-Augmented Generation)
- Knowledge base integration
- Multi-collection support
- Semantic document search
- Context-aware response generation
- Configurable top-k retrieval

#### API Key Management
- Virtual API key generation
- User-scoped key management
- Key revocation (irreversible)
- Key status tracking (active/revoked/expired)
- Usage tracking per key

#### Billing & Cost Tracking
- Token usage tracking
- Multi-provider cost calculation
- Per-key monthly quotas (dollar and token limits)
- Budget enforcement
- Spend analytics by provider and model
- Usage statistics and reports

#### Rate Limiting
- Per-user rate limits (RPM/TPM)
- Per-provider rate enforcement
- Token bucket algorithm
- Burst handling
- Rate limit middleware integration

#### Security
- Prompt injection detection
- Jailbreak attempt prevention
- Credential harvesting detection
- Prompt sanitization
- Security policy levels (strict/balanced/permissive)
- Threat statistics tracking
- Security event logging

#### Monitoring & Observability
- `/healthz` Kubernetes health check
- `/metrics` Prometheus endpoint
- Routing statistics endpoint
- Request success/failure tracking
- Provider-specific metrics
- Latency tracking (p50, p95, p99)

### Architecture

- FastAPI HTTP server (Port 8080)
- gRPC service (Port 50051)
- Async/await pattern for concurrency
- Connection pooling to all providers
- Structured logging with JSON output

### API Endpoints

**HTTP API:**
- `POST /v1/chat/completions` - Chat completions
- `GET /v1/models` - List available models
- `GET /api/routing/stats` - Routing statistics
- `GET /healthz` - Health check
- `GET /metrics` - Prometheus metrics
- `POST /api/keys` - Create API key
- `GET /api/keys` - List keys
- `GET /api/keys/{key_id}` - Get key details
- `PATCH /api/keys/{key_id}` - Update key
- `DELETE /api/keys/{key_id}` - Revoke key
- `GET /api/billing/spend` - Spending summary
- `POST /api/billing/budget` - Set budget
- `GET /api/billing/budget/{key_id}` - Check budget status

**gRPC API:**
- `GetStatus()` - Module health
- `GetMetrics()` - Performance metrics
- `SetTrafficWeight()` - Blue/green deployment
- `ApplyRateLimit()` - Rate limit configuration
- `GetRoutes()` - Route information
- `Reload()` - Configuration reload

### Testing

- 74 prompt security tests
- 63 token manager tests
- Comprehensive edge case coverage
- No external API mocking required

### Documentation

- `ARCHITECTURE.md` - System design and data flow
- `API.md` - Complete API reference
- `CONFIGURATION.md` - Environment variables and setup
- `TESTING.md` - Test running and structure
- `USAGE.md` - Usage examples and integration patterns
- `README.md` - Overview and quick start

### Docker

- Multi-stage build for minimal image size
- Python 3.11-slim base image
- Health checks integrated
- Persistent volumes for memory/RAG
- Dockerfile provided

### Known Limitations

1. **Token Counting:** Estimates based on character count (4 chars per token)
   - Future: Provider-specific token counters

2. **Metrics:** Endpoint structure ready but data collection incomplete
   - Future: Full Prometheus integration

3. **Authentication:** Bearer token validation only
   - Future: OAuth2, SAML, API key rotation

4. **Conversation Memory:**
   - Single instance storage (non-distributed)
   - Future: Distributed memory backend (Redis, Memcached)

5. **RAG:**
   - Document ingestion API not yet implemented
   - Future: Document upload and indexing endpoints

### Dependencies

- FastAPI 0.104+
- Pydantic 2.0+
- gRPC 1.60+
- ChromaDB 0.4+
- sentence-transformers 2.2+
- openai 1.3+
- anthropic 0.7+
- uvicorn 0.24+
- prometheus-client 0.19+
- structlog 24.1+

### Performance

- Request latency: 100-500ms (provider-dependent)
- Memory lookup: 10-50ms
- RAG search: 50-200ms
- Routing decision: <1ms
- Concurrent sessions: Limited by provider rate limits

### Deployment

- Docker container ready
- gRPC integration with MarchProxy NLB
- Health check compatible with Kubernetes
- Stateless design for horizontal scaling

### Bug Fixes

- Initial release: No bugs to fix

### Breaking Changes

- None (initial release)

### Deprecations

- None

### Security Updates

- Prompt security scanner with multiple threat detection types
- Rate limiting per user and provider
- Budget enforcement to prevent runaway costs
- API key revocation support

### Contributors

- Initial implementation team

---

## Version History

| Version | Release Date | Status | Notes |
|---------|-------------|--------|-------|
| v0.1.0 | 2024-01-20 | Stable | Initial release |

---

## Upgrade Guide

### From Pre-Release

1. Build new image: `docker build -t marchproxy/ailb:v0.1.0 .`
2. Update Docker Compose version tag
3. Restart containers with persistent volumes
4. Test API endpoints with curl or client SDK

## Migration Notes

### First Time Setup

1. Configure provider API keys (OpenAI/Anthropic)
2. Enable conversation memory if needed
3. Create initial API keys for users
4. Configure rate limits and budgets
5. Test routing with multiple providers

---

**Last Updated:** 2024-01-20
