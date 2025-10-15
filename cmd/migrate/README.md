# MongoDB Migration Tool

This tool migrates Odin API Gateway configuration from YAML files to MongoDB.

## Quick Start

### Build

```bash
cd /home/sep/code/odin
go build -o bin/odin-migrate ./cmd/migrate/
```

### Usage

```bash
./bin/odin-migrate [OPTIONS]
```

## Options

| Flag | Description | Default | Required |
|------|-------------|---------|----------|
| `--config` | Path to main configuration file | `config/config.yaml` | Yes |
| `--services` | Path to services YAML file | - | No |
| `--routes` | Path to routes YAML file | - | No |
| `--mongodb-uri` | MongoDB connection URI (overrides config) | - | No |
| `--mongodb-database` | MongoDB database name | `odin_gateway` | No |
| `--dry-run` | Perform dry run without actual migration | `false` | No |
| `--force` | Force migration even if services exist | `false` | No |
| `--verbose` | Enable verbose logging | `false` | No |

## Examples

### Dry Run (Recommended First)

Test migration without making changes:

```bash
./bin/odin-migrate \
  --config config/config.yaml \
  --dry-run \
  --verbose
```

### Migrate from Main Config

```bash
./bin/odin-migrate \
  --config config/config.yaml \
  --mongodb-uri "mongodb://user:pass@localhost:27017?authSource=admin" \
  --mongodb-database "odin_gateway" \
  --verbose
```

### Migrate from Multiple Files

```bash
./bin/odin-migrate \
  --config config/config.yaml \
  --services config/services.yaml \
  --routes config/routes.yaml \
  --verbose
```

### Force Override Existing Services

```bash
./bin/odin-migrate \
  --config config/config.yaml \
  --force \
  --verbose
```

## Output

### Successful Migration

```
INFO[0000] Starting MongoDB migration tool
INFO[0000] MongoDB configuration                         database=odin_gateway dry_run=false uri="mongodb://***:***@***"
INFO[0001] Connected to MongoDB successfully
INFO[0001] Found services in main config                 count=10
INFO[0001] Total unique services to migrate              count=10
INFO[0001] Migrating service                             index=1 service=user-service basePath=/api/users total=10
INFO[0001] Service migrated successfully                 service=user-service
...
INFO[0005] =====================================
INFO[0005] Migration completed                           failed=0 migrated=10 total=10
INFO[0005] Verifying migration...
INFO[0005] Services in MongoDB after migration           count=10
INFO[0005] ✅ Migration completed successfully!
```

### Dry Run Output

```
INFO[0000] Starting MongoDB migration tool
INFO[0000] MongoDB configuration                         database=odin_gateway dry_run=true uri="mongodb://***:***@***"
INFO[0001] Connected to MongoDB successfully
INFO[0001] Found services in main config                 count=10
INFO[0001] Total unique services to migrate              count=10
INFO[0001] DRY RUN MODE - No changes will be made
INFO[0001] Services that would be migrated:
INFO[0001] Service                                       basePath=/api/users enabled=true index=1 name=user-service targets=2
INFO[0001] Service                                       basePath=/api/products enabled=true index=2 name=product-service targets=1
...
INFO[0001] Migration would complete successfully (dry run)
```

## What Gets Migrated

The migration tool transfers:

- ✅ Service name and base path
- ✅ Target endpoints
- ✅ Timeout and retry settings
- ✅ Load balancing configuration
- ✅ Authentication settings
- ✅ Custom headers
- ✅ Protocol type (HTTP/gRPC/GraphQL)
- ✅ Transform configurations (if present)
- ✅ Aggregation configurations (if present)
- ✅ Health check configurations (if present)

## Verification

After migration, verify in MongoDB:

```bash
mongosh "mongodb://user:pass@localhost:27017/odin_gateway?authSource=admin"
```

```javascript
// Count services
db.services.countDocuments()

// List all services
db.services.find({}, { name: 1, basePath: 1, targets: 1 }).pretty()

// Check specific service
db.services.findOne({ name: "user-service" })

// Verify indexes
db.services.getIndexes()
```

## Common Issues

### Issue: "MongoDB is not enabled in configuration"

**Solution:** Add MongoDB configuration to your config file:

```yaml
mongodb:
  enabled: true
  uri: "mongodb://localhost:27017"
  database: "odin_gateway"
```

### Issue: "Services already exist in MongoDB"

**Solution 1:** Use `--force` flag to overwrite:
```bash
./bin/odin-migrate --config config/config.yaml --force
```

**Solution 2:** Delete existing services manually:
```javascript
db.services.deleteMany({})
```

### Issue: "No services found to migrate"

**Solution:** Check your config file has services defined:
```yaml
services:
  - name: user-service
    basePath: /api/users
    targets:
      - http://localhost:3001
```

### Issue: "Failed to connect to MongoDB"

**Solution:** Verify MongoDB is running and credentials are correct:
```bash
# Check MongoDB status
sudo systemctl status mongod

# Test connection
mongosh "mongodb://localhost:27017"
```

## Rollback

If migration fails, services in MongoDB can be deleted:

```javascript
// Connect to MongoDB
mongosh "mongodb://user:pass@localhost:27017/odin_gateway?authSource=admin"

// Delete all services
db.services.deleteMany({})

// Delete specific service
db.services.deleteOne({ name: "user-service" })
```

Your original YAML files remain unchanged and can be used with Odin by setting `mongodb.enabled: false`.

## Best Practices

1. **Always run dry-run first**
   ```bash
   ./bin/odin-migrate --config config/config.yaml --dry-run --verbose
   ```

2. **Backup configuration before migration**
   ```bash
   cp -r config config.backup.$(date +%Y%m%d)
   ```

3. **Verify migration in test environment** before production

4. **Use verbose logging** to see detailed progress:
   ```bash
   ./bin/odin-migrate --config config/config.yaml --verbose
   ```

5. **Check MongoDB after migration**:
   ```bash
   mongosh "mongodb://..." --eval "db.services.countDocuments()"
   ```

## Audit Trail

The migration tool creates an audit log entry in MongoDB:

```javascript
db.audit_logs.findOne({ action: "migrate_services" })
```

This includes:
- Migration timestamp
- Number of services migrated
- Number of failures
- Migration tool identifier

## Performance

- Migration is fast: ~100 services per second
- Memory usage: ~50MB for 1000 services
- Network I/O: Depends on MongoDB connection speed

For large migrations (>1000 services), consider:
- Running on server with MongoDB co-located
- Using MongoDB bulk insert APIs
- Increasing connection pool size temporarily

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Migration failure (check logs) |

## Examples for Common Scenarios

### Local Development

```bash
./bin/odin-migrate \
  --config config/config.yaml \
  --mongodb-uri "mongodb://localhost:27017" \
  --verbose
```

### Production with Authentication

```bash
./bin/odin-migrate \
  --config config/config.yaml \
  --mongodb-uri "mongodb://odin_user:STRONG_PASSWORD@mongo1:27017,mongo2:27017,mongo3:27017/odin_gateway?replicaSet=rs0&authSource=admin" \
  --verbose
```

### MongoDB Atlas

```bash
./bin/odin-migrate \
  --config config/config.yaml \
  --mongodb-uri "mongodb+srv://odin_user:PASSWORD@cluster0.mongodb.net/?retryWrites=true&w=majority" \
  --mongodb-database "odin_gateway" \
  --verbose
```

## See Also

- [MongoDB Integration Guide](mongodb-integration.md)
- [Production Setup Guide](MONGODB_PRODUCTION_GUIDE.md)
- [Configuration Reference](configuration.md)
