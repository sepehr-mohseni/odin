# gRPC Protocol Support

Odin API Gateway provides HTTP-to-gRPC transcoding, allowing REST clients to communicate with gRPC services through the gateway.

## Features

- **HTTP-to-gRPC Transcoding**: Convert HTTP/JSON requests to gRPC calls
- **Automatic Error Mapping**: Convert gRPC status codes to appropriate HTTP status codes
- **Metadata Conversion**: Forward HTTP headers as gRPC metadata
- **Connection Management**: Persistent gRPC connections with health checks
- **Message Size Limits**: Configurable maximum message sizes for security

## Configuration

Add a gRPC service to your `config.yaml`:

```yaml
services:
  - name: user-service
    basePath: /grpc/users
    protocol: grpc
    targets:
      - localhost:50051
    authentication: true
    timeout: 30s
    grpc:
      enableReflection: true      # Enable gRPC reflection
      maxMessageSize: 4194304     # 4MB message size limit
      enableTLS: false            # Use TLS for gRPC connection
      tlsCertFile: ""             # TLS certificate file (if TLS enabled)
      tlsKeyFile: ""              # TLS key file (if TLS enabled)
```

## Request Format

HTTP requests are transcoded to gRPC calls using this URL pattern:

```
POST /grpc/{service}/{method}
```

For example:
- `POST /grpc/users/GetUser` → calls `UserService.GetUser`
- `POST /grpc/users/CreateUser` → calls `UserService.CreateUser`

### Request Body

Send JSON in the request body that matches the protobuf message structure:

```bash
curl -X POST http://localhost:8080/grpc/users/GetUser \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "id": "12345"
  }'
```

## Response Format

Responses are returned as JSON with the protobuf message structure:

```json
{
  "id": "12345",
  "name": "John Doe",
  "email": "john@example.com",
  "created_at": "2023-01-01T00:00:00Z"
}
```

## Error Handling

gRPC errors are automatically converted to appropriate HTTP status codes:

| gRPC Code | HTTP Status | Description |
|-----------|-------------|-------------|
| OK | 200 | Success |
| INVALID_ARGUMENT | 400 | Bad Request |
| UNAUTHENTICATED | 401 | Unauthorized |
| PERMISSION_DENIED | 403 | Forbidden |
| NOT_FOUND | 404 | Not Found |
| ALREADY_EXISTS | 409 | Conflict |
| RESOURCE_EXHAUSTED | 429 | Too Many Requests |
| INTERNAL | 500 | Internal Server Error |
| UNAVAILABLE | 503 | Service Unavailable |
| DEADLINE_EXCEEDED | 504 | Gateway Timeout |

Error response format:

```json
{
  "error": "User not found",
  "code": "NOT_FOUND",
  "details": []
}
```

## Metadata Forwarding

HTTP headers are automatically converted to gRPC metadata:

- Header names are converted to lowercase
- Most standard HTTP headers are forwarded
- Authentication headers are preserved
- Custom headers starting with `X-` are forwarded

## Health Checks

Each gRPC service endpoint provides a health check:

```bash
curl http://localhost:8080/grpc/users/health
```

Response:

```json
{
  "status": "UP",
  "grpc_connection_state": "READY"
}
```

## Limitations

Current implementation limitations:

- **No streaming support**: Only unary RPCs are supported (streaming planned for future)
- **Dynamic invocation**: Uses generic JSON marshaling (may not work with all protobuf features)
- **No proto file parsing**: Relies on JSON structure matching protobuf schema

## Best Practices

- **Use gRPC reflection** for better debugging and tooling support
- **Set appropriate message size limits** to prevent abuse
- **Enable TLS** for production deployments
- **Monitor connection state** using health check endpoints
- **Handle errors gracefully** in client applications

## Future Enhancements

Planned improvements:

- **Server streaming** support via Server-Sent Events
- **Client streaming** support via WebSocket connections
- **Bidirectional streaming** support
- **Proto file parsing** for better type safety
- **gRPC-Web** support for browser clients