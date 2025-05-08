# Configuration Guide

Odin API Gateway is configured using YAML files stored in the `config/` directory.

## Core Configuration

The main configuration file (`config.yaml`) defines the gateway's behavior:

```yaml
server:
  port: 8080 # API gateway port
  readTimeout: 5s # HTTP read timeout
  writeTimeout: 10s # HTTP write timeout
  gracefulTimeout: 15s # Graceful shutdown timeout
  compression: true # Enable response compression

logging:
  level: info # Logging level (debug, info, warn, error)
  json: false # Use JSON format for logs

auth:
  jwtSecret: 'your-secret' # JWT secret for token validation
  accessTokenTTL: 1h # Access token time-to-live
  refreshTokenTTL: 24h # Refresh token time-to-live
  ignorePathRegexes: # Paths to exclude from authentication
    - ^/health$
    - ^/metrics$
    - ^/api/public/.*$

rateLimit:
  enabled: true # Enable rate limiting
  limit: 100 # Requests per duration
  duration: 1m # Rate limit window
  strategy: local # Strategy (local, redis)
  redisUrl: 'redis://localhost:6379'

cache:
  enabled: true # Enable response caching
  ttl: 5m # Cache time-to-live
  strategy: local # Strategy (local, redis)
  maxSizeInMB: 100 # Maximum cache size (local strategy)
  redisUrl: 'redis://localhost:6379'

monitoring:
  enabled: true # Enable Prometheus metrics
  path: /metrics # Metrics endpoint

admin:
  enabled: true # Enable admin interface
  username: admin # Admin username
  password: admin # Admin password (change this!)
```

## Service Configuration

Service configurations define how API requests are routed to backend services.

```yaml
services:
  - name: users # Service name
    basePath: /api/users # Base path for routing
    stripBasePath: true # Remove base path when forwarding
    targets: # Backend service URLs
      - http://users-service:8081
      - http://users-service-replica:8081
    timeout: 5s # Request timeout
    retryCount: 3 # Retry count on failures
    retryDelay: 100ms # Delay between retries
    authentication: true # Require authentication
    loadBalancing: round-robin # Load balancing strategy

    # HTTP headers to add to forwarded requests
    headers:
      X-Source: odin-gateway
      X-Version: 1.0

    # Transform requests and responses
    transform:
      request:
        - from: $.user.id # Source field (JSONPath)
          to: $.userId # Target field
          default: 'anonymous' # Default value if source missing
      response:
        - from: $.data # Source field
          to: $.users # Target field

    # Data aggregation configuration
    aggregation:
      dependencies:
        - service: products # Service to query
          path: /api/products/by-user/{userId}
          # Map source data to path parameters
          parameterMapping:
            - from: $.id
              to: userId
          # Map response data to final response
          resultMapping:
            - from: $
              to: $.products
```

## Reloading Configuration

Configuration can be reloaded without restarting the gateway:

1. Through the admin interface
2. By sending a SIGHUP signal to the process
3. Using the API endpoint: `POST /admin/config/reload`

## Environment Variables

Configuration values can be overridden using environment variables:

- `GATEWAY_PORT`: Override server port
- `GATEWAY_LOG_LEVEL`: Override log level
- `ODIN_JWT_SECRET`: Override JWT secret

For more information, see the [Environment Variables Guide](environment-variables.md).
