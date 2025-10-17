# Plugin Binary Upload & Management - User Guide

**Quick Start Guide for Administrators**

---

## 📖 Table of Contents

1. [Overview](#overview)
2. [Uploading a Plugin](#uploading-a-plugin)
3. [Managing Plugins](#managing-plugins)
4. [Configuration](#configuration)
5. [Troubleshooting](#troubleshooting)
6. [Best Practices](#best-practices)

---

## Overview

The Plugin Binary Upload system allows you to upload compiled Go plugins (.so files) and manage them through the admin interface without restarting the server.

### What You Can Do

✅ Upload plugin binaries  
✅ Enable/disable plugins dynamically  
✅ Edit plugin configuration  
✅ View statistics and metadata  
✅ Delete unused plugins  

### Requirements

- Admin access to Odin Gateway
- Plugin compiled with **Go 1.25** (exact version)
- Plugin file must be `.so` format
- Maximum file size: 50 MB

---

## Uploading a Plugin

### Step 1: Access Upload Page

1. Log in to the admin panel
2. Navigate to **Plugin Binaries** in the top menu
3. Click **"+ Upload New Plugin"** button

### Step 2: Select Plugin File

**Option A: Drag and Drop**
- Drag your `.so` file directly into the upload area
- The file will be automatically validated

**Option B: Browse**
- Click **"Browse Files"** button
- Select your `.so` file from the file picker

### Step 3: Fill in Metadata

#### Required Fields

**Name**  
Unique identifier for the plugin (auto-filled from filename)
```
Example: rate-limiter
```

**Version**  
Semantic version number
```
Example: 1.0.0
Recommended format: MAJOR.MINOR.PATCH
```

#### Optional Fields

**Description**  
What does this plugin do?
```
Example: Implements token bucket rate limiting with configurable limits
```

**Author**  
Who created this plugin?
```
Example: Your Name or Team Odin
```

**Configuration (JSON)**  
Plugin-specific configuration
```json
{
  "max_requests": 100,
  "window": "1m",
  "burst": 10
}
```

**Routes**  
Which routes should this plugin apply to?
```
Examples:
  /*              All routes
  /api/*          API routes only
  /users/*        User routes only
  /api/*,/v2/*    Multiple route patterns
```

**Priority**  
Execution order (0-1000, lower runs first)
```
Default: 100
Example: 50 (runs before priority 100)
```

**Phase**  
When to execute the plugin
```
Options:
  - pre-routing    Before routing decision
  - post-routing   After routing decision
  - pre-response   Before sending response
```

### Step 4: Upload

1. Click **"Upload Plugin"** button
2. Wait for upload progress bar
3. Success message will appear
4. You'll be redirected to the management page

### Upload Success Indicators

✅ Green success message  
✅ Redirect to plugin list  
✅ New plugin appears in the list  

### Common Upload Errors

❌ **"Please select a .so file"**  
→ Wrong file type selected

❌ **"File size exceeds 50 MB"**  
→ Plugin file too large

❌ **"Plugin with this name and version already exists"**  
→ Duplicate plugin, change version or delete existing

❌ **"Invalid Go version"**  
→ Plugin compiled with wrong Go version (need Go 1.25)

---

## Managing Plugins

### Viewing Plugin List

Navigate to **Plugin Binaries** to see all uploaded plugins.

**Statistics Cards** (top of page):
- **Total Plugins**: All uploaded plugins
- **Enabled**: Currently active plugins
- **Disabled**: Inactive plugins
- **Total Size**: Combined size of all plugins

**Plugin Table** shows:
- Name and version
- Status badge (enabled/disabled)
- Author
- File size
- Upload date
- Enable/disable toggle
- Action buttons

### Searching and Filtering

**Search Box**  
Type to search by name or description
```
Example: "rate" will find "rate-limiter"
```

**Status Filter**  
Dropdown to filter by status
```
Options:
  - All Status
  - Enabled
  - Disabled
```

### Enabling a Plugin

**What Happens**:
1. Plugin binary is loaded from storage
2. Validated for compatibility
3. Registered in middleware chain
4. Starts processing requests immediately

**How To**:
1. Find plugin in the list
2. Toggle the switch in "Enabled" column to ON (green)
3. Wait for confirmation message
4. Plugin is now active

**Requirements**:
- Plugin must be uploaded first
- No compilation errors
- Correct Go version
- Valid configuration

### Disabling a Plugin

**What Happens**:
1. Plugin is unregistered from middleware chain
2. Stops processing new requests
3. Can be re-enabled anytime

**How To**:
1. Find plugin in the list
2. Toggle the switch in "Enabled" column to OFF (gray)
3. Wait for confirmation message
4. Plugin is now inactive

### Viewing Plugin Details

Click the **👁️ View** button to see full metadata:

- Plugin ID
- Name and version
- Description
- Author
- Go version used
- Platform (OS/Architecture)
- SHA256 hash (integrity check)
- File size
- Upload timestamp
- Configuration (formatted JSON)

### Editing Configuration

**How To**:
1. Click the **⚙️ Config** button
2. JSON editor opens with current configuration
3. Make changes to the JSON
4. Click **"Save Configuration"**

**Example**:
```json
{
  "max_requests": 200,
  "window": "1m",
  "burst": 20,
  "whitelist": ["192.168.1.0/24"]
}
```

**Validation**:
- JSON must be valid
- Configuration is validated before saving
- Invalid JSON shows error message

**Effect**:
- If plugin is enabled, it will reload with new config
- If plugin is disabled, config is saved for next enable

### Deleting a Plugin

**Requirements**:
- Plugin must be disabled first
- Cannot delete enabled plugins

**How To**:
1. Ensure plugin is disabled
2. Click the **🗑️ Delete** button
3. Confirm deletion in modal
4. Plugin is permanently removed

**Warning**: This action cannot be undone!

**What Gets Deleted**:
- Plugin binary file
- Metadata
- Configuration
- Upload history

---

## Configuration

### Plugin Configuration Format

All plugin configurations use JSON format:

```json
{
  "key": "value",
  "nested": {
    "key": "value"
  },
  "array": [1, 2, 3]
}
```

### Common Configuration Patterns

#### Rate Limiting
```json
{
  "max_requests": 100,
  "window": "1m",
  "burst": 10,
  "key": "ip"
}
```

#### Authentication
```json
{
  "secret_key": "your-secret-key",
  "algorithm": "HS256",
  "expiry": "24h"
}
```

#### Caching
```json
{
  "ttl": "5m",
  "max_size": "100MB",
  "key_prefix": "cache:"
}
```

#### Logging
```json
{
  "level": "info",
  "format": "json",
  "output": "/var/log/plugin.log"
}
```

### Environment Variables

Configuration can reference environment variables:

```json
{
  "api_key": "${API_KEY}",
  "endpoint": "${SERVICE_URL}"
}
```

---

## Troubleshooting

### Upload Issues

**Problem**: "Failed to upload plugin"  
**Solutions**:
- Check file size (< 50 MB)
- Verify file extension (.so)
- Ensure file is not corrupted
- Check network connection

**Problem**: "Invalid Go version"  
**Solutions**:
- Recompile plugin with Go 1.25
- Check Go version: `go version`
- Use exact version: `go1.25.x`

**Problem**: "Missing required symbol: New"  
**Solutions**:
- Plugin must export `New` function
- Check function signature matches interface
- Recompile plugin with correct exports

### Enable/Disable Issues

**Problem**: "Failed to enable plugin"  
**Solutions**:
- Check plugin is uploaded correctly
- Verify Go version compatibility
- Review plugin logs for errors
- Check plugin configuration is valid

**Problem**: "Plugin not processing requests"  
**Solutions**:
- Verify plugin is enabled (green toggle)
- Check routes configuration matches your URLs
- Verify priority order (lower runs first)
- Check phase is correct for your use case

### Configuration Issues

**Problem**: "Invalid JSON"  
**Solutions**:
- Use JSON validator (jsonlint.com)
- Check for missing commas, quotes, brackets
- Remove trailing commas
- Use proper escaping for special characters

**Problem**: "Configuration not taking effect"  
**Solutions**:
- Save configuration after editing
- Disable and re-enable plugin
- Check plugin logs for config errors
- Verify configuration keys match plugin expectations

### Performance Issues

**Problem**: "Requests are slow after enabling plugin"  
**Solutions**:
- Check plugin has no blocking operations
- Review plugin priority (may be running too early)
- Disable plugin and test performance
- Contact plugin developer

---

## Best Practices

### Naming Conventions

**Plugin Names**:
- Use lowercase with hyphens
- Be descriptive and specific
- Examples: `rate-limiter`, `jwt-auth`, `response-cache`

**Versions**:
- Use semantic versioning (MAJOR.MINOR.PATCH)
- Increment MAJOR for breaking changes
- Increment MINOR for new features
- Increment PATCH for bug fixes

### Testing

**Before Production**:
1. Upload plugin to staging environment
2. Enable and test thoroughly
3. Monitor performance and errors
4. Review logs for issues
5. Load test if handling high traffic

**After Upload**:
1. Enable on low-traffic routes first
2. Monitor for errors
3. Gradually expand to more routes
4. Keep previous version as backup

### Configuration Management

**Security**:
- Never commit secrets in configuration
- Use environment variables for sensitive data
- Rotate credentials regularly
- Limit access to admin panel

**Organization**:
- Document configuration schema
- Use consistent naming conventions
- Version control configuration files
- Maintain configuration backups

### Monitoring

**What to Monitor**:
- Plugin enable/disable events
- Configuration changes
- Error rates
- Performance metrics
- Request latency

**Tools**:
- Admin dashboard statistics
- Application logs
- Monitoring integration (Prometheus, etc.)
- Error tracking (Sentry, etc.)

### Maintenance

**Regular Tasks**:
- Review enabled plugins monthly
- Update plugins when new versions available
- Remove unused plugins
- Monitor plugin performance
- Review configuration for optimization

**Update Process**:
1. Upload new version
2. Test thoroughly
3. Disable old version
4. Enable new version
5. Monitor for issues
6. Delete old version if successful

---

## Quick Reference

### File Requirements
- Format: `.so` (shared object)
- Max size: 50 MB
- Go version: 1.25
- Must export: `New` function

### Priority Guidelines
- 0-50: Critical early processing (auth, security)
- 51-100: Standard middleware (rate limiting, logging)
- 101-200: Enhancement (caching, transformation)
- 201-1000: Post-processing (response modification)

### Phases
- **pre-routing**: Before route matching (security, rate limiting)
- **post-routing**: After route matched (logging, metrics)
- **pre-response**: Before response sent (caching, transformation)

### Route Patterns
- `/*`: All routes
- `/api/*`: API routes only
- `/v1/*`: Version-specific
- `/api/users/*`: Specific path
- `/*,!/health`: All except health checks (with exclusion)

---

## Support

Need help?

- **Documentation**: See `/docs` for detailed guides
- **API Reference**: `docs/GOAL-7-SUMMARY.md`
- **Examples**: Check `/examples/plugins`
- **Issues**: Report bugs on GitHub
- **Contact**: Reach out to your admin team

---

**Quick Tips**:

💡 Always test plugins in staging first  
💡 Keep plugin files backed up  
💡 Document your configuration  
💡 Monitor performance after enabling  
💡 Disable problematic plugins immediately  

---

*Last Updated: 2025-01-XX*  
*Odin API Gateway - Plugin Management*
