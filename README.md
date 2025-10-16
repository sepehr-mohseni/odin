<div align="center">

# ⚡ Odin API Gateway

[![CI](https://github.com/sepehr-mohseni/odin/actions/workflows/ci.yml/badge.svg)](https://github.com/sepehr-mohseni/odin/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/sepehr-mohseni/odin/branch/main/graph/badge.svg)](https://codecov.io/gh/sepehr-mohseni/odin)
[![Go Report Card](https://goreportcard.com/badge/github.com/sepehr-mohseni/odin)](https://goreportcard.com/report/github.com/sepehr-mohseni/odin)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![GitHub stars](https://img.shields.io/github/stars/sepehr-mohseni/odin?style=social)](https://github.com/sepehr-mohseni/odin/stargazers)

_A lightning-fast, feature-rich API Gateway built with Go_

[Features](#-features) • [Quick Start](#-quick-start) • [Documentation](#-documentation) • [Contributing](#-contributing)

[![Star this repository](https://img.shields.io/badge/⭐-Star%20this%20repository-yellow?style=for-the-badge&logo=github)](https://github.com/sepehr-mohseni/odin)

</div>

---

## 🚀 Features

<table>
<tr>
<td width="50%">

### 🔧 **Core Features**

- **🔄 Load Balancing** - Round-robin, random, and weighted algorithms
- **🔐 Authentication** - JWT and OAuth2 support with role-based access
- **⚡ Rate Limiting** - Token bucket, sliding window, fixed window algorithms
- **💾 Caching** - TTL, conditional, and user context strategies
- **🛡️ Circuit Breaker** - Fault tolerance and automatic recovery
- **🌐 WebSocket Proxying** - Full-duplex communication support

</td>
<td width="50%">

### 📊 **Advanced Features**

- **🔗 Response Aggregation** - Combine multiple service responses
- **⚙️ Admin Interface** - Web-based configuration management
- **📈 Monitoring** - Prometheus metrics and health checks
- **🔄 Request/Response Transformation** - JSONPath-based data mapping
- **🏗️ Service Discovery** - Dynamic service registration
- **📝 API Documentation** - Auto-generated OpenAPI specs
- **🧩 Plugin System** - Extensible architecture with custom plugins
- **📊 GraphQL Proxy** - Query validation, caching, and security
- **⚡ gRPC Support** - HTTP-to-gRPC transcoding
- **🗄️ MongoDB Integration** - Centralized storage for config and metrics
- **🔌 WASM Extensions** - Lightweight, secure plugin runtime
- **🌐 Multi-Cluster** - Global load balancing across clusters

</td>
</tr>
</table>

## 🏃 Quick Start

### Prerequisites

- Go 1.21 or higher
- Redis (optional, for distributed rate limiting and caching)
- MongoDB (optional, for persistent storage and metrics)

### Installation

```bash
# Clone the repository
git clone https://github.com/sepehr-mohseni/odin.git
cd odin

# Build the binary
make build

# Run with sample configuration
make run
```

### Docker Setup

```bash
# Build and run with Docker Compose
docker-compose up -d

# Or build just the gateway
make docker
docker run -p 8080:8080 -v ./config:/app/config odin-gateway:latest
```

## ⚙️ Configuration

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
    rateLimit:
      limit: 100
      window: '1m'

  - name: 'product-service'
    basePath: '/api/products'
    targets:
      - 'http://localhost:3002'
      - 'http://localhost:3003'
    loadBalancing: 'weighted'
    cache:
      enabled: true
      ttl: '5m'
```

### Advanced Configuration

```yaml
# Enable response aggregation
services:
  - name: 'user-profile'
    basePath: '/api/profile'
    targets:
      - 'http://users-service:8080'
    aggregation:
      dependencies:
        - service: 'user-preferences'
          path: '/preferences/{user_id}'
          parameterMapping:
            - from: '$.user.id'
              to: '{user_id}'
        - service: 'user-activity'
          path: '/activity/{user_id}/recent'
```

## 🛠️ Features in Detail

### Load Balancing

- **Round Robin**: Distributes requests evenly across all targets
- **Random**: Randomly selects a target for each request
- **Weighted**: Routes based on configured weights
- **Health Checks**: Automatic failover for unhealthy targets

### Authentication & Authorization

- **JWT Authentication**: Stateless token-based auth with configurable claims
- **OAuth2 Integration**: Support for major providers (Google, GitHub, etc.)
- **Role-Based Access Control**: Fine-grained permissions per route
- **API Key Management**: Simple API key authentication

### Rate Limiting

- **Multiple Algorithms**: Token bucket, sliding window, fixed window
- **Per-User/IP/API Key**: Flexible rate limiting strategies
- **Redis Backend**: Distributed rate limiting across multiple instances
- **Custom Headers**: Include custom headers in rate limit calculations

### Caching

- **Multiple Strategies**: TTL-based, conditional (ETag/Last-Modified), user context
- **Storage Backends**: In-memory, Redis, or custom implementations
- **Cache Invalidation**: Tag-based and pattern-based invalidation
- **Compression**: Automatic response compression

## 📊 Monitoring & Observability

### Prometheus Metrics

- Request count, duration, and size distributions
- Error rates and status code breakdowns
- Rate limiting and cache hit/miss ratios
- Circuit breaker state transitions
- Active WebSocket connections

### Health Checks

```bash
# Gateway health
curl http://localhost:8080/health

# Readiness probe
curl http://localhost:8080/ready

# Metrics endpoint
curl http://localhost:8080/metrics
```

### Admin Dashboard

Access the web-based admin interface at `http://localhost:8080/admin`

- Real-time service monitoring
- Configuration management
- Route testing and debugging
- Performance analytics

## 🔧 Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test suites
make test-unit
make test-integration
make test-oauth2
make test-websocket
```

### Benchmarks

```bash
# Run performance benchmarks
make benchmark

# Load testing
hey -n 10000 -c 100 http://localhost:8080/api/users
```

## 🚀 Deployment

### Kubernetes with Helm

```bash
# Package and install
make helm-package
make helm-install

# Or customize values
helm install odin deployments/helm/odin \
  --set ingress.enabled=true \
  --set ingress.hostname=gateway.example.com \
  --set auth.jwtSecret=your-secret-key
```

### Docker Compose

```bash
# Development environment
make docker-compose-dev

# Production environment
make docker-compose
```

## 📖 Documentation

### User Documentation
- [📋 Configuration Guide](docs/configuration.md) - Complete configuration reference
- [🔌 API Reference](docs/api.md) - REST API documentation
- [🚀 Deployment Guide](docs/deployment.md) - Production deployment strategies
- [🔧 Plugin Development](docs/plugins.md) - Extending gateway functionality
- [📊 Monitoring Setup](docs/monitoring.md) - Observability and alerting
- [🔒 Security Guide](docs/security.md) - Security best practices
- [🧠 AI Traffic Analysis](docs/ai-analysis.md) - AI-powered anomaly detection

### Project Planning
- [🗺️ Roadmap](ROADMAP.md) - Current and planned features
- [📋 Implementation Plan](docs/IMPLEMENTATION_PLAN.md) - Detailed 7-month plan (50+ pages)
- [✅ Implementation Checklist](docs/IMPLEMENTATION_CHECKLIST.md) - Progress tracking
- [📊 Visual Roadmap](docs/VISUAL_ROADMAP.md) - Timeline and milestones
- [📄 Executive Summary](docs/EXECUTIVE_SUMMARY.md) - High-level overview for stakeholders

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Start for Contributors

```bash
# Fork and clone the repository
git clone https://github.com/sepehr-mohseni/odin.git

# Install development tools
make install-tools

# Run tests and linting
make test lint

# Submit a pull request
```

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Echo](https://echo.labstack.com/) web framework
- Monitoring powered by [Prometheus](https://prometheus.io/)
- WebSocket support via [Gorilla WebSocket](https://github.com/gorilla/websocket)

---

<div align="center">

**[⭐ Star this repository](https://github.com/sepehr-mohseni/odin/stargazers)** if you find it useful!

[![Stargazers repo roster for @sepehr-mohseni/odin](https://reporoster.com/stars/sepehr-mohseni/odin)](https://github.com/sepehr-mohseni/odin/stargazers)

Made with ❤️ by [Sepehr Mohseni](https://github.com/sepehr-mohseni)

</div>
