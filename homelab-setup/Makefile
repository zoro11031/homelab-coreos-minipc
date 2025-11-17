.PHONY: build install lint clean run help

# Build variables
BINARY_NAME=homelab-setup
BUILD_DIR=bin
MAIN_PATH=./cmd/homelab-setup

# Version information (will be set by build flags)
VERSION ?= 0.1.0-dev
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "\
	-X github.com/zoro11031/homelab-coreos-minipc/homelab-setup/pkg/version.Version=$(VERSION) \
	-X github.com/zoro11031/homelab-coreos-minipc/homelab-setup/pkg/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/zoro11031/homelab-coreos-minipc/homelab-setup/pkg/version.BuildDate=$(BUILD_DATE)"

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) $(MAIN_PATH)
	@echo "Installed to: $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## lint: Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
	golangci-lint run; \
	else \
	echo "golangci-lint not installed. Install from: https://golangci-lint.run/usage/install/"; \
	fi

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean

## run: Build and run the binary
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

## tidy: Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download

## build-all: Build for multiple architectures
build-all:
	@echo "Building for multiple architectures..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@echo "Multi-arch binaries built in $(BUILD_DIR)/"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help
