# Docker Registry Manager Makefile

.PHONY: build run clean test deps help

# Variables
BINARY_NAME=docker-registry-manager
BUILD_DIR=build
CMD_DIR=cmd
CONFIG_FILE=config.yaml

# Default target
all: build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
run: build
	@echo "Starting $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG_FILE)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf data/uploads/*
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Install dependencies
install-deps:
	@echo "Installing dependencies..."
	go get github.com/gorilla/mux@v1.8.1
	go get github.com/gorilla/handlers@v1.5.2
	go get github.com/sirupsen/logrus@v1.9.3
	go get gopkg.in/yaml.v2@v2.4.0

# Development mode (with auto-restart)
dev: build
	@echo "Starting development mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG_FILE) &
	@echo "Development server started. Press Ctrl+C to stop."

# Create release build
release:
	@echo "Building release version..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Release build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@mkdir -p data/{blobs,repositories,uploads}
	@mkdir -p web/{static/{css,js},templates}
	@echo "Development environment ready"

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  run          - Build and run the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  deps         - Download dependencies"
	@echo "  install-deps - Install dependencies"
	@echo "  dev          - Start development mode"
	@echo "  release      - Create release build"
	@echo "  setup        - Setup development environment"
	@echo "  help         - Show this help message"

