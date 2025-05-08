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
- `/test/unit/pkg/auth` - Tests for authentication
- `/test/unit/pkg/gateway` - Tests for core gateway functionality
- etc.

## Writing Tests

When writing new tests, please follow these guidelines:

1. Test files should be named `*_test.go`
2. Use table-driven tests when appropriate
3. Mock external dependencies
4. Aim for high coverage of edge cases
5. Include both positive and negative test cases

## Test Utilities

Common test utilities are available in the `/test/unit/utils` directory.

## Benchmarks

Performance benchmarks are included for critical path operations. Run them with:

```bash
go test -bench=. ./...
```
