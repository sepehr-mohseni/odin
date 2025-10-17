# Postman Integration Testing Guide

## Overview

This guide covers testing strategies and examples for the Postman integration in Odin API Gateway.

## Test Categories

### 1. Unit Tests
### 2. Integration Tests
### 3. End-to-End Tests
### 4. Manual Testing

## Unit Testing

### Testing the Transformer

Create a test file: `pkg/integrations/postman/transformer_test.go`

```go
package postman

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformCollection_BasicRequest(t *testing.T) {
	collection := &Collection{
		Info: CollectionInfo{
			Name:        "Test API",
			Description: "Test collection",
		},
		Item: []Item{
			{
				Name: "Get User",
				Request: Request{
					Method: "GET",
					URL: URL{
						Raw:  "{{baseUrl}}/users/:id",
						Path: []string{"users", ":id"},
					},
					Header: []Header{
						{Key: "Accept", Value: "application/json"},
					},
				},
			},
		},
		Variable: []Variable{
			{Key: "baseUrl", Value: "https://api.example.com"},
		},
	}

	transformer := NewTransformer()
	service, err := transformer.TransformCollection(collection, "test-service")

	require.NoError(t, err)
	assert.Equal(t, "test-service", service.Name)
	assert.Equal(t, "/api/test", service.BasePath)
	assert.Len(t, service.Routes, 1)
	
	route := service.Routes[0]
	assert.Equal(t, "GET", route.Method)
	assert.Equal(t, "/users/:id", route.Path)
}

func TestTransformCollection_WithAuthentication(t *testing.T) {
	collection := &Collection{
		Info: CollectionInfo{
			Name: "Protected API",
		},
		Auth: &Auth{
			Type: "bearer",
			Bearer: []AuthValue{
				{Key: "token", Value: "{{bearerToken}}"},
			},
		},
		Item: []Item{
			{
				Name: "Protected Endpoint",
				Request: Request{
					Method: "GET",
					URL:    URL{Raw: "{{baseUrl}}/protected"},
				},
			},
		},
	}

	transformer := NewTransformer()
	service, err := transformer.TransformCollection(collection, "protected-service")

	require.NoError(t, err)
	require.NotNil(t, service.Authentication)
	assert.Equal(t, "bearer", service.Authentication.Type)
}

func TestTransformCollection_WithMultipleMethods(t *testing.T) {
	collection := &Collection{
		Info: CollectionInfo{Name: "CRUD API"},
		Item: []Item{
			{
				Name:    "Get Users",
				Request: Request{Method: "GET", URL: URL{Raw: "/users"}},
			},
			{
				Name:    "Create User",
				Request: Request{Method: "POST", URL: URL{Raw: "/users"}},
			},
			{
				Name:    "Update User",
				Request: Request{Method: "PUT", URL: URL{Raw: "/users/:id"}},
			},
			{
				Name:    "Delete User",
				Request: Request{Method: "DELETE", URL: URL{Raw: "/users/:id"}},
			},
		},
	}

	transformer := NewTransformer()
	service, err := transformer.TransformCollection(collection, "crud-service")

	require.NoError(t, err)
	assert.Len(t, service.Routes, 4)

	methods := make(map[string]bool)
	for _, route := range service.Routes {
		methods[route.Method] = true
	}

	assert.True(t, methods["GET"])
	assert.True(t, methods["POST"])
	assert.True(t, methods["PUT"])
	assert.True(t, methods["DELETE"])
}

func TestVariableSubstitution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		vars     map[string]string
		expected string
	}{
		{
			name:     "simple variable",
			input:    "{{baseUrl}}/users",
			vars:     map[string]string{"baseUrl": "https://api.example.com"},
			expected: "https://api.example.com/users",
		},
		{
			name:     "multiple variables",
			input:    "{{protocol}}://{{host}}:{{port}}/{{path}}",
			vars:     map[string]string{"protocol": "https", "host": "api.example.com", "port": "443", "path": "v1"},
			expected: "https://api.example.com:443/v1",
		},
		{
			name:     "no variables",
			input:    "https://api.example.com/users",
			vars:     map[string]string{},
			expected: "https://api.example.com/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteVariables(tt.input, tt.vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

Run unit tests:

```bash
cd pkg/integrations/postman
go test -v
```

### Testing the Newman Runner

Create: `pkg/integrations/postman/newman_test.go`

```go
package postman

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sirupsen/logrus"
)

