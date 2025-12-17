# MarchProxy API Server - Testing Guide

Comprehensive testing documentation for the MarchProxy API Server.

## Overview

Testing strategy includes:
- **Unit Tests**: Individual component testing with mocked dependencies
- **Integration Tests**: Component interaction testing
- **API Tests**: HTTP endpoint testing
- **Performance Tests**: Load and stress testing
- **Security Tests**: Vulnerability and compliance testing

## Test Coverage Requirements

Minimum coverage thresholds:
- **Overall**: 80%
- **Critical paths**: 95%
- **Security**: 100%

## Unit Testing

### Running Unit Tests

```bash
# Run all tests
pytest

# Run with coverage report
pytest --cov=app --cov-report=html --cov-report=term

# Run specific test file
pytest tests/unit/test_auth.py

# Run specific test
pytest tests/unit/test_auth.py::TestLoginEndpoint::test_login_success

# Run with verbose output
pytest -v

# Run with markers
pytest -m "auth"
```

### Test File Structure

```
tests/
├── unit/                    # Unit tests
│   ├── test_auth.py        # Authentication tests
│   ├── test_services.py    # Service tests
│   ├── test_clusters.py    # Cluster tests
│   ├── test_proxies.py     # Proxy tests
│   └── test_certificates.py
├── integration/            # Integration tests
│   ├── test_service_flow.py
│   ├── test_xds_bridge.py
│   └── test_database.py
├── api/                    # API endpoint tests
│   ├── test_auth_endpoints.py
│   ├── test_service_endpoints.py
│   └── test_cluster_endpoints.py
├── conftest.py            # Pytest fixtures and configuration
└── factories.py           # Test data factories
```

### Example Unit Test

```python
import pytest
from app.models.sqlalchemy.user import User
from app.core.security import hash_password, verify_password


class TestPasswordHashing:
    """Test password hashing functions"""

    def test_hash_password(self):
        """Test password hashing"""
        password = "test_password_123"
        hashed = hash_password(password)

        assert hashed != password
        assert verify_password(password, hashed)

    def test_verify_password_fails_with_wrong_password(self):
        """Test password verification fails with wrong password"""
        password = "test_password_123"
        hashed = hash_password(password)

        assert not verify_password("wrong_password", hashed)


class TestUserModel:
    """Test User SQLAlchemy model"""

    def test_user_creation(self):
        """Test user model instantiation"""
        user = User(
            email="test@example.com",
            username="testuser",
            password_hash=hash_password("password123"),
            is_admin=False
        )

        assert user.email == "test@example.com"
        assert user.username == "testuser"
        assert user.is_admin is False
```

### Fixtures and Factories

Use pytest fixtures for common setup:

```python
# conftest.py
import pytest
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker


@pytest.fixture(scope="session")
def db():
    """Create test database"""
    engine = create_engine("sqlite:///:memory:")
    Base.metadata.create_all(engine)
    SessionLocal = sessionmaker(bind=engine)
    return SessionLocal()


@pytest.fixture
def client(db):
    """FastAPI test client"""
    from fastapi.testclient import TestClient
    from app.main import app

    # Override database dependency
    from app.dependencies import get_db
    app.dependency_overrides[get_db] = lambda: db

    return TestClient(app)


@pytest.fixture
def admin_user(db):
    """Create admin test user"""
    from app.models.sqlalchemy.user import User
    from app.core.security import hash_password

    user = User(
        email="admin@test.com",
        username="admin",
        password_hash=hash_password("admin123"),
        is_admin=True,
        is_verified=True,
        is_active=True
    )
    db.add(user)
    db.commit()
    return user


@pytest.fixture
def admin_token(admin_user):
    """Generate admin JWT token"""
    from app.core.security import create_access_token
    return create_access_token(admin_user.id)
```

## Integration Testing

### Running Integration Tests

```bash
# Run all integration tests
pytest tests/integration/ -v

# Run specific integration test
pytest tests/integration/test_service_flow.py

# Run with database setup/teardown
pytest tests/integration/ --setup-show

# Run with timeout
pytest tests/integration/ --timeout=30
```

### Example Integration Test

```python
class TestServiceCreationFlow:
    """Test service creation and xDS update flow"""

    async def test_create_service_triggers_xds_update(
        self,
        client,
        admin_token,
        test_cluster,
        xds_mock
    ):
        """Test creating service updates xDS configuration"""
        response = client.post(
            "/api/v1/services",
            json={
                "cluster_id": str(test_cluster.id),
                "name": "test-service",
                "destination_ip": "10.0.1.100",
                "destination_port": 443,
                "protocol": "https"
            },
            headers={"Authorization": f"Bearer {admin_token}"}
        )

        assert response.status_code == 201
        service_id = response.json()["id"]

        # Verify xDS update was triggered
        xds_mock.update_snapshot.assert_called_once()
        call_args = xds_mock.update_snapshot.call_args
        assert str(test_cluster.id) in call_args
```

