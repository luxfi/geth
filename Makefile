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

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(BUILDFLAGS) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/geth

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILDFLAGS) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/geth
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILDFLAGS) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/geth

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILDFLAGS) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/geth
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILDFLAGS) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/geth

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILDFLAGS) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe ./cmd/geth

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

# Run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Install
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install ./cmd/geth

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
	@echo "  make build          Build the binary"
	@echo "  make build-all      Build for all platforms"
	@echo "  make test           Run tests"
	@echo "  make test-coverage  Run tests with coverage"
	@echo "  make bench          Run benchmarks"
	@echo "  make clean          Clean build files"
	@echo "  make deps           Install dependencies"
	@echo "  make update-deps    Update dependencies"
	@echo "  make fmt            Format code"
	@echo "  make lint           Run linter"
	@echo "  make run            Build and run"
	@echo "  make install        Install the binary"
	@echo "  make security       Check for vulnerabilities"
	@echo "  make mocks          Generate mocks"
	@echo "  make verify         Verify modules"
	@echo "  make help           Show this help"

.PHONY: build build-all build-linux build-darwin build-windows test test-coverage bench clean deps update-deps fmt lint run install security mocks verify help