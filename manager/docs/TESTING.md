# Testing Guide

Comprehensive testing documentation for MarchProxy Manager.

## Test Structure

```
tests/
├── __init__.py
├── test_auth_utils.py          # Authentication tests
├── fixtures/                   # Test fixtures and data
├── integration/                # Integration tests
└── unit/                       # Unit tests
```

## Running Tests

### Via Docker

**Run all tests:**
```bash
docker build -t marchproxy-manager:testing --target testing .
docker run --rm marchproxy-manager:testing
```

**Run specific test file:**
```bash
docker run --rm marchproxy-manager:testing \
  python -m pytest tests/test_auth_utils.py -v
```

**Run with coverage report:**
```bash
docker run --rm marchproxy-manager:testing \
  python -m pytest tests/ -v --cov=. --cov-report=html
```

### Locally

**Install test dependencies:**
```bash
pip install -r requirements-dev.txt
```

**Run all tests:**
```bash
pytest tests/ -v
```

**Run specific test:**
```bash
pytest tests/test_auth_utils.py::TestAuthUtils::test_hash_password -v
```

**With coverage:**
```bash
pytest tests/ --cov=. --cov-report=term-missing
```

## Test Categories

### Unit Tests

Test individual functions and classes in isolation.

**Location:** `tests/unit/`

**Example:**
```python
def test_user_password_hashing():
    """Test password hashing functionality"""
    password = "secure_password123"
    hashed = UserModel.hash_password(password)

    assert UserModel.verify_password(password, hashed)
    assert not UserModel.verify_password("wrong_password", hashed)
```

### Integration Tests

Test interaction between components (API endpoints, database, services).

**Location:** `tests/integration/`

**Example:**
```python
def test_user_registration_and_login(client):
    """Test complete registration and login flow"""

    # Register user
    response = client.post('/api/auth/register', json={
        'username': 'testuser',
        'email': 'test@example.com',
        'password': 'TestPass123!'
    })
    assert response.status_code == 201

    # Login
    response = client.post('/api/auth/login', json={
        'username': 'testuser',
        'password': 'TestPass123!'
    })
    assert response.status_code == 200
    assert 'token' in response.json()
```

### Database Tests

Test database models and operations.

**Setup:**
```python
@pytest.fixture
def test_db():
    """Create test database"""
    db = DAL('sqlite:memory:', migrate=True)
    # Define tables
    UserModel.define_table(db)
    ClusterModel.define_table(db)
    yield db
    db.close()
```

### API Tests

Test API endpoints with mocked external services.

**Requirements:**
- In-memory SQLite database
- Mocked license server
- Test JWT tokens

**Example:**
```python
def test_create_cluster_unauthorized(client):
    """Test cluster creation requires admin role"""
    response = client.post('/api/clusters', json={
        'name': 'Test Cluster'
    })
    assert response.status_code == 401
```

## Test Fixtures

### Database Fixtures

Create reusable test data:

```python
@pytest.fixture
def test_user(test_db):
    """Create test user"""
    user_id = test_db.users.insert(
        username='testuser',
        email='test@example.com',
        password_hash=UserModel.hash_password('password123'),
        is_active=True
    )
    test_db.commit()
    return test_db.users[user_id]

@pytest.fixture
def test_cluster(test_db, test_user):
    """Create test cluster"""
    cluster_id = test_db.clusters.insert(
        name='Test Cluster',
        created_by=test_user.id,
        is_active=True
    )
    test_db.commit()
    return test_db.clusters[cluster_id]
```

### Client Fixtures

Create authenticated test clients:

```python
@pytest.fixture
def authenticated_client(client, jwt_manager):
    """Client with authentication token"""
    token = jwt_manager.create_token(user_id=1)
    client.headers = {'Authorization': f'Bearer {token}'}
    return client
```

## Test Coverage Requirements

**Minimum coverage: 80%**

### Coverage Targets

- **Models**: 85%+ (authentication, database operations)
- **API endpoints**: 80%+ (request/response handling)
- **Services**: 85%+ (business logic)
- **Utilities**: 75%+ (helper functions)

### Exclude from Coverage

```python
# pragma: no cover
def debug_function():
    pass
```

## Mocking External Services

### License Server

```python
@pytest.fixture
def mock_license_server(monkeypatch):
    """Mock PenguinTech license server"""
    def mock_validate(key):
        return {
            'is_valid': True,
            'tier': 'enterprise',
            'max_proxies': 100,
            'expires_at': '2026-01-01'
        }

    monkeypatch.setattr(
        'services.license.LicenseManager.validate',
        mock_validate
    )
```

### SAML Provider

```python
@pytest.fixture
def mock_saml_response():
    """Mock SAML authentication response"""
    return {
        'email': 'user@example.com',
        'name': 'Test User',
        'groups': ['admins']
    }
```

