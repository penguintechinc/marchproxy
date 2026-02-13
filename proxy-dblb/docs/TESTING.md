# DBLB Testing Guide

Comprehensive testing guide for MarchProxy Database Load Balancer (DBLB) including unit tests, integration tests, and performance testing.

## Prerequisites

- Go 1.24.x or later
- Docker (for containerized testing)
- Make (optional, for convenience commands)
- `golangci-lint` (for linting)

## Unit Testing

### Running All Unit Tests

```bash
go test ./...
```

### Running with Coverage

```bash
go test -v -cover ./...
```

Generate detailed coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Running Specific Test Package

```bash
go test -v ./internal/handlers/
go test -v ./internal/pool/
go test -v ./internal/security/
```

### Running Specific Test

```bash
go test -v -run TestConnectionPooling ./internal/pool/
```

### Test Naming Convention

All tests follow Go conventions:
- File suffix: `_test.go`
- Function prefix: `Test`
- Benchmarks prefix: `Benchmark`
- Examples prefix: `Example`

Example test file: `internal/handlers/sqlite_test.go`

## Database-Specific Tests

### SQLite Tests

SQLite is bundled with DBLB for testing and standalone deployments:

```bash
go test -v ./internal/handlers/ -run TestSQLite
```

### MySQL Handler Tests

Requires MySQL test instance. Use Docker:

```bash
docker run -d \
  -e MYSQL_ROOT_PASSWORD=testpass \
  -e MYSQL_DATABASE=testdb \
  -p 3306:3306 \
  mysql:8.0
```

Then run tests:
```bash
MYSQL_HOST=localhost MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=testpass go test -v ./internal/handlers/ -run TestMySQL
```

### PostgreSQL Handler Tests

Requires PostgreSQL test instance:

```bash
docker run -d \
  -e POSTGRES_PASSWORD=testpass \
  -e POSTGRES_DB=testdb \
  -p 5432:5432 \
  postgres:15
```

Then run tests:
```bash
POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_USER=postgres POSTGRES_PASSWORD=testpass go test -v ./internal/handlers/ -run TestPostgreSQL
```

### MongoDB Handler Tests

Requires MongoDB test instance:

```bash
docker run -d \
  -p 27017:27017 \
  mongo:6.0
```

Then run tests:
```bash
MONGODB_HOST=localhost MONGODB_PORT=27017 go test -v ./internal/handlers/ -run TestMongoDB
```

## Integration Tests

Integration tests verify DBLB behavior with real or mocked database backends.

### Running Integration Tests

```bash
go test -v -tags=integration ./...
```

### Docker Compose Integration Testing

A complete test environment can be started with Docker Compose:

```bash
docker-compose -f docker-compose.test.yml up -d
```

This starts:
- MySQL instance (port 3306)
- PostgreSQL instance (port 5432)
- MongoDB instance (port 27017)
- Redis instance (port 6379)
- MSSQL instance (port 1433)

Run full integration suite:
```bash
go test -v -tags=integration ./... -timeout=5m
```

### Test Data Setup

Databases are automatically initialized with test schemas:

