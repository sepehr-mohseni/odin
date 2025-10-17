# 🔌 Plugin Upload Quick Reference

**Goal #7: Plugin Binary Upload & Management System**

---

## ⚡ Quick Start (3 Steps)

### 1️⃣ Start MongoDB

```bash
docker run -d --name odin-mongodb -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=password123 \
  mongo:7.0
```

### 2️⃣ Configure Odin

Add to `config/config.yaml`:

```yaml
mongodb:
  enabled: true
  uri: "mongodb://admin:password123@localhost:27017"
  database: "odin"
```

### 3️⃣ Upload Plugin

- Go to: http://localhost:8080/admin/plugin-binaries/upload
- Drag & drop your `.so` file
- Click "Upload"
- Done! ✅

---

## 🛠️ Build a Plugin

### Required Structure

```go
package main

type MyPlugin struct {
    config map[string]interface{}
}

// REQUIRED: New function
func New(config map[string]interface{}) (interface{}, error) {
    return &MyPlugin{config: config}, nil
}

// REQUIRED: Handle method
func (p *MyPlugin) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Your logic here
        return next(c)
    }
}
```

### Build Command

⚠️ **Must use same Go version as Odin!**

```bash
go build -buildmode=plugin -o my-plugin-1.0.0.so
```

---

## 📊 Management Dashboard

### Access

```
URL: http://localhost:8080/admin/plugin-binaries
```

### Features

| Action | Button | Result |
|--------|--------|--------|
| **Enable** | Toggle ON | Hot-load plugin (no restart) |
| **Disable** | Toggle OFF | Unload plugin immediately |
| **View** | 👁️ View | Show full metadata |
| **Config** | ⚙️ Config | Edit JSON configuration |
| **Delete** | 🗑️ Delete | Remove plugin completely |

---

## 🔧 Configuration Options

### Upload Form Fields

| Field | Required | Example | Description |
|-------|----------|---------|-------------|
| **Name** | ✅ | `rate-limiter` | Plugin identifier |
| **Version** | ✅ | `1.0.0` | Semantic version |
| **Description** | ❌ | `Rate limit API requests` | Brief description |
| **Author** | ❌ | `Your Name` | Author/organization |
| **Routes** | ❌ | `/api/*` | Route patterns |
| **Priority** | ❌ | `100` | Execution order (0-1000) |
| **Phase** | ❌ | `post-routing` | When to execute |
| **Config** | ❌ | `{"rate": 100}` | JSON configuration |

### Execution Phases

| Phase | When | Use Case |
|-------|------|----------|
| `pre-routing` | Before route matching | Auth, rate limit |
| `post-routing` | After route, before backend | Transform request |
| `pre-response` | Before sending response | Transform response |

---

## 🔍 Validation Checks

Upload validation (automatic):

✅ **1. File Extension** - Must be `.so`  
✅ **2. File Size** - 0 < size ≤ 50 MB  
✅ **3. ELF Magic** - Valid shared object (`0x7f 0x45 0x4c 0x46`)  
✅ **4. Go Version** - Matches Odin's version  
✅ **5. Required Symbol** - Has `New` function  
✅ **6. Test Load** - Can be loaded with `plugin.Open()`  

---

## 📡 API Endpoints

### Upload

```bash
curl -X POST http://localhost:8080/admin/api/plugin-binaries/upload \
  -F "file=@plugin.so" \
  -F "name=my-plugin" \
  -F "version=1.0.0"
```

### List All

```bash
curl http://localhost:8080/admin/api/plugin-binaries
```

### Get Details

```bash
curl http://localhost:8080/admin/api/plugin-binaries/{id}
```

### Enable

```bash
curl -X POST http://localhost:8080/admin/api/plugin-binaries/{id}/enable
```

### Disable

```bash
curl -X POST http://localhost:8080/admin/api/plugin-binaries/{id}/disable
```

### Update Config

```bash
curl -X PUT http://localhost:8080/admin/api/plugin-binaries/{id}/config \
  -H "Content-Type: application/json" \
  -d '{"config": {"key": "value"}}'
```

### Delete

```bash
curl -X DELETE http://localhost:8080/admin/api/plugin-binaries/{id}
```

---

## 🚨 Troubleshooting

### Error: "Go version mismatch"

**Cause**: Plugin compiled with different Go version

**Fix**: Rebuild with matching version

```bash
# Check Odin's version
go version  # e.g., go1.25.3

# Rebuild plugin with same version
go build -buildmode=plugin -o plugin.so
```

---

### Error: "MongoDB not connected"

**Cause**: MongoDB not running or misconfigured

**Fix**: Verify MongoDB

```bash
# Check MongoDB
docker ps | grep mongodb

# Test connection
mongosh "mongodb://admin:password@localhost:27017"

# Check config.yaml
mongodb:
  enabled: true
  uri: "mongodb://..."
```

