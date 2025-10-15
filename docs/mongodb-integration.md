# MongoDB Integration

This document describes Odin's MongoDB integration for storing dynamic configuration, metrics, traces, and other gateway data.

## Overview

Odin can use MongoDB as its primary data store for all dynamic information, replacing file-based storage. This enables:

- **Dynamic Configuration**: Update services and routes without restarting
- **Centralized Storage**: Shared configuration across multiple gateway instances
- **Scalability**: Handle large volumes of metrics and traces
- **Persistence**: Durable storage with automatic backups
- **Query Capabilities**: Advanced querying and analytics
- **Automatic Cleanup**: TTL indexes for time-series data

## Architecture

### Data Model

Odin uses a document-based model with 13 collections:

| Collection | Purpose | TTL | Key Features |
|-----------|---------|-----|--------------|
| `services` | Service configurations | No | Unique name, versioning |
| `config` | Gateway configuration | No | Version history, active flag |
| `metrics` | Performance metrics | 30 days | Time-series, labels |
| `traces` | Distributed tracing | 7 days | Span data, correlation |
| `alerts` | Alert notifications | No | Status tracking, resolution |
| `health_checks` | Health status | 24 hours | Service monitoring |
| `clusters` | Multi-cluster config | No | Cluster state |
| `plugins` | WASM plugins | No | Plugin metadata |
| `users` | Admin users | No | Authentication, roles |
| `api_keys` | API authentication | No | Scoped access |
| `rate_limits` | Rate limit state | No | Counter windows |
| `cache` | Response cache | Custom | TTL per entry |
| `audit_logs` | Audit trail | 90 days | Change tracking |

### Repository Pattern

The MongoDB integration uses the repository pattern for clean separation:

```go
type Repository interface {
    // Service operations
    CreateService(ctx context.Context, service *ServiceDocument) error
    GetService(ctx context.Context, id string) (*ServiceDocument, error)
    ListServices(ctx context.Context, enabled *bool) ([]*ServiceDocument, error)
    UpdateService(ctx context.Context, id string, service *ServiceDocument) error
    DeleteService(ctx context.Context, id string) error
    
    // Config operations
    SaveConfig(ctx context.Context, config *ConfigDocument) error
    GetActiveConfig(ctx context.Context) (*ConfigDocument, error)
    
    // And 40+ more methods...
}
```

## Configuration

### Basic Setup

```yaml
mongodb:
  enabled: true
  uri: "mongodb://localhost:27017"
  database: "odin_gateway"
  maxPoolSize: 100
  minPoolSize: 10
  connectTimeout: "10s"
```

### Production Setup with Authentication

```yaml
mongodb:
  enabled: true
  uri: "mongodb://mongodb1.example.com:27017,mongodb2.example.com:27017,mongodb3.example.com:27017/?replicaSet=rs0"
  database: "odin_gateway"
  maxPoolSize: 200
  minPoolSize: 20
  connectTimeout: "15s"
  auth:
    username: "odin_admin"
    password: "SecurePassword123"
    authDB: "admin"
  tls:
    enabled: true
    caFile: "/etc/odin/certs/ca.pem"
    certFile: "/etc/odin/certs/client.pem"
    keyFile: "/etc/odin/certs/client-key.pem"
```

### MongoDB Atlas

```yaml
mongodb:
  enabled: true
  uri: "mongodb+srv://odin_user:password@cluster0.mongodb.net/?retryWrites=true&w=majority"
  database: "odin_gateway"
  maxPoolSize: 50
  minPoolSize: 5
  connectTimeout: "30s"
  tls:
    enabled: true
```

## Installation

### 1. Install MongoDB

**Docker:**
```bash
docker run -d \
  --name mongodb \
  -p 27017:27017 \
  -v mongodb_data:/data/db \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=password \
  mongo:7
```

**Linux:**
```bash
# Ubuntu/Debian
sudo apt-get install -y mongodb-org

# RHEL/CentOS
sudo yum install -y mongodb-org

# Start MongoDB
sudo systemctl start mongod
sudo systemctl enable mongod
```