```sql
-- MySQL test schema
CREATE DATABASE IF NOT EXISTS testdb;
USE testdb;
CREATE TABLE IF NOT EXISTS test_table (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

```sql
-- PostgreSQL test schema
CREATE DATABASE testdb;
\c testdb
CREATE TABLE test_table (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Security Testing

### SQL Injection Detection Tests

Test SQL injection pattern detection:

```bash
go test -v -run TestSQLInjection ./internal/security/
```

Test cases include:
- UNION-based injection: `SELECT * FROM users WHERE id = 1 UNION SELECT ...`
- Error-based injection: `SELECT * FROM users WHERE id = 1 AND 1=CAST(...)`
- Time-based blind injection: `SELECT * FROM users WHERE id = 1 AND SLEEP(5)`
- Comment injection: `SELECT * FROM users WHERE id = 1 -- ...`
- Stacked queries: `SELECT ...; DROP TABLE ...;`

### Threat Intelligence Tests

Test threat intelligence integration:

```bash
go test -v ./internal/security/threat_intelligence_test.go
```

## Performance Testing

### Benchmarking

Run benchmarks to measure performance:

```bash
go test -bench=. -benchmem ./...
```

Run specific benchmark:
```bash
go test -bench=BenchmarkConnectionPool -benchmem ./internal/pool/
```

### Load Testing

Test DBLB under load using `wrk` or similar tool:

```bash
# Install wrk
brew install wrk  # macOS
# or
apt-get install wrk  # Linux

# Run load test
wrk -t12 -c400 -d30s --script=load_test.lua http://localhost:3306
```

### Connection Pool Stress Testing

Test connection pool behavior under high concurrency:

```bash
go test -bench=BenchmarkConcurrentConnections -benchmem ./internal/pool/ -benchtime=30s
```

## Race Detection

Use Go's race detector to find potential race conditions:

```bash
go test -race ./...
```

Run with coverage:
```bash
go test -race -coverprofile=coverage.out ./...
```

## Test Coverage Requirements

Minimum coverage targets:
- `internal/handlers/`: 85%+
- `internal/pool/`: 90%+
- `internal/security/`: 80%+
- Overall: 80%+

Check current coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1
```

## Linting

### Run All Linters

```bash
golangci-lint run ./...
```

### Run Specific Linter

```bash
golangci-lint run --disable-all -E vet ./...
golangci-lint run --disable-all -E gosec ./...
golangci-lint run --disable-all -E golint ./...
```

### Auto-fix Issues

Some issues can be auto-fixed:

```bash
golangci-lint run --fix ./...
```

### Linting Configuration

Linting is configured in `.golangci.yml`:

```yaml
linters:
  enable:
    - vet
    - vetshadow
    - golint
    - gosec
    - govet
    - ineffassign
    - staticcheck
    - unused
    - unconvert
    - misspell
```

## Docker Testing

### Build Test Image

```bash
docker build -t marchproxy/dblb:test \
  --build-arg VERSION=test-$(date +%s) .
```

### Run Tests in Container

```bash
docker run --rm \
  -v $(pwd):/app \
  golang:1.24-bookworm \
  bash -c "cd /app && go test -v -cover ./..."
```

### Integration Test with Docker Compose

```bash
docker-compose -f docker-compose.test.yml run dblb-tests
```

## CI/CD Testing

Tests are automatically run in GitHub Actions on every commit:

1. **Lint Stage**: Checks code quality with golangci-lint
2. **Test Stage**: Runs all tests with coverage reporting
3. **Build Stage**: Builds Docker images for multiple architectures
4. **Security Scan**: Scans for vulnerabilities with gosec and Trivy

See `.github/workflows/proxy-dblb-ci.yml` for full pipeline details.

## Debugging Tests

### Verbose Test Output

```bash
go test -v ./...
```

### Print Debugging

Add print statements in code:
```go
fmt.Printf("DEBUG: value=%v\n", someValue)
```

Then run with verbose flag to see output.

### Using Delve Debugger

```bash
go install github.com/go-delve/delve/cmd/dlv@latest

dlv test ./internal/handlers/
(dlv) break TestMySQL
(dlv) continue
(dlv) print someValue
```

### Environment Variables for Testing

```bash
# Verbose logging
DBLB_LOG_LEVEL=debug go test -v ./...

# Skip slow tests
go test -short ./...

# Run only specific tests
go test -run "TestConnection" ./...

# Timeout control
go test -timeout 10m ./...
```

## Test Database Cleanup

After running tests, clean up test containers:

```bash
docker-compose -f docker-compose.test.yml down -v
```

Or manually:
```bash
docker rm -f $(docker ps -q -f "label=test=dblb")
```

## Continuous Integration

### GitHub Actions

Tests run automatically on:
- Push to main branch
- Push to develop branch
- Pull requests
- Manual workflow trigger

### Local CI Simulation

Run the full CI pipeline locally:

```bash
# Install act (GitHub Actions runner)
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | bash

# Run workflow
act push
```

## Troubleshooting

### Test Hangs

If tests hang, increase timeout:
```bash
go test -timeout 30s ./...
```

### Database Connection Issues

Verify database is running and accessible:
```bash
# MySQL
mysql -h localhost -u root -p -e "SELECT 1"

# PostgreSQL
psql -h localhost -U postgres -c "SELECT 1"

# MongoDB
mongosh "mongodb://localhost:27017"
```

### Race Condition Failures

Race detector may be flaky. Run multiple times:
```bash
for i in {1..10}; do go test -race ./...; done
```

### Coverage Not Meeting Requirements

Identify uncovered code:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
# Open coverage.html in browser to see uncovered lines
```

## Test Maintenance

### Regular Updates

- Update test dependencies monthly
- Review and refactor tests quarterly
- Update test data and schemas with production changes
- Monitor test execution time and optimize slow tests

### Test Review Checklist

Before committing tests:
- [ ] Tests pass locally
- [ ] Coverage meets requirements
- [ ] Linting passes
- [ ] Race detector passes
- [ ] Performance benchmarks acceptable
- [ ] Documentation updated
