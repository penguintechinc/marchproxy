# Makefile for MarchProxy ALB

VERSION ?= v1.0.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
IMAGE_NAME := marchproxy/proxy-alb
IMAGE_TAG := $(VERSION)

.PHONY: all proto build build-docker run clean test help

all: proto build

## help: Display this help message
help:
	@echo "MarchProxy ALB Build System"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

## proto: Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	cd ../proto/marchproxy && \
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       module_service.proto
	@echo "Protobuf generation complete"

## build: Build the Go supervisor binary
build:
	@echo "Building ALB supervisor..."
	go build -ldflags="-w -s \
		-X main.version=$(VERSION) \
		-X main.buildTime=$(BUILD_TIME) \
		-X main.gitCommit=$(GIT_COMMIT)" \
		-o bin/alb-supervisor main.go
	@echo "Build complete: bin/alb-supervisor"

## build-docker: Build Docker image
build-docker:
	@echo "Building Docker image: $(IMAGE_NAME):$(IMAGE_TAG)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-t $(IMAGE_NAME):latest \
		-f Dockerfile \
		..
	@echo "Docker build complete"

## build-multi: Build multi-architecture Docker image
build-multi:
	@echo "Building multi-arch Docker image: $(IMAGE_NAME):$(IMAGE_TAG)"
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-t $(IMAGE_NAME):latest \
		-f Dockerfile \
		--push \
		..
	@echo "Multi-arch build complete"

## run: Run the ALB locally
run: build
	@echo "Running ALB supervisor..."
	./bin/alb-supervisor

## run-docker: Run ALB in Docker
run-docker:
	@echo "Running ALB container..."
	docker run -d \
		--name marchproxy-alb \
		-p 10000:10000 \
		-p 9901:9901 \
		-p 50051:50051 \
		-p 9090:9090 \
		-p 8080:8080 \
		-e XDS_SERVER=api-server:18000 \
		-e MODULE_ID=alb-1 \
		-e LOG_LEVEL=info \
		$(IMAGE_NAME):$(IMAGE_TAG)
	@echo "Container started: marchproxy-alb"

## test: Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

## lint: Run linters
lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf vendor/
	go clean -cache -testcache -modcache
	@echo "Clean complete"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

## verify: Verify dependencies
verify:
	@echo "Verifying dependencies..."
	go mod verify

## docker-logs: View container logs
docker-logs:
	docker logs -f marchproxy-alb

## docker-stop: Stop and remove container
docker-stop:
	docker stop marchproxy-alb || true
	docker rm marchproxy-alb || true

## docker-shell: Open shell in container
docker-shell:
	docker exec -it marchproxy-alb /bin/sh
