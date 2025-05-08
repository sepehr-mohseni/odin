# Authentication & Authorization

This guide explains how to configure and use authentication and authorization features in Odin API Gateway.

## Overview

Odin provides the following authentication mechanisms:

1. **JWT (JSON Web Token)** - Token-based authentication for APIs
2. **Basic Authentication** - For the admin interface
3. **Custom Authentication** - Via middleware extensions

## JWT Authentication

### Configuration

Configure JWT authentication in your `config.yaml`:

```yaml
auth:
  jwtSecret: 'your-secure-secret-here' # Use a strong secret in production!
  accessTokenTTL: 1h # Access token time-to-live
  refreshTokenTTL: 24h # Refresh token time-to-live
  ignorePathRegexes: # Paths to exclude from authentication
    - ^/health$
    - ^/metrics$
    - ^/api/public/.*$
```

For enhanced security, we recommend:

1. Setting the JWT secret via environment variable: `ODIN_JWT_SECRET`
2. Storing the secret in a secrets manager (for Kubernetes: use secrets)

### Service-Level Authentication

Enable authentication for specific services in `services.yaml`:

```yaml
services:
  - name: users-service
    basePath: /api/users
    targets:
      - http://users-service:8081
    authentication: true # Enable JWT authentication for this service
```

### Token Structure

Odin expects JWTs with the following claims:

```json
{
  "user_id": "usr-123",
  "username": "john_doe",
  "role": "admin",
  "exp": 1639340000,
  "iat": 1639336400
}
```

Required claims:

- `user_id`: Unique identifier for the user
- `role`: User role for authorization
- `exp`: Expiration time

### Generating Tokens

Use the JWT utility in Odin to generate tokens:

```bash
# Using the CLI tool
go run cmd/tools/jwt-generator.go --user-id=usr-123 --username=john_doe --role=admin

# Using the API
curl -X POST http://localhost:8080/admin/auth/token \
  -H "Content-Type: application/json" \
  -d '{"user_id": "usr-123", "username": "john_doe", "role": "admin"}'
```

### Using Tokens

Include the JWT token in the `Authorization` header:

```bash
curl -X GET http://localhost:8080/api/protected-resource \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Role-Based Access Control

### Configure RBAC

Define roles and permissions in your configuration:

```yaml
auth:
  # ...JWT configuration...
  roles:
    admin:
      - '*' # Full access
    user:
      - 'GET:/api/products/*'
      - 'GET:/api/categories/*'
      - 'GET,POST:/api/users/{user_id}' # Where {user_id} matches JWT user_id
    guest:
      - 'GET:/api/products'
```

### Path Variables and Claims Matching

For paths with variables like `/api/users/{user_id}`, Odin can match the variable against JWT claims:

```yaml
authz:
  claimMatching:
    enabled: true
    rules:
      - pathVar: 'user_id'
        claim: 'user_id'
```

This ensures users can only access their own resources (unless they have admin privileges).

## Admin Authentication

The admin interface uses basic authentication:

```yaml
admin:
  enabled: true
  username: admin # Change this in production!
  password: admin123 # Change this in production!
```

For production, use strong passwords or integrate with an identity provider.

## Custom Authentication

You can implement custom authentication by:

1. Creating a middleware file in `pkg/middleware/`
2. Implementing the Echo middleware interface
3. Registering your middleware in the gateway initialization

Example custom authentication middleware:

```go
func CustomAuthMiddleware(config YourConfig) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Your authentication logic here

            if !authenticated {
                return echo.NewHTTPError(http.StatusUnauthorized, "Authentication failed")
            }

            return next(c)
        }
    }
}
```

## Troubleshooting Authentication

Common issues and solutions:

1. **Invalid token errors**: Check if the token is expired or uses the wrong secret
2. **Authentication bypass**: Verify your `ignorePathRegexes` aren't too permissive
3. **JWT verification failures**: Ensure the same signing method and secret are used

## Security Best Practices

1. Always use HTTPS in production
2. Rotate JWT secrets periodically
3. Set appropriate token expiration times (short for access tokens)
4. Use environment variables for secrets, not configuration files
5. Implement rate limiting to prevent brute force attacks
6. Consider using a proper identity provider for production systems
