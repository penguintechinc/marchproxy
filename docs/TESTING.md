# MarchProxy Testing Infrastructure

Comprehensive testing documentation for MarchProxy integration, end-to-end, performance, and security tests.

## Table of Contents

- [Overview](#overview)
- [Test Suites](#test-suites)
- [Running Tests](#running-tests)
- [CI/CD Integration](#cicd-integration)
- [Coverage Requirements](#coverage-requirements)
- [Test Development](#test-development)

## Overview

MarchProxy implements a comprehensive testing strategy with multiple test levels:

1. **Unit Tests** - Individual component testing
2. **Integration Tests** - API and database integration
3. **End-to-End Tests** - Full system deployment
4. **Performance Tests** - Load and stress testing
5. **Security Tests** - Vulnerability and penetration testing
6. **UI Tests** - Browser automation with Playwright

## Test Suites

### API Server Integration Tests

**Location**: `api-server/tests/integration/`

Tests complete workflows for:
- Authentication flow (login, 2FA, token refresh)
- Cluster lifecycle (CRUD, license validation)
- Service management (creation, routing, tokens)
- Proxy registration (heartbeat, metrics)
- Certificate management (upload, rotation, expiry)
- xDS configuration (generation, updates)

**Run**:
```bash
cd api-server
pytest tests/integration/ -v --cov=app
```

### WebUI Integration Tests

**Location**: `webui/tests/integration/`

Playwright tests for:
- Login flow with 2FA
- Cluster management UI
- Service CRUD operations
- Real-time proxy monitoring
- Dashboard functionality

**Run**:
```bash
cd webui
npm run test
```

### End-to-End Tests

**Location**: `tests/e2e/`

Full deployment tests:
- All 4 containers startup
- Proxy → API → xDS → Envoy flow
- Service routing configuration
- Certificate rotation propagation
- License enforcement

**Run**:
```bash
./scripts/run-e2e-tests.sh
```

### Performance Tests

**Location**: `tests/performance/`

Load testing with Locust:
- API server load (10K+ req/s target)
- Proxy throughput benchmarks
- Concurrent connections handling
- Response time SLAs

**Run**:
```bash
./scripts/run-performance-tests.sh
# Or manually:
cd tests/performance
locust -f locustfile.py --host=http://localhost:8000
```

### Security Tests

**Location**: `tests/security/`

OWASP Top 10 validation:
- Authentication security
- Authorization/RBAC enforcement
- SQL injection prevention
- XSS prevention
- Command injection prevention
- Rate limiting
- Secret management

**Run**:
```bash
pytest tests/security/ -v -m security
```

## Running Tests

### Run All Tests

```bash
./scripts/run-tests.sh
```

This script runs:
1. API integration tests with coverage
2. End-to-end tests
3. Security tests
4. Performance benchmarks
5. WebUI Playwright tests

### Run Specific Test Suites

**API Integration Only**:
```bash
cd api-server
pytest tests/integration/ -v
```

**E2E Only**:
```bash
./scripts/run-e2e-tests.sh
```

**Security Only**:
```bash
pytest tests/security/ -v -m security
```

**Performance Only**:
```bash
./scripts/run-performance-tests.sh
```

**WebUI Only**:
```bash
cd webui
npm run test
npm run test:headed  # Watch tests run
npm run test:ui      # Interactive UI
```

### Test Markers

Tests are organized with pytest markers:

```bash
pytest -m integration    # Integration tests
pytest -m e2e           # End-to-end tests
pytest -m security      # Security tests
pytest -m performance   # Performance tests
pytest -m "not slow"    # Exclude slow tests
```

## CI/CD Integration

### GitHub Actions Workflow

**File**: `.github/workflows/tests.yml`

Automatically runs on:
- Push to `main` or `develop`
- Pull requests to `main` or `develop`

**Jobs**:
1. `api-integration-tests` - API server integration tests
2. `webui-tests` - Playwright UI tests
3. `e2e-tests` - Full deployment E2E tests
4. `security-tests` - Security scans and tests

### Test Reports

Reports are generated and uploaded as artifacts:
- **API Coverage**: HTML coverage report
- **Playwright Report**: Visual test execution
- **E2E Report**: HTML test results
- **Security Reports**: Bandit and Safety JSON output

Access reports in GitHub Actions artifacts (retained for 30 days).

## Coverage Requirements

### Minimum Coverage Thresholds

- **API Server**: 80% code coverage
- **Services**: 85% code coverage
- **Critical paths**: 95% code coverage

### Coverage Reports

**API Server**:
```bash
cd api-server
pytest tests/integration/ --cov=app --cov-report=html
open htmlcov/index.html
```

**Coverage by Module**:
- `app/routers/`: API endpoints
- `app/services/`: Business logic
- `app/models/`: Database models
- `app/dependencies/`: Dependency injection

## Test Development

### Writing Integration Tests

**Example**:
```python
@pytest.mark.asyncio
async def test_create_cluster(async_client, auth_headers):
    """Test cluster creation."""
    response = await async_client.post(
        "/api/v1/clusters",
        headers=auth_headers,
        json={
            "name": "test-cluster",
            "tier": "community"
        }
    )
    
    assert response.status_code == 201
    data = response.json()
    assert data["name"] == "test-cluster"
    assert "api_key" in data
```

### Writing E2E Tests

**Example**:
```python
@pytest.mark.e2e
def test_full_flow(docker_services, api_base_url):
    """Test complete flow."""
    # Create cluster
    response = requests.post(f"{api_base_url}/api/v1/clusters", ...)
    
    # Register proxy
    proxy_response = requests.post(f"{api_base_url}/api/v1/proxies/register", ...)
    
    # Verify
    assert response.status_code == 201
```

### Writing Playwright Tests

**Example**:
```typescript
test('should create cluster', async ({ page }) => {
  await page.goto('/dashboard/clusters');
  await page.click('button:has-text("Create Cluster")');
  await page.fill('input[name="name"]', 'test-cluster');
  await page.click('button[type="submit"]');
  
  await expect(page.locator('text=Success')).toBeVisible();
});
```

### Test Fixtures

**Common Fixtures**:
- `admin_user` - Admin user with credentials
- `test_cluster` - Pre-created test cluster
- `auth_headers` - Authorization headers
- `db_session` - Database session
- `docker_services` - Running Docker services

### Test Database

Tests use a separate test database:
- **URL**: `postgresql://marchproxy:marchproxy@localhost:5432/marchproxy_test`
- Automatically created/dropped per test session
- Isolated from development/production databases

## Performance Benchmarks

### Target Performance Metrics

- **Health endpoint**: < 50ms p99
- **Authentication**: < 500ms average
- **List operations**: < 200ms average
- **Metrics endpoint**: < 100ms average
- **Concurrent requests**: 100+ simultaneous without errors

### Load Testing Scenarios

**Locust scenarios**:
1. `MarchProxyUser` - Simulates admin operations
2. `ProxyHeartbeatUser` - Simulates proxy heartbeats

**Run load test**:
```bash
cd tests/performance
locust -f locustfile.py --users 100 --spawn-rate 10 --run-time 5m
```

## Security Testing

### Automated Security Scans

**Bandit** - Python code security:
```bash
bandit -r api-server/app
```

**Safety** - Dependency vulnerabilities:
```bash
safety check
```

**OWASP ZAP** - Web application scanning (manual):
```bash
docker run -t owasp/zap2docker-stable zap-baseline.py -t http://localhost:8000
```

### Security Test Categories

1. **Authentication** - JWT validation, token expiry, 2FA
2. **Authorization** - RBAC, permission checks
3. **Injection** - SQL, XSS, command injection
4. **Rate Limiting** - Brute force protection
5. **Secrets** - No hardcoded credentials

## Troubleshooting

### Common Issues

**Database connection errors**:
```bash
# Ensure PostgreSQL is running
docker-compose -f docker-compose.test.yml up -d postgres
```

**Port conflicts**:
```bash
# Stop conflicting services
docker-compose -f docker-compose.test.yml down
```

**Playwright installation**:
```bash
cd webui
npx playwright install --with-deps
```

**Missing dependencies**:
```bash
pip install -r api-server/requirements-test.txt
pip install -r tests/requirements.txt
cd webui && npm ci
```

## Best Practices

1. **Test Isolation** - Each test should be independent
2. **Cleanup** - Always cleanup resources after tests
3. **Realistic Data** - Use realistic test data
4. **Error Cases** - Test both success and failure paths
5. **Performance** - Keep tests fast (< 5s per test)
6. **Documentation** - Document complex test scenarios
7. **Fixtures** - Reuse fixtures for common setup

## Contributing

When adding new features:

1. Write tests first (TDD)
2. Ensure 80%+ coverage
3. Add integration tests for APIs
4. Add E2E tests for critical flows
5. Update test documentation
6. Run full test suite before PR

## Resources

- [Pytest Documentation](https://docs.pytest.org/)
- [Playwright Documentation](https://playwright.dev/)
- [Locust Documentation](https://docs.locust.io/)
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
