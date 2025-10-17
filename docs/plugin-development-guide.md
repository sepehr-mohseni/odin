# Odin API Gateway Plugin System Guide

## Overview

The Odin API Gateway plugin system allows you to extend gateway functionality through dynamically loaded Go plugins. Similar to Traefik's middleware system, plugins can be registered and applied to specific routes or globally.

## Plugin Types

### 1. Hook-based Plugins
Traditional lifecycle hooks that execute at specific points in request processing:
- **Pre-Request**: Before forwarding to backend
- **Post-Request**: After receiving backend response
- **Pre-Response**: Before sending response to client
- **Post-Response**: After sending response to client

### 2. Middleware Plugins (Traefik-style)
Wrap HTTP requests in a middleware chain, similar to Traefik:
- Full control over request/response cycle
- Can short-circuit requests
- Chain multiple middlewares
- Compatible with Echo framework

## Quick Start

### Creating Your First Plugin

#### Option 1: Using the Admin Panel (Easiest)

1. Navigate to `http://localhost:8080/admin/plugins/new`
2. Choose "From Template"
3. Select a template (Auth, Rate Limit, Logging, Transform, Cache)
4. Configure the plugin
5. Click "Create Plugin" - it builds automatically!

#### Option 2: Upload Pre-built .so File

1. Build your plugin: `go build -buildmode=plugin -o myplugin.so plugin.go`
2. Go to `http://localhost:8080/admin/plugins/new`
3. Choose "Upload .so File"
4. Upload your binary
5. Configure and enable

#### Option 3: Manual Development

Create a plugin from scratch following the examples below.

## Hook-based Plugin Development

### Basic Structure

```go
package main

import (
	"context"
)

// PluginContext is provided by Odin
type PluginContext struct {
	RequestID   string
	ServiceName string
	Path        string
	Method      string
	Headers     map[string][]string
	Body        []byte
	UserID      string
	Metadata    map[string]interface{}
}

// Your plugin struct
type MyPlugin struct {
	config map[string]interface{}
}

// Export as "Plugin"
var Plugin MyPlugin

// Required interface methods
func (p *MyPlugin) Name() string {
	return "my-plugin"
}

func (p *MyPlugin) Version() string {
	return "1.0.0"
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Your pre-request logic here
	return nil
}

func (p *MyPlugin) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Your post-request logic here
	return nil
}

func (p *MyPlugin) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	// Your pre-response logic here
	return nil
}

func (p *MyPlugin) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	// Your post-response logic here
	return nil
}

func (p *MyPlugin) Cleanup() error {
	// Cleanup resources
	return nil
}
```

### Building the Plugin

```bash
go build -buildmode=plugin -o myplugin.so plugin.go
```

## Middleware Plugin Development

### Basic Structure

```go
package main

import (
	"github.com/labstack/echo/v4"
)

// Your middleware struct
type MyMiddleware struct {
	config map[string]interface{}
}

// Export as "Middleware"
var Middleware MyMiddleware

// Required interface methods
func (m *MyMiddleware) Name() string {
	return "my-middleware"
}

func (m *MyMiddleware) Version() string {
	return "1.0.0"
}

func (m *MyMiddleware) Initialize(config map[string]interface{}) error {
	m.config = config
	return nil
}

// Main middleware handler - wraps the next handler
func (m *MyMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Pre-processing
		// You can:
		// - Modify request
		// - Check authentication
		// - Rate limit
		// - Short-circuit (return error)
		
		// Call next handler
		err := next(c)
		
		// Post-processing
		// You can:
		// - Modify response
		// - Log metrics
		// - Add headers
		
		return err
	}
}

func (m *MyMiddleware) Cleanup() error {
	return nil
}
```

### Building the Middleware

```bash
go build -buildmode=plugin -o mymiddleware.so middleware.go
```

## Example Plugins

### 1. Authentication Plugin (Hook-based)

