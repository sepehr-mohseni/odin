# Odin API Gateway

[![CI](https://github.com/YOUR_USERNAME/odin/actions/workflows/ci.yml/badge.svg)](https://github.com/YOUR_USERNAME/odin/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/YOUR_USERNAME/odin/branch/main/graph/badge.svg)](https://codecov.io/gh/YOUR_USERNAME/odin)
[![Go Report Card](https://goreportcard.com/badge/github.com/YOUR_USERNAME/odin)](https://goreportcard.com/report/github.com/YOUR_USERNAME/odin)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A high-performance, feature-rich API Gateway written in Go.

## Features

- **Load Balancing**: Round-robin and random algorithms
- **Authentication**: JWT and OAuth2 support
- **Rate Limiting**: Multiple algorithms (token bucket, sliding window, fixed window)
- **Caching**: TTL, conditional, and user context strategies
- **Circuit Breaker**: Fault tolerance and resilience
- **WebSocket Proxying**: Full WebSocket support
- **Response Aggregation**: Combine multiple service responses
- **Admin Interface**: Web-based configuration management
- **Monitoring**: Prometheus metrics integration
- **Health Checks**: Readiness and liveness probes

## Quick Start

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/odin.git
cd odin

# Build the binary
go build -o bin/odin cmd/odin/main.go

# Run with default configuration
./bin/odin -config=config/config.yaml
```

## Configuration

Create a `config/config.yaml` file:

```yaml
server:
  port: 8080
  timeout: 30s

logging:
  level: info
  json: false

services:
  - name: 'user-service'
    basePath: '/api/users'
    targets:
      - 'http://localhost:3001'
    authentication: true
    loadBalancing: 'round-robin'
```

## Documentation

- [Configuration Guide](docs/configuration.md)
- [API Reference](docs/api.md)
- [Deployment Guide](docs/deployment.md)
- [Contributing](CONTRIBUTING.md)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
