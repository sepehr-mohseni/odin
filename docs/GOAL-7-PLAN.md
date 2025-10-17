# Goal #7: Go Plugin Upload & Registration from Admin Panel

## 🎯 Objective
Enable users to upload Go plugin (`.so`) files through the admin panel and register them as middleware, similar to Traefik's plugin registration system. This builds directly on Goal #5's middleware chain system.

## 📊 Current State

**Already Implemented (Goal #5):**
- ✅ Middleware chain system (Traefik-style)
- ✅ Plugin registration API
- ✅ Middleware testing framework
- ✅ Rollback capabilities
- ✅ Metrics collection
- ✅ MongoDB storage for middleware configs

**What's Missing:**
- ❌ Plugin file upload via admin UI
- ❌ Plugin validation system
- ❌ Binary file storage (GridFS)
- ❌ UI for uploading and managing plugins
- ❌ Security validation for uploaded plugins
- ❌ Version management for plugins

## 🎨 Design Overview

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Admin Panel UI                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │
│  │ Upload Page  │  │ Plugin List  │  │ Middleware   │     │
│  │              │  │              │  │ Chain        │     │
│  │ - Drag/Drop  │  │ - Enable/    │  │ (Goal #5)    │     │
│  │ - Validate   │  │   Disable    │  │              │     │
│  │ - Configure  │  │ - Version    │  │              │     │
│  └──────────────┘  └──────────────┘  └──────────────┘     │
└────────────┬────────────────────────────────────────────────┘
             │
             │ HTTPS
             ▼
┌─────────────────────────────────────────────────────────────┐
│                    Gateway Backend                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Plugin Upload API                       │   │
│  │  - POST /admin/api/plugins/upload                   │   │
│  │  - GET  /admin/api/plugins                          │   │
│  │  - DELETE /admin/api/plugins/:id                    │   │
│  │  - POST /admin/api/plugins/:id/enable               │   │
│  │  - POST /admin/api/plugins/:id/disable              │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │           Plugin Validation System                   │   │
│  │  - File type check (.so only)                       │   │
│  │  - Size validation (max 50MB)                       │   │
│  │  - Go version compatibility check                   │   │
│  │  - Symbol validation (required functions)           │   │
│  │  - Interface compliance check                       │   │
│  │  - Security scan (basic)                            │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Plugin Manager                          │   │
│  │  - Load plugins dynamically                         │   │
│  │  - Register with middleware chain (Goal #5)         │   │
│  │  - Version management                               │   │
│  │  - Enable/Disable plugins                           │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────┬────────────────────────────────────────────────┘
             │
             │
             ▼
┌─────────────────────────────────────────────────────────────┐
│                     MongoDB Storage                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  plugins collection                                  │   │
│  │  {                                                   │   │
│  │    id: ObjectId,                                     │   │
│  │    name: string,                                     │   │
│  │    version: string,                                  │   │
│  │    file_id: ObjectId (GridFS),                      │   │
│  │    enabled: boolean,                                 │   │
│  │    config: object,                                   │   │
│  │    go_version: string,                              │   │
│  │    uploaded_at: timestamp,                          │   │
│  │    updated_at: timestamp,                           │   │
│  │    metadata: {...}                                   │   │
│  │  }                                                   │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  fs.files & fs.chunks (GridFS)                      │   │
│  │  - Binary storage for .so files                     │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 📋 Task Breakdown

### Task 1: Planning & Design ⏳ (IN PROGRESS)
**Goal:** Create comprehensive plan and design documents

- [x] Define architecture
- [x] Design API endpoints
- [ ] Create UI mockups/wireframes
- [ ] Define security requirements
- [ ] Plan MongoDB schema
- [ ] Identify integration points with Goal #5

**Deliverables:**
- GOAL-7-PLAN.md (this document)
- API specification
- Security requirements doc

### Task 2: Backend API - Plugin Upload ⏳
**Goal:** Implement secure file upload endpoint

**Implementation:**
```go
// pkg/plugins/upload.go

type PluginUploadRequest struct {
    File     multipart.File
    Name     string
    Version  string
    Config   map[string]interface{}
}

type PluginInfo struct {
    ID         string
    Name       string
    Version    string
    FileID     string // GridFS file ID
    Enabled    bool
    Config     map[string]interface{}
    GoVersion  string
    UploadedAt time.Time
    UpdatedAt  time.Time
}

// POST /admin/api/plugins/upload
func (pm *PluginManager) UploadPlugin(c echo.Context) error {
    // 1. Parse multipart form
    // 2. Validate file (type, size)
    // 3. Extract metadata
    // 4. Validate Go version compatibility
    // 5. Store in GridFS
    // 6. Save metadata to MongoDB
    // 7. Return plugin info
}
```

**Validation Rules:**
- File extension: `.so` only
- Max file size: 50MB
- Go version: Must match gateway's Go version (1.25.3)
- Required symbols: Check for standard plugin functions

**API Endpoints:**
```
POST   /admin/api/plugins/upload
  - Body: multipart/form-data
  - Fields: file, name, version, config (JSON)
  - Response: { id, name, version, status, file_id }
  
GET    /admin/api/plugins
  - Query: ?enabled=true/false, ?name=xxx
  - Response: [{ id, name, version, enabled, ... }]
  
GET    /admin/api/plugins/:id
  - Response: { id, name, version, config, enabled, ... }
  
DELETE /admin/api/plugins/:id
  - Response: { success: true }
  
POST   /admin/api/plugins/:id/enable
  - Response: { success: true, enabled: true }
  
POST   /admin/api/plugins/:id/disable
  - Response: { success: true, enabled: false }
  
PUT    /admin/api/plugins/:id/config
  - Body: { config: {...} }
  - Response: { success: true }
```

### Task 3: Backend API - Plugin Management ⏳
**Goal:** Implement plugin lifecycle management

**Features:**
- List all plugins (with filters)
- Get plugin details
- Enable/Disable plugins
- Delete plugins (with confirmation)
- Update plugin configuration
- Version management

**Integration with Goal #5:**
```go
// When plugin is enabled, register with middleware chain
func (pm *PluginManager) EnablePlugin(id string) error {
    plugin := pm.GetPlugin(id)
    
    // 1. Load plugin from GridFS
    // 2. Open plugin (.so file)
    // 3. Lookup required symbols
    // 4. Register with middleware chain (Goal #5)
    // 5. Update status in MongoDB
    
    return pm.middlewareChain.RegisterMiddleware(plugin.Name, plugin.Handler)
}
```

### Task 4: Plugin Validation System ⏳
**Goal:** Ensure uploaded plugins are safe and compatible

**Validation Checks:**

1. **File Type Validation**
   ```go
   func ValidateFileType(filename string) error {
       if !strings.HasSuffix(filename, ".so") {
           return errors.New("only .so files allowed")
       }
       return nil
   }
   ```

2. **Size Validation**
   ```go
   func ValidateFileSize(size int64) error {
       maxSize := int64(50 * 1024 * 1024) // 50MB
       if size > maxSize {
           return errors.New("file too large")
       }
       return nil
   }
   ```

3. **Go Version Check**
   ```go
   func ValidateGoVersion(pluginPath string) error {
       // Use debug/elf or debug/macho to read build info
       // Ensure plugin was built with same Go version as gateway
       return nil
   }
   ```

4. **Symbol Validation**
   ```go
   func ValidateSymbols(pluginPath string) error {
       p, err := plugin.Open(pluginPath)
       if err != nil {
           return err
       }
       
       // Check for required functions
       requiredSymbols := []string{"New", "Handler"}
       for _, symbol := range requiredSymbols {
           _, err := p.Lookup(symbol)
           if err != nil {
               return fmt.Errorf("missing symbol: %s", symbol)
           }
       }
       return nil
   }
   ```

5. **Security Scan**
   - Check for suspicious strings
   - Verify plugin doesn't import dangerous packages
   - Basic static analysis

### Task 5: MongoDB Schema & Storage ⏳
**Goal:** Design database schema for plugin storage

**Collections:**

1. **plugins** collection:
```json
{
  "_id": ObjectId("..."),
  "name": "rate-limiter",
  "version": "1.0.0",
  "description": "Custom rate limiting plugin",
  "file_id": ObjectId("..."),  // GridFS reference
  "filename": "rate-limiter.so",
  "file_size": 2048576,
  "enabled": true,
  "config": {
    "max_requests": 100,
    "window": "1m"
  },
  "go_version": "1.25.3",
  "go_os": "linux",
  "go_arch": "amd64",
  "author": "admin",
  "uploaded_by": "user_id",
  "uploaded_at": ISODate("2025-10-17T10:00:00Z"),
  "updated_at": ISODate("2025-10-17T10:00:00Z"),
  "last_enabled_at": ISODate("2025-10-17T10:00:00Z"),
  "usage_count": 42,
  "status": "active",  // active, disabled, error
  "error_message": null,
  "metadata": {
    "tags": ["rate-limiting", "security"],
    "routes": ["/*"],
    "priority": 100
  }
}
```

2. **plugin_versions** collection (for version history):
```json
{
  "_id": ObjectId("..."),
  "plugin_id": ObjectId("..."),
  "version": "1.0.0",
  "file_id": ObjectId("..."),
  "uploaded_at": ISODate("2025-10-17T10:00:00Z"),
  "is_active": true
}
```

**GridFS for Binary Storage:**
- Collection: `fs.files` and `fs.chunks`
- Stores `.so` binary files
- Automatic chunking for large files
- Efficient retrieval

**Indexes:**
```javascript
db.plugins.createIndex({ "name": 1, "version": 1 }, { unique: true })
db.plugins.createIndex({ "enabled": 1 })
db.plugins.createIndex({ "uploaded_at": -1 })
db.plugin_versions.createIndex({ "plugin_id": 1, "version": 1 })
```

### Task 6: Frontend - Plugin Upload UI ⏳
**Goal:** Create user-friendly upload interface

**Page: `/admin/plugins/upload`**

**Features:**
- Drag-and-drop file upload
- File browser fallback
- Real-time validation feedback
- Progress indicator during upload
- Configuration form (JSON editor)
- Success/Error messaging

**HTML Structure:**
```html
<div class="plugin-upload-container">
  <h2>Upload New Plugin</h2>
  
  <!-- Upload Area -->
  <div class="upload-area" id="dropzone">
    <i class="icon-upload"></i>
    <p>Drag and drop .so file here</p>
    <p>or</p>
    <button class="btn btn-primary">Browse Files</button>
    <input type="file" id="fileInput" accept=".so" hidden>
  </div>
  
  <!-- Plugin Details Form -->
  <form id="pluginForm" style="display:none;">
    <div class="form-group">
      <label>Plugin Name *</label>
      <input type="text" name="name" required>
    </div>
    
    <div class="form-group">
      <label>Version *</label>
      <input type="text" name="version" placeholder="1.0.0" required>
    </div>
    
    <div class="form-group">
      <label>Description</label>
      <textarea name="description"></textarea>
    </div>
    
    <div class="form-group">
      <label>Configuration (JSON)</label>
      <textarea name="config" rows="10">{}</textarea>
    </div>
    
    <div class="form-group">
      <label>Apply to Routes</label>
      <input type="text" name="routes" placeholder="/*">
    </div>
    
    <button type="submit" class="btn btn-success">Upload Plugin</button>
  </form>
  
  <!-- Progress Indicator -->
  <div class="upload-progress" style="display:none;">
    <progress value="0" max="100"></progress>
    <span class="progress-text">Uploading...</span>
  </div>
</div>
```

**JavaScript (`plugin-upload.js`):**
```javascript
// Drag and drop functionality
const dropzone = document.getElementById('dropzone');
dropzone.addEventListener('drop', handleFileDrop);
dropzone.addEventListener('dragover', (e) => e.preventDefault());

async function uploadPlugin(formData) {
    const response = await fetch('/admin/api/plugins/upload', {
        method: 'POST',
        body: formData,
        onUploadProgress: (progressEvent) => {
            const percent = (progressEvent.loaded / progressEvent.total) * 100;
            updateProgress(percent);
        }
    });
    
    if (response.ok) {
        showSuccess('Plugin uploaded successfully!');
        redirectTo('/admin/plugins');
    } else {
        showError(await response.text());
    }
}
```

### Task 7: Frontend - Plugin Management UI ⏳
**Goal:** Build comprehensive plugin management interface

**Page: `/admin/plugins`**

**Features:**
- Table view of all plugins
- Enable/Disable toggle switches
- Delete with confirmation
- Edit configuration modal
- Filter by status (enabled/disabled)
- Search by name
- Version display
- Integration with middleware chain UI (Goal #5)

**HTML Structure:**
```html
<div class="plugins-container">
  <div class="plugins-header">
    <h2>Plugins</h2>
    <button class="btn btn-primary" onclick="location.href='/admin/plugins/upload'">
      <i class="icon-plus"></i> Upload New Plugin
    </button>
  </div>
  
  <!-- Filters -->
  <div class="filters">
    <input type="text" placeholder="Search plugins..." id="searchInput">
    <select id="statusFilter">
      <option value="">All Status</option>
      <option value="enabled">Enabled</option>
      <option value="disabled">Disabled</option>
    </select>
  </div>
  
  <!-- Plugins Table -->
  <table class="plugins-table">
    <thead>
      <tr>
        <th>Name</th>
        <th>Version</th>
        <th>Status</th>
        <th>Uploaded</th>
        <th>Actions</th>
      </tr>
    </thead>
    <tbody id="pluginsTableBody">
      <!-- Populated via JavaScript -->
    </tbody>
  </table>
</div>

<!-- Config Edit Modal -->
<div id="configModal" class="modal">
  <div class="modal-content">
    <h3>Edit Plugin Configuration</h3>
    <textarea id="configEditor" rows="20"></textarea>
    <button onclick="saveConfig()">Save</button>
    <button onclick="closeModal()">Cancel</button>
  </div>
</div>
```

### Task 8: Integration & Testing ⏳
**Goal:** Ensure seamless integration with existing systems

**Integration Points:**

1. **With Goal #5 Middleware Chain:**
   ```go
   // When plugin is enabled, add to chain
   plugin := loadPlugin(pluginPath)
   middlewareChain.RegisterMiddleware(plugin.Name, plugin.Handler)
   ```

2. **With Admin Panel:**
   - Add plugin management menu item
   - Integrate with existing auth system
   - Use consistent styling

3. **With MongoDB:**
   - Use existing connection pool
   - Follow naming conventions
   - Add proper indexes

**Testing Requirements:**

1. **Unit Tests:**
   ```go
   func TestPluginUpload(t *testing.T) {
       // Test valid plugin upload
       // Test invalid file type
       // Test file too large
       // Test duplicate plugin
   }
   
   func TestPluginValidation(t *testing.T) {
       // Test Go version mismatch
       // Test missing symbols
       // Test malformed .so file
   }
   
   func TestPluginEnableDisable(t *testing.T) {
       // Test enabling plugin
       // Test disabling plugin
       // Test enabling already enabled plugin
   }
   ```

2. **Integration Tests:**
   - Upload plugin via API
   - Enable plugin and verify middleware chain
   - Send request through plugin
   - Verify plugin metrics
   - Disable plugin and verify removal

3. **UI Tests:**
   - Test drag-and-drop upload
   - Test form validation
   - Test enable/disable toggles
   - Test delete confirmation

### Task 9: Security Hardening ⏳
**Goal:** Implement comprehensive security measures

**Security Measures:**

1. **File Upload Security:**
   - Max file size: 50MB
   - Allowed extensions: `.so` only
   - Virus scanning (ClamAV integration optional)
   - Content-Type validation
   - File signature verification

2. **Access Control:**
   - Only admin users can upload plugins
   - JWT authentication required
   - Rate limiting on upload endpoint (5 uploads/hour)
   - Audit logging for all plugin operations

3. **Plugin Sandbox (Future):**
   - Run plugins in isolated environment
   - Resource limits (CPU, memory)
   - Network access restrictions
   - File system access restrictions

4. **Validation:**
   - Go version must match exactly
   - No suspicious imports
   - Symbol validation
   - Binary signature check

**Security Audit Log:**
```go
type PluginAuditLog struct {
    Timestamp  time.Time
    UserID     string
    Action     string // upload, enable, disable, delete
    PluginID   string
    PluginName string
    Success    bool
    ErrorMsg   string
    IPAddress  string
}
```

### Task 10: Documentation ⏳
**Goal:** Create comprehensive documentation

**Documents to Create:**

1. **Developer Guide** (`docs/plugin-upload-guide.md`):
   - How to create compatible plugins
   - Plugin interface requirements
   - Building plugins with correct Go version
   - Testing plugins locally
   - Best practices

2. **User Guide** (`docs/plugin-management-user-guide.md`):
   - How to upload plugins via admin panel
   - How to configure plugins
   - How to enable/disable plugins
   - Troubleshooting common issues

3. **API Documentation** (`docs/plugin-api.md`):
   - All API endpoints
   - Request/response formats
   - Error codes
   - Examples using curl

4. **Security Guide** (`docs/plugin-security.md`):
   - Security considerations
   - Validation requirements
   - Best practices for plugin authors
   - Audit logging

5. **Goal #7 Summary** (`docs/GOAL-7-SUMMARY.md`):
   - Implementation details
   - Metrics and achievements
   - Lessons learned
   - Future improvements

## 🔍 Success Criteria

- [ ] Users can upload .so files via admin panel
- [ ] Plugin validation prevents invalid uploads
- [ ] Plugins are stored securely in MongoDB GridFS
- [ ] Plugins can be enabled/disabled from UI
- [ ] Enabled plugins integrate with middleware chain (Goal #5)
- [ ] Plugin configuration can be updated via UI
- [ ] All operations are logged for audit
- [ ] Security measures prevent malicious uploads
- [ ] Comprehensive tests cover all functionality
- [ ] Documentation is complete and clear

## 📊 Estimated Timeline

| Task | Estimated Time |
|------|----------------|
| Task 1: Planning | 1 hour (in progress) |
| Task 2: Backend Upload API | 2 hours |
| Task 3: Backend Management API | 1.5 hours |
| Task 4: Validation System | 1.5 hours |
| Task 5: MongoDB Schema | 1 hour |
| Task 6: Frontend Upload UI | 2 hours |
| Task 7: Frontend Management UI | 2 hours |
| Task 8: Integration & Testing | 2 hours |
| Task 9: Security Hardening | 1.5 hours |
| Task 10: Documentation | 1.5 hours |
| **Total** | **~16 hours** |

## 🎯 Key Differentiators from Goal #5

**Goal #5 (Completed):**
- Middleware chain system
- Manual plugin registration via config
- Testing and rollback
- Metrics collection

**Goal #7 (New):**
- **Web-based upload** (drag & drop)
- **Binary file storage** (GridFS)
- **Validation system** (Go version, symbols)
- **UI for management** (enable/disable, configure)
- **Security scanning**
- **Version management**

## 🚀 Next Steps

1. Review and approve this plan
2. Start Task 2: Backend Upload API implementation
3. Implement incrementally with testing
4. Regular commits after each task completion

---

**Status:** 🟡 PLANNING COMPLETE - Ready for Implementation  
**Priority:** HIGH  
**Dependencies:** Goal #5 (Middleware Chain) ✅  
**Estimated Completion:** 16 hours
