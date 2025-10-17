# MongoDB Setup for Plugin Binary Upload System

This guide explains how to set up MongoDB for the plugin binary upload and management system.

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Integration with Odin](#integration-with-odin)
5. [Database Schema](#database-schema)
6. [Testing](#testing)
7. [Troubleshooting](#troubleshooting)

---

## Overview

The plugin upload system uses MongoDB with GridFS for:
- **Binary Storage**: Plugin `.so` files stored in GridFS
- **Metadata**: Plugin information in the `plugins` collection
- **Indexing**: Fast queries and duplicate detection

### Why MongoDB + GridFS?

✅ **Efficient Binary Storage**: GridFS handles large files (up to 50MB per plugin)  
✅ **Chunked Storage**: Files split into 255KB chunks for efficient streaming  
✅ **Metadata Integration**: Binary and metadata in same database  
✅ **Scalability**: Supports replication and sharding  
✅ **No Filesystem Issues**: No file permissions, paths, or cleanup needed  

---

## Installation

### Option 1: Docker (Recommended)

```bash
# Start MongoDB container
docker run -d \
  --name odin-mongodb \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=password123 \
  -v odin_mongodb_data:/data/db \
  mongo:7.0
```

### Option 2: Docker Compose

Add to your `docker-compose.yml`:

```yaml
version: '3.8'

services:
  mongodb:
    image: mongo:7.0
    container_name: odin-mongodb
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: password123
    volumes:
      - odin_mongodb_data:/data/db
    networks:
      - odin-network

volumes:
  odin_mongodb_data:

networks:
  odin-network:
    driver: bridge
```

Start with:
```bash
docker-compose up -d mongodb
```

### Option 3: Local Installation

**Ubuntu/Debian**:
```bash
# Import MongoDB GPG key
wget -qO - https://www.mongodb.org/static/pgp/server-7.0.asc | sudo apt-key add -

# Add MongoDB repository
echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu $(lsb_release -cs)/mongodb-org/7.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list

# Install
sudo apt-get update
sudo apt-get install -y mongodb-org

# Start service
sudo systemctl start mongod
sudo systemctl enable mongod
```

**macOS**:
```bash
# Using Homebrew
brew tap mongodb/brew
brew install mongodb-community@7.0
brew services start mongodb-community@7.0
```

**Windows**:
1. Download MongoDB Community Server from [mongodb.com](https://www.mongodb.com/try/download/community)
2. Run the installer
3. Start MongoDB service

---

## Configuration

### 1. Create MongoDB User for Odin

```bash
# Connect to MongoDB
mongosh

# Switch to admin database
use admin

# Create Odin database user
db.createUser({
  user: "odin",
  pwd: "odin_secure_password",
  roles: [
    { role: "readWrite", db: "odin" }
  ]
})

# Exit
exit
```

### 2. Configure Odin Gateway

Add MongoDB configuration to `config/config.yaml`:

```yaml
server:
  port: 8080
  timeout: 30s

admin:
  enabled: true
  username: admin
  password: admin123

# MongoDB Configuration
mongodb:
  enabled: true
  uri: "mongodb://odin:odin_secure_password@localhost:27017"
  database: "odin"
  options:
    maxPoolSize: 100
    minPoolSize: 10
    connectTimeoutMS: 5000
    serverSelectionTimeoutMS: 5000

# Plugin System Configuration
plugins:
  enabled: true
  upload:
    enabled: true
    maxFileSize: 52428800  # 50MB in bytes
    allowedExtensions:
      - ".so"
  storage:
    type: "gridfs"
    gridfs:
      bucketName: "plugin_binaries"
      chunkSizeKB: 255

# ... rest of config
```

### 3. Environment Variables (Alternative)

For production, use environment variables:

```bash
export ODIN_MONGODB_URI="mongodb://odin:odin_secure_password@localhost:27017"
export ODIN_MONGODB_DATABASE="odin"
export ODIN_PLUGIN_UPLOAD_ENABLED="true"
export ODIN_PLUGIN_MAX_FILE_SIZE="52428800"
```

Update `config/config.yaml` to use env vars:

```yaml
mongodb:
  enabled: true
  uri: ${ODIN_MONGODB_URI}
  database: ${ODIN_MONGODB_DATABASE}
```

---

## Integration with Odin

### 1. Initialize MongoDB Connection

Update `cmd/odin/main.go` (or `cmd/gateway/main.go`):

```go
package main

import (
    "context"
    "log"
    "time"
    
    "odin/pkg/admin"
    "odin/pkg/config"
    "odin/pkg/plugins"
    
    "github.com/labstack/echo/v4"
    "github.com/sirupsen/logrus"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    // Load configuration
    cfg, err := config.Load("config/config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    logger := logrus.New()
    
    // Initialize MongoDB
    var mongoClient *mongo.Client
    var db *mongo.Database
    
    if cfg.MongoDB.Enabled {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        
        mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDB.URI))
        if err != nil {
            logger.Fatalf("Failed to connect to MongoDB: %v", err)
        }
        
        // Verify connection
        err = mongoClient.Ping(ctx, nil)
        if err != nil {
            logger.Fatalf("Failed to ping MongoDB: %v", err)
        }
        
        db = mongoClient.Database(cfg.MongoDB.Database)
        logger.Info("MongoDB connected successfully")
    }
    
    // Initialize Echo
    e := echo.New()
    
    // Initialize plugin manager
    pluginManager := plugins.NewPluginManager(logger)
    pluginRepo := plugins.NewPluginRepository()
    
    // Initialize admin handler
    adminHandler := admin.New(cfg, "config/config.yaml", logger)
    adminHandler.SetPluginHandler(pluginManager, pluginRepo)
    adminHandler.SetMiddlewareAPIHandler(pluginManager, pluginRepo)
    
    // Initialize plugin upload handler if MongoDB is enabled
    if db != nil && cfg.Plugins.Upload.Enabled {
        pluginUploadHandler, err := admin.NewPluginUploadHandler(db, pluginManager, logger)
        if err != nil {
            logger.Warnf("Failed to initialize plugin upload handler: %v", err)
        } else {
            adminHandler.SetPluginUploadHandler(pluginUploadHandler)
            logger.Info("Plugin upload handler initialized")
        }
    }
    
    // Register admin routes
    adminHandler.Register(e)
    
    // Start server
    logger.Infof("Starting Odin API Gateway on port %d", cfg.Server.Port)
    if err := e.Start(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
        logger.Fatalf("Failed to start server: %v", err)
    }
    
    // Cleanup on shutdown
    if mongoClient != nil {
        mongoClient.Disconnect(context.Background())
    }
}
```

### 2. Add MongoDB Config Structure

Update `pkg/config/config.go`:

```go
type Config struct {
    Server     ServerConfig     `yaml:"server"`
    Admin      AdminConfig      `yaml:"admin"`
    MongoDB    MongoDBConfig    `yaml:"mongodb"`     // Add this
    Plugins    PluginsConfig    `yaml:"plugins"`     // Add this
    // ... other fields
}

type MongoDBConfig struct {
    Enabled  bool              `yaml:"enabled"`
    URI      string            `yaml:"uri"`
    Database string            `yaml:"database"`
    Options  MongoOptions      `yaml:"options"`
}

type MongoOptions struct {
    MaxPoolSize              int `yaml:"maxPoolSize"`
    MinPoolSize              int `yaml:"minPoolSize"`
    ConnectTimeoutMS         int `yaml:"connectTimeoutMS"`
    ServerSelectionTimeoutMS int `yaml:"serverSelectionTimeoutMS"`
}

type PluginsConfig struct {
    Enabled bool               `yaml:"enabled"`
    Upload  PluginUploadConfig `yaml:"upload"`
    Storage PluginStorageConfig `yaml:"storage"`
}

type PluginUploadConfig struct {
    Enabled           bool     `yaml:"enabled"`
    MaxFileSize       int64    `yaml:"maxFileSize"`
    AllowedExtensions []string `yaml:"allowedExtensions"`
}

type PluginStorageConfig struct {
    Type    string          `yaml:"type"`
    GridFS  GridFSConfig    `yaml:"gridfs"`
}

type GridFSConfig struct {
    BucketName  string `yaml:"bucketName"`
    ChunkSizeKB int    `yaml:"chunkSizeKB"`
}
```

---

## Database Schema

### Collections

#### 1. `plugins` Collection

Stores plugin metadata:

```javascript
{
    _id: ObjectId("..."),
    name: "rate-limiter",
    version: "1.0.0",
    description: "Rate limiting middleware",
    author: "Team Odin",
    go_version: "go1.25",
    go_os: "linux",
    go_arch: "amd64",
    file_size: 2048576,
    sha256: "abc123...",
    config: {
        max_requests: 100,
        window: "1m"
    },
    enabled: false,
    uploaded_at: ISODate("2025-10-17T10:30:00Z"),
    enabled_at: null,
    routes: ["/*"],
    priority: 100,
    phase: "pre-routing",
    gridfs_file_id: ObjectId("...")
}
```

**Indexes**:
```javascript
db.plugins.createIndex({ "name": 1, "version": 1 }, { unique: true })
db.plugins.createIndex({ "enabled": 1 })
db.plugins.createIndex({ "uploaded_at": -1 })
db.plugins.createIndex({ "name": "text", "description": "text" })
```

#### 2. `fs.files` Collection (GridFS)

Stores file metadata:

```javascript
{
    _id: ObjectId("..."),
    filename: "rate-limiter-1.0.0.so",
    length: 2048576,
    chunkSize: 261120,
    uploadDate: ISODate("2025-10-17T10:30:00Z"),
    metadata: {
        plugin_id: ObjectId("..."),
        sha256: "abc123...",
        content_type: "application/octet-stream"
    }
}
```

#### 3. `fs.chunks` Collection (GridFS)

Stores binary data chunks:

```javascript
{
    _id: ObjectId("..."),
    files_id: ObjectId("..."),
    n: 0,
    data: BinData(0, "...")
}
```

### Initialize Collections and Indexes

Run this script to set up the database:

```javascript
// Connect to MongoDB
use odin

// Create plugins collection with indexes
db.plugins.createIndex({ "name": 1, "version": 1 }, { unique: true })
db.plugins.createIndex({ "enabled": 1 })
db.plugins.createIndex({ "uploaded_at": -1 })
db.plugins.createIndex({ "name": "text", "description": "text" })

// GridFS collections are created automatically
print("Database initialized successfully")
```

---

## Testing

### 1. Test MongoDB Connection

```bash
# Using mongosh
mongosh "mongodb://odin:odin_secure_password@localhost:27017/odin"

# Run test query
db.plugins.find()
```

### 2. Test Plugin Upload

```bash
# Build a test plugin
cd examples/plugins/hello-world
go build -buildmode=plugin -o hello-world-1.0.0.so

# Upload via API
curl -X POST http://localhost:8080/admin/api/plugin-binaries/upload \
  -H "Authorization: Basic YWRtaW46YWRtaW4xMjM=" \
  -F "file=@hello-world-1.0.0.so" \
  -F "name=hello-world" \
  -F "version=1.0.0" \
  -F "description=Test plugin" \
  -F "config={}"
```

### 3. Verify Storage

```bash
# Check GridFS files
mongosh "mongodb://odin:odin_secure_password@localhost:27017/odin" --eval "
db.fs.files.find().pretty()
"

# Check plugin metadata
mongosh "mongodb://odin:odin_secure_password@localhost:27017/odin" --eval "
db.plugins.find().pretty()
"

# Check file chunks count
mongosh "mongodb://odin:odin_secure_password@localhost:27017/odin" --eval "
db.fs.chunks.countDocuments()
"
```

---

## Troubleshooting

### Connection Issues

**Problem**: `Failed to connect to MongoDB`

**Solutions**:
1. Check MongoDB is running:
   ```bash
   # Docker
   docker ps | grep mongo
   
   # System service
   sudo systemctl status mongod
   ```

2. Verify connection string:
   ```bash
   # Test with mongosh
   mongosh "mongodb://odin:odin_secure_password@localhost:27017/odin"
   ```

3. Check firewall:
   ```bash
   # Allow port 27017
   sudo ufw allow 27017
   ```

### Authentication Errors

**Problem**: `Authentication failed`

**Solutions**:
1. Recreate user with correct credentials
2. Check username/password in config
3. Verify user has correct permissions

### Upload Failures

**Problem**: `Failed to upload plugin`

**Solutions**:
1. Check MongoDB disk space:
   ```bash
   df -h /data/db
   ```

2. Verify GridFS bucket:
   ```javascript
   db.fs.files.find()
   ```

3. Check file size limits in config

### Performance Issues

**Problem**: Slow queries or uploads

**Solutions**:
1. Add indexes if missing
2. Increase connection pool size
3. Monitor with:
   ```javascript
   db.serverStatus()
   db.currentOp()
   ```

---

## Production Recommendations

### 1. Security

- **Authentication**: Always use strong passwords
- **Encryption**: Enable TLS/SSL for connections
- **Network**: Restrict MongoDB to internal network only
- **Firewall**: Block external access to port 27017

```yaml
# config.yaml - Production MongoDB
mongodb:
  enabled: true
  uri: "mongodb://odin:STRONG_PASSWORD@mongodb-server:27017/odin?authSource=admin&ssl=true"
  database: "odin"
```

### 2. Backup

```bash
# Daily backup script
mongodump --uri="mongodb://odin:password@localhost:27017/odin" \
  --out="/backup/odin-$(date +%Y%m%d)"

# Restore
mongorestore --uri="mongodb://odin:password@localhost:27017/odin" \
  /backup/odin-20251017
```

### 3. Monitoring

```javascript
// Enable profiling
db.setProfilingLevel(1, { slowms: 100 })

// Check slow queries
db.system.profile.find().limit(10).sort({ ts: -1 }).pretty()
```

### 4. Replication (High Availability)

```yaml
# docker-compose.yml - Replica Set
version: '3.8'
services:
  mongo1:
    image: mongo:7.0
    command: mongod --replSet rs0
    
  mongo2:
    image: mongo:7.0
    command: mongod --replSet rs0
    
  mongo3:
    image: mongo:7.0
    command: mongod --replSet rs0
```

---

## Summary

✅ MongoDB provides efficient binary storage for plugins  
✅ GridFS handles large files with chunking  
✅ Integrated metadata and binary storage  
✅ Easy to set up and configure  
✅ Production-ready with replication support  

For more information:
- [MongoDB Documentation](https://docs.mongodb.com/)
- [GridFS Guide](https://docs.mongodb.com/manual/core/gridfs/)
- [Go MongoDB Driver](https://pkg.go.dev/go.mongodb.org/mongo-driver)

---

*Last Updated: October 17, 2025*
