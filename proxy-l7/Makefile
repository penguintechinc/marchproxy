.PHONY: all build build-xdp build-filters build-docker clean test help

# Project variables
PROJECT_NAME := marchproxy-proxy-l7
DOCKER_IMAGE := marchproxy/proxy-l7
DOCKER_TAG := latest
BUILD_DIR := build

all: clean build

help:
	@echo "MarchProxy L7 Proxy Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all           - Clean and build everything"
	@echo "  build         - Build XDP + WASM filters"
	@echo "  build-xdp     - Build XDP program only"
	@echo "  build-filters - Build WASM filters only"
	@echo "  build-docker  - Build Docker image"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  help          - Show this help"

build: build-xdp build-filters
	@echo "✓ All components built successfully"

build-xdp:
	@echo "Building XDP program..."
	@./scripts/build_xdp.sh

build-filters:
	@echo "Building WASM filters..."
	@./scripts/build_filters.sh

build-docker:
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	docker build -f envoy/Dockerfile -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@cd xdp && make clean || true
	@find filters -type d -name target -exec rm -rf {} + 2>/dev/null || true
	@echo "✓ Clean complete"

test:
	@echo "Running tests..."
	@echo "TODO: Implement tests"

# Development targets
dev-build: build
	@echo "Development build complete"

dev-run: dev-build
	@echo "Starting development Envoy..."
	docker run -it --rm \
		-p 10000:10000 \
		-p 9901:9901 \
		--cap-add=NET_ADMIN \
		-e XDS_SERVER=host.docker.internal:18000 \
		-e LOGLEVEL=debug \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

# Install targets
install-deps:
	@echo "Installing build dependencies..."
	@./scripts/install_deps.sh || echo "Note: Run scripts/install_deps.sh manually if needed"

# Docker targets
docker-push:
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-pull:
	docker pull $(DOCKER_IMAGE):$(DOCKER_TAG)
