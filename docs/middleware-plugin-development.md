# Middleware Plugin Development Guide

## Overview

Odin API Gateway supports dynamic middleware plugins that can be loaded at runtime to extend the request processing pipeline. Middleware plugins provide a Traefik-style approach to building composable request/response processing chains.

## Table of Contents

1. [Understanding Middleware Plugins](#understanding-middleware-plugins)
2. [Middleware Interface](#middleware-interface)
3. [Plugin Lifecycle](#plugin-lifecycle)
4. [Creating Your First Middleware](#creating-your-first-middleware)
5. [Building and Deploying](#building-and-deploying)
6. [Best Practices](#best-practices)
7. [Example Plugins](#example-plugins)
8. [Troubleshooting](#troubleshooting)

---

## Understanding Middleware Plugins

### What is a Middleware Plugin?

A middleware plugin is a Go plugin (.so file) that implements the `Middleware` interface. Unlike hook-based plugins, middleware plugins wrap the entire request handler chain, allowing you to:

- Intercept requests before they reach the handler
- Modify request/response data
- Short-circuit request processing
- Add custom logic to the processing pipeline

### Middleware vs Hooks

| Feature | Middleware | Hooks |
|---------|-----------|-------|
| **Execution Model** | Wraps handler chain | Callback at specific points |
| **Request Control** | Can short-circuit | Cannot stop request |
| **Next Handler** | Must call `next()` | Automatic continuation |
| **Use Cases** | Auth, rate limiting, logging | Request enrichment, monitoring |

---

## Middleware Interface

All middleware plugins must implement this interface:

```go
type Middleware interface {
    // Name returns the middleware name
    Name() string

    // Version returns the middleware version
    Version() string

    // Initialize initializes the middleware with configuration
    Initialize(config map[string]interface{}) error

    // Handle wraps the next handler in the chain
    Handle(next echo.HandlerFunc) echo.HandlerFunc

    // Cleanup is called when the middleware is being unloaded
    Cleanup() error
}
```

### Method Descriptions

#### `Name() string`
Returns a unique identifier for your middleware.

**Example:**
```go
func (m *MyMiddleware) Name() string {
    return "custom-auth"
}
```

#### `Version() string`
Returns the version of your middleware (semantic versioning recommended).

**Example:**
```go
func (m *MyMiddleware) Version() string {
    return "1.0.0"
}
```

#### `Initialize(config map[string]interface{}) error`
Called once when the middleware is loaded. Use this to:
- Parse configuration
- Initialize resources (database connections, caches, etc.)
- Validate settings

**Example:**
```go
func (m *MyMiddleware) Initialize(config map[string]interface{}) error {
    if apiKey, ok := config["apiKey"].(string); ok {
        m.apiKey = apiKey
    }
    
    if timeout, ok := config["timeout"].(float64); ok {
        m.timeout = time.Duration(timeout) * time.Second
    }
    
    return nil
}
```

#### `Handle(next echo.HandlerFunc) echo.HandlerFunc`
The core middleware logic. Returns a handler function that:
1. Processes the request
2. Calls `next(c)` to continue the chain
3. Processes the response

**Example:**
```go
func (m *MyMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Pre-processing
        startTime := time.Now()
        
        // Call next handler
        err := next(c)
        
        // Post-processing
        duration := time.Since(startTime)
        c.Response().Header().Set("X-Response-Time", duration.String())
        
        return err
    }
}
```

#### `Cleanup() error`
Called when the middleware is unloaded. Use this to:
- Close connections
- Release resources
- Save state

**Example:**
```go
func (m *MyMiddleware) Cleanup() error {
    if m.db != nil {
        return m.db.Close()
    }
    return nil
}
```

---

## Plugin Lifecycle

```
┌─────────────────┐
│  Plugin Loaded  │
│   (.so file)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Initialize()   │
│  - Parse config │
│  - Setup resources
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Registered in  │
│ Middleware Chain│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Handle() called│
│  for each request
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Unregister or  │
│   Hot Reload    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Cleanup()     │
│ - Close connections
│ - Release resources
└─────────────────┘
```

---

## Creating Your First Middleware

### Step 1: Create Plugin Directory

```bash
mkdir -p plugins/request-logger
cd plugins/request-logger
```

### Step 2: Create Plugin Code

**File: `plugin.go`**

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/labstack/echo/v4"
    "github.com/sirupsen/logrus"
)

// RequestLogger logs all incoming requests
type RequestLogger struct {
    logger *logrus.Logger
    prefix string
}

// Export the middleware instance
var Middleware RequestLogger

// Name returns the middleware name
func (m *RequestLogger) Name() string {
    return "request-logger"
}

// Version returns the middleware version
func (m *RequestLogger) Version() string {
    return "1.0.0"
}

// Initialize sets up the middleware
func (m *RequestLogger) Initialize(config map[string]interface{}) error {
    m.logger = logrus.New()
    
    // Parse configuration
    if prefix, ok := config["prefix"].(string); ok {
        m.prefix = prefix
    } else {
        m.prefix = "[REQUEST]"
    }
    
    if logLevel, ok := config["logLevel"].(string); ok {
        level, err := logrus.ParseLevel(logLevel)
        if err != nil {
            return fmt.Errorf("invalid log level: %w", err)
        }
        m.logger.SetLevel(level)
    }
    
    m.logger.Info("Request logger middleware initialized")
    return nil
}

// Handle processes requests
func (m *RequestLogger) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        start := time.Now()
        
        // Log request
        m.logger.WithFields(logrus.Fields{
            "method": c.Request().Method,
            "path":   c.Request().URL.Path,
            "ip":     c.RealIP(),
        }).Info(fmt.Sprintf("%s Incoming request", m.prefix))
        
        // Process request
        err := next(c)
        
        // Log response
        duration := time.Since(start)
        status := c.Response().Status
        
        logEntry := m.logger.WithFields(logrus.Fields{
            "method":   c.Request().Method,
            "path":     c.Request().URL.Path,
            "status":   status,
            "duration": duration.String(),
        })
        
        if status >= 500 {
            logEntry.Error(fmt.Sprintf("%s Request failed", m.prefix))
        } else if status >= 400 {
            logEntry.Warn(fmt.Sprintf("%s Client error", m.prefix))
        } else {
            logEntry.Info(fmt.Sprintf("%s Request completed", m.prefix))
        }
        
        return err
    }
}

// Cleanup releases resources
func (m *RequestLogger) Cleanup() error {
    m.logger.Info("Request logger middleware cleaned up")
    return nil
}

// Required for Go plugins
func main() {}
```

### Step 3: Create go.mod

```bash
go mod init request-logger
go get github.com/labstack/echo/v4
go get github.com/sirupsen/logrus
```

### Step 4: Build the Plugin

```bash
go build -buildmode=plugin -o request-logger.so plugin.go
```

### Step 5: Upload and Register

#### Via Admin UI:
1. Navigate to **Middleware Chain** page
2. Click **Register Middleware**
3. Upload `request-logger.so`
4. Configure:
   - **Priority**: 10 (executes early)
   - **Routes**: `*` (all routes) or `/api/*` (specific)
   - **Phase**: pre-auth
   - **Config**:
     ```json
     {
       "prefix": "[API]",
       "logLevel": "info"
     }
     ```

#### Via API:
```bash
# Upload plugin
curl -X POST http://localhost:8080/admin/api/plugins/upload \
  -H "Authorization: Basic YWRtaW46YWRtaW4x" \
  -F "file=@request-logger.so" \
  -F 'metadata={"name":"request-logger","version":"1.0.0","pluginType":"middleware"}'

# Register in chain
curl -X POST http://localhost:8080/admin/api/middleware/request-logger/register \
  -H "Authorization: Basic YWRtaW46YWRtaW4x" \
  -H "Content-Type: application/json" \
  -d '{
    "priority": 10,
    "routes": ["*"],
    "phase": "pre-auth"
  }'
```

---

## Building and Deploying

### Build Requirements

- Go 1.19+ (must match gateway Go version)
- CGO enabled (for plugin support)
- Same OS/architecture as gateway server

### Build Commands

**Development build:**
```bash
go build -buildmode=plugin -o middleware.so plugin.go
```

**Production build with optimizations:**
```bash
go build -buildmode=plugin -ldflags="-s -w" -o middleware.so plugin.go
```

**Cross-compilation note:**
Go plugins cannot be cross-compiled. Build on the target platform or use Docker:

```dockerfile
FROM golang:1.21-alpine

WORKDIR /build
COPY . .

RUN go build -buildmode=plugin -o middleware.so plugin.go
```

### Deployment Strategies

#### 1. Direct Upload (Development)
Upload via admin UI for quick testing.

#### 2. File System (Production)
```bash
# Copy to plugins directory
cp middleware.so /var/odin/plugins/

# Register via API
curl -X POST http://localhost:8080/admin/api/plugins \
  -d '{"name":"my-middleware","binaryPath":"/var/odin/plugins/middleware.so",...}'
```

#### 3. CI/CD Pipeline
```yaml
# Example GitHub Actions
- name: Build Plugin
  run: go build -buildmode=plugin -o middleware.so

- name: Deploy Plugin
  run: |
    scp middleware.so server:/var/odin/plugins/
    curl -X POST http://server/admin/api/middleware/reload-all
```

---

## Best Practices

### 1. Priority Ordering

Recommended priority ranges:

| Range | Use Case | Examples |
|-------|----------|----------|
| 0-50 | Critical security | CORS, security headers |
| 50-150 | Authentication | JWT validation, OAuth |
| 150-300 | Request preprocessing | Logging, request ID |
| 300-500 | Business logic | Rate limiting, API versioning |
| 500-800 | Response processing | Compression, caching |
| 800-1000 | Cleanup | Metrics, monitoring |

### 2. Error Handling

Always handle errors gracefully:

```go
func (m *MyMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Validate request
        if err := m.validateRequest(c); err != nil {
            // Return error response, don't call next
            return echo.NewHTTPError(http.StatusBadRequest, err.Error())
        }
        
        // Process request
        if err := next(c); err != nil {
            // Log error but don't swallow it
            m.logger.WithError(err).Error("Request failed")
            return err
        }
        
        return nil
    }
}
```

### 3. Configuration Validation

Validate configuration in `Initialize()`:

```go
func (m *MyMiddleware) Initialize(config map[string]interface{}) error {
    // Required field
    apiKey, ok := config["apiKey"].(string)
    if !ok || apiKey == "" {
        return fmt.Errorf("apiKey is required")
    }
    
    // Optional with default
    timeout, ok := config["timeout"].(float64)
    if !ok {
        timeout = 30.0 // default
    }
    
    if timeout < 0 || timeout > 300 {
        return fmt.Errorf("timeout must be between 0 and 300 seconds")
    }
    
    m.apiKey = apiKey
    m.timeout = time.Duration(timeout) * time.Second
    return nil
}
```

### 4. Thread Safety

Middleware instances are shared across requests:

```go
type MyMiddleware struct {
    // Safe: immutable after initialization
    config Config
    logger *logrus.Logger
    
    // Safe: concurrent-safe types
    cache sync.Map
    
    // UNSAFE: shared mutable state
    // requestCount int // DON'T DO THIS
    
    // Safe: use atomics for counters
    requestCount atomic.Int64
}
```

### 5. Route Patterns

Use specific routes for better performance:

```go
// Good: specific routes
routes: ["/api/v1/*", "/api/v2/*"]

// Less efficient: catches everything
routes: ["*"]

// Flexible: mix specific and wildcards
routes: ["/api/*", "/webhooks/*", "/public/login"]
```

### 6. Resource Management

Always clean up in `Cleanup()`:

```go
type MyMiddleware struct {
    db    *sql.DB
    cache *redis.Client
    file  *os.File
}

func (m *MyMiddleware) Cleanup() error {
    var errs []error
    
    if m.db != nil {
        if err := m.db.Close(); err != nil {
            errs = append(errs, fmt.Errorf("db close: %w", err))
        }
    }
    
    if m.cache != nil {
        if err := m.cache.Close(); err != nil {
            errs = append(errs, fmt.Errorf("cache close: %w", err))
        }
    }
    
    if m.file != nil {
        if err := m.file.Close(); err != nil {
            errs = append(errs, fmt.Errorf("file close: %w", err))
        }
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("cleanup errors: %v", errs)
    }
    
    return nil
}
```

---

## Example Plugins

### 1. API Key Authentication

```go
type APIKeyAuth struct {
    validKeys map[string]bool
}

var Middleware APIKeyAuth

func (m *APIKeyAuth) Name() string { return "api-key-auth" }
func (m *APIKeyAuth) Version() string { return "1.0.0" }

func (m *APIKeyAuth) Initialize(config map[string]interface{}) error {
    keys, ok := config["keys"].([]interface{})
    if !ok {
        return fmt.Errorf("keys configuration required")
    }
    
    m.validKeys = make(map[string]bool)
    for _, key := range keys {
        if keyStr, ok := key.(string); ok {
            m.validKeys[keyStr] = true
        }
    }
    
    return nil
}

func (m *APIKeyAuth) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        apiKey := c.Request().Header.Get("X-API-Key")
        
        if !m.validKeys[apiKey] {
            return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API key")
        }
        
        return next(c)
    }
}

func (m *APIKeyAuth) Cleanup() error { return nil }
func main() {}
```

### 2. Request Size Limiter

```go
type SizeLimiter struct {
    maxSize int64
}

var Middleware SizeLimiter

func (m *SizeLimiter) Name() string { return "size-limiter" }
func (m *SizeLimiter) Version() string { return "1.0.0" }

func (m *SizeLimiter) Initialize(config map[string]interface{}) error {
    if maxSize, ok := config["maxSizeBytes"].(float64); ok {
        m.maxSize = int64(maxSize)
    } else {
        m.maxSize = 10 * 1024 * 1024 // 10MB default
    }
    return nil
}

func (m *SizeLimiter) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        if c.Request().ContentLength > m.maxSize {
            return echo.NewHTTPError(http.StatusRequestEntityTooLarge, 
                "Request body too large")
        }
        
        // Wrap body reader with limit
        c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, m.maxSize)
        
        return next(c)
    }
}

func (m *SizeLimiter) Cleanup() error { return nil }
func main() {}
```

### 3. Response Transformer

```go
type ResponseTransformer struct {
    addHeaders map[string]string
}

var Middleware ResponseTransformer

func (m *ResponseTransformer) Name() string { return "response-transformer" }
func (m *ResponseTransformer) Version() string { return "1.0.0" }

func (m *ResponseTransformer) Initialize(config map[string]interface{}) error {
    m.addHeaders = make(map[string]string)
    
    if headers, ok := config["addHeaders"].(map[string]interface{}); ok {
        for key, value := range headers {
            if valueStr, ok := value.(string); ok {
                m.addHeaders[key] = valueStr
            }
        }
    }
    
    return nil
}

func (m *ResponseTransformer) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Process request
        err := next(c)
        
        // Add custom headers to response
        for key, value := range m.addHeaders {
            c.Response().Header().Set(key, value)
        }
        
        return err
    }
}

func (m *ResponseTransformer) Cleanup() error { return nil }
func main() {}
```

---

## Troubleshooting

### Plugin Won't Load

**Error: "plugin was built with a different version of package X"**

**Solution:** Rebuild plugin with exact same Go version and dependencies as gateway.

```bash
# Check gateway Go version
go version

# Rebuild plugin with matching version
go build -buildmode=plugin -o plugin.so
```

### Plugin Crashes Gateway

**Symptom:** Gateway crashes when loading plugin

**Solutions:**
1. Check `Initialize()` for panics - use recovery:
```go
func (m *MyMiddleware) Initialize(config map[string]interface{}) (err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("initialization panic: %v", r)
        }
    }()
    // ... initialization code
    return nil
}
```

2. Validate all type assertions:
```go
// BAD
value := config["key"].(string)

// GOOD
value, ok := config["key"].(string)
if !ok {
    return fmt.Errorf("invalid configuration")
}
```

### Middleware Not Executing

**Check:**
1. Middleware is registered in chain: `/admin/middleware-chain`
2. Routes match request path
3. Priority is correct (lower = earlier)
4. Plugin is enabled and loaded

### Performance Issues

**Profile middleware:**
```go
func (m *MyMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        start := time.Now()
        defer func() {
            duration := time.Since(start)
            if duration > 100*time.Millisecond {
                log.Printf("SLOW: %s took %v", m.Name(), duration)
            }
        }()
        
        return next(c)
    }
}
```

---

## Additional Resources

- [Odin Plugin System Documentation](../plugins.md)
- [Echo Framework Documentation](https://echo.labstack.com/)
- [Go Plugin Package](https://pkg.go.dev/plugin)
- [Example Plugins Repository](https://github.com/example/odin-plugins)

---

**Need help?** Open an issue on GitHub or join our community Discord!
