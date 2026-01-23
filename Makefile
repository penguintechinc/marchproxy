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
