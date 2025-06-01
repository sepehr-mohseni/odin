# Odin API Gateway - Package Documentation

This directory contains the core packages for the Odin API Gateway.

## Package Overview

### Core Packages

- **`admin/`** - Web-based administration interface
- **`auth/`** - Authentication and authorization components (JWT, OAuth2)
- **`gateway/`** - Core gateway functionality and request routing
- **`cache/`** - Response caching strategies and implementations
- **`ratelimit/`** - Rate limiting algorithms and middleware
- **`websocket/`** - WebSocket proxy functionality
- **`circuit/`** - Circuit breaker implementation
- **`middleware/`** - Common middleware components
- **`errors/`** - Error handling utilities

### Package Dependencies

```
gateway/
├── auth/         (authentication)
├── cache/        (response caching)
├── ratelimit/    (request limiting)
├── websocket/    (websocket proxying)
├── circuit/      (circuit breaker)
├── middleware/   (common middleware)
└── errors/       (error handling)

admin/
├── auth/         (admin authentication)
└── gateway/      (configuration management)
```

## Usage Examples

### Basic Gateway Setup

```go
import (
    "odin/pkg/gateway"
    "odin/pkg/auth"
    "odin/pkg/cache"
)

// Initialize gateway with authentication and caching
gw := gateway.New(config)
gw.Use(auth.JWTMiddleware(authConfig))
gw.Use(cache.Middleware(cacheConfig))
```

### Admin Interface

```go
import "odin/pkg/admin"

// Start admin interface
adminHandler := admin.NewHandler(config)
adminHandler.Start(":8081")
```

## Testing

Each package includes comprehensive unit tests. Run tests with:

```bash
go test ./pkg/...
```

## Contributing

When adding new packages:

1. Follow the existing package structure
2. Include comprehensive tests
3. Add package documentation
4. Update this README with package descriptions