**macOS:**
```bash
brew install mongodb-community
brew services start mongodb-community
```

### 2. Create Database and User

```javascript
// Connect to MongoDB
mongosh

// Switch to admin database
use admin

// Create Odin user
db.createUser({
  user: "odin_user",
  pwd: "SecurePassword123",
  roles: [
    { role: "readWrite", db: "odin_gateway" },
    { role: "dbAdmin", db: "odin_gateway" }
  ]
})

// Create database
use odin_gateway

// Verify access
db.runCommand({ connectionStatus: 1 })
```

### 3. Configure Odin

Create or update `config/config.yaml`:

```yaml
mongodb:
  enabled: true
  uri: "mongodb://odin_user:SecurePassword123@localhost:27017"
  database: "odin_gateway"
  maxPoolSize: 100
  minPoolSize: 10
  connectTimeout: "10s"
  auth:
    authDB: "admin"
```

### 4. Start Odin

```bash
./odin --config config/config.yaml
```

Check logs for successful connection:
```
INFO[0000] Connected to MongoDB database=odin_gateway uri=mongodb://***:***@***
```

## Migration from File-Based Storage

### Automatic Migration

Odin provides a migration tool to import existing YAML configurations:

```bash
# Migrate all services from routes.yaml
./odin migrate \
  --from-yaml config/routes.yaml \
  --mongodb-uri "mongodb://localhost:27017" \
  --database "odin_gateway"

# Migrate service definitions
./odin migrate \
  --from-yaml config/services.yaml \
  --mongodb-uri "mongodb://localhost:27017" \
  --database "odin_gateway"
```

### Manual Migration

1. **Export existing configuration:**
```bash
# Backup current config
cp config/routes.yaml config/routes.yaml.backup
cp config/services.yaml config/services.yaml.backup
```

2. **Start with MongoDB disabled:**
```yaml
mongodb:
  enabled: false
```

3. **Use Admin UI to migrate:**
- Navigate to `http://localhost:8080/admin`
- Go to "MongoDB Migration"
- Upload YAML files
- Review and confirm migration

4. **Enable MongoDB:**
```yaml
mongodb:
  enabled: true
```

5. **Restart Odin**

## Usage

### Service Management via API

**Create a service:**
```bash
curl -X POST http://localhost:8080/admin/api/services \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-service",
    "host": "users.internal:8080",
    "basePath": "/api/users",
    "enabled": true,
    "healthCheck": {
      "enabled": true,
      "path": "/health",
      "interval": "30s"
    }
  }'
```

**List services:**
```bash
curl http://localhost:8080/admin/api/services
```

**Update a service:**
```bash
curl -X PUT http://localhost:8080/admin/api/services/user-service \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-service",
    "host": "users.internal:8081",
    "enabled": true
  }'
```

**Delete a service:**
```bash
curl -X DELETE http://localhost:8080/admin/api/services/user-service
```

### Direct MongoDB Queries

**Connect to MongoDB:**
```bash
mongosh "mongodb://odin_user:password@localhost:27017/odin_gateway?authSource=admin"
```

**List all services:**
```javascript
db.services.find().pretty()
```

**Find enabled services:**
```javascript
db.services.find({ enabled: true }).pretty()
```

**Get active configuration:**
```javascript
db.config.findOne({ active: true })
```

**Query metrics:**
```javascript
// Get metrics from last hour
db.metrics.find({
  name: "http_requests_total",
  timestamp: { 
    $gte: new Date(Date.now() - 3600000) 
  }
}).sort({ timestamp: -1 })
```

**Query traces:**
```javascript
// Get traces for a service
db.traces.find({
  serviceName: "user-service",
  startTime: {
    $gte: new Date("2024-01-01")
  }
}).sort({ startTime: -1 }).limit(10)
```

**Check health history:**
```javascript
// Get recent health checks
db.health_checks.find({
  serviceName: "user-service"
}).sort({ checkedAt: -1 }).limit(20)
```

