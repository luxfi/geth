# Makefile for Lux Geth

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=geth
BINARY_UNIX=$(BINARY_NAME)_unix

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILDFLAGS=-v

# Default target
.DEFAULT_GOAL := build

# Build target is a no-op in this repo; see the lux/node README for build instructions
build:
	@echo "This repository is a Geth plugin for Lux; it must be built via the Lux node build script."
	@echo "Run the following from the lux/node directory:"
	@echo "  ./scripts/build.sh"

# Build-all is an alias for build
build-all: build

# Test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOGET) -v -t -d ./...

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Please install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

# Run and install targets are no longer supported; use the build script directly

# Check for security vulnerabilities
security:
	@echo "Checking for vulnerabilities..."
	$(GOCMD) list -json -m all | nancy sleuth

# Generate mocks
mocks:
	@echo "Generating mocks..."
	$(GOCMD) generate ./...

# Verify modules
verify:
	@echo "Verifying modules..."
	$(GOMOD) verify

# Show help
help:
	@echo "Makefile for Lux Geth"
	@echo ""
	@echo "Usage:"
		@echo "  make build          Print Lux C-Chain plugin build instructions"
		@echo "  make build-all      Alias for make build"
	@echo "  make test           Run tests"
	@echo "  make test-coverage  Run tests with coverage"
	@echo "  make bench          Run benchmarks"
	@echo "  make clean          Clean build files"
	@echo "  make deps           Install dependencies"
	@echo "  make update-deps    Update dependencies"
	@echo "  make fmt            Format code"
	@echo "  make lint           Run linter"
	@echo "  make security       Check for vulnerabilities"
	@echo "  make mocks          Generate mocks"
	@echo "  make verify         Verify modules"
	@echo "  make help           Show this help"

.PHONY: build build-all test test-coverage bench clean deps update-deps fmt lint security mocks verify help
