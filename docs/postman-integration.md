# Postman API Platform Integration

## Overview

The Postman integration enables seamless synchronization between your Postman workspace and Odin API Gateway. This integration allows you to:

- **Import Postman collections** as Odin service definitions
- **Auto-sync** collections with scheduled updates
- **Run Newman tests** directly from the admin panel
- **Monitor sync status** and test results in real-time
- **Export Odin services** back to Postman format

## Architecture

### Components

1. **Postman Client** (`pkg/integrations/postman/client.go`)
   - Communicates with Postman API
   - Fetches collections, environments, and workspaces
   - Handles authentication with API key

2. **Transformer** (`pkg/integrations/postman/transformer.go`)
   - Converts Postman collections to Odin service definitions
   - Maps request/response transformations
   - Handles authentication schemes (Basic, Bearer, OAuth2, API Key)

3. **Sync Engine** (`pkg/integrations/postman/sync.go`)
   - Background synchronization service
   - Scheduled auto-sync with configurable intervals
   - Tracks sync history and error handling

4. **Newman Runner** (`pkg/integrations/postman/newman.go`)
   - Executes Postman tests using Newman CLI
   - Parses test results and assertions
   - Reports detailed test metrics

5. **MongoDB Repository** (`pkg/integrations/postman/repository.go`)
   - Persists integration configuration
   - Stores collection mappings
   - Maintains sync history

6. **Admin API Handler** (`pkg/admin/integration_handler.go`)
   - REST API for integration management
   - 25+ endpoints for all operations

7. **Admin UI** (`pkg/admin/templates/integrations_postman.html`)
   - Web interface for configuration
   - Real-time status monitoring
   - Collection browser and test runner

## Prerequisites

### 1. Postman API Key

You need a Postman API key to access the Postman API:

