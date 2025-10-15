.PHONY: build run test clean docker docker-compose helm-package helm-install

BINARY_NAME=odin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

build:
	@echo "Building Odin API Gateway..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/odin/main.go

migrate-dry-run:
	@echo "Running MongoDB migration (dry run)..."
	./bin/odin-migrate --config config/config.yaml --dry-run --verbose

migrate:
	@echo "Running MongoDB migration..."
	./bin/odin-migrate --config config/config.yaml --verbose

migrate-force:
	@echo "Running MongoDB migration (force override)..."
	./bin/odin-migrate --config config/config.yaml --force --verbose

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
	go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@echo "Coverage report generated: coverage.html"

coverage-badge:
	@echo "Generating coverage badge..."
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE >= 80" | bc -l) -eq 1 ]; then \
		echo "Coverage is good ($$COVERAGE% >= 80%)"; \
	else \
		echo "Coverage is below threshold ($$COVERAGE% < 80%)"; \
		exit 1; \
	fi

test-unit:
	@echo "Running unit tests..."
	go test -v ./pkg/...

test-integration:
	@echo "Running integration tests..."
	go test -v ./test/integration/...

test-oauth2:
	@echo "Running OAuth2 tests..."
	go test -v ./test/unit/pkg/auth/

test-circuit-breaker:
	@echo "Running circuit breaker tests..."
	go test -v ./test/unit/pkg/circuit/

test-websocket:
	@echo "Running WebSocket tests..."
	go test -v ./test/unit/pkg/websocket/

lint:
	@echo "Running linter..."
	golangci-lint run ./...

clean: clean-plugins
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

helm-package:
	@echo "Packaging Helm chart..."
	helm package deployments/helm/odin

helm-install:
	@echo "Installing Helm chart..."
	helm install odin deployments/helm/odin \
		--set config.auth.jwtSecret="development-secret" \
		--set ingress.enabled=false

helm-upgrade:
	@echo "Upgrading Helm chart..."
	helm upgrade odin deployments/helm/odin

helm-uninstall:
	@echo "Uninstalling Helm chart..."
	helm uninstall odin

generate-token:
	@echo "Generating JWT token..."
	cd test/auth && go run jwt-generator.go

install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

security-scan:
	@echo "Running security scan..."
	gosec ./...

build-plugins:
	@echo "Building example plugins..."
	@mkdir -p plugins
	go build -buildmode=plugin -o plugins/request_logger.so examples/plugins/request_logger.go

clean-plugins:
	@echo "Cleaning plugin artifacts..."
	rm -rf plugins/*.so

build-all-tools:
	@echo "Building Odin API Gateway and migration tool..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) cmd/odin/main.go
	go build $(LDFLAGS) -o bin/odin-gateway cmd/gateway/main.go
	go build $(LDFLAGS) -o bin/odin-migrate cmd/migrate/main.go
	@echo "✅ Built: bin/$(BINARY_NAME), bin/odin-gateway, bin/odin-migrate"

build-all: build build-plugins
	@echo "Built gateway and plugins successfully"

help:
	@echo "Odin API Gateway Make commands:"
	@echo "  build                 - Build the binary"
	@echo "  run                   - Build and run the binary"
	@echo "  test                  - Run all tests"
	@echo "  test-coverage         - Run tests with coverage report"
	@echo "  test-unit             - Run unit tests only"
	@echo "  test-integration      - Run integration tests only"
	@echo "  test-oauth2           - Run OAuth2 tests"
	@echo "  test-circuit-breaker  - Run circuit breaker tests"
	@echo "  test-websocket        - Run WebSocket tests"
	@echo "  lint                  - Run linter"
	@echo "  clean                 - Clean build artifacts"
	@echo "  docker                - Build Docker image"
	@echo "  docker-compose        - Start all services with Docker Compose"
	@echo "  docker-compose-dev    - Start development environment"
	@echo "  helm-package          - Package Helm chart"
	@echo "  helm-install          - Install Helm chart"
	@echo "  helm-upgrade          - Upgrade Helm chart"
	@echo "  helm-uninstall        - Uninstall Helm chart"
	@echo "  generate-token        - Generate JWT token for testing"
	@echo "  install-tools         - Install development tools"
	@echo "  benchmark             - Run performance benchmarks"
	@echo "  security-scan         - Run security analysis"
	@echo "  build-plugins         - Build example plugins"
	@echo "  clean-plugins         - Clean plugin artifacts"
	@echo "  build-all             - Build gateway and plugins"

ci-local:
	@echo "Running local CI simulation..."
	@./scripts/ci-local.sh

ci-quick:
	@echo "Quick CI check..."
	@go mod verify
	@go build ./...
	@go test ./...
	@golangci-lint run --fast

pre-commit: ci-quick
	@echo "Pre-commit checks passed!"
