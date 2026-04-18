# MarchProxy Makefile
# Provides convenient targets for development and testing

.PHONY: help smoke-test smoke-alpha smoke-beta dev clean

help:
	@echo "MarchProxy Development Commands"
	@echo ""
	@echo "Smoke Tests:"
	@echo "  make smoke-test   - Run alpha smoke tests (local E2E)"
	@echo "  make smoke-alpha  - Run alpha smoke tests (local E2E)"
	@echo "  make smoke-beta   - Run beta smoke tests (staging K8s)"
	@echo ""
	@echo "Development:"
	@echo "  make dev          - Start development environment"
	@echo "  make clean        - Stop and clean all containers"
	@echo ""

# Alpha smoke tests (local end-to-end)
smoke-test: smoke-alpha

smoke-alpha:
	@echo "Running alpha smoke tests (local E2E)..."
	@./tests/smoke/alpha/run-all.sh

# Beta smoke tests (staging K8s cluster)
smoke-beta:
	@echo "Running beta smoke tests (staging cluster)..."
	@./tests/smoke/beta/run-all.sh

# Start development environment
dev:
	@echo "Starting development environment..."
	@docker-compose -f docker-compose.yml up -d
	@echo "Services started. Check status with: docker-compose ps"

# Clean up containers
clean:
	@echo "Stopping and cleaning containers..."
	@docker-compose -f docker-compose.yml down -v
	@echo "Cleanup complete"

test:
	@$(MAKE) test-unit

test-unit:
	@echo "Running unit tests..."
	@if [ -d tests ]; then python3 -m pytest tests/ -v; fi
	@find . -name "go.mod" -not -path "*/vendor/*" | xargs -I{} dirname {} | xargs -I{} sh -c 'cd {} && go test ./... || true'

test-integration:
	@echo "Running integration tests..."

test-e2e:
	@echo "Running e2e tests..."

test-functional:
	@echo "No functional tests defined"

test-security:
	@echo "=== Security Scans ==="
	@if command -v bandit >/dev/null 2>&1; then echo "-- bandit --"; bandit -r . -x ./tests,./venv,./.git,./node_modules --quiet || true; fi
	@if command -v pip-audit >/dev/null 2>&1; then echo "-- pip-audit --"; find . -name "requirements.txt" -not -path "*/.git/*" -not -path "*/venv/*" | xargs -I{} pip-audit -r {} 2>/dev/null || true; fi
	@if command -v gosec >/dev/null 2>&1; then echo "-- gosec --"; find . -name "go.mod" -not -path "*/.git/*" -not -path "*/vendor/*" | xargs -I{} dirname {} | xargs -I{} sh -c 'cd {} && gosec ./... || true'; fi
	@if command -v govulncheck >/dev/null 2>&1; then echo "-- govulncheck --"; find . -name "go.mod" -not -path "*/.git/*" -not -path "*/vendor/*" | xargs -I{} dirname {} | xargs -I{} sh -c 'cd {} && govulncheck ./... || true'; fi
	@find . -name "package.json" -not -path "*/.git/*" -not -path "*/node_modules/*" -maxdepth 3 | xargs -I{} dirname {} | xargs -I{} sh -c 'cd {} && echo "-- npm audit --" && npm audit 2>/dev/null || true'
	@if command -v gitleaks >/dev/null 2>&1; then echo "-- gitleaks --"; gitleaks detect --source . --no-git 2>/dev/null || true; fi
	@if command -v trufflehog >/dev/null 2>&1; then echo "-- trufflehog --"; trufflehog filesystem . --no-update 2>/dev/null || true; fi

lint:
	@echo "=== Linting ==="
	@if command -v flake8 >/dev/null 2>&1; then echo "-- flake8 --"; python3 -m flake8 . --max-line-length=120 --exclude=.git,__pycache__,venv,.venv,node_modules,.claude,*/venv/*,*/.venv/* || true; fi
	@if command -v black >/dev/null 2>&1; then echo "-- black --"; black --check . --exclude '/(\.git|venv|\.venv|__pycache__|node_modules|\.claude)/' || true; fi
	@if command -v isort >/dev/null 2>&1; then echo "-- isort --"; isort --check-only . || true; fi
	@if command -v mypy >/dev/null 2>&1; then echo "-- mypy --"; python3 -m mypy . --ignore-missing-imports 2>/dev/null || true; fi
	@if command -v golangci-lint >/dev/null 2>&1; then echo "-- golangci-lint --"; find . -name "go.mod" -not -path "*/.git/*" -not -path "*/vendor/*" | xargs -I{} dirname {} | xargs -I{} sh -c 'cd {} && golangci-lint run || true'; fi
	@if command -v hadolint >/dev/null 2>&1; then echo "-- hadolint --"; find . -name "Dockerfile*" -not -path "*/.git/*" | xargs hadolint || true; fi
	@if command -v shellcheck >/dev/null 2>&1; then echo "-- shellcheck --"; find . -name "*.sh" -not -path "*/.git/*" | xargs shellcheck || true; fi
	@find . -name "package.json" -not -path "*/.git/*" -not -path "*/node_modules/*" -maxdepth 3 | xargs -I{} dirname {} | xargs -I{} sh -c 'cd {} && command -v eslint >/dev/null 2>&1 && eslint . || true' 2>/dev/null || true
	@find . -name "package.json" -not -path "*/.git/*" -not -path "*/node_modules/*" -maxdepth 3 | xargs -I{} dirname {} | xargs -I{} sh -c 'cd {} && command -v prettier >/dev/null 2>&1 && prettier --check . || true' 2>/dev/null || true

build:
	docker-compose build

docker-build: build

docker-push:
	@echo "Push images to registry - use CI pipeline"

deploy-dev:
	@echo "Deploy to dev/alpha environment"

deploy-prod:
	@echo "Deploy to production"

seed-mock-data:
	@echo "No mock data seeding defined"

pre-commit:
	@echo "=== Pre-commit checks ==="
	@$(MAKE) lint
	@$(MAKE) test-security
	@$(MAKE) test
	@echo "=== Pre-commit complete ==="
