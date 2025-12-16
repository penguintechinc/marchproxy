.PHONY: all build test clean docker docker-nvidia docker-amd docker-all run help

# Variables
BINARY_NAME=rtmp-proxy
VERSION?=1.0.0
BUILD_DIR=build
DOCKER_REGISTRY?=ghcr.io/penguintech
IMAGE_NAME=marchproxy-rtmp

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -s -w"

all: test build

help:
	@echo "MarchProxy RTMP Container - Makefile targets:"
	@echo "  build          - Build the binary"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker         - Build CPU Docker image"
	@echo "  docker-nvidia  - Build NVIDIA GPU Docker image"
	@echo "  docker-amd     - Build AMD GPU Docker image"
	@echo "  docker-all     - Build all Docker images"
	@echo "  run            - Run locally"
	@echo "  tidy           - Tidy go.mod"

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/rtmp
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

tidy:
	@echo "Tidying go.mod..."
	$(GOMOD) tidy

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.txt

# Docker builds
docker:
	@echo "Building CPU Docker image..."
	docker build -f Dockerfile -t $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)-cpu .
	docker tag $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)-cpu $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest

docker-nvidia:
	@echo "Building NVIDIA GPU Docker image..."
	docker build -f Dockerfile.nvidia -t $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)-nvidia .

docker-amd:
	@echo "Building AMD GPU Docker image..."
	docker build -f Dockerfile.amd -t $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)-amd .

docker-all: docker docker-nvidia docker-amd

# Run locally
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) \
		--host 0.0.0.0 \
		--port 1935 \
		--grpc-port 50053 \
		--encoder auto \
		--preset medium \
		--log-level debug

# Push to registry
push:
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)-cpu
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)-nvidia
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)-amd
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest

# Development
dev: tidy build run

# Install dependencies
deps:
	$(GOGET) -v -t -d ./...