func TestNewmanRunner_FindExecutable(t *testing.T) {
	logger := logrus.New()
	runner, err := NewNewmanRunner("", logger)

	if err != nil {
		t.Skip("Newman not installed, skipping test")
	}

	require.NoError(t, err)
	assert.NotEmpty(t, runner.newmanPath)
}

func TestNewmanRunner_InvalidPath(t *testing.T) {
	logger := logrus.New()
	_, err := NewNewmanRunner("/nonexistent/newman", logger)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "newman executable not found")
}

func TestParseNewmanOutput(t *testing.T) {
	// Sample Newman JSON output
	output := `{
		"collection": {
			"info": {
				"name": "Test Collection",
				"_postman_id": "test-id"
			}
		},
		"run": {
			"stats": {
				"iterations": {"total": 1, "pending": 0, "failed": 0},
				"requests": {"total": 5, "pending": 0, "failed": 1},
				"assertions": {"total": 10, "pending": 0, "failed": 2}
			},
			"timings": {
				"completed": 1234,
				"started": 0
			},
			"executions": [
				{
					"item": {"name": "Test Request 1"},
					"response": {"code": 200},
					"assertions": [
						{"assertion": "Status code is 200", "error": null}
					]
				},
				{
					"item": {"name": "Test Request 2"},
					"response": {"code": 500},
					"assertions": [
						{"assertion": "Status code is 200", "error": {"message": "Expected 200, got 500"}}
					]
				}
			]
		}
	}`

	result := parseNewmanOutput([]byte(output))

	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, 10, result.TotalTests)
	assert.Equal(t, 8, result.Passed)
	assert.Equal(t, 2, result.Failed)
	assert.Equal(t, 1234, result.Duration)
	assert.Len(t, result.Assertions, 2)
	assert.Len(t, result.Failures, 1)
}
```

### Testing the Sync Engine

Create: `pkg/integrations/postman/sync_test.go`

```go
package postman

import (
	"context"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sirupsen/logrus"
)

func TestSyncEngine_Start_Stop(t *testing.T) {
	logger := logrus.New()
	client := &Client{apiKey: "test-key", logger: logger}
	repo := &MockRepository{}
	
	engine := NewSyncEngine(client, repo, logger)
	
	err := engine.Start(context.Background(), 1*time.Second)
	require.NoError(t, err)
	
	time.Sleep(2 * time.Second)
	
	engine.Stop()
	
	// Verify sync was called at least once
	assert.True(t, repo.syncCalled)
}

// MockRepository for testing
type MockRepository struct {
	syncCalled bool
	mappings   []CollectionMapping
}

func (m *MockRepository) GetCollectionMappings() ([]CollectionMapping, error) {
	m.syncCalled = true
	return m.mappings, nil
}

func (m *MockRepository) SaveCollectionMapping(mapping *CollectionMapping) error {
	return nil
}

func (m *MockRepository) GetSyncHistory(limit int) ([]SyncRecord, error) {
	return []SyncRecord{}, nil
}

func (m *MockRepository) SaveSyncRecord(record *SyncRecord) error {
	return nil
}
```

## Integration Testing

### Testing with Real Postman API

Create: `test/integration/postman_integration_test.go`

```go
//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"odin/pkg/integrations/postman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sirupsen/logrus"
)

func TestPostmanClient_GetUser(t *testing.T) {
	apiKey := os.Getenv("POSTMAN_API_KEY")
	if apiKey == "" {
		t.Skip("POSTMAN_API_KEY not set, skipping integration test")
	}

	logger := logrus.New()
	client := postman.NewClient(apiKey, logger)

	user, err := client.GetUser(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.Username)
}

func TestPostmanClient_ListWorkspaces(t *testing.T) {
	apiKey := os.Getenv("POSTMAN_API_KEY")
	if apiKey == "" {
		t.Skip("POSTMAN_API_KEY not set")
	}

	logger := logrus.New()
	client := postman.NewClient(apiKey, logger)

	workspaces, err := client.ListWorkspaces(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, workspaces)
}

