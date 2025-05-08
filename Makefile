.PHONY: build run test clean docker docker-compose

BINARY_NAME=odin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

build:
	@echo "Building Odin API Gateway..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/odin/main.go

run: build
	@echo "Running Odin API Gateway..."
	./bin/$(BINARY_NAME) --config config/config.yaml

test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint:
	@echo "Running linter..."
	golangci-lint run ./...

clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -f coverage.out coverage.html

docker:
	@echo "Building Docker image..."
	docker build -t odin-gateway:latest -f deployments/docker/Dockerfile.prod .

docker-compose:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-compose-dev:
	@echo "Starting development environment with Docker Compose..."
	docker-compose -f docker-compose.dev.yml up -d

generate-token:
	@echo "Generating JWT token..."
	cd test/auth && go run jwt-generator.go

install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

help:
	@echo "Odin API Gateway Make commands:"
	@echo "  build           - Build the binary"
	@echo "  run             - Build and run the binary"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  lint            - Run linter"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker          - Build Docker image"
	@echo "  docker-compose  - Start all services with Docker Compose"
	@echo "  docker-compose-dev - Start development environment"
	@echo "  generate-token  - Generate JWT token for testing"
	@echo "  install-tools   - Install development tools"