```go
package main

import (
	"context"
	"fmt"
	"strings"
)

type PluginContext struct {
	RequestID   string
	ServiceName string
	Path        string
	Method      string
	Headers     map[string][]string
	Body        []byte
	UserID      string
	Metadata    map[string]interface{}
}

type AuthPlugin struct {
	config map[string]interface{}
	apiKey string
}

var Plugin AuthPlugin

func (p *AuthPlugin) Name() string {
	return "auth-plugin"
}

func (p *AuthPlugin) Version() string {
	return "1.0.0"
}

func (p *AuthPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	
	// Get API key from config
	if key, ok := config["api_key"].(string); ok {
		p.apiKey = key
	}
	
	return nil
}

func (p *AuthPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Check Authorization header
	authHeader := pluginCtx.Headers["Authorization"]
	if len(authHeader) == 0 {
		return fmt.Errorf("missing Authorization header")
	}
	
	token := strings.TrimPrefix(authHeader[0], "Bearer ")
	if token == "" {
		return fmt.Errorf("invalid token format")
	}
	
	// Validate token (simplified - use proper validation in production)
	if token != p.apiKey {
		return fmt.Errorf("invalid token")
	}
	
	// Add user context
	pluginCtx.UserID = "authenticated-user"
	pluginCtx.Metadata["authenticated"] = true
	pluginCtx.Metadata["auth_time"] = time.Now()
	
	return nil
}

func (p *AuthPlugin) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *AuthPlugin) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *AuthPlugin) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *AuthPlugin) Cleanup() error {
	return nil
}
```

### 2. Rate Limiting Middleware

```go
package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type RateLimiter struct {
	config   map[string]interface{}
	counters map[string]*counter
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

type counter struct {
	count     int
	resetTime time.Time
}

var Middleware RateLimiter

func (r *RateLimiter) Name() string {
	return "rate-limiter"
}

func (r *RateLimiter) Version() string {
	return "1.0.0"
}

func (r *RateLimiter) Initialize(config map[string]interface{}) error {
	r.config = config
	r.counters = make(map[string]*counter)
	
	// Get limit from config
	if limit, ok := config["limit"].(float64); ok {
		r.limit = int(limit)
	} else {
		r.limit = 100 // default
	}
	
	// Get window from config
	if window, ok := config["window_seconds"].(float64); ok {
		r.window = time.Duration(window) * time.Second
	} else {
		r.window = time.Minute // default
	}
	
	return nil
}

func (r *RateLimiter) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get client identifier (IP address)
		clientIP := c.RealIP()
		
		r.mu.Lock()
		defer r.mu.Unlock()
		
		now := time.Now()
		cnt, exists := r.counters[clientIP]
		
		if !exists || now.After(cnt.resetTime) {
			// New window
			r.counters[clientIP] = &counter{
				count:     1,
				resetTime: now.Add(r.window),
			}
			return next(c)
		}
		
		if cnt.count >= r.limit {
			// Rate limit exceeded
			return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
				"error": "rate limit exceeded",
				"retry_after": cnt.resetTime.Sub(now).Seconds(),
			})
		}
		
		cnt.count++
		return next(c)
	}
}

func (r *RateLimiter) Cleanup() error {
	r.counters = nil
	return nil
}
```

### 3. Request Logger Middleware

```go
package main

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
)

type RequestLogger struct {
	config map[string]interface{}
}

var Middleware RequestLogger

func (l *RequestLogger) Name() string {
	return "request-logger"
}

func (l *RequestLogger) Version() string {
	return "1.0.0"
}

func (l *RequestLogger) Initialize(config map[string]interface{}) error {
	l.config = config
	return nil
}

func (l *RequestLogger) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		
		// Log request
		fmt.Printf("[%s] --> %s %s\n", 
			start.Format(time.RFC3339), 
			c.Request().Method, 
			c.Request().URL.Path)
		
		// Process request
		err := next(c)
		
		// Log response
		duration := time.Since(start)
		status := c.Response().Status
		
		fmt.Printf("[%s] <-- %s %s %d (%v)\n",
			time.Now().Format(time.RFC3339),
			c.Request().Method,
			c.Request().URL.Path,
			status,
			duration)
		
		return err
	}
}

func (l *RequestLogger) Cleanup() error {
	return nil
}
```

