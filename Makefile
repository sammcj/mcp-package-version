# Makefile for mcp-package-version

# Variables
BINARY_NAME=mcp-package-version
BINARY_PATH=bin/$(BINARY_NAME)
GO=go
GOFLAGS=-v
GOFMT=$(GO) fmt
GOTEST=$(GO) test
DOCKER=docker
DOCKER_IMAGE=$(BINARY_NAME)

# Default target
.PHONY: all
all: build

# Build the server
.PHONY: build
build:
	mkdir -p bin
	$(GO) build $(GOFLAGS) -o $(BINARY_PATH) \
		-ldflags "-X github.com/sammcj/mcp-package-version/pkg/version.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo '0.1.0-dev') \
		-X github.com/sammcj/mcp-package-version/pkg/version.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown') \
		-X github.com/sammcj/mcp-package-version/pkg/version.BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")" \
		.

# Run the server with stdio transport (default)
.PHONY: run
run: build
	./$(BINARY_PATH)

# Run the server with SSE transport
.PHONY: run-sse
run-sse: build
	./$(BINARY_PATH) --transport sse --port 18080 --base-url http://localhost

# Run tests
.PHONY: test
test:
	$(GOTEST) $(GOFLAGS) ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/

# Format code
.PHONY: fmt
fmt:
	$(GOFMT) ./...

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Install dependencies
.PHONY: deps
deps:
	$(GO) mod download

# Update dependencies
.PHONY: update-deps
update-deps:
	$(GO) get -u ./...
	$(GO) mod tidy

# Build Docker image
.PHONY: docker-build
docker-build:
	$(DOCKER) build -t $(DOCKER_IMAGE) .

# Run Docker container
.PHONY: docker-run
docker-run: docker-build
	$(DOCKER) run -p 18080:18080 $(DOCKER_IMAGE)

# Create a new release
.PHONY: release
release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required. Use: make release VERSION=x.y.z"; exit 1; fi
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)

# Bump version (similar to standard-version in JS)
.PHONY: bump-version
bump-version:
	npx -y standard-version --skip.tag && git add . && git commit -m "chore: bump version" && git push

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all          : Build the server (default)"
	@echo "  build        : Build the server"
	@echo "  run          : Run the server with stdio transport (default)"
	@echo "  run-sse      : Run the server with SSE transport"
	@echo "  test         : Run tests"
	@echo "  clean        : Clean build artifacts"
	@echo "  fmt          : Format code"
	@echo "  lint         : Lint code"
	@echo "  deps         : Install dependencies"
	@echo "  update-deps  : Update dependencies"
	@echo "  docker-build : Build Docker image"
	@echo "  docker-run   : Run Docker container with SSE transport"
	@echo "  release      : Create a new release (requires VERSION=x.y.z)"
	@echo "  bump-version : Automatically bump version, update CHANGELOG.md and push changes"
	@echo "  help         : Show this help message"
