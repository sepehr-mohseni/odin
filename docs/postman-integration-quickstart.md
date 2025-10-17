# Postman Integration Quick Start Guide

Get up and running with Postman integration in 5 minutes!

## Prerequisites

- ✅ Odin API Gateway installed and running
- ✅ MongoDB configured and accessible
- ✅ Postman account with API access
- ⚠️  Newman CLI (optional, for test execution)

## Step 1: Get Postman API Key (2 minutes)

1. Log in to [Postman](https://www.postman.com/)
2. Click your profile icon → Settings
3. Navigate to "API Keys" section
4. Click "Generate API Key"
5. Name it (e.g., "Odin Integration")
6. **Copy the key immediately** (it won't be shown again!)

## Step 2: Access Integration Panel (30 seconds)

1. Start Odin gateway:
   ```bash
   ./odin
   ```

2. Open your browser to:
   ```
   http://localhost:8080/admin/integrations/postman
   ```

3. You should see the Postman Integration dashboard

## Step 3: Configure Integration (1 minute)

In the "Integration Configuration" section:

1. **API Key**: Paste your Postman API key
2. Click "Test Connection" button
3. ✅ You should see: "Connected successfully"
4. **Select Workspace**: Choose from dropdown
5. **Auto-sync**: Enable if you want automatic updates
6. **Sync Interval**: 300 seconds (5 minutes) recommended
7. Click "Save Configuration"

## Step 4: Import Your First Collection (1 minute)

1. Click "Load Collections" button
2. Your collections will appear in the table
3. Find a collection to import
4. Click "Import as Service"
5. Enter a service name (e.g., "my-api")
6. Click "Import"
7. ✅ Collection imported!

## Step 5: Verify Service Created (30 seconds)

Your collection is now an Odin service!

Check it:
```bash
curl http://localhost:8080/admin/services
```

You should see your newly imported service in the list.

## Testing Your Imported Service

### Example: Imported Collection "Users API"

If your collection had this request:
```
GET /users/:id
Host: api.example.com
```

It's now available through Odin:
```bash
curl http://localhost:8080/api/users/123
```

Odin routes it to your backend automatically!

## Next Steps

### Enable Auto-Sync

Auto-sync keeps your services updated automatically:

1. In config section, check "Enable Auto-sync"
2. Set sync interval (300 seconds = 5 minutes)
3. Click "Save Configuration"
4. Monitor "Sync History" section for updates

### Run Tests with Newman

If you have test scripts in Postman:

1. Install Newman (if not already):
   ```bash
   npm install -g newman
   ```

2. In Integration dashboard:
   - Find your collection
   - Click "Run Tests"
   - View results in "Test Results" section

### Manual Sync

Update a specific collection:
```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/sync/COLLECTION_ID
```

Or sync all:
```bash
curl -X POST http://localhost:8080/admin/api/integrations/postman/sync
```

## Common First-Time Issues

### "Connection Failed"

❌ **Problem**: Test connection returns 401 Unauthorized

✅ **Solution**: 
- Verify API key is correct (no extra spaces)
- Check key hasn't expired
- Ensure you copied the full key

### "No Collections Found"

❌ **Problem**: Load Collections returns empty list

✅ **Solution**:
- Verify correct workspace is selected
- Check collections exist in that workspace
- Ensure API key has access to workspace

### "Import Failed"

❌ **Problem**: Import button returns error

✅ **Solution**:
- Check service name is unique
- Verify collection has valid requests
- Review gateway logs for details:
  ```bash
  tail -f /var/log/odin/gateway.log | grep -i postman
  ```

### "Newman Tests Won't Run"

❌ **Problem**: Run Tests returns error

✅ **Solution**:
- Install Newman: `npm install -g newman`
- Verify installation: `newman --version`
- Check collection has test scripts
- Enable Newman in config

## Configuration Options Explained

### Sync Interval

How often to check for collection updates:

- **60-300 seconds**: Rapid development, frequent changes
- **300-900 seconds**: Normal operations (recommended)
- **900-3600 seconds**: Stable production, rare changes

### Auto-sync Behavior

When enabled:
- ✅ Automatically fetches collection updates
- ✅ Compares with stored version
- ✅ Updates Odin service if changes detected
- ✅ Records sync history

When disabled:
- Manual sync only via "Sync All Collections" button
- Or via API endpoint

## Quick Reference: Key Endpoints

```bash
# Test connection
curl -X POST http://localhost:8080/admin/api/integrations/postman/connect

# List collections
curl http://localhost:8080/admin/api/integrations/postman/collections

# Import collection
curl -X POST http://localhost:8080/admin/api/integrations/postman/collections/COL_ID/import \
  -H "Content-Type: application/json" \
  -d '{"service_name": "my-service"}'

# Sync all
curl -X POST http://localhost:8080/admin/api/integrations/postman/sync

# Run tests
curl -X POST http://localhost:8080/admin/api/integrations/postman/test/COL_ID

# Get status
curl http://localhost:8080/admin/api/integrations/postman/status
```

## Video Tutorial (Coming Soon)

We're working on a video walkthrough! Check our docs for updates.

## Need Help?

- 📖 **Full Documentation**: `docs/postman-integration.md`
- 🧪 **Testing Guide**: `docs/postman-integration-testing.md`
- 🐛 **Issues**: Check gateway logs first
- 💬 **Support**: Open an issue on GitHub

## Success Checklist

After completing this guide, you should have:

- [x] Postman API key configured
- [x] Connection tested successfully
- [x] At least one collection imported
- [x] Service accessible through Odin
- [x] Sync history showing successful operations
- [x] Understanding of auto-sync behavior

## What's Next?

Explore advanced features:

1. **Multiple Collections**: Import your entire API portfolio
2. **Environments**: Use Postman environments for config
3. **Testing**: Automate API testing with Newman
4. **Monitoring**: Track sync history and test results
5. **Export**: Share Odin services back to Postman

---

**Congratulations! 🎉** You've successfully integrated Postman with Odin API Gateway!

Your Postman collections are now powerful Odin services with:
- ⚡ High-performance routing
- 🔒 Built-in authentication
- 📊 Metrics and monitoring
- 🔄 Automatic synchronization
- 🧪 Integrated testing

Happy gateway-ing! 🚀
