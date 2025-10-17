# 🚀 Odin API Gateway - Complete Deployment Guide

**Version**: 1.0  
**Last Updated**: October 17, 2025  
**Status**: Production Ready

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [MongoDB Setup](#mongodb-setup)
4. [Gateway Configuration](#gateway-configuration)
5. [Building & Running](#building--running)
6. [Plugin Upload System](#plugin-upload-system)
7. [Admin Panel Access](#admin-panel-access)
8. [Production Deployment](#production-deployment)
9. [Monitoring & Maintenance](#monitoring--maintenance)
10. [Troubleshooting](#troubleshooting)

---

## Overview

This guide provides step-by-step instructions for deploying the Odin API Gateway with all features enabled, including the new **Plugin Binary Upload & Management System** (Goal #7).

### What's Included

✅ MongoDB setup for persistent storage  
✅ Plugin upload system with GridFS  
✅ Admin panel configuration  
✅ Security hardening  
✅ Production best practices  
✅ Monitoring and alerts  

---

## Prerequisites

### System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| **OS** | Linux, macOS, Windows | Ubuntu 22.04+ / RHEL 8+ |
| **CPU** | 2 cores | 4+ cores |
| **RAM** | 2 GB | 4+ GB |
| **Disk** | 10 GB | 50+ GB SSD |
| **Go** | 1.25.0 | 1.25.3 (latest) |

### Software Dependencies

```bash
# Required
- Go 1.25+
- Git

# Optional (but recommended)
- MongoDB 7.0+ (for plugin upload, metrics, config storage)
- Redis 7.0+ (for caching, rate limiting)
- Docker 24.0+ (for containerized deployment)
- Docker Compose 2.0+ (for orchestration)
```

### Network Requirements

| Port | Service | Purpose |
|------|---------|---------|
| 8080 | Gateway | Main API gateway port |
| 9090 | Metrics | Prometheus metrics endpoint |
| 27017 | MongoDB | Database (if not using Docker) |
| 6379 | Redis | Cache/rate limit (if not using Docker) |

---

## MongoDB Setup

MongoDB is **required** for the Plugin Binary Upload System (Goal #7) and recommended for production deployments.

### Option 1: Docker (Recommended)

**Single Instance (Development)**

```bash
# Start MongoDB container
docker run -d \
  --name odin-mongodb \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=your-secure-password \
  -v odin-mongo-data:/data/db \
  mongo:7.0

# Create Odin database and user
docker exec -it odin-mongodb mongosh -u admin -p your-secure-password --authenticationDatabase admin

# In mongosh:
use odin
db.createUser({
  user: "odin",
  pwd: "odin-password",
  roles: [
    { role: "readWrite", db: "odin" }
  ]
})
exit
```

**Docker Compose (Production-like)**

Create `docker-compose.mongodb.yml`:

```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:7.0
    container_name: odin-mongodb
    restart: unless-stopped
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_ROOT_PASSWORD}
    ports:
      - "27017:27017"
    volumes:
      - mongo-data:/data/db
      - mongo-config:/data/configdb
      - ./mongo-init:/docker-entrypoint-initdb.d
    networks:
      - odin-network
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  mongo-data:
    driver: local
  mongo-config:
    driver: local

networks:
  odin-network:
    driver: bridge
```

Create `mongo-init/init-odin.js`:

```javascript
db = db.getSiblingDB('odin');

// Create odin user
db.createUser({
  user: 'odin',
  pwd: 'odin-password', // Change in production
  roles: [
    { role: 'readWrite', db: 'odin' }
  ]
});

// Create collections
db.createCollection('plugins');
db.createCollection('services');
db.createCollection('metrics');
db.createCollection('audit_logs');

// Create indexes for plugin binary upload system
db.plugins.createIndex({ "name": 1, "version": 1 }, { unique: true });
db.plugins.createIndex({ "enabled": 1 });
db.plugins.createIndex({ "uploaded_at": -1 });
db.plugins.createIndex({ "name": "text", "description": "text" });

// Create indexes for GridFS (binary storage)
db.fs.files.createIndex({ "filename": 1 });
db.fs.files.createIndex({ "uploadDate": -1 });
db.fs.chunks.createIndex({ "files_id": 1, "n": 1 }, { unique: true });

print('Odin database initialized successfully');
```

Start MongoDB:

```bash
export MONGO_ROOT_PASSWORD="your-secure-password"
docker-compose -f docker-compose.mongodb.yml up -d
```

### Option 2: Local Installation

**Ubuntu/Debian**

```bash
# Import MongoDB public key
curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | \
  sudo gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor

# Add MongoDB repository
echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" | \
  sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list

# Install MongoDB
sudo apt-get update
sudo apt-get install -y mongodb-org

# Start MongoDB
sudo systemctl start mongod
sudo systemctl enable mongod
```

**macOS (Homebrew)**

```bash
brew tap mongodb/brew
brew install mongodb-community@7.0
brew services start mongodb-community@7.0
```

**Configure MongoDB**

```bash
# Connect to MongoDB
mongosh

# Create database and user
use odin
db.createUser({
  user: "odin",
  pwd: "odin-password",
  roles: [{ role: "readWrite", db: "odin" }]
})

# Create indexes (same as Docker init script)
db.plugins.createIndex({ "name": 1, "version": 1 }, { unique: true });
db.plugins.createIndex({ "enabled": 1 });
db.plugins.createIndex({ "uploaded_at": -1 });
db.fs.files.createIndex({ "filename": 1 });
db.fs.chunks.createIndex({ "files_id": 1, "n": 1 }, { unique: true });

exit
```

### Option 3: MongoDB Atlas (Cloud)

1. Sign up at https://www.mongodb.com/cloud/atlas
2. Create a free M0 cluster
3. Create database user: `odin` / `your-password`
4. Whitelist your IP address (or use `0.0.0.0/0` for testing)
5. Get connection string:
   ```
   mongodb+srv://odin:<password>@cluster0.xxxxx.mongodb.net/odin?retryWrites=true&w=majority
   ```

---

## Gateway Configuration

### Step 1: Clone Repository

```bash
git clone https://github.com/sepehr-mohseni/odin.git
cd odin
```

### Step 2: Configure MongoDB

Edit `config/config.yaml` and add MongoDB configuration:

```yaml
# Server Configuration
server:
  port: 8080
  timeout: 30s
  readTimeout: 30s
  writeTimeout: 30s
  gracefulTimeout: 30s
  compression: true

# Logging Configuration
logging:
  level: info  # debug, info, warn, error
  json: false  # Set true for production

# Authentication
auth:
  jwtSecret: "your-jwt-secret-key-change-in-production"
  accessTokenTTL: 15m
  refreshTokenTTL: 7d
  ignorePathRegexes:
    - "^/health$"
    - "^/metrics$"
    - "^/admin/login$"

# Admin Panel
admin:
  enabled: true
  username: admin
  password: admin  # CHANGE IN PRODUCTION!

# MongoDB Configuration (REQUIRED for Plugin Upload)
mongodb:
  enabled: true
  uri: "mongodb://odin:odin-password@localhost:27017"
  database: "odin"
  connectTimeout: 10s
  maxPoolSize: 100
  minPoolSize: 10
  
  # Authentication (if using auth)
  auth:
    username: "odin"
    password: "odin-password"
    authDB: "admin"
  
  # TLS (for production)
  tls:
    enabled: false
    certFile: ""
    keyFile: ""
    caFile: ""

# Plugin System (File-based - backward compatibility)
plugins:
  enabled: true
  directory: "./plugins"
  plugins: []

# Monitoring
monitoring:
  enabled: true
  path: "/metrics"
  webhookUrl: ""  # Optional: Slack/Discord webhook for alerts

# Rate Limiting
rateLimit:
  enabled: true
  limit: 100
  duration: 1m
  strategy: "token-bucket"
  redisUrl: ""  # Optional: redis://localhost:6379

# Caching
cache:
  enabled: true
  ttl: 5m
  strategy: "ttl"
  maxSizeInMB: 100
  redisUrl: ""  # Optional

# Tracing (Optional)
tracing:
  enabled: false
  serviceName: "odin-gateway"
  serviceVersion: "1.0.0"
  environment: "production"
  endpoint: "http://localhost:4318"
  sampleRate: 0.1
  insecure: true

# Services (Your backend services)
services:
  - name: example-service
    basePath: /api/v1
    targets:
      - url: http://localhost:3000
    stripBasePath: false
    timeout: 30s
    retryCount: 3
    retryDelay: 1s
    loadBalancing: round-robin
```

### Step 3: Environment Variables (Optional)

Create `.env` file for sensitive data:

```bash
# MongoDB
MONGO_URI=mongodb://odin:odin-password@localhost:27017
MONGO_DATABASE=odin

# Admin Credentials
ADMIN_USERNAME=admin
ADMIN_PASSWORD=your-secure-admin-password

# JWT Secret
JWT_SECRET=your-long-random-jwt-secret-key

# Gateway
GATEWAY_PORT=8080
LOG_LEVEL=info
```

Load environment variables:

```bash
export $(cat .env | xargs)
```

---

## Building & Running

### Development Mode

```bash
# Install dependencies
go mod download

# Run directly (with config file)
go run cmd/odin/main.go -config config/config.yaml

# Or use Makefile
make run
```

### Production Build

```bash
# Build optimized binary
make build

# Binary will be created at ./bin/odin
./bin/odin -config config/config.yaml
```

### Docker Deployment

**Build Image**

```bash
# Build Docker image
make docker

# Or manually
docker build -t odin-gateway:latest -f Dockerfile .
```

**Run Container**

```bash
# Run with external MongoDB
docker run -d \
  --name odin-gateway \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/config:/app/config \
  -e MONGO_URI=mongodb://odin:odin-password@host.docker.internal:27017 \
  odin-gateway:latest
```

**Docker Compose (Full Stack)**

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:7.0
    container_name: odin-mongodb
    restart: unless-stopped
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: ${MONGO_ROOT_PASSWORD:-admin}
    ports:
      - "27017:27017"
    volumes:
      - mongo-data:/data/db
    networks:
      - odin-network
    healthcheck:
      test: ["CMD", "mongosh", "--eval", "db.adminCommand('ping')"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: odin-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    networks:
      - odin-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  gateway:
    build: .
    container_name: odin-gateway
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      MONGO_URI: mongodb://admin:${MONGO_ROOT_PASSWORD:-admin}@mongodb:27017
      MONGO_DATABASE: odin
      REDIS_URL: redis://redis:6379
      LOG_LEVEL: info
    volumes:
      - ./config:/app/config
      - ./plugins:/app/plugins
    depends_on:
      mongodb:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - odin-network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  mongo-data:
    driver: local

networks:
  odin-network:
    driver: bridge
```

Start everything:

```bash
export MONGO_ROOT_PASSWORD="your-secure-password"
docker-compose up -d
```

---

## Plugin Upload System

The Plugin Binary Upload System (Goal #7) allows you to upload, manage, and hot-reload Go plugins through the admin web interface.

### Accessing the Plugin Upload UI

1. **Start Gateway** (ensure MongoDB is running)
   ```bash
   ./bin/odin -config config/config.yaml
   ```

2. **Open Admin Panel**
   - URL: http://localhost:8080/admin
   - Username: `admin`
   - Password: `admin` (or your configured password)

3. **Navigate to Plugin Upload**
   - Click "Plugin Binaries" in the navigation menu
   - Or go directly to: http://localhost:8080/admin/plugin-binaries/upload

### Building a Plugin

Create a plugin following the Odin plugin interface:

```go
// plugin/my-plugin/main.go
package main

import (
    "net/http"
    "github.com/labstack/echo/v4"
)

type MyPlugin struct {
    config map[string]interface{}
}

// Required: New function for plugin initialization
func New(config map[string]interface{}) (interface{}, error) {
    return &MyPlugin{config: config}, nil
}

// Middleware handler
func (p *MyPlugin) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Your middleware logic here
        // Example: Add custom header
        c.Response().Header().Set("X-My-Plugin", "active")
        
        // Call next handler
        return next(c)
    }
}
```

**Build the Plugin**

⚠️ **CRITICAL**: Plugin must be built with the **exact same Go version** as Odin!

```bash
# Check Odin's Go version
go version  # Should be go1.25.3 or your Odin version

# Build plugin
cd plugin/my-plugin
go build -buildmode=plugin -o my-plugin-1.0.0.so

# Verify it's a valid shared object
file my-plugin-1.0.0.so
# Output: my-plugin-1.0.0.so: ELF 64-bit LSB shared object, ...
```

### Uploading a Plugin

**Via Web UI (Recommended)**

1. Go to http://localhost:8080/admin/plugin-binaries/upload
2. Drag-and-drop your `.so` file or click to browse
3. Fill in the form:
   - **Name**: `my-plugin` (auto-filled from filename)
   - **Version**: `1.0.0` (semantic versioning)
   - **Description**: Brief description
   - **Author**: Your name/org
   - **Routes**: Route patterns to apply plugin (e.g., `/api/*`)
   - **Priority**: 0-1000 (default: 100, higher = earlier)
   - **Phase**: When to execute
     - `pre-routing`: Before route matching
     - `post-routing`: After route, before backend
     - `pre-response`: Before sending response
   - **Configuration**: JSON config for plugin
     ```json
     {
       "enabled": true,
       "timeout": "30s",
       "custom_value": "example"
     }
     ```
4. Click "Upload Plugin"

**Via API (cURL)**

```bash
curl -X POST http://localhost:8080/admin/api/plugin-binaries/upload \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "file=@my-plugin-1.0.0.so" \
  -F "name=my-plugin" \
  -F "version=1.0.0" \
  -F "description=My custom plugin" \
  -F "author=YourName" \
  -F 'routes=["/api/*"]' \
  -F "priority=100" \
  -F "phase=post-routing" \
  -F 'config={"enabled":true}'
```

### Managing Plugins

**List All Plugins**

- Web UI: http://localhost:8080/admin/plugin-binaries
- API: `GET /admin/api/plugin-binaries`

**Enable Plugin** (Hot-reload)

1. Click the toggle switch in the UI (OFF → ON)
2. Plugin is immediately loaded and active
3. No gateway restart required!

**Disable Plugin**

1. Click the toggle switch (ON → OFF)
2. Plugin is unloaded immediately
3. Requests no longer processed by plugin

**Update Configuration**

1. Click "Config" button for a plugin
2. Edit JSON configuration
3. Click "Save Changes"
4. Configuration updates immediately (if plugin is enabled)

**Delete Plugin**

1. Click "Delete" button
2. Confirm deletion
3. Plugin binary removed from GridFS
4. Metadata deleted from database

---

## Admin Panel Access

### Default Credentials

```
URL: http://localhost:8080/admin
Username: admin
Password: admin
```

⚠️ **CHANGE PASSWORD IN PRODUCTION!**

### Available Features

| Section | URL | Description |
|---------|-----|-------------|
| **Dashboard** | `/admin` | Overview, metrics, status |
| **Services** | `/admin/services` | Manage backend services |
| **Routes** | `/admin/routes` | Route configuration |
| **Plugins (File)** | `/admin/plugins` | File-based plugins (legacy) |
| **Plugin Binaries** | `/admin/plugin-binaries` | Binary plugin management |
| **Upload Plugin** | `/admin/plugin-binaries/upload` | Upload new plugin |
| **Integrations** | `/admin/integrations` | Postman, API management |
| **Middleware API** | `/admin/middleware` | Middleware configuration |

### Authentication

Admin panel uses JWT authentication:

1. **Login**: POST `/admin/api/login`
   ```bash
   curl -X POST http://localhost:8080/admin/api/login \
     -H "Content-Type: application/json" \
     -d '{"username":"admin","password":"admin"}'
   ```

2. **Response**: Returns JWT token
   ```json
   {
     "token": "eyJhbGciOiJIUzI1NiIs...",
     "expiresAt": "2025-10-17T12:00:00Z"
   }
   ```

3. **Use Token**: Include in subsequent requests
   ```bash
   curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
     http://localhost:8080/admin/api/plugin-binaries
   ```

---

## Production Deployment

### Security Checklist

- [ ] **Change default admin password**
  ```yaml
  admin:
    username: admin
    password: "use-strong-password-here"
  ```

- [ ] **Use strong JWT secret**
  ```yaml
  auth:
    jwtSecret: "generate-random-256-bit-key"
  ```

- [ ] **Enable MongoDB authentication**
  ```yaml
  mongodb:
    auth:
      username: "odin"
      password: "strong-mongodb-password"
  ```

- [ ] **Enable TLS/SSL**
  ```yaml
  mongodb:
    tls:
      enabled: true
      certFile: "/path/to/cert.pem"
      keyFile: "/path/to/key.pem"
      caFile: "/path/to/ca.pem"
  ```

- [ ] **Restrict admin panel access**
  - Use firewall rules
  - VPN/bastion host
  - IP whitelisting

- [ ] **Enable rate limiting**
  ```yaml
  rateLimit:
    enabled: true
    limit: 100
    duration: 1m
  ```

- [ ] **Set up monitoring alerts**
  ```yaml
  monitoring:
    webhookUrl: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
  ```

### Performance Tuning

**MongoDB Connection Pool**

```yaml
mongodb:
  maxPoolSize: 100  # Concurrent connections
  minPoolSize: 10   # Minimum idle connections
  connectTimeout: 10s
```

**Server Timeouts**

```yaml
server:
  timeout: 30s
  readTimeout: 30s
  writeTimeout: 30s
  gracefulTimeout: 30s
```

**Caching**

```yaml
cache:
  enabled: true
  ttl: 5m
  maxSizeInMB: 500  # Increase for better hit rate
  redisUrl: "redis://localhost:6379"
```

### High Availability

**MongoDB Replica Set**

```bash
# Configure 3-node replica set
docker-compose -f docker-compose.replica.yml up -d

# In config:
mongodb:
  uri: "mongodb://node1:27017,node2:27017,node3:27017/?replicaSet=rs0"
```

**Load Balancer Setup**

```nginx
# nginx.conf
upstream odin_gateway {
    server gateway1:8080;
    server gateway2:8080;
    server gateway3:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://odin_gateway;
    }
}
```

### Backup Strategy

**MongoDB Backups**

```bash
# Daily backup script
#!/bin/bash
DATE=$(date +%Y%m%d)
mongodump --uri="mongodb://odin:password@localhost:27017/odin" \
  --out="/backups/odin-$DATE"

# Compress
tar -czf "/backups/odin-$DATE.tar.gz" "/backups/odin-$DATE"
rm -rf "/backups/odin-$DATE"

# Keep last 7 days
find /backups -name "odin-*.tar.gz" -mtime +7 -delete
```

**Restore from Backup**

```bash
# Extract backup
tar -xzf /backups/odin-20251017.tar.gz -C /tmp

# Restore
mongorestore --uri="mongodb://odin:password@localhost:27017/odin" \
  /tmp/odin-20251017/odin
```

---

## Monitoring & Maintenance

### Prometheus Metrics

Access metrics at: http://localhost:9090/metrics

**Key Metrics**

```promql
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# Latency (95th percentile)
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Plugin upload count
plugin_uploads_total

# Active plugins
plugin_active_count

# MongoDB operations
mongodb_operations_total
```

### Health Checks

**Gateway Health**

```bash
curl http://localhost:8080/health
```

**MongoDB Health**

```bash
curl http://localhost:8080/admin/api/health/mongodb
```

**Plugin System Health**

```bash
curl http://localhost:8080/admin/api/plugin-binaries/stats
```

### Log Management

**Enable JSON Logging (Production)**

```yaml
logging:
  level: info
  json: true
```

**Log Rotation**

```bash
# logrotate config: /etc/logrotate.d/odin
/var/log/odin/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 odin odin
}
```

### Database Maintenance

**Create Indexes**

```javascript
// Connect to MongoDB
use odin

// Plugin indexes
db.plugins.createIndex({ "name": 1, "version": 1 }, { unique: true });
db.plugins.createIndex({ "enabled": 1 });
db.plugins.createIndex({ "uploaded_at": -1 });

// GridFS indexes
db.fs.files.createIndex({ "filename": 1 });
db.fs.chunks.createIndex({ "files_id": 1, "n": 1 }, { unique: true });

// Metrics indexes (with TTL)
db.metrics.createIndex({ "timestamp": 1 }, { expireAfterSeconds: 2592000 }); // 30 days
```

**Compact Collections**

```javascript
// Reclaim space
db.runCommand({ compact: "plugins" });
db.runCommand({ compact: "fs.files" });
db.runCommand({ compact: "fs.chunks" });
```

---

## Troubleshooting

### Common Issues

**Issue 1: Plugin Upload Fails**

```
Error: MongoDB not connected
```

**Solution**: Verify MongoDB is running and configured

```bash
# Check MongoDB
docker ps | grep mongodb

# Check connection in config.yaml
mongodb:
  enabled: true
  uri: "mongodb://odin:password@localhost:27017"
```

---

**Issue 2: Plugin Enable Fails**

```
Error: Plugin validation failed: Go version mismatch
```

**Solution**: Rebuild plugin with matching Go version

```bash
# Check Odin's Go version
./bin/odin --version

# Rebuild plugin with same version
go1.25.3 build -buildmode=plugin -o plugin.so
```

---

**Issue 3: Cannot Access Admin Panel**

```
Error: 401 Unauthorized
```

**Solution**: Check admin credentials

```yaml
admin:
  enabled: true
  username: admin
  password: admin  # Verify this matches
```

---

**Issue 4: MongoDB Connection Timeout**

```
Error: connection timeout
```

**Solution**: Check network and credentials

```bash
# Test connection
mongosh "mongodb://odin:password@localhost:27017/odin"

# Check firewall
sudo ufw allow 27017/tcp
```

---

**Issue 5: High Memory Usage**

**Solution**: Tune connection pools and cache

```yaml
mongodb:
  maxPoolSize: 50  # Reduce if memory limited
  
cache:
  maxSizeInMB: 100  # Reduce cache size
```

---

### Debug Mode

Enable debug logging:

```yaml
logging:
  level: debug
  json: false
```

Or via environment:

```bash
LOG_LEVEL=debug ./bin/odin -config config/config.yaml
```

### Support Resources

- **Documentation**: `/docs` directory
- **Examples**: `/examples` directory
- **GitHub Issues**: https://github.com/sepehr-mohseni/odin/issues
- **User Guide**: `docs/GOAL-7-USER-GUIDE.md`
- **MongoDB Setup**: `docs/GOAL-7-MONGODB-SETUP.md`

---

## Next Steps

✅ **Deployment Complete!**

1. **Upload Your First Plugin**
   - Build a custom plugin
   - Upload via admin panel
   - Enable and test

2. **Configure Services**
   - Add backend services in `config.yaml`
   - Test routing and load balancing

3. **Set Up Monitoring**
   - Configure Prometheus
   - Set up Grafana dashboards
   - Enable alerts

4. **Optimize Performance**
   - Tune connection pools
   - Enable caching
   - Configure rate limiting

5. **Explore Advanced Features**
   - GraphQL proxy
   - gRPC transcoding
   - Service mesh integration
   - WASM extensions

---

**Congratulations! Your Odin API Gateway is now deployed and ready for production! 🎉**

For detailed feature documentation, see the `/docs` directory.
