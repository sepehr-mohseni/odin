# Installation Guide

This guide covers different ways to install and set up the Odin API Gateway.

## Prerequisites

- Go 1.25 or later
- Git
- Docker and Docker Compose (optional, for containerized deployment)

## Installing from Source

### 1. Clone the Repository

```bash
git clone https://github.com/sepehr-mohseni/odin.git
cd odin
```

### 2. Build the Gateway

```bash
# Install dependencies
go mod download

# Build the binary
go build -o bin/odin cmd/odin/main.go

# Or use the provided Makefile
make build
```

### 3. Configuration

```bash
# Copy the example configuration
cp config/default_config.yaml config/config.yaml

# Copy the auth secrets template and edit with your secure JWT key
cp config/auth_secrets.yaml.example config/auth_secrets.yaml
```

Edit `config/config.yaml` to suit your needs.

### 4. Run the Gateway

```bash
./bin/odin --config config/config.yaml
```

## Docker Installation

### Using Pre-built Image

```bash
docker pull sepehr-mohseni/odin:latest

docker run -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/config:/app/config \
  sepehr-mohseni/odin:latest
```

### Building Custom Image

```bash
# Build the image
docker build -t odin-gateway .

# Run the container
docker run -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/config:/app/config \
  odin-gateway
```

## Docker Compose Setup

For a complete development environment with Redis, Prometheus, and Grafana:

```bash
# Start all services
docker-compose up -d

# Or for production setup
docker-compose -f deployments/docker/docker-compose.prod.yml up -d
```

### Accessing Services

- API Gateway: http://localhost:8080
- Admin Interface: http://localhost:8081
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000

## Development Environment

For setting up a development environment:

```bash
# Install dependencies
go mod download

# Install development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run the test services
cd test/services
npm install
npm run start:all
```

## Verifying Installation

After installation, verify your setup:

```bash
# Check the health endpoint
curl http://localhost:8080/health

# Check metrics
curl http://localhost:8080/metrics
```

## Next Steps

- [Configuration Guide](configuration.md)
- [Tutorial: Your First API](tutorial.md)
- [Security Guide](security.md)
