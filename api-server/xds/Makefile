.PHONY: all build run clean test docker-build docker-run

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=xds-server

# Docker parameters
DOCKER_IMAGE=marchproxy/xds-server
DOCKER_TAG=latest

all: clean deps build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

deps:
	$(GOMOD) download
	$(GOMOD) verify

run: build
	./$(BINARY_NAME) -port=18000 -http-port=19000 -nodeID=marchproxy-node -debug

docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run:
	docker run -p 18000:18000 -p 19000:19000 $(DOCKER_IMAGE):$(DOCKER_TAG)

tidy:
	$(GOMOD) tidy
