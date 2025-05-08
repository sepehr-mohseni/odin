# Project Structure

This document outlines the structure of the Odin API Gateway codebase to help contributors understand the organization.

## Directory Structure

```
odin/
├── api/                  # API definitions (OpenAPI/Swagger)
├── cmd/                  # Command line applications
│   └── odin/             # Main API gateway application
├── config/               # Configuration files and examples
├── deployments/          # Deployment configurations
│   ├── docker/           # Docker-specific configuration
│   └── kubernetes/       # Kubernetes manifests
├── docs/                 # Documentation
├── monitoring/           # Monitoring configuration
│   ├── grafana/          # Grafana dashboards
│   └── prometheus/       # Prometheus configuration
├── pkg/                  # Reusable packages
│   ├── admin/            # Admin interface
│   ├── aggregator/       # Response aggregation
│   ├── auth/             # Authentication
│   ├── cache/            # Caching
│   ├── config/           # Configuration loading
│   ├── gateway/          # Core gateway functionality
│   ├── health/           # Health checks
│   ├── logging/          # Logging
│   ├── middleware/       # Request middleware
│   ├── monitoring/       # Metrics and monitoring
│   ├── proxy/            # Request proxying
│   ├── routing/          # Request routing
│   ├── service/          # Service management
│   └── utils/            # Utility functions
└── test/                 # Test code and mock services
    ├── auth/             # Authentication tests
    ├── integration/      # Integration tests
    ├── postman/          # Postman collections
    ├── services/         # Mock microservices
    └── unit/             # Unit tests
```

## Key Components

### Gateway Core

The heart of Odin is in `pkg/gateway`, which initializes and coordinates all other components.

### API Routing

Request routing is handled in `pkg/routing`, which determines where requests should be forwarded.

### Service Configuration

Service definitions and configuration are managed in `pkg/service` and loaded from YAML config files.

### Authentication and Authorization

JWT-based authentication is provided by `pkg/auth`.

### Response Processing

- `pkg/aggregator`: Combines multiple service responses
- `pkg/middleware`: Processes requests and responses with middleware patterns

### Monitoring and Telemetry

- `pkg/monitoring`: Prometheus metrics
- `pkg/logging`: Structured logging

### Administration

`pkg/admin`: Provides a web-based admin interface for configuration

## Data Flow

1. Requests enter through the main Echo server
2. Middleware processes the request (auth, logging, metrics)
3. The router determines destination service
4. The proxy forwards the request to backend service
5. Response is processed (transform, aggregate)
6. Response is returned to client
