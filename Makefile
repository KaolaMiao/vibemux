# VibeMux Makefile

APP_NAME := vibemux
VERSION := 0.1.0
BUILD_DIR := bin
GO := go

# Build flags
LDFLAGS := -ldflags "-s -w -X main.appVersion=$(VERSION)"

.PHONY: all build run clean test fmt lint help

## Default target
all: build

## Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) .

## Run in development mode
run:
	$(GO) run .

## Run with race detection (for debugging)
run-race:
	$(GO) run -race .

## Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GO) clean

## Run tests
test:
	$(GO) test -v ./...

## Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## Format code
fmt:
	$(GO) fmt ./...

## Run linter (requires golangci-lint)
lint:
	golangci-lint run

## Tidy dependencies
tidy:
	$(GO) mod tidy

## Install the application
install: build
	@echo "Installing $(APP_NAME)..."
	@cp $(BUILD_DIR)/$(APP_NAME) $(GOPATH)/bin/

## Build for multiple platforms
release:
	@echo "Building releases..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 .

## Display help
help:
	@echo "VibeMux - AI Agent Orchestration Terminal"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build         Build the application"
	@echo "  run           Run in development mode"
	@echo "  run-race      Run with race detection"
	@echo "  clean         Clean build artifacts"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  fmt           Format code"
	@echo "  lint          Run linter"
	@echo "  tidy          Tidy dependencies"
	@echo "  install       Install to GOPATH/bin"
	@echo "  release       Build for multiple platforms"
	@echo "  help          Display this help"