## API Testing

### Testing with TestClient

```python
from fastapi.testclient import TestClient
from app.main import app

client = TestClient(app)

# Test successful login
response = client.post(
    "/api/v1/auth/login",
    json={
        "email": "admin@example.com",
        "password": "securepassword"
    }
)

assert response.status_code == 200
assert "access_token" in response.json()
assert response.json()["token_type"] == "bearer"
```

### End-to-End API Tests

```python
class TestAuthenticationFlow:
    """Test complete authentication flow"""

    def test_full_auth_flow(self, client):
        """Test registration -> login -> auth request"""
        # Register user
        reg_response = client.post(
            "/api/v1/auth/register",
            json={
                "email": "newuser@example.com",
                "username": "newuser",
                "password": "SecurePass123!",
                "full_name": "New User"
            }
        )
        assert reg_response.status_code == 201

        # Login
        login_response = client.post(
            "/api/v1/auth/login",
            json={
                "email": "newuser@example.com",
                "password": "SecurePass123!"
            }
        )
        assert login_response.status_code == 200
        token = login_response.json()["access_token"]

        # Use token for authenticated request
        me_response = client.get(
            "/api/v1/auth/me",
            headers={"Authorization": f"Bearer {token}"}
        )
        assert me_response.status_code == 200
        assert me_response.json()["email"] == "newuser@example.com"
```

## Performance Testing

### Load Testing with Locust

```bash
# Install locust
pip install locust

# Run load test
locust -f tests/performance/locustfile.py --host=http://localhost:8000
```

### Example Load Test

```python
# tests/performance/locustfile.py
from locust import HttpUser, task, between


class APIUser(HttpUser):
    wait_time = between(1, 3)

    @task(3)
    def list_services(self):
        token = "your_test_token"
        self.client.get(
            "/api/v1/services",
            headers={"Authorization": f"Bearer {token}"}
        )

    @task(1)
    def create_service(self):
        token = "your_test_token"
        self.client.post(
            "/api/v1/services",
            json={
                "cluster_id": "test-cluster-id",
                "name": f"service-{random.randint(1000, 9999)}",
                "destination_ip": "10.0.1.100",
                "destination_port": 443,
                "protocol": "https"
            },
            headers={"Authorization": f"Bearer {token}"}
        )
```

### Benchmarking

```bash
# Run with timing
pytest tests/unit/test_auth.py -v --durations=10

# Profile with pytest-benchmark
pip install pytest-benchmark

pytest tests/unit/test_auth.py --benchmark-only
```

## Security Testing

### OWASP Top 10 Tests

```python
class TestSecurityVulnerabilities:
    """Test for common security vulnerabilities"""

    def test_sql_injection_protection(self, client, admin_token):
        """Test SQL injection protection"""
        response = client.get(
            "/api/v1/clusters?name='; DROP TABLE users; --",
            headers={"Authorization": f"Bearer {admin_token}"}
        )
        # Should handle safely, not execute injection
        assert response.status_code in [200, 400, 404]

    def test_xss_protection(self, client, admin_token):
        """Test XSS protection in responses"""
        response = client.post(
            "/api/v1/services",
            json={
                "cluster_id": "test",
                "name": "<script>alert('xss')</script>",
                "destination_ip": "10.0.1.100",
                "destination_port": 443,
                "protocol": "https"
            },
            headers={"Authorization": f"Bearer {admin_token}"}
        )
        # Should reject or sanitize
        assert response.status_code in [400, 422]

    def test_csrf_protection(self, client):
        """Test CSRF token validation"""
        response = client.post(
            "/api/v1/auth/logout"
            # Missing CSRF token - should be rejected
        )
        # Without proper CSRF header
        assert response.status_code == 401

    def test_authentication_bypass(self, client):
        """Test authentication cannot be bypassed"""
        # Try accessing protected endpoint without token
        response = client.get("/api/v1/clusters")
        assert response.status_code == 401

    def test_authorization_enforcement(self, client, user_token):
        """Test user cannot access admin-only endpoints"""
        # Regular user token trying admin operation
        response = client.post(
            "/api/v1/clusters",
            json={"name": "test"},
            headers={"Authorization": f"Bearer {user_token}"}
        )
        assert response.status_code == 403
```

### Dependency Vulnerability Scanning

```bash
# Check Python dependencies
pip install safety
safety check

# Check Go dependencies
go install github.com/golang/vuln/cmd/govulncheck@latest
cd xds && govulncheck ./...

# Run security linting
pip install bandit
bandit -r app/ -ll
```

## Docker Testing

### Test Docker Build