**Audit logs:**
```javascript
// Get recent changes
db.audit_logs.find().sort({ timestamp: -1 }).limit(50)
```

## Data Management

### Backup

**Using mongodump:**
```bash
# Full backup
mongodump \
  --uri="mongodb://odin_user:password@localhost:27017" \
  --db=odin_gateway \
  --out=/backup/odin/$(date +%Y%m%d)

# Backup specific collections
mongodump \
  --uri="mongodb://odin_user:password@localhost:27017" \
  --db=odin_gateway \
  --collection=services \
  --out=/backup/odin/services
```

**Automated backups:**
```bash
# Add to crontab
0 2 * * * /usr/local/bin/mongodump --uri="mongodb://localhost:27017" --db=odin_gateway --out=/backup/odin/$(date +\%Y\%m\%d)
```

### Restore

```bash
# Restore full backup
mongorestore \
  --uri="mongodb://odin_user:password@localhost:27017" \
  --db=odin_gateway \
  /backup/odin/20240101/odin_gateway

# Restore specific collection
mongorestore \
  --uri="mongodb://odin_user:password@localhost:27017" \
  --db=odin_gateway \
  --collection=services \
  /backup/odin/services/odin_gateway/services.bson
```

### Data Cleanup

**Manual cleanup:**
```javascript
// Remove old metrics (older than 30 days)
db.metrics.deleteMany({
  timestamp: { $lt: new Date(Date.now() - 30 * 24 * 3600000) }
})

// Remove old traces (older than 7 days)
db.traces.deleteMany({
  startTime: { $lt: new Date(Date.now() - 7 * 24 * 3600000) }
})

// Remove old audit logs (older than 90 days)
db.audit_logs.deleteMany({
  timestamp: { $lt: new Date(Date.now() - 90 * 24 * 3600000) }
})
```

**Automatic cleanup** is handled by TTL indexes (configured automatically).

### Indexes

Odin automatically creates indexes for optimal performance:

```javascript
// View all indexes
db.services.getIndexes()
db.metrics.getIndexes()
db.traces.getIndexes()

// Create custom indexes if needed
db.metrics.createIndex({ "labels.endpoint": 1, timestamp: -1 })
db.traces.createIndex({ "tags.http_status": 1 })
```

## Performance Tuning

### Connection Pool

Adjust pool size based on load:

```yaml
mongodb:
  maxPoolSize: 200  # High traffic
  minPoolSize: 20   # Maintain minimum connections
```

### Write Concern

For high-throughput scenarios:

```yaml
mongodb:
  uri: "mongodb://localhost:27017/?w=1&journal=true"
```

Options:
- `w=1`: Acknowledge after writing to primary (default)
- `w=majority`: Acknowledge after majority of replica set
- `journal=true`: Ensure write to journal

### Read Preference

For read-heavy workloads with replica sets:

```yaml
mongodb:
  uri: "mongodb://localhost:27017/?readPreference=secondaryPreferred"
```

Options:
- `primary`: Read from primary only
- `primaryPreferred`: Prefer primary
- `secondary`: Read from secondary only
- `secondaryPreferred`: Prefer secondary
- `nearest`: Lowest latency

## Monitoring

### Health Check

Check MongoDB connection status:

```bash
curl http://localhost:8080/health
```

Response includes MongoDB status:
```json
{
  "status": "healthy",
  "mongodb": {
    "connected": true,
    "database": "odin_gateway",
    "latency_ms": 2
  }
}
```

### Prometheus Metrics

Odin exports MongoDB metrics:

```
# Connection pool
mongodb_connection_pool_size{state="available"} 95
mongodb_connection_pool_size{state="in_use"} 5

# Operations
mongodb_operations_total{operation="insert"} 1543
mongodb_operations_total{operation="update"} 892
mongodb_operations_total{operation="find"} 4521

# Latency
mongodb_operation_duration_seconds{operation="find",quantile="0.99"} 0.015
```

### MongoDB Monitoring

**Current operations:**
```javascript
db.currentOp()
```

**Database stats:**
```javascript
db.stats()
```

