# MongoDB Integration - Production Setup Guide

## ⚠️ CRITICAL - For Production Use with 1000+ Users

This guide provides **battle-tested** MongoDB integration for Odin API Gateway. Follow these steps carefully to ensure zero downtime and data integrity.

## Table of Contents

1. [Pre-Migration Checklist](#pre-migration-checklist)
2. [Step-by-Step Migration](#step-by-step-migration)
3. [Verification & Testing](#verification--testing)
4. [Rollback Procedure](#rollback-procedure)
5. [Monitoring & Alerts](#monitoring--alerts)
6. [Troubleshooting](#troubleshooting)

---

## Pre-Migration Checklist

### 1. Backup Current Configuration

```bash
# Backup all config files
mkdir -p backup/$(date +%Y%m%d)
cp -r config/* backup/$(date +%Y%m%d)/

# Verify backup
ls -la backup/$(date +%Y%m%d)/
```

### 2. MongoDB Installation

**For Production (Replica Set - Recommended):**

```bash
# Install MongoDB 7.0+
wget -qO - https://www.mongodb.org/static/pgp/server-7.0.asc | sudo apt-key add -
echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/7.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list
sudo apt-get update
sudo apt-get install -y mongodb-org

# Start MongoDB
sudo systemctl start mongod
sudo systemctl enable mongod
sudo systemctl status mongod
```

**For Cloud (MongoDB Atlas - Easiest):**

1. Create cluster at https://cloud.mongodb.com
2. Configure network access (allowlist your IPs)
3. Create database user with readWrite permissions
4. Get connection string

### 3. Create MongoDB User

```javascript
// Connect to MongoDB
mongosh

// Switch to admin database
use admin

// Create Odin user with strong password
db.createUser({
  user: "odin_production",
  pwd: "CHANGE_THIS_TO_STRONG_PASSWORD",  // IMPORTANT: Change this!
  roles: [
    { role: "readWrite", db: "odin_gateway" },
    { role: "dbAdmin", db: "odin_gateway" }
  ]
})

// Verify user creation
db.getUsers()

// Create the database
use odin_gateway
db.createCollection("services")

// Verify
show collections
```

### 4. Test MongoDB Connection

```bash
# Test connection
mongosh "mongodb://odin_production:YOUR_PASSWORD@localhost:27017/odin_gateway?authSource=admin"

# Should see MongoDB prompt if successful
```

---

## Step-by-Step Migration

### Step 1: Configure MongoDB (DO NOT ENABLE YET)

Create or update `config/config.yaml`:

```yaml
# Gateway-specific configuration (stays in YAML)
server:
  port: 8080
  timeout: 30s
  readTimeout: 30s
  writeTimeout: 30s
  gracefulTimeout: 10s

logging:
  level: info
  json: true  # Important for production

# MongoDB Configuration (KEEP DISABLED FOR NOW)
mongodb:
  enabled: false  # Start with false!
  uri: "mongodb://odin_production:YOUR_PASSWORD@localhost:27017?authSource=admin"
  database: "odin_gateway"
  maxPoolSize: 200  # Adjust based on your load
  minPoolSize: 20
  connectTimeout: "10s"
  auth:
    username: "odin_production"
    password: "YOUR_PASSWORD"  # Use environment variable in production!
    authDB: "admin"
  tls:
    enabled: false  # Enable for production!
    caFile: ""
    certFile: ""
    keyFile: ""

# Your existing services configuration
services:
  - name: user-service
    basePath: /api/users
    targets:
      - http://users.internal:8080
    # ... rest of config
```

**⚠️ SECURITY WARNING:** In production, use environment variables for sensitive data:

```yaml
mongodb:
  uri: "${MONGODB_URI}"
  auth:
    password: "${MONGODB_PASSWORD}"
```

### Step 2: Dry Run Migration

```bash
# Build migration tool
cd /home/sep/code/odin
go build -o bin/odin-migrate ./cmd/migrate/

# Perform DRY RUN (no actual changes)
./bin/odin-migrate \
  --config config/config.yaml \
  --mongodb-uri "mongodb://odin_production:YOUR_PASSWORD@localhost:27017?authSource=admin" \
  --mongodb-database "odin_gateway" \
  --dry-run \
  --verbose

# Review output carefully!
# Expected output:
# - DRY RUN MODE - No changes will be made
# - Services that would be migrated: [list of services]
# - Migration would complete successfully (dry run)
```

### Step 3: Perform Actual Migration

```bash
# Run migration (for real this time)
./bin/odin-migrate \
  --config config/config.yaml \
  --mongodb-uri "mongodb://odin_production:YOUR_PASSWORD@localhost:27017?authSource=admin" \
  --mongodb-database "odin_gateway" \
  --verbose

# Expected output:
# ✅ Migration completed successfully!
# Total: XX services
# Migrated: XX
# Failed: 0
```

### Step 4: Verify Migration

```bash
# Connect to MongoDB and verify
mongosh "mongodb://odin_production:YOUR_PASSWORD@localhost:27017/odin_gateway?authSource=admin"
```

```javascript
// Check services collection
db.services.countDocuments()  // Should match your service count

// View all services
db.services.find().pretty()

// Check specific service
db.services.findOne({ name: "user-service" })

// Verify indexes
db.services.getIndexes()
```

### Step 5: Enable MongoDB in Configuration

Update `config/config.yaml`:

```yaml
mongodb:
  enabled: true  # ← CHANGE THIS
  uri: "mongodb://odin_production:YOUR_PASSWORD@localhost:27017?authSource=admin"
  # ... rest stays the same
```

### Step 6: Build and Deploy New Version

```bash
# Build with MongoDB integration
go build -o bin/odin ./cmd/odin/

# Test configuration
./bin/odin --config config/config.yaml --test-config

# If test passes, proceed with deployment
```

### Step 7: Rolling Restart (Zero Downtime)

**For Single Instance:**

```bash
# Stop current instance gracefully
sudo systemctl stop odin

# Start new version
sudo systemctl start odin

# Check logs
sudo journalctl -u odin -f --since "1 minute ago"

# Look for:
# "Connected to MongoDB successfully"
# "Loaded services from MongoDB"
```

**For Multiple Instances (Kubernetes/Docker):**

```bash
# Update deployment
kubectl set image deployment/odin odin=odin:latest

# Watch rollout
kubectl rollout status deployment/odin

# Verify pods
kubectl get pods -l app=odin
kubectl logs -f deployment/odin --since=1m
```

---

## Verification & Testing

### 1. Health Check

```bash
# Check gateway health
curl http://localhost:8080/health

# Check MongoDB health
curl http://localhost:8080/admin/api/mongodb/health \
  -u admin:your_admin_password

# Expected response:
# {
#   "status": "healthy",
#   "message": "MongoDB connection is healthy"
# }
```

### 2. Test Service Loading

```bash
# List all services via API
curl http://localhost:8080/admin/api/mongodb/services \
  -u admin:your_admin_password | jq .

# Get specific service
curl http://localhost:8080/admin/api/mongodb/services/user-service \
  -u admin:your_admin_password | jq .
```

### 3. Test Service Management

```bash
# Create test service
curl -X POST http://localhost:8080/admin/api/mongodb/services \
  -u admin:your_admin_password \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-service",
    "basePath": "/api/test",
    "targets": ["http://test.internal:8080"],
    "timeout": "30s",
    "retryCount": 3,
    "loadBalancing": "round_robin"
  }'

# Verify it works
curl http://localhost:8080/api/test/health

# Delete test service
curl -X DELETE http://localhost:8080/admin/api/mongodb/services/test-service \
  -u admin:your_admin_password
```

### 4. Test with Production Traffic

```bash
# Monitor response times
watch -n 1 'curl -w "\nTime: %{time_total}s\n" -s http://localhost:8080/api/users | head -1'

# Check error rates
curl http://localhost:8080/metrics | grep http_requests_total
```

---

## Rollback Procedure

If anything goes wrong, follow these steps:

### Immediate Rollback (< 5 minutes)

```bash
# 1. Stop new version
sudo systemctl stop odin

# 2. Disable MongoDB in config
sed -i 's/enabled: true/enabled: false/' config/config.yaml

# 3. Start old version
sudo systemctl start odin

# 4. Verify services are working
curl http://localhost:8080/health
```

### Data Loss Prevention

```bash
# Services remain in MongoDB even after rollback
# You can migrate back at any time

# Export services from MongoDB (just in case)
mongoexport --uri="mongodb://odin_production:PASSWORD@localhost:27017/odin_gateway?authSource=admin" \
  --collection=services \
  --out=services_backup_$(date +%Y%m%d_%H%M%S).json
```

---

## Monitoring & Alerts

### Prometheus Metrics

```yaml
# Add to prometheus.yml
scrape_configs:
  - job_name: 'odin-mongodb'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

**Key Metrics to Monitor:**

```
# Connection pool
mongodb_connection_pool_size{state="available"}
mongodb_connection_pool_size{state="in_use"}

# Operations
mongodb_operations_total{operation="find"}
mongodb_operations_total{operation="insert"}
mongodb_operations_total{operation="update"}

# Latency
mongodb_operation_duration_seconds{operation="find",quantile="0.99"}
```

### Alert Rules

```yaml
groups:
  - name: mongodb
    rules:
      - alert: MongoDBConnectionPoolExhausted
        expr: mongodb_connection_pool_size{state="available"} < 10
        for: 5m
        annotations:
          summary: "MongoDB connection pool running low"

      - alert: MongoDBHighLatency
        expr: mongodb_operation_duration_seconds{quantile="0.99"} > 1
        for: 5m
        annotations:
          summary: "MongoDB queries are slow"

      - alert: MongoDBConnectionFailed
        expr: up{job="odin-mongodb"} == 0
        for: 1m
        annotations:
          summary: "Cannot connect to MongoDB"
```

---

## Troubleshooting

### Issue: "Failed to connect to MongoDB"

**Symptoms:**
```
ERROR Failed to connect to MongoDB
```

**Solutions:**

1. Check MongoDB is running:
```bash
sudo systemctl status mongod
```

2. Test connection manually:
```bash
mongosh "mongodb://localhost:27017"
```

3. Check firewall:
```bash
sudo ufw status
sudo ufw allow 27017/tcp
```

4. Verify credentials:
```javascript
use admin
db.auth("odin_production", "PASSWORD")
```

### Issue: "Service not found"

**Symptoms:**
```
404 Service not found: user-service
```

**Solutions:**

1. Check services in MongoDB:
```javascript
db.services.find({ name: "user-service" })
```

2. Re-run migration:
```bash
./bin/odin-migrate --config config/config.yaml --force
```

### Issue: High Memory Usage

**Symptoms:**
- Memory continuously increasing
- OOM kills

**Solutions:**

1. Reduce connection pool:
```yaml
mongodb:
  maxPoolSize: 50  # Reduce from 200
  minPoolSize: 5   # Reduce from 20
```

2. Enable connection timeout:
```yaml
mongodb:
  connectTimeout: "5s"
```

### Issue: Slow Queries

**Symptoms:**
- High latency on service lookups
- Timeout errors

**Solutions:**

1. Check indexes:
```javascript
db.services.getIndexes()
```

2. Analyze slow queries:
```javascript
db.setProfilingLevel(1, { slowms: 100 })
db.system.profile.find().sort({ ts: -1 }).limit(10)
```

3. Add custom indexes if needed:
```javascript
db.services.createIndex({ basePath: 1 })
db.services.createIndex({ enabled: 1, name: 1 })
```

---

## Best Practices for 1000+ Users

### 1. Use Replica Sets

For high availability with 1000+ users, use MongoDB replica set:

```yaml
mongodb:
  uri: "mongodb://odin_production:PASSWORD@mongo1:27017,mongo2:27017,mongo3:27017/odin_gateway?replicaSet=rs0&authSource=admin"
```

### 2. Enable TLS

```yaml
mongodb:
  tls:
    enabled: true
    caFile: "/etc/odin/certs/ca.pem"
    certFile: "/etc/odin/certs/client.pem"
    keyFile: "/etc/odin/certs/client-key.pem"
```

### 3. Use Environment Variables

```bash
# /etc/systemd/system/odin.service
[Service]
Environment="MONGODB_URI=mongodb://..."
Environment="MONGODB_PASSWORD=..."
ExecStart=/usr/local/bin/odin --config /etc/odin/config.yaml
```

### 4. Regular Backups

```bash
# Add to crontab
0 2 * * * /usr/bin/mongodump --uri="mongodb://..." --db=odin_gateway --out=/backup/$(date +\%Y\%m\%d)
```

### 5. Monitor Disk Usage

```bash
# Check MongoDB data size
du -sh /var/lib/mongodb/

# Set up alerts when > 80% full
```

---

## Summary Checklist

Before going live:

- [ ] MongoDB installed and running
- [ ] Strong passwords configured
- [ ] Backup of current configuration
- [ ] Dry run migration successful
- [ ] Actual migration successful
- [ ] Services verified in MongoDB
- [ ] Health checks passing
- [ ] Test traffic successful
- [ ] Monitoring configured
- [ ] Alerts set up
- [ ] Rollback procedure tested
- [ ] Team notified of changes
- [ ] Documentation updated

---

## Support

If you encounter issues:

1. Check logs: `sudo journalctl -u odin -f`
2. Check MongoDB logs: `sudo tail -f /var/log/mongodb/mongod.log`
3. Review this guide's troubleshooting section
4. Check GitHub issues: https://github.com/sepehr-mohseni/odin/issues

**Emergency Rollback:** Disable MongoDB in config and restart. Services will load from YAML.

---

✅ **You're ready for production!**

Remember: Start with `enabled: false`, test thoroughly, then enable MongoDB.