---

### Error: "Symbol 'New' not found"

**Cause**: Plugin missing required `New` function

**Fix**: Add New function

```go
func New(config map[string]interface{}) (interface{}, error) {
    return &MyPlugin{config: config}, nil
}
```

---

### Error: "Upload failed: file too large"

**Cause**: File exceeds 50 MB limit

**Fix**: 
1. Optimize plugin (remove unused code)
2. Build with `-ldflags="-s -w"` to strip symbols
3. Use `upx` to compress

```bash
go build -buildmode=plugin -ldflags="-s -w" -o plugin.so
upx --best plugin.so
```

---

## 💡 Best Practices

### ✅ DO

- **Version your plugins** - Use semantic versioning (1.0.0, 1.1.0, etc.)
- **Test locally first** - Load plugin with `plugin.Open()` before upload
- **Use descriptive names** - `rate-limiter` not `plugin1`
- **Document configuration** - Explain config JSON in description
- **Set appropriate priority** - Higher = earlier execution
- **Handle errors gracefully** - Return proper error messages

### ❌ DON'T

- **Don't skip validation** - Build errors will fail at enable
- **Don't hardcode values** - Use config JSON for flexibility
- **Don't block requests** - Keep plugin logic fast
- **Don't ignore Go version** - Version mismatch = won't load
- **Don't upload untested plugins** - Test enable/disable locally

---

## 📚 Example Plugins

### Rate Limiter

```go
package main

import (
    "net/http"
    "github.com/labstack/echo/v4"
)

type RateLimiter struct {
    rate int
}

func New(config map[string]interface{}) (interface{}, error) {
    rate := 100
    if r, ok := config["rate"].(float64); ok {
        rate = int(r)
    }
    return &RateLimiter{rate: rate}, nil
}

func (p *RateLimiter) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Rate limiting logic
        return next(c)
    }
}
```

**Config JSON:**
```json
{
  "rate": 100,
  "window": "1m"
}
```

### Request Logger

```go
package main

import (
    "log"
    "github.com/labstack/echo/v4"
)

type RequestLogger struct{}

func New(config map[string]interface{}) (interface{}, error) {
    return &RequestLogger{}, nil
}

func (p *RequestLogger) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        log.Printf("%s %s", c.Request().Method, c.Request().URL.Path)
        return next(c)
    }
}
```

### Header Injector

```go
package main

import (
    "github.com/labstack/echo/v4"
)

type HeaderInjector struct {
    headers map[string]string
}

func New(config map[string]interface{}) (interface{}, error) {
    headers := make(map[string]string)
    if h, ok := config["headers"].(map[string]interface{}); ok {
        for k, v := range h {
            if s, ok := v.(string); ok {
                headers[k] = s
            }
        }
    }
    return &HeaderInjector{headers: headers}, nil
}

func (p *HeaderInjector) Handle(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        for k, v := range p.headers {
            c.Response().Header().Set(k, v)
        }
        return next(c)
    }
}
```

**Config JSON:**
```json
{
  "headers": {
    "X-Custom-Header": "value",
    "X-API-Version": "v1"
  }
}
```

---

## 📈 Performance Tips

### Build Optimization

```bash
# Strip symbols and debug info
go build -buildmode=plugin -ldflags="-s -w" -o plugin.so

# Compress with UPX (optional)
upx --best plugin.so
```

### Runtime Optimization

- ✅ Cache expensive operations
- ✅ Use connection pools
- ✅ Minimize allocations
- ✅ Avoid blocking operations
- ✅ Use context for timeouts

---

## 🔗 Related Documentation

- **Complete Guide**: `docs/GOAL-7-SUMMARY.md`
- **User Guide**: `docs/GOAL-7-USER-GUIDE.md`
- **MongoDB Setup**: `docs/GOAL-7-MONGODB-SETUP.md`
- **Deployment Guide**: `docs/DEPLOYMENT-GUIDE.md`
- **API Reference**: In GOAL-7-SUMMARY.md

---

## ✅ Checklist

Before uploading a plugin:

- [ ] Plugin compiles with matching Go version
- [ ] Has `New(map[string]interface{}) (interface{}, error)` function
- [ ] Has `Handle(echo.HandlerFunc) echo.HandlerFunc` method
- [ ] Tested locally with `plugin.Open()`
- [ ] File size under 50 MB
- [ ] Semantic version number (e.g., 1.0.0)
- [ ] Configuration documented
- [ ] Routes and priority set appropriately

---

**Need Help?**

- 📖 Read: `docs/GOAL-7-USER-GUIDE.md`
- 🐛 Issues: https://github.com/sepehr-mohseni/odin/issues
- 💬 Discussions: GitHub Discussions

---

**Plugin Upload System - Ready to Use! 🚀**
