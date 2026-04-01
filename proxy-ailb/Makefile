.PHONY: build test lint run docker-build proto clean fmt vet

VERSION ?= $(shell cat ../.version 2>/dev/null || echo "dev")
XDP_BUILD_TAGS ?= noxdp
BINARY := ailb
DOCKER_IMAGE := marchproxy/proxy-ailb

build:
	CGO_ENABLED=0 go build \
		-tags "$(XDP_BUILD_TAGS)" \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o bin/$(BINARY) ./cmd/ailb

run: build
	./bin/$(BINARY)

test:
	go test -v -race -cover ./...

lint: vet
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found"; exit 1; }
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -s -w .

proto:
	cd ../proto && protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		marchproxy/*.proto

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg XDP_BUILD_TAGS=$(XDP_BUILD_TAGS) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-f Dockerfile ..

clean:
	rm -rf bin/

tidy:
	go mod tidy