### Syslog Server

```python
@pytest.fixture
def mock_syslog_server(monkeypatch):
    """Mock UDP syslog server"""
    messages = []

    def mock_send(message):
        messages.append(message)
        return True

    monkeypatch.setattr(
        'services.syslog_client.SyslogClient.send',
        mock_send
    )

    return messages
```

## Performance Tests

Load testing for critical endpoints:

```python
def test_auth_endpoint_performance(benchmark, client):
    """Benchmark authentication endpoint"""
    result = benchmark(
        client.post,
        '/api/auth/login',
        json={'username': 'admin', 'password': 'password'}
    )
    assert result.status_code == 200
```

**Run performance tests:**
```bash
pytest tests/ -v --benchmark-only
```

## End-to-End Tests

Full system tests with Docker Compose:

```bash
docker-compose -f docker-compose.test.yml up
pytest tests/e2e/ -v --timeout=30
docker-compose -f docker-compose.test.yml down
```

## Continuous Integration

Tests run automatically on:
- Push to main/develop branches
- Pull requests
- Manual trigger

### GitHub Actions

Configuration in `.github/workflows/manager-ci.yml`:

1. **Lint Stage** - Code quality checks
2. **Test Stage** - Unit and integration tests
3. **Build Stage** - Docker image build
4. **Security Stage** - Vulnerability scanning

## Troubleshooting Tests

### Database Lock Issues

```python
# Use WAL mode for better concurrency
DAL('sqlite:memory:', pool_size=1, check_reserved=['all'])
```

### Async Test Issues

```python
@pytest.mark.asyncio
async def test_async_function():
    result = await async_function()
    assert result is not None
```

### Timezone Issues

```python
import os
os.environ['TZ'] = 'UTC'
```

## Best Practices

1. **Isolation**: Each test should be independent
2. **Clarity**: Test names describe what is tested
3. **Simplicity**: Test one thing per test
4. **Setup/Teardown**: Clean up after each test
5. **Assertions**: Use clear, specific assertions
6. **Mocking**: Mock external dependencies
7. **Coverage**: Aim for high coverage on critical paths
8. **Speed**: Keep tests fast (< 1 second each)

## Example Test Suite

```python
import pytest
from models.auth import UserModel
from models.cluster import ClusterModel

class TestUserAuthentication:
    """Test user authentication workflows"""

    def test_create_user_with_validation(self, test_db):
        """Test user creation with input validation"""
        user_id = test_db.users.insert(
            username='validuser',
            email='user@example.com',
            password_hash=UserModel.hash_password('SecurePass123!'),
            is_active=True
        )

        assert user_id is not None
        user = test_db.users[user_id]
        assert user.username == 'validuser'

    def test_password_verification(self):
        """Test password hashing and verification"""
        password = 'MySecurePassword123!'
        hashed = UserModel.hash_password(password)

        assert UserModel.verify_password(password, hashed)
        assert not UserModel.verify_password('WrongPassword', hashed)

    def test_duplicate_username_rejected(self, test_db):
        """Test that duplicate usernames are rejected"""
        test_db.users.insert(
            username='duplicate',
            email='first@example.com',
            password_hash='hash123',
            is_active=True
        )

        with pytest.raises(ValueError):
            test_db.users.insert(
                username='duplicate',
                email='second@example.com',
                password_hash='hash456',
                is_active=True
            )

class TestClusterManagement:
    """Test cluster creation and management"""

    def test_create_cluster(self, test_db, test_user):
        """Test cluster creation"""
        cluster_id = test_db.clusters.insert(
            name='Test Cluster',
            created_by=test_user.id,
            is_active=True
        )

        assert cluster_id is not None
        cluster = test_db.clusters[cluster_id]
        assert cluster.name == 'Test Cluster'

    def test_cluster_api_key_generation(self, test_db, test_cluster):
        """Test automatic API key generation for cluster"""
        assert test_cluster.api_key is not None
        assert len(test_cluster.api_key) > 20
```

## Debugging Tests

### Enable verbose output
```bash
pytest tests/ -vv -s
```

### Run with debugger
```bash
pytest tests/ --pdb
```

### Show local variables on failure
```bash
pytest tests/ -l
```

### Filter tests by marker
```bash
pytest tests/ -m "not slow"
```

## Test Markers

Define test categories:

```python
@pytest.mark.unit
def test_password_hashing():
    pass

@pytest.mark.integration
def test_full_auth_flow():
    pass

@pytest.mark.slow
def test_large_dataset_processing():
    pass
```

Run specific markers:
```bash
pytest tests/ -m "unit" -v
pytest tests/ -m "not slow" -v
```