## Registering Plugins

### Via Admin UI

1. Go to `http://localhost:8080/admin/plugins`
2. Click "Add Plugin"
3. Fill in the form:
   - **Name**: Unique identifier
   - **Version**: Semantic version
   - **Plugin Source**: Upload, Path, or Template
   - **Plugin Type**: Hooks or Middleware
   - **Configuration**: JSON configuration
   - **Applied To Routes**: Specific routes or leave empty for all
4. Enable the plugin

### Via API

```bash
# Create plugin from uploaded file
curl -X POST http://localhost:8080/admin/api/plugins/upload \
  -F "file=@myplugin.so" \
  -F 'metadata={
    "name": "my-plugin",
    "version": "1.0.0",
    "pluginType": "hooks",
    "hooks": ["pre-request"],
    "enabled": true,
    "config": {}
  }'

# Build from template
curl -X POST http://localhost:8080/admin/api/plugins/build \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-auth-plugin",
    "version": "1.0.0",
    "template": "auth",
    "pluginType": "hooks",
    "hooks": ["pre-request"],
    "enabled": true,
    "config": {
      "api_key": "secret-key"
    }
  }'
```

## Testing Plugins

Before enabling in production, test your plugin:

```bash
curl -X POST http://localhost:8080/admin/api/plugins/test/my-plugin \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GET",
    "path": "/api/users",
    "headers": {
      "Authorization": ["Bearer test-token"]
    },
    "body": ""
  }'
```

Response:
```json
{
  "plugin": "my-plugin",
  "results": {
    "pre-request": {
      "success": true,
      "error": ""
    }
  },
  "context": {
    "...": "..."
  }
}
```

## Plugin Configuration

### Configuration Format

Plugins receive configuration as a map:

```json
{
  "api_key": "secret",
  "timeout": 30,
  "enabled_features": ["feature1", "feature2"],
  "nested": {
    "setting": "value"
  }
}
```

### Accessing Configuration

```go
func (p *MyPlugin) Initialize(config map[string]interface{}) error {
	// String
	if apiKey, ok := config["api_key"].(string); ok {
		p.apiKey = apiKey
	}
	
	// Number
	if timeout, ok := config["timeout"].(float64); ok {
		p.timeout = int(timeout)
	}
	
	// Array
	if features, ok := config["enabled_features"].([]interface{}); ok {
		for _, f := range features {
			if feature, ok := f.(string); ok {
				p.features = append(p.features, feature)
			}
		}
	}
	
	// Nested object
	if nested, ok := config["nested"].(map[string]interface{}); ok {
		if setting, ok := nested["setting"].(string); ok {
			p.setting = setting
		}
	}
	
	return nil
}
```

## Route Filtering

Apply plugins to specific routes:

### Via Admin UI

In the "Applied To Routes" field:
```
/api/users
/api/products/*
/api/v1/**
```

### Wildcard Patterns

- `/api/users` - Exact match
- `/api/users/*` - Single-level wildcard
- `/api/v1/**` - Multi-level wildcard

## Best Practices

### 1. Error Handling

```go
func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Return errors to reject requests
	if !isValid(pluginCtx) {
		return fmt.Errorf("validation failed: %s", reason)
	}
	
	// Log warnings but don't fail
	if hasWarning(pluginCtx) {
		log.Warn("Warning condition detected")
	}
	
	return nil // Success
}
```

### 2. Use Plugin Context Metadata

