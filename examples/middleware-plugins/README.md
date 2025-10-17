# Example Middleware Plugins

This directory contains example middleware plugins for the Odin API Gateway demonstrating various middleware patterns and use cases.

## Available Examples

### 1. Request Logger (`request-logger`)
A simple middleware that logs all incoming requests with timing information.

**Features:**
- Logs request method, path, and client IP
- Tracks response time
- Configurable log level and prefix
- Color-coded status responses

### 2. API Key Authentication (`api-key-auth`)
Header-based API key authentication middleware.

**Features:**
- Validates X-API-Key header
- Configurable API keys list
- Custom error messages
- Fast lookup using hash map

### 3. Rate Limiter (`rate-limiter`)
Token bucket rate limiting middleware with Redis backend.

**Features:**
- Per-IP rate limiting
- Configurable requests per second
- Redis-backed for distributed setups
- Burst allowance support

### 4. Request Transformer (`request-transformer`)
Modifies requests before forwarding to backend services.

**Features:**
- Add/remove/replace headers
- Body transformation
- URL rewriting
- Query parameter manipulation

## Building Examples

Each example can be built as a Go plugin:

```bash
cd examples/middleware-plugins/request-logger
go build -buildmode=plugin -o request-logger.so plugin.go
```

## Loading Examples

### Via Admin UI

1. Navigate to **Middleware Chain** page
2. Click **Register Middleware**
3. Upload the `.so` file
4. Configure priority, routes, and phase
5. Provide configuration JSON

### Via API

```bash
# Upload plugin
curl -X POST http://localhost:8080/admin/api/plugins/upload \
  -H "Authorization: Basic YWRtaW46YWRtaW4x" \
  -F "file=@request-logger.so" \
  -F 'metadata={"name":"request-logger","version":"1.0.0","pluginType":"middleware"}'

# Register in middleware chain
curl -X POST http://localhost:8080/admin/api/middleware/request-logger/register \
  -H "Authorization: Basic YWRtaW46YWRtaW4x" \
  -H "Content-Type: application/json" \
  -d '{
    "priority": 10,
    "routes": ["*"],
    "phase": "pre-auth",
    "config": {
      "prefix": "[API]",
      "logLevel": "info"
    }
  }'
```

## Configuration Examples

### Request Logger

```json
{
  "prefix": "[REQUEST]",
  "logLevel": "info",
  "includeHeaders": true,
  "includeBody": false
}
```

### API Key Auth

```json
{
  "keys": ["key1", "key2", "key3"],
  "headerName": "X-API-Key",
  "errorMessage": "Invalid or missing API key"
}
```

### Rate Limiter

```json
{
  "requestsPerSecond": 100,
  "burst": 50,
  "redisAddr": "localhost:6379",
  "keyPrefix": "ratelimit"
}
```

### Request Transformer

```json
{
  "addHeaders": {
    "X-Gateway": "Odin",
    "X-Processed-At": "${timestamp}"
  },
  "removeHeaders": ["X-Internal-Only"],
  "replaceHeaders": {
    "User-Agent": "Odin-Gateway/1.0"
  }
}
```

## Testing Examples

Use the middleware test endpoint:

```bash
curl -X POST http://localhost:8080/admin/api/middleware/request-logger/test \
  -H "Authorization: Basic YWRtaW46YWRtaW4x" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GET",
    "path": "/api/test",
    "headers": {
      "X-Test": "value"
    }
  }'
```

## Best Practices

1. **Priority Ordering**: 
   - Security middleware: 0-50
   - Authentication: 50-150
   - Logging: 150-300
   - Business logic: 300-500
   - Response processing: 500-800

2. **Route Patterns**:
   - Use specific routes when possible: `/api/v1/*`
   - Avoid overly broad patterns: `*`
   - Combine patterns for flexibility: `["/api/*", "/webhooks/*"]`

3. **Error Handling**:
   - Always return proper HTTP errors
   - Don't panic - return errors gracefully
   - Log errors for debugging

4. **Performance**:
   - Keep middleware lightweight
   - Avoid blocking operations
   - Use connection pooling for external services
   - Implement proper timeouts

## Development Workflow

1. **Create** plugin using one of the examples as template
2. **Build** as Go plugin (`.so` file)
3. **Test** locally with test endpoint
4. **Upload** via admin UI or API
5. **Register** in middleware chain
6. **Monitor** health and metrics
7. **Adjust** priority and routes as needed
8. **Rollback** if issues arise

## Troubleshooting

### Plugin Won't Load

- Ensure Go version matches gateway
- Check for missing dependencies
- Verify plugin export symbol
- Review initialization logic

### Middleware Not Executing

- Check route patterns match request path
- Verify middleware is enabled
- Check priority ordering
- Review health status

### Performance Issues

- Check metrics for slow middleware
- Review execution time
- Consider route-specific application
- Optimize configuration

## Additional Resources

- [Middleware Development Guide](../../docs/middleware-plugin-development.md)
- [Plugin System Documentation](../../docs/plugins.md)
- [API Documentation](../../docs/api.md)