func TestPostmanClient_GetCollection(t *testing.T) {
	apiKey := os.Getenv("POSTMAN_API_KEY")
	collectionID := os.Getenv("TEST_COLLECTION_ID")
	
	if apiKey == "" || collectionID == "" {
		t.Skip("Required environment variables not set")
	}

	logger := logrus.New()
	client := postman.NewClient(apiKey, logger)

	collection, err := client.GetCollection(context.Background(), collectionID)
	require.NoError(t, err)
	assert.NotEmpty(t, collection.Info.Name)
	assert.NotEmpty(t, collection.Item)
}
```

Run integration tests:

```bash
# Set environment variables
export POSTMAN_API_KEY="your-api-key-here"
export TEST_COLLECTION_ID="your-collection-id"

# Run tests
go test -v -tags=integration ./test/integration/...
```

### Testing with MongoDB

Create: `test/integration/repository_test.go`

```go
//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"
	"odin/pkg/integrations/postman"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDBRepository(t *testing.T) {
	// Connect to test MongoDB
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	require.NoError(t, err)
	defer client.Disconnect(ctx)

	db := client.Database("odin_test")
	defer db.Drop(ctx)

	repo, err := postman.NewMongoDBRepository(db)
	require.NoError(t, err)

	// Test saving configuration
	config := &postman.IntegrationConfig{
		APIKey:        "test-key",
		WorkspaceID:   "workspace-123",
		Enabled:       true,
		AutoSync:      true,
		SyncInterval:  300,
		NewmanEnabled: true,
	}

	err = repo.SaveConfig(config)
	require.NoError(t, err)

	// Test loading configuration
	loadedConfig, err := repo.GetConfig()
	require.NoError(t, err)
	assert.Equal(t, config.APIKey, loadedConfig.APIKey)
	assert.Equal(t, config.WorkspaceID, loadedConfig.WorkspaceID)

	// Test saving collection mapping
	mapping := &postman.CollectionMapping{
		CollectionID:   "col-123",
		CollectionName: "Test API",
		ServiceName:    "test-service",
		LastSynced:     time.Now(),
	}

	err = repo.SaveCollectionMapping(mapping)
	require.NoError(t, err)

	// Test getting mappings
	mappings, err := repo.GetCollectionMappings()
	require.NoError(t, err)
	assert.Len(t, mappings, 1)
	assert.Equal(t, mapping.CollectionID, mappings[0].CollectionID)

	// Test sync history
	record := &postman.SyncRecord{
		CollectionID:     "col-123",
		CollectionName:   "Test API",
		Status:           "success",
		Timestamp:        time.Now(),
		ChangesDetected:  true,
	}

	err = repo.SaveSyncRecord(record)
	require.NoError(t, err)

	history, err := repo.GetSyncHistory(10)
	require.NoError(t, err)
	assert.Len(t, history, 1)
}
```

## End-to-End Testing

### Full Workflow Test

Create: `test/e2e/postman_e2e_test.go`

```go
//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	gatewayURL = "http://localhost:8080"
	adminURL   = gatewayURL + "/admin/api/integrations/postman"
)

