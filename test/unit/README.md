# Unit Tests for Odin API Gateway

This directory contains unit tests for the Odin API Gateway components.

## Running Tests

To run all tests:

```bash
cd /path/to/odin
go test ./...
```

To run tests with coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Test Organization

Tests are organized to mirror the package structure of the main code:

- `/test/unit/pkg/admin` - Tests for admin functionality
- `/test/unit/pkg/auth` - Tests for authentication (JWT and OAuth2)
- `/test/unit/pkg/gateway` - Tests for core gateway functionality
- `/test/unit/pkg/circuit` - Tests for circuit breaker functionality
- `/test/unit/pkg/websocket` - Tests for WebSocket support
- `/test/unit/pkg/ratelimit` - Tests for rate limiting functionality
- `/test/unit/pkg/cache` - Tests for caching strategies
- etc.

## Writing Tests

When writing new tests, please follow these guidelines:

1. Test files should be named `*_test.go`
2. Use table-driven tests when appropriate
3. Mock external dependencies
4. Aim for high coverage of edge cases
5. Include both positive and negative test cases
6. Test concurrent scenarios where applicable

## Test Utilities

Common test utilities are available in the `/test/unit/utils` directory.

## Coverage Goals

We maintain a minimum of 80% test coverage across all packages. Critical paths should have 90%+ coverage.

## Benchmarks

Performance benchmarks are included for critical path operations. Run them with:

```bash
go test -bench=. ./...
```

## Integration with Circuit Breaker

Tests include scenarios for circuit breaker state transitions and failure handling.

## OAuth2 Testing

OAuth2 tests include mock providers and token validation scenarios.

## WebSocket Testing

WebSocket tests verify proxy functionality and connection handling.

## Rate Limiting Tests

Rate limiting tests verify different algorithms and configuration scenarios.

## Cache Strategy Tests

Cache tests verify TTL, conditional, and user context caching strategies.
