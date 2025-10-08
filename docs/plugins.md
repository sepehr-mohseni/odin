# Plugin System

Odin API Gateway provides a flexible plugin system that allows you to extend gateway functionality with custom Go plugins.

## Overview

The plugin system allows you to:

- **Hook into request/response lifecycle** at multiple points
- **Add custom logic** for authentication, logging, transformation, etc.
- **Load plugins dynamically** without recompiling the gateway
- **Configure plugins** with custom settings

## Plugin Interface

All plugins must implement the following interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Initialize(config map[string]interface{}) error
    PreRequest(ctx context.Context, pluginCtx *PluginContext) error
    PostRequest(ctx context.Context, pluginCtx *PluginContext) error
    PreResponse(ctx context.Context, pluginCtx *PluginContext) error
    PostResponse(ctx context.Context, pluginCtx *PluginContext) error
    Cleanup() error
}
```

## Hook Points

- **PreRequest**: Called before forwarding request to backend
- **PostRequest**: Called after receiving response from backend
- **PreResponse**: Called before sending response to client
- **PostResponse**: Called after sending response to client

## Plugin Context

Plugins receive a `PluginContext` with request information:

```go
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
```

## Configuration

Enable plugins in your `config.yaml`:

```yaml
plugins:
  enabled: true
  directory: "./plugins"
  plugins:
    - name: "request_logger"
      path: "./plugins/request_logger.so"
      enabled: true
      hooks: ["pre-request", "post-response"]
      config:
        log_level: "info"
        include_headers: true
    - name: "rate_limiter"
      path: "./plugins/rate_limiter.so"
      enabled: false
      hooks: ["pre-request"]
      config:
        requests_per_minute: 100
```

## Creating a Plugin

1. **Implement the Plugin interface**:

```go
package main

import (
    "context"
    "github.com/sirupsen/logrus"
)

type MyPlugin struct {
    logger *logrus.Logger
    config map[string]interface{}
}

var Plugin MyPlugin

func (p *MyPlugin) Name() string {
    return "my_plugin"
}

func (p *MyPlugin) Version() string {
    return "1.0.0"
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    p.logger = logrus.New()
    return nil
}

func (p *MyPlugin) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
    p.logger.Infof("Processing request: %s %s", pluginCtx.Method, pluginCtx.Path)
    return nil
}

// Implement other hook methods...

func main() {}
```

2. **Build as plugin**:

```bash
go build -buildmode=plugin -o my_plugin.so my_plugin.go
```

3. **Configure in gateway**:

```yaml
plugins:
  enabled: true
  plugins:
    - name: "my_plugin"
      path: "./plugins/my_plugin.so"
      enabled: true
      hooks: ["pre-request"]
      config:
        custom_setting: "value"
```

## Example: Request Logger Plugin

See `examples/plugins/request_logger.go` for a complete example that:

- Logs incoming requests with timing information
- Configurable log level and header inclusion
- Demonstrates metadata usage for cross-hook communication

## Best Practices

- **Handle errors gracefully** - plugin errors can break request processing
- **Use context for cancellation** - respect request timeouts
- **Keep plugins lightweight** - avoid blocking operations
- **Use metadata** to share data between hooks
- **Test thoroughly** - plugin bugs affect all requests

## Security Considerations

- **Validate plugin sources** - only load trusted plugins
- **Sandbox if possible** - consider running plugins in restricted environments
- **Monitor resource usage** - plugins can impact gateway performance
- **Audit plugin access** - plugins have access to all request data