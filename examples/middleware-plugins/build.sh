#!/bin/bash

# Build all example middleware plugins

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building example middleware plugins..."

# Request Logger
echo "Building request-logger..."
cd request-logger
go mod init request-logger 2>/dev/null || true
go get github.com/labstack/echo/v4
go get github.com/sirupsen/logrus
go build -buildmode=plugin -o request-logger.so plugin.go
echo "✓ request-logger.so built successfully"
cd ..

# API Key Auth
echo "Building api-key-auth..."
cd api-key-auth
go mod init api-key-auth 2>/dev/null || true
go get github.com/labstack/echo/v4
go build -buildmode=plugin -o api-key-auth.so plugin.go
echo "✓ api-key-auth.so built successfully"
cd ..

# Request Transformer
echo "Building request-transformer..."
cd request-transformer
go mod init request-transformer 2>/dev/null || true
go get github.com/labstack/echo/v4
go build -buildmode=plugin -o request-transformer.so plugin.go
echo "✓ request-transformer.so built successfully"
cd ..

echo ""
echo "All middleware plugins built successfully!"
echo ""
echo "Plugin files:"
ls -lh */*.so 2>/dev/null || echo "No .so files found (they may have been built in subdirectories)"
echo ""
echo "To use these plugins:"
echo "1. Upload via admin UI at /admin/middleware-chain"
echo "2. Or use the API: curl -X POST http://localhost:8080/admin/api/plugins/upload -F 'file=@plugin.so'"