**Collection stats:**
```javascript
db.services.stats()
db.metrics.stats()
```

**Slow query log:**
```javascript
// Enable profiling
db.setProfilingLevel(1, { slowms: 100 })

// View slow queries
db.system.profile.find().sort({ ts: -1 }).limit(10)
```

## Troubleshooting

### Connection Issues

**Problem:** Cannot connect to MongoDB

**Solution:**
1. Check MongoDB is running: `systemctl status mongod`
2. Verify URI and credentials
3. Check firewall rules: `telnet localhost 27017`
4. Review MongoDB logs: `/var/log/mongodb/mongod.log`

### Authentication Errors

**Problem:** Authentication failed

**Solution:**
```javascript
// Verify user exists
use admin
db.getUsers()

// Check user permissions
db.getUser("odin_user")

// Update password
db.changeUserPassword("odin_user", "NewPassword123")
```

### Performance Issues

**Problem:** Slow queries

**Solution:**
1. **Check indexes:**
```javascript
db.metrics.find({ name: "http_requests_total" }).explain("executionStats")
```

2. **Add missing indexes:**
```javascript
db.metrics.createIndex({ name: 1, timestamp: -1 })
```

3. **Optimize connection pool:**
```yaml
mongodb:
  maxPoolSize: 200
  minPoolSize: 50
```

### Disk Space

**Problem:** MongoDB using too much disk

**Solution:**
1. **Check collection sizes:**
```javascript
db.stats()
db.metrics.stats().size
db.traces.stats().size
```

2. **Verify TTL indexes are working:**
```javascript
db.metrics.getIndexes()
// Look for expireAfterSeconds: 0
```

3. **Manual cleanup if needed:**
```javascript
db.metrics.deleteMany({ 
  timestamp: { $lt: new Date(Date.now() - 30 * 24 * 3600000) }
})
```

4. **Compact collections:**
```javascript
db.runCommand({ compact: "metrics" })
```

## Security

### Network Security

1. **Bind to localhost only:**
```yaml
# /etc/mongod.conf
net:
  bindIp: 127.0.0.1
```

2. **Use firewall rules:**
```bash
# Allow only from gateway servers
sudo ufw allow from 10.0.1.0/24 to any port 27017
```

### Authentication

1. **Enable authentication:**
```yaml
# /etc/mongod.conf
security:
  authorization: enabled
```

2. **Use strong passwords:**
- Minimum 16 characters
- Mix of uppercase, lowercase, numbers, symbols
- Use password manager

3. **Principle of least privilege:**
```javascript
// Create read-only user for monitoring
db.createUser({
  user: "odin_monitor",
  pwd: "MonitorPassword123",
  roles: [{ role: "read", db: "odin_gateway" }]
})
```

### Encryption

1. **TLS for connections:**
```yaml
mongodb:
  tls:
    enabled: true
    caFile: "/etc/odin/certs/ca.pem"
    certFile: "/etc/odin/certs/client.pem"
    keyFile: "/etc/odin/certs/client-key.pem"
```

2. **Encryption at rest:**
```yaml
# /etc/mongod.conf
security:
  enableEncryption: true
  encryptionKeyFile: /etc/mongodb/encryption-key
```

### Audit Logging

Enable MongoDB audit log:
```yaml
# /etc/mongod.conf
auditLog:
  destination: file
  format: JSON
  path: /var/log/mongodb/audit.json
```

## Best Practices

1. **Use replica sets in production** for high availability
2. **Enable TLS** for all connections
3. **Regular backups** with automated testing
4. **Monitor disk usage** and set up alerts
5. **Use strong authentication** with unique users per service
6. **Keep MongoDB updated** with latest security patches
7. **Test disaster recovery** procedures regularly
8. **Document your schema** and indexing strategy
9. **Use connection pooling** efficiently
10. **Monitor slow queries** and optimize indexes

## See Also

- [Configuration Guide](configuration.md)
- [Admin Dashboard](../pkg/admin/README.md)
- [Monitoring](monitoring.md)
- [MongoDB Documentation](https://docs.mongodb.com/)