1. Log in to [Postman](https://www.postman.com/)
2. Go to [API Keys Settings](https://go.postman.co/settings/me/api-keys)
3. Click "Generate API Key"
4. Copy the key (it won't be shown again)

### 2. Newman CLI (Optional)

For test execution functionality, install Newman:

```bash
npm install -g newman
```

Verify installation:

```bash
newman --version
```

### 3. MongoDB (Required)

MongoDB is required for storing integration configuration and sync history:

- Configured in `config/config.yaml` under `mongodb` section
- Must be running before enabling Postman integration

## Configuration

### Basic Setup

1. **Access Admin Panel**
   
   Navigate to `http://localhost:8080/admin/integrations/postman`

2. **Enter Postman API Key**
   
   In the "Integration Configuration" section:
   - Enter your Postman API key
   - Click "Test Connection" to verify

3. **Select Workspace**
   
   - After connection, available workspaces will load
   - Select your target workspace from the dropdown
   - Click "Save Configuration"

### Advanced Configuration

#### Auto-Sync Settings

Enable automatic synchronization:

```yaml
# In MongoDB or via Admin UI
enabled: true
auto_sync: true
sync_interval: 300  # seconds (5 minutes)
```

Options:
- **Sync Interval**: How often to check for updates (in seconds)
  - Default: 300 (5 minutes)
  - Minimum: 60 (1 minute)
  - Recommended: 300-900 (5-15 minutes)

#### Newman Testing

Enable automated test execution:

```yaml
newman_enabled: true
newman_path: "/usr/local/bin/newman"  # auto-detected if not specified
```

## Usage Guide

### Importing Collections

#### Via Admin UI

1. Navigate to "Collections" section
2. Click "Load Collections" to fetch from Postman
3. Select a collection
4. Click "Import as Service"
5. Enter service name for Odin
6. Click "Import"

#### Via API

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/collections/{collection_id}/import \
  -H "Content-Type: application/json" \
  -d '{"service_name": "my-api-service"}'
```

### Manual Sync

#### Via Admin UI

Click "Sync All Collections" button in the dashboard

#### Via API

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/sync
```

### Running Tests

#### Via Admin UI

1. Navigate to collection in the table
2. Click "Run Tests" button
3. View results in "Test Results" section

#### Via API

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/test/{collection_id}
```

Get results:

```bash
curl http://localhost:8080/admin/api/integrations/postman/test/results/{collection_id}
```

### Exporting Services

Export an Odin service back to Postman format:

```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/collections/export/my-service
```

This creates a Postman collection from the Odin service definition.

## API Reference

### Configuration Endpoints

#### Get Configuration
```
GET /admin/api/integrations/postman/config
```

Response:
```json
{
  "api_key": "PMAK-xxxxx",
  "workspace_id": "workspace-id-here",
  "enabled": true,
  "auto_sync": true,
  "sync_interval": 300,
  "newman_enabled": true
}
```

#### Save Configuration
```
POST /admin/api/integrations/postman/config
Content-Type: application/json

{
  "api_key": "PMAK-xxxxx",
  "workspace_id": "workspace-id-here",
  "enabled": true,
  "auto_sync": true,
  "sync_interval": 300,
  "newman_enabled": true
}
```

#### Test Connection
```
POST /admin/api/integrations/postman/connect
```

Response:
```json
{
  "success": true,
  "message": "Connected successfully",
  "user": {
    "id": "12345",
    "username": "john.doe",
    "email": "john.doe@example.com"
  }
}
```

### Collection Endpoints

#### List Collections
```
GET /admin/api/integrations/postman/collections
```

Response:
```json
{
  "collections": [
    {
      "id": "collection-id-1",
      "name": "My API",
      "owner": "12345",
      "uid": "12345-collection-id-1"
    }
  ]
}
```

#### Get Collection Details
```
GET /admin/api/integrations/postman/collections/{id}
```

#### Import Collection
```
POST /admin/api/integrations/postman/collections/{id}/import
Content-Type: application/json

{
  "service_name": "my-api-service"
}
```

#### Export Service to Collection
```
POST /admin/api/integrations/postman/collections/export/{service_name}
```

### Sync Endpoints

#### Sync All Collections
```
POST /admin/api/integrations/postman/sync
```

#### Sync Single Collection
```
POST /admin/api/integrations/postman/sync/{collection_id}
```

#### Get Sync History
```
GET /admin/api/integrations/postman/sync/history?limit=50
```

Response:
```json
{
  "history": [
    {
      "collection_id": "collection-id-1",
      "collection_name": "My API",
      "status": "success",
      "timestamp": "2024-01-15T10:30:00Z",
      "changes_detected": true,
      "error": ""
    }
  ]
}
```

#### Start Auto-Sync
```
POST /admin/api/integrations/postman/sync/start
```

#### Stop Auto-Sync
```
POST /admin/api/integrations/postman/sync/stop
```

### Test Endpoints

#### Run Tests for Collection
```
POST /admin/api/integrations/postman/test/{collection_id}
```

Response:
```json
{
  "success": true,
  "message": "Tests started",
  "run_id": "test-run-123"
}
```

#### Get Test Results
```
GET /admin/api/integrations/postman/test/results/{collection_id}
```

Response:
```json
{
  "collection_id": "collection-id-1",
  "collection_name": "My API",
  "status": "completed",
  "total_tests": 25,
  "passed": 23,
  "failed": 2,
  "skipped": 0,
  "duration": 1234,
  "timestamp": "2024-01-15T10:30:00Z",
  "assertions": [
    {
      "name": "Status code is 200",
      "passed": true
    }
  ],
  "failures": [
    {
      "name": "Response time < 200ms",
      "message": "Expected response time to be below 200ms, got 250ms"
    }
  ]
}
```

#### Get Test Statistics
```
GET /admin/api/integrations/postman/test/stats
```

### Environment Endpoints

#### List Environments
```
GET /admin/api/integrations/postman/environments
```

#### Get Environment Details
```
GET /admin/api/integrations/postman/environments/{id}
```

### Workspace Endpoints

#### List Workspaces
```
GET /admin/api/integrations/postman/workspaces
```

### Status Endpoint

#### Get Integration Status
```
GET /admin/api/integrations/postman/status
```

Response:
```json
{
  "connected": true,
  "auto_sync_running": true,
  "last_sync": "2024-01-15T10:30:00Z",
  "collections_count": 5,
  "workspace_name": "My Workspace",
  "newman_available": true
}
```

## Transformation Details

### Request Mapping

Postman requests are transformed to Odin service routes:

**Postman:**
```json
{
  "method": "GET",
  "url": "{{baseUrl}}/users/:id",
  "header": [
    {"key": "Authorization", "value": "Bearer {{token}}"}
  ]
}
```

**Odin Service:**
```yaml
name: users-api
base_path: /api/users
targets:
  - url: https://api.example.com
routes:
  - path: /:id
    method: GET
authentication:
  type: bearer
  token_source: header
  header_name: Authorization
```

### Authentication Schemes

#### Basic Auth
```yaml
authentication:
  type: basic
  username: "{{username}}"
  password: "{{password}}"
```

#### Bearer Token
```yaml
authentication:
  type: bearer
  token_source: header
  header_name: Authorization
```

#### API Key
```yaml
authentication:
  type: apikey
  key_name: X-API-Key
  key_location: header
  key_value: "{{api_key}}"
```

#### OAuth 2.0
```yaml
authentication:
  type: oauth2
  grant_type: client_credentials
  token_url: https://auth.example.com/token
  client_id: "{{client_id}}"
  client_secret: "{{client_secret}}"
  scopes:
    - read
    - write
```

### Variable Substitution

Postman variables are mapped to Odin configuration:

- `{{baseUrl}}` → Service `targets[0].url`
- `{{token}}` → Authentication token configuration
- Custom variables → Odin service headers or environment

## Monitoring & Troubleshooting

### Viewing Logs

Check gateway logs for integration activity:

```bash
# Look for Postman integration logs
grep "Postman" /var/log/odin/gateway.log

# Check sync operations
grep "sync" /var/log/odin/gateway.log | grep -i postman
```

### Common Issues

#### Connection Failed

**Symptom:** "Test Connection" fails or returns 401 Unauthorized

**Solutions:**
1. Verify API key is correct and not expired
2. Check Postman API is accessible from your network
3. Ensure API key has proper permissions

#### Sync Errors

**Symptom:** Auto-sync reports errors in sync history

**Solutions:**
1. Check collection still exists in Postman workspace
2. Verify workspace ID is correct
3. Review error message in sync history for details

#### Newman Tests Fail to Run

**Symptom:** "Run Tests" returns error or no results

**Solutions:**
1. Verify Newman is installed: `newman --version`
2. Check Newman path in configuration
3. Ensure collection has test scripts
4. Review Newman output in test results

#### Collections Not Appearing

**Symptom:** "Load Collections" returns empty list

**Solutions:**
1. Verify correct workspace is selected
2. Check collections exist in Postman workspace
3. Ensure API key has access to workspace

### Health Checks

The integration provides health status:

```bash
curl http://localhost:8080/admin/api/integrations/postman/status
```

Check for:
- `connected: true` - API connection is working
- `auto_sync_running: true` - Background sync is active
- `newman_available: true` - Newman CLI is installed

### Sync History

Review sync operations:

```bash
curl http://localhost:8080/admin/api/integrations/postman/sync/history?limit=100
```

Look for:
- Failed syncs with error messages
- Collections that frequently fail
- Sync duration anomalies

## Best Practices

### 1. Workspace Organization

- Use dedicated workspace for Odin synced collections
- Organize collections by environment or API domain
- Use consistent naming conventions

### 2. Sync Intervals

- **Development**: 5-10 minutes for rapid iteration
- **Staging**: 15-30 minutes for moderate updates
- **Production**: 30-60 minutes for stability

### 3. Testing Strategy

- Include comprehensive test scripts in Postman
- Use environment variables for configuration
- Test both success and error cases
- Monitor test results regularly

### 4. Version Control

- Export collections periodically as backup
- Track changes in sync history
- Document major collection updates

### 5. Security

- Rotate API keys regularly
- Use environment-specific API keys
- Restrict workspace access appropriately
- Store sensitive data in Postman environments

### 6. Performance

- Avoid extremely large collections (>500 requests)
- Use reasonable sync intervals
- Monitor MongoDB storage usage
- Clean old sync history periodically

## Integration Patterns

### Pattern 1: Dev → Staging → Prod

1. Develop APIs in Postman
2. Import to Odin dev environment
3. Test with Newman
4. Export to Postman for QA review
5. Import to Odin staging
6. Final import to production

### Pattern 2: Continuous Sync

1. Enable auto-sync for development
2. Postman becomes source of truth
3. Odin automatically updates routes
4. Continuous testing with Newman
5. Monitor sync history for issues

### Pattern 3: Manual Control

1. Disable auto-sync
2. Review Postman changes manually
3. Import specific collections explicitly
4. Full control over deployments

## Limitations

- **Collection Size**: Very large collections (>1000 requests) may slow sync operations
- **Newman Requirement**: Test execution requires Newman CLI installed
- **MongoDB Dependency**: Integration requires MongoDB for state persistence
- **API Rate Limits**: Postman API has rate limits (check Postman documentation)
- **Real-time Updates**: Not real-time, relies on polling interval
- **Variable Resolution**: Some complex Postman variables may require manual configuration

## Roadmap

Future enhancements planned:

- [ ] Webhook support for real-time updates
- [ ] Bidirectional sync (Odin → Postman)
- [ ] Team collaboration features
- [ ] Advanced conflict resolution
- [ ] Collection versioning and rollback
- [ ] GraphQL collection support
- [ ] Custom transformer plugins

## Support

For issues or questions:

- Check sync history for error details
- Review gateway logs for diagnostic information
- Consult [API Reference](#api-reference) for endpoint details
- See [Troubleshooting](#monitoring--troubleshooting) for common issues

## Contributing

When contributing to Postman integration:

1. Follow existing code patterns
2. Add tests for new functionality
3. Update documentation
4. Test with real Postman collections
5. Consider backward compatibility

## License

This integration is part of Odin API Gateway. See main LICENSE file.
