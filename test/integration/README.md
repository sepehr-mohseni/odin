# Integration Tests

This directory contains integration tests for the Odin API Gateway. These tests verify the end-to-end functionality of the gateway by testing the interaction between different components.

## Test Structure

- **API Tests**: Test the complete request/response flow through the gateway
- **Authentication Tests**: Verify JWT and OAuth2 authentication flows
- **Service Integration**: Test service discovery and load balancing
- **WebSocket Tests**: Test WebSocket proxy functionality
- **Cache Tests**: Verify caching behavior across requests
- **Rate Limiting Tests**: Test rate limiting enforcement

## Running Integration Tests

```bash
# Run all integration tests
make test-integration

# Run specific test categories
go test -v ./test/integration/api/
go test -v ./test/integration/auth/
go test -v ./test/integration/websocket/
```

## Prerequisites

Before running integration tests, ensure you have:

1. **Test Services Running**: The test services in `test/services/` should be running
2. **Redis Available**: For cache and rate limiting tests
3. **Valid Configuration**: Test configuration files should be present

## Test Services

The integration tests use mock services located in `test/services/`:

- **users-service**: Mock user management service (port 8081)
- **products-service**: Mock product catalog service (port 8083)
- **orders-service**: Mock order processing service (port 8084)
- **categories-service**: Mock category service (port 8085)

## Environment Setup

```bash
# Start test services with Docker Compose
docker-compose -f docker-compose.test.yml up -d

# Or start individual services
cd test/services
node users-service/server.js &
node products-service/server.js &
node orders-service/server.js &
node categories-service/server.js &
```

## Test Configuration

Integration tests use configuration files from `test/config/`:

- `test-config.yaml`: Main gateway configuration for testing
- `test-services.yaml`: Service definitions for testing
- `test-auth.yaml`: Authentication configuration for testing

## Writing Integration Tests

When writing new integration tests:

1. **Use Real HTTP Requests**: Test the actual HTTP interface
2. **Test Error Scenarios**: Verify error handling and edge cases
3. **Clean Up Resources**: Ensure tests clean up after themselves
4. **Use Parallel Testing**: Where possible, run tests in parallel
5. **Mock External Dependencies**: Use the test services for consistency

## Example Test

```go
func TestGatewayRouting(t *testing.T) {
    // Setup gateway with test configuration
    gateway := setupTestGateway(t)
    defer gateway.Shutdown()

    // Test request routing
    resp, err := http.Get("http://localhost:8080/api/users")
    require.NoError(t, err)
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Verify response
    var users []User
    err = json.NewDecoder(resp.Body).Decode(&users)
    require.NoError(t, err)
    assert.NotEmpty(t, users)
}
```

## Debugging Tests

To debug failing integration tests:

1. **Check Service Logs**: Review logs from test services
2. **Enable Debug Logging**: Set `LOG_LEVEL=debug` in test configuration
3. **Use Test Isolation**: Run individual tests to isolate issues
4. **Verify Service Availability**: Ensure all required services are running

## Continuous Integration

Integration tests are designed to run in CI/CD pipelines:

- Tests use ephemeral ports to avoid conflicts
- Services are started/stopped automatically
- Test data is cleaned up after each test run
- Tests are designed to be deterministic and repeatable