```bash
# Build image
docker build -t marchproxy-api-server:test .

# Test image
docker run --rm marchproxy-api-server:test python -c "
from app.core.config import settings
print(f'Version: {settings.APP_VERSION}')
"

# Run tests in container
docker run --rm \
  -e DATABASE_URL="sqlite:////:memory:" \
  marchproxy-api-server:test \
  pytest --cov=app
```

### Integration Tests with Docker Compose

```bash
# Run full stack with test database
docker-compose -f docker-compose.test.yml up --abort-on-container-exit

# View logs
docker-compose -f docker-compose.test.yml logs -f

# Cleanup
docker-compose -f docker-compose.test.yml down -v
```

### Example docker-compose.test.yml

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-bookworm
    environment:
      POSTGRES_USER: test
      POSTGRES_PASSWORD: test
      POSTGRES_DB: marchproxy_test
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "test"]
      interval: 5s
      timeout: 5s
      retries: 5

  api-server:
    build: .
    environment:
      DATABASE_URL: postgresql+asyncpg://test:test@postgres:5432/marchproxy_test
      SECRET_KEY: test-secret-key-min-32-chars-here
      DEBUG: "true"
    depends_on:
      postgres:
        condition: service_healthy
    command: pytest --cov=app --cov-report=term-missing
```

## Continuous Integration

### GitHub Actions Test Workflow

```yaml
name: API Server Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15-bookworm
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: marchproxy_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Set up Python
        uses: actions/setup-python@v4
        with:
          python-version: '3.12'

      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          pip install -r requirements-test.txt

      - name: Run linting
        run: |
          flake8 app/
          black --check app/
          isort --check-only app/

      - name: Run tests
        env:
          DATABASE_URL: postgresql+asyncpg://test:test@localhost:5432/marchproxy_test
          SECRET_KEY: test-secret-key
        run: pytest --cov=app --cov-report=xml

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.xml
```

## Test Markers

Use pytest markers for test organization:

```python
import pytest

@pytest.mark.auth
def test_login(): pass

@pytest.mark.service
def test_create_service(): pass

@pytest.mark.integration
def test_service_xds_flow(): pass

@pytest.mark.slow
def test_large_dataset(): pass

@pytest.mark.security
def test_sql_injection(): pass
```

Run by marker:
```bash
pytest -m "auth"           # Run auth tests
pytest -m "not slow"       # Skip slow tests
pytest -m "integration and not slow"  # Specific combination
```

## Debugging Tests

### Using pdb

```python
def test_something():
    import pdb; pdb.set_trace()
    # Code execution pauses here for debugging
```

### Pytest debugging options

```bash
# Drop into pdb on failure
pytest --pdb

# Drop into pdb on first error
pytest -x --pdb

# Print captured output
pytest -s

# Show local variables on failure
pytest -l
```

## Test Data Management

### Factories for Test Data

```python
# tests/factories.py
from factory import Factory, Faker, SubFactory
from app.models.sqlalchemy.user import User
from app.core.security import hash_password


class UserFactory(Factory):
    class Meta:
        model = User

    email = Faker('email')
    username = Faker('user_name')
    password_hash = factory.LazyFunction(
        lambda: hash_password('password123')
    )
    is_admin = False
    is_active = True
    is_verified = True


class ClusterFactory(Factory):
    class Meta:
        model = Cluster

    name = Faker('word')
    description = Faker('sentence')
    max_proxies = 10
```

## Coverage Reports

### Generate HTML coverage report

```bash
pytest --cov=app --cov-report=html --cov-report=term-missing
open htmlcov/index.html
```

### View coverage by module

```bash
pytest --cov=app --cov-report=term-missing:skip-covered
```

## Troubleshooting Tests

### Database Issues

```bash
# Reset test database
rm -f test.db
pytest --create-db

# View database state
sqlite3 test.db ".tables"
sqlite3 test.db "SELECT * FROM users;"
```

### Fixture Issues

```bash
# Show fixture setup/teardown
pytest --setup-show

# Verbose fixture info
pytest --fixtures
```

### Async Test Issues

```bash
# Install pytest-asyncio
pip install pytest-asyncio

# Mark async tests
import pytest

@pytest.mark.asyncio
async def test_async_function():
    result = await some_async_function()
    assert result == expected
```

## Best Practices

1. **Test Organization**: Group related tests in classes
2. **Descriptive Names**: Use clear test names describing what's tested
3. **Setup/Teardown**: Use fixtures for common setup
4. **Mocking**: Mock external dependencies (databases, APIs)
5. **Assertions**: Use clear, specific assertions
6. **Coverage**: Aim for >80% code coverage
7. **Performance**: Keep tests fast; use mocks over real I/O
8. **Isolation**: Each test should be independent
9. **Documentation**: Document complex test scenarios
10. **Maintenance**: Keep tests updated with code changes