func TestE2E_FullWorkflow(t *testing.T) {
	// 1. Save configuration
	t.Run("SaveConfiguration", func(t *testing.T) {
		config := map[string]interface{}{
			"api_key":        "PMAK-test-key",
			"workspace_id":   "workspace-123",
			"enabled":        true,
			"auto_sync":      false,
			"sync_interval":  300,
			"newman_enabled": true,
		}

		body, _ := json.Marshal(config)
		resp, err := http.Post(adminURL+"/config", "application/json", bytes.NewBuffer(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// 2. Test connection
	t.Run("TestConnection", func(t *testing.T) {
		resp, err := http.Post(adminURL+"/connect", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result["success"].(bool))
	})

	// 3. List collections
	t.Run("ListCollections", func(t *testing.T) {
		resp, err := http.Get(adminURL + "/collections")
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		collections := result["collections"].([]interface{})
		assert.NotEmpty(t, collections)
	})

	// 4. Import collection
	t.Run("ImportCollection", func(t *testing.T) {
		importReq := map[string]string{
			"service_name": "test-e2e-service",
		}

		body, _ := json.Marshal(importReq)
		resp, err := http.Post(
			adminURL+"/collections/test-collection-id/import",
			"application/json",
			bytes.NewBuffer(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// 5. Verify service was created
	t.Run("VerifyService", func(t *testing.T) {
		time.Sleep(2 * time.Second) // Wait for async processing

		resp, err := http.Get(gatewayURL + "/admin/services")
		require.NoError(t, err)
		defer resp.Body.Close()

		var services []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&services)

		found := false
		for _, svc := range services {
			if svc["name"] == "test-e2e-service" {
				found = true
				break
			}
		}

		assert.True(t, found, "Service should be created")
	})

	// 6. Run tests
	t.Run("RunTests", func(t *testing.T) {
		resp, err := http.Post(adminURL+"/test/test-collection-id", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Wait for tests to complete
		time.Sleep(5 * time.Second)

		// Get results
		resp, err = http.Get(adminURL + "/test/results/test-collection-id")
		require.NoError(t, err)
		defer resp.Body.Close()

		var results map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&results)

		assert.Equal(t, "completed", results["status"])
		assert.NotZero(t, results["total_tests"])
	})

	// 7. Sync collection
	t.Run("SyncCollection", func(t *testing.T) {
		resp, err := http.Post(adminURL+"/sync/test-collection-id", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// 8. Check sync history
	t.Run("CheckSyncHistory", func(t *testing.T) {
		resp, err := http.Get(adminURL + "/sync/history?limit=10")
		require.NoError(t, err)
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		history := result["history"].([]interface{})
		assert.NotEmpty(t, history)
	})
}
```

Run E2E tests:

```bash
# Start Odin gateway
./odin &

# Run E2E tests
go test -v -tags=e2e ./test/e2e/...
```

## Manual Testing

### Using curl

#### 1. Configure Integration

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/config \
  -H "Content-Type: application/json" \
  -d '{
    "api_key": "PMAK-your-api-key-here",
    "workspace_id": "your-workspace-id",
    "enabled": true,
    "auto_sync": true,
    "sync_interval": 300,
    "newman_enabled": true
  }'
```

#### 2. Test Connection

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/connect
```

#### 3. List Collections

```bash
curl http://localhost:8080/admin/api/integrations/postman/collections
```

#### 4. Import Collection

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/collections/COLLECTION_ID/import \
  -H "Content-Type: application/json" \
  -d '{"service_name": "my-api"}'
```

#### 5. Sync Collections

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/sync
```

#### 6. Run Tests

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/test/COLLECTION_ID
```

#### 7. Get Test Results

```bash
curl http://localhost:8080/admin/api/integrations/postman/test/results/COLLECTION_ID
```

### Using Postman (Ironically)

Import this collection to test the integration:

```json
{
  "info": {
    "name": "Odin Postman Integration Tests",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Get Status",
      "request": {
        "method": "GET",
        "url": "{{odinUrl}}/admin/api/integrations/postman/status"
      }
    },
    {
      "name": "Save Config",
      "request": {
        "method": "POST",
        "url": "{{odinUrl}}/admin/api/integrations/postman/config",
        "body": {
          "mode": "raw",
          "raw": "{\n  \"api_key\": \"{{postmanApiKey}}\",\n  \"workspace_id\": \"{{workspaceId}}\",\n  \"enabled\": true\n}"
        }
      }
    },
    {
      "name": "List Collections",
      "request": {
        "method": "GET",
        "url": "{{odinUrl}}/admin/api/integrations/postman/collections"
      }
    }
  ],
  "variable": [
    {"key": "odinUrl", "value": "http://localhost:8080"},
    {"key": "postmanApiKey", "value": ""},
    {"key": "workspaceId", "value": ""}
  ]
}
```

## Performance Testing

### Load Test Sync Operations

Use Apache Bench or similar:

```bash
# Test concurrent sync requests
ab -n 100 -c 10 -p /dev/null \
  http://localhost:8080/admin/api/integrations/postman/sync

# Test collection listing
ab -n 1000 -c 50 \
  http://localhost:8080/admin/api/integrations/postman/collections
```

### Stress Test Auto-Sync

```bash
# Set very short sync interval (testing only!)
curl -X POST http://localhost:8080/admin/api/integrations/postman/config \
  -H "Content-Type: application/json" \
  -d '{"sync_interval": 10}'  # 10 seconds

# Monitor logs for performance issues
tail -f /var/log/odin/gateway.log | grep -i sync

# Reset to normal interval
curl -X POST http://localhost:8080/admin/api/integrations/postman/config \
  -H "Content-Type: application/json" \
  -d '{"sync_interval": 300}'
```

## Test Data Setup

### Creating Test Collections

1. **Simple Collection**

```json
{
  "info": {"name": "Simple Test API"},
  "item": [
    {
      "name": "Hello World",
      "request": {
        "method": "GET",
        "url": "https://httpbin.org/get"
      }
    }
  ]
}
```

2. **Complex Collection with Auth**

```json
{
  "info": {"name": "Auth Test API"},
  "auth": {
    "type": "bearer",
    "bearer": [{"key": "token", "value": "{{token}}"}]
  },
  "item": [
    {
      "name": "Protected Resource",
      "request": {
        "method": "GET",
        "url": "https://httpbin.org/bearer",
        "header": [
          {"key": "Authorization", "value": "Bearer {{token}}"}
        ]
      },
      "event": [
        {
          "listen": "test",
          "script": {
            "exec": [
              "pm.test('Status code is 200', function() {",
              "  pm.response.to.have.status(200);",
              "});",
              "pm.test('Response has token', function() {",
              "  pm.expect(pm.response.json()).to.have.property('token');",
              "});"
            ]
          }
        }
      ]
    }
  ]
}
```

## CI/CD Integration

### GitHub Actions

Create `.github/workflows/postman-integration-test.yml`:

```yaml
name: Postman Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      mongodb:
        image: mongo:6
        ports:
          - 27017:27017
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install Newman
        run: npm install -g newman
      
      - name: Run Unit Tests
        run: go test -v ./pkg/integrations/postman/...
      
      - name: Run Integration Tests
        env:
          POSTMAN_API_KEY: ${{ secrets.POSTMAN_API_KEY }}
          TEST_COLLECTION_ID: ${{ secrets.TEST_COLLECTION_ID }}
        run: go test -v -tags=integration ./test/integration/...
      
      - name: Start Odin Gateway
        run: |
          go build -o odin ./cmd/odin
          ./odin &
          sleep 5
      
      - name: Run E2E Tests
        run: go test -v -tags=e2e ./test/e2e/...
```

## Troubleshooting Tests

### Common Test Failures

1. **"Newman not found"**
   - Install Newman: `npm install -g newman`
   - Verify: `newman --version`

2. **"MongoDB connection failed"**
   - Ensure MongoDB is running
   - Check connection string in tests
   - Verify network connectivity

3. **"Postman API rate limit"**
   - Add delays between API calls
   - Use mock data for unit tests
   - Consider dedicated test API key

4. **"Collection not found"**
   - Verify collection ID is correct
   - Ensure API key has access
   - Check workspace ID

## Best Practices

1. **Use Test Tags**
   - `//go:build integration` for integration tests
   - `//go:build e2e` for E2E tests
   - Keep unit tests fast and isolated

2. **Mock External Dependencies**
   - Mock Postman API for unit tests
   - Use test MongoDB database
   - Avoid hitting real APIs in CI

3. **Clean Up Test Data**
   - Drop test databases after tests
   - Remove imported test collections
   - Clear sync history

4. **Use Environment Variables**
   - Never commit API keys
   - Use `.env` files locally
   - Configure secrets in CI/CD

5. **Test Error Cases**
   - Invalid API keys
   - Network failures
   - Malformed collections
   - Rate limiting

## Running All Tests

```bash
# Unit tests only
go test ./pkg/integrations/postman/...

# Integration tests (requires setup)
export POSTMAN_API_KEY="your-key"
export TEST_COLLECTION_ID="collection-id"
go test -tags=integration ./test/integration/...

# E2E tests (requires running gateway)
go test -tags=e2e ./test/e2e/...

# All tests
go test -tags="integration e2e" ./...

# With coverage
go test -cover -coverprofile=coverage.out ./pkg/integrations/postman/...
go tool cover -html=coverage.out
```
