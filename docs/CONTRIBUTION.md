# Contributing to MarchProxy

Thank you for your interest in contributing to MarchProxy! This guide covers everything needed to contribute effectively.

## Table of Contents

- [Getting Started](#getting-started)
- [Branch Naming](#branch-naming)
- [Commit Format](#commit-format)
- [PR Process](#pr-process)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)

## Getting Started

### Prerequisites

- Git
- Docker and Docker Compose
- Go 1.24+ (for proxy development)
- Python 3.12+ (for manager development)
- PostgreSQL 13+ (local development)

### Setup

1. **Fork and clone**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/marchproxy.git
   cd marchproxy
   git remote add upstream https://github.com/marchproxy/marchproxy.git
   ```

2. **Setup development environment**:
   ```bash
   docker-compose -f docker-compose.dev.yml up -d
   ./scripts/setup-dev.sh
   ```

3. **Install pre-commit hooks**:
   ```bash
   cd manager
   pre-commit install
   ```

## Branch Naming

Use clear, descriptive branch names following this format:

- **Features**: `feature/add-oauth-support`
- **Bug fixes**: `bugfix/fix-connection-leak`
- **Releases**: `release/v1.2.0`
- **Hotfixes**: `hotfix/critical-security-fix`

Pattern: `{type}/{description}` with hyphens, lowercase, max 60 chars.

## Commit Format

Follow Conventional Commits specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Code style (formatting)
- `refactor`: Code refactoring
- `test`: Test additions/updates
- `chore`: Maintenance tasks

### Scopes

- `manager`: Manager/backend
- `proxy`: Proxy services
- `proxy-egress`: Egress proxy
- `proxy-ingress`: Ingress proxy
- `webui`: Web interface
- `docs`: Documentation

### Examples

```
feat(manager): add SAML authentication support

Implement SAML 2.0 authentication with IdP integration.
Includes user provisioning and attribute mapping.

Closes #123
```

```
fix(proxy): resolve memory leak in connection pooling

Connection objects were not properly released on error paths,
causing gradual memory increase under load.

Closes #456
```

## PR Process

### Before Creating PR

1. **Create issue first** - Describe problem, solution, breaking changes
2. **Update from upstream**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```
3. **Pass all local checks**:
   - Linting passes
   - Tests pass with 80%+ coverage
   - No hardcoded secrets
   - Security scans clean

### Creating PR

1. **Push branch**: `git push origin feature/your-feature`
2. **Use PR template** with:
   - Clear description
   - Changes checklist
   - Testing steps
   - Related issues
3. **Link related issues**: "Closes #123"

### PR Requirements

- ✅ All CI checks pass
- ✅ Code review approved (1+ maintainer)
- ✅ Tests pass
- ✅ Documentation updated
- ✅ No breaking changes (unless discussed)

## Code Standards

### Python Standards (Manager)

**Formatting & Linting**:
```bash
black --line-length=100 manager/
isort manager/
flake8 manager/
mypy --strict manager/
bandit -r manager/
```

**Requirements**:
- PEP 8 compliance
- Type hints on all functions
- Docstrings (PEP 257)
- No unused imports/variables
- Explicit error handling

**Example**:
```python
def create_service(self, config: Dict[str, Any]) -> int:
    """Create a new service.

    Args:
        config: Service configuration with required keys

    Returns:
        Service ID of created service

    Raises:
        ValidationError: If config invalid
        ServiceExistsError: If name already exists
    """
    if not self._validate_config(config):
        raise ValidationError("Invalid configuration")
    return self.db.services.create(**config)
```

### Go Standards (Proxy Services)

**Formatting & Linting**:
```bash
go fmt ./...
golangci-lint run
go vet ./...
go test -race ./...
```

**Requirements**:
- Standard Go conventions
- godoc comments for exported functions
- Explicit error handling
- No unused imports/variables
- Race-free code

**Example**:
```go
// Start starts the proxy server and begins accepting connections.
func (s *Server) Start(ctx context.Context) error {
    if s.listener == nil {
        return fmt.Errorf("listener not configured")
    }

    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        return s.listen()
    }
}
```

### TypeScript Standards (Web UI)

**Formatting & Linting**:
```bash
eslint webui/src/
prettier --write webui/src/
```

**Requirements**:
- Strict TypeScript mode
- No any types without justification
- React/component best practices
- Tests for components
- Clear prop types

## Testing Requirements

### Coverage Targets

- **Minimum**: 80% code coverage
- **Target**: 90%+ coverage
- **Exceptions**: Generated code, vendor code

### Running Tests

**Python (Manager)**:
```bash
cd manager
pytest tests/ -v --cov=apps --cov-report=term-missing
```

**Go (Proxy Services)**:
```bash
cd proxy-egress
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**TypeScript (Web UI)**:
```bash
cd webui
npm test -- --coverage
```

### Test Requirements

✅ **Unit Tests**: All new functions must have tests
✅ **Error Cases**: Test both success and failure paths
✅ **Edge Cases**: Test boundary conditions
✅ **Mocking**: Mock external dependencies
✅ **Fixtures**: Use reusable test fixtures
✅ **Clear Names**: Test names describe what they verify

### Test Example (Go)

```go
func TestServerStart(t *testing.T) {
    tests := []struct {
        name    string
        config  *Config
        wantErr bool
    }{
        {
            name:    "valid config",
            config:  &Config{ListenAddr: ":8080"},
            wantErr: false,
        },
        {
            name:    "invalid config",
            config:  &Config{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := NewServer(tt.config)
            err := server.Start(context.Background())
            if (err != nil) != tt.wantErr {
                t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Example (Python)

```python
import unittest
from unittest.mock import Mock, patch

class TestAuthenticationManager(unittest.TestCase):
    def setUp(self):
        self.auth = AuthenticationManager(jwt_secret="test")

    def test_authenticate_user_success(self):
        with patch('manager.auth.verify_password') as mock_verify:
            mock_verify.return_value = True
            result = self.auth.authenticate_user("user", "pass")
            self.assertIsNotNone(result)

    def test_authenticate_user_failure(self):
        with patch('manager.auth.verify_password') as mock_verify:
            mock_verify.return_value = False
            result = self.auth.authenticate_user("user", "pass")
            self.assertIsNone(result)
```

## Security & Quality Checklist

Before submitting PR, verify:

- [ ] No hardcoded secrets/credentials
- [ ] No debug statements or console.log
- [ ] Input validation on all API inputs
- [ ] Error messages don't leak sensitive info
- [ ] No SQL injection vulnerabilities
- [ ] Dependencies have no known vulnerabilities
- [ ] CSRF/XSS protection in place
- [ ] Tests pass with adequate coverage
- [ ] No TODO comments (complete or remove)
- [ ] Linting passes without warnings

## Getting Help

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and general discussion
- **Security Issues**: security@marchproxy.io

## License

By contributing, you agree your contributions are licensed under the Limited AGPL3 license with preamble for fair use.

---

**Last Updated**: 2025-12-16
**Version**: 1.0.0