```go
func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Store data for later hooks
	pluginCtx.Metadata["start_time"] = time.Now()
	pluginCtx.Metadata["user_id"] = extractUserID(pluginCtx)
	return nil
}

func (p *MyPlugin) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	// Retrieve stored data
	if startTime, ok := pluginCtx.Metadata["start_time"].(time.Time); ok {
		duration := time.Since(startTime)
		log.Printf("Request took %v", duration)
	}
	return nil
}
```

### 3. Thread Safety

```go
type MyPlugin struct {
	config  map[string]interface{}
	counter int
	mu      sync.Mutex // Protect shared state
}

func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	p.mu.Lock()
	p.counter++
	count := p.counter
	p.mu.Unlock()
	
	log.Printf("Request #%d", count)
	return nil
}
```

### 4. Resource Cleanup

```go
type MyPlugin struct {
	db     *sql.DB
	cache  *redis.Client
	ticker *time.Ticker
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
	// Initialize resources
	p.db, _ = sql.Open("postgres", connStr)
	p.cache = redis.NewClient(&redis.Options{})
	p.ticker = time.NewTicker(time.Minute)
	
	// Start background tasks
	go p.cleanupTask()
	
	return nil
}

func (p *MyPlugin) Cleanup() error {
	// Clean up resources
	if p.ticker != nil {
		p.ticker.Stop()
	}
	if p.db != nil {
		p.db.Close()
	}
	if p.cache != nil {
		p.cache.Close()
	}
	return nil
}
```

### 5. Graceful Degradation

```go
func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Try to enhance request, but don't fail if unable
	if err := p.enrichWithUserData(pluginCtx); err != nil {
		log.Warn("Failed to enrich request: %v", err)
		// Continue anyway
	}
	
	// Only fail for critical errors
	if err := p.validateAuth(pluginCtx); err != nil {
		return err // This is critical
	}
	
	return nil
}
```

## Debugging Plugins

### Enable Verbose Logging

```go
func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	if p.config["debug"].(bool) {
		log.Printf("Processing request: %+v", pluginCtx)
	}
	// ... rest of logic
}
```

### Test Endpoint

Use the test endpoint to debug without affecting production:

```bash
curl -X POST http://localhost:8080/admin/api/plugins/test/my-plugin \
  -H "Content-Type: application/json" \
  -d '{
    "method": "POST",
    "path": "/api/users",
    "headers": {
      "Content-Type": ["application/json"],
      "Authorization": ["Bearer token"]
    },
    "body": "{\"name\":\"test\"}"
  }'
```

## Performance Considerations

### 1. Minimize Allocations

```go
// Bad - allocates on every request
func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	pattern := regexp.MustCompile(`\d+`) // Compiles every time!
	// ...
}

// Good - compile once
type MyPlugin struct {
	pattern *regexp.Regexp
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
	p.pattern = regexp.MustCompile(`\d+`) // Compile once
	return nil
}
```

### 2. Use Context Deadlines

```go
func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Respect context deadlines
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return p.externalCall(ctx)
}
```

### 3. Cache Expensive Operations

```go
type MyPlugin struct {
	cache map[string]interface{}
	mu    sync.RWMutex
}

func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	key := pluginCtx.Path
	
	// Check cache first
	p.mu.RLock()
	if cached, ok := p.cache[key]; ok {
		p.mu.RUnlock()
		pluginCtx.Metadata["cached"] = cached
		return nil
	}
	p.mu.RUnlock()
	
	// Compute
	result := p.expensiveOperation(pluginCtx)
	
	// Store in cache
	p.mu.Lock()
	p.cache[key] = result
	p.mu.Unlock()
	
	return nil
}
```

## Security Considerations

### 1. Validate Input

```go
func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Validate all inputs
	if !isValidPath(pluginCtx.Path) {
		return fmt.Errorf("invalid path")
	}
	
	if !isValidMethod(pluginCtx.Method) {
		return fmt.Errorf("invalid method")
	}
	
	// Sanitize user input
	pluginCtx.Path = sanitize(pluginCtx.Path)
	
	return nil
}
```

### 2. Protect Sensitive Data

```go
func (p *MyPlugin) Initialize(config map[string]interface{}) error {
	// Store sensitive data securely
	if apiKey, ok := config["api_key"].(string); ok {
		// Don't log sensitive data
		p.apiKey = apiKey
		log.Info("API key configured") // Don't log the actual key!
	}
	return nil
}
```

### 3. Rate Limit External Calls

```go
type MyPlugin struct {
	limiter *rate.Limiter
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
	// Limit external API calls
	p.limiter = rate.NewLimiter(rate.Every(time.Second), 10)
	return nil
}

func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Wait for rate limiter
	if err := p.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limit exceeded")
	}
	
	return p.callExternalAPI(ctx)
}
```

## Troubleshooting

### Plugin Won't Load

**Error**: `failed to open plugin: plugin was built with a different version of package X`

**Solution**: Rebuild plugin with same Go version and dependencies as gateway:
```bash
# Check gateway Go version
go version

# Rebuild plugin
go build -buildmode=plugin -o plugin.so plugin.go
```

### Symbol Not Found

**Error**: `plugin does not export 'Plugin' symbol`

**Solution**: Ensure you export the plugin variable:
```go
var Plugin MyPlugin // Must be exported (capital P)
```

### Interface Not Satisfied

**Error**: `Plugin symbol is not of type Plugin`

**Solution**: Implement all required methods with exact signatures.

## Advanced Topics

### Dynamic Route Registration

Plugins can request routes to be added dynamically:

```go
func (p *MyPlugin) Initialize(config map[string]interface{}) error {
	// Request a custom route
	p.config["custom_routes"] = []map[string]interface{}{
		{
			"path":    "/plugin/status",
			"method":  "GET",
			"handler": "GetStatus",
		},
	}
	return nil
}
```

### Plugin Dependencies

Plugins can declare dependencies on other plugins:

```go
func (p *MyPlugin) Initialize(config map[string]interface{}) error {
	// Check if required plugin is loaded
	if _, ok := config["auth_plugin"]; !ok {
		return fmt.Errorf("requires auth-plugin to be loaded first")
	}
	return nil
}
```

### Metrics Collection

```go
func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Track metrics
	pluginCtx.Metadata["metrics_start"] = time.Now()
	return nil
}

func (p *MyPlugin) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	if start, ok := pluginCtx.Metadata["metrics_start"].(time.Time); ok {
		duration := time.Since(start)
		// Export to Prometheus, etc.
		p.recordMetric("request_duration", duration.Seconds())
	}
	return nil
}
```

## Examples Repository

Check out `/var/odin/plugins/examples/` for complete working examples:

- `auth-plugin/` - Full authentication plugin
- `rate-limiter/` - Rate limiting middleware
- `cache-plugin/` - Response caching
- `transform-plugin/` - Request/response transformation
- `logger-plugin/` - Structured logging

## API Reference

### Plugin Management Endpoints

- `GET /admin/api/plugins` - List all plugins
- `POST /admin/api/plugins` - Create plugin from path
- `POST /admin/api/plugins/upload` - Upload plugin binary
- `POST /admin/api/plugins/build` - Build from template
- `GET /admin/api/plugins/:name` - Get plugin details
- `PUT /admin/api/plugins/:name` - Update plugin
- `DELETE /admin/api/plugins/:name` - Delete plugin
- `POST /admin/api/plugins/:name/enable` - Enable plugin
- `POST /admin/api/plugins/:name/disable` - Disable plugin
- `POST /admin/api/plugins/:name/load` - Load plugin
- `POST /admin/api/plugins/:name/unload` - Unload plugin
- `POST /admin/api/plugins/test/:name` - Test plugin

## Contributing

Share your plugins with the community! Submit to the Odin Plugin Registry.

## Support

- Documentation: `/docs/plugins`
- Examples: `/var/odin/plugins/examples/`
- Issues: GitHub Issues
- Community: Discord/Forum

---

**Happy Plugin Development!** 🚀
