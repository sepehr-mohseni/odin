# WebAssembly (WASM) Extensions

The Odin API Gateway supports WebAssembly (WASM) plugins for dynamic, high-performance extensibility without recompiling the gateway. WASM plugins can implement custom logic for authentication, request/response transformation, rate limiting, and more.

## Overview

### Why WASM Extensions?

- **Dynamic Loading**: Load and unload plugins without restarting the gateway
- **Safety**: Sandboxed execution environment prevents plugins from accessing system resources
- **Performance**: Near-native performance with minimal overhead
- **Language Agnostic**: Write plugins in Go, Rust, C++, AssemblyScript, or any language that compiles to WASM
- **Portability**: WASM modules run on any platform
- **Isolation**: Each plugin runs in its own memory space

### Supported Plugin Types

| Type | Purpose | Use Cases |
|------|---------|-----------|
| `request` | Request transformation | Header injection, body modification, validation |
| `response` | Response transformation | Format conversion, data enrichment |
| `auth` | Authentication/Authorization | Custom auth schemes, token validation |
| `ratelimit` | Rate limiting | Advanced rate limiting algorithms |
| `middleware` | General middleware | Logging, monitoring, security headers |
| `aggregation` | Response aggregation | Combine multiple responses |

## Configuration

### Basic Configuration

```yaml
wasm:
  enabled: true
  pluginDir: /etc/odin/plugins      # Directory containing WASM files
  maxMemoryPages: 100                # Max memory per plugin (100 * 64KB = 6.4MB)
  maxInstances: 10                   # Max concurrent plugin instances
  cacheEnabled: true                 # Cache compiled modules
  
  plugins:
    - name: my-plugin
      path: plugin.wasm               # Relative to pluginDir or absolute
      type: request
      enabled: true
      priority: 10                    # Lower runs first
      timeout: 5s
      allowedUrls:                    # URL patterns (regex)
        - /api/.*
      services:                       # Service names
        - users-service
      config:                         # Plugin-specific configuration
        key: value
```

### Complete Example

```yaml
wasm:
  enabled: true
  pluginDir: /opt/odin/wasm-plugins
  maxMemoryPages: 200
  maxInstances: 50
  cacheEnabled: true
  
  plugins:
    # Authentication
    - name: jwt-auth
      path: auth/jwt.wasm
      type: auth
      enabled: true
      priority: 1
      timeout: 2s
      allowedUrls:
        - /api/.*
      config:
        publicKey: /etc/odin/keys/public.pem
        
    # Rate limiting
    - name: rate-limiter
      path: ratelimit.wasm
      type: ratelimit
      enabled: true
      priority: 2
      timeout: 1s
      config:
        limit: 100
        window: 60s
        
    # Request transformation
    - name: header-injector
      path: transform/headers.wasm
      type: request
      enabled: true
      priority: 5
      timeout: 1s
      services:
        - users-service
      config:
        addHeaders:
          X-Gateway: Odin
          X-Version: v1
```

## Writing WASM Plugins

### Plugin Interface

Every WASM plugin must export these functions:

```
init()         - Called when plugin is loaded (optional)
execute()      - Called for each request (required)
cleanup()      - Called when plugin is unloaded (optional)
```

### Input/Output Format

**Input** (JSON):
```json
{
  "context": {
    "requestId": "abc-123",
    "serviceId": "users-service",
    "routePath": "/api/users",
    "timestamp": "2024-01-01T00:00:00Z",
    "config": { "key": "value" },
    "metadata": { "key": "value" }
  },
  "input": {
    "method": "GET",
    "url": "http://example.com/api/users",
    "headers": { "Content-Type": ["application/json"] },
    "body": "...",
    "query": { "limit": ["10"] }
  },
  "config": { "pluginConfig": "value" }
}
```

**Output** (JSON):
```json
{
  "modified": true,
  "continue": true,
  "response": {
    "statusCode": 200,
    "headers": { "X-Custom": ["value"] },
    "body": "..."
  },
  "error": "",
  "metadata": { "key": "value" }
}
```

### Example: Go Plugin

```go
package main

import (
    "encoding/json"
    "unsafe"
)

type Input struct {
    Context struct {
        RequestID string `json:"requestId"`
        Config    map[string]interface{} `json:"config"`
    } `json:"context"`
    Input map[string]interface{} `json:"input"`
}

type Result struct {
    Modified bool `json:"modified"`
    Continue bool `json:"continue"`
    Response interface{} `json:"response,omitempty"`
    Error    string `json:"error,omitempty"`
}

//export execute
func execute(inputPtr uint32, inputSize uint32) uint64 {
    // Read input
    inputData := readMemory(inputPtr, inputSize)
    
    var input Input
    json.Unmarshal(inputData, &input)
    
    // Process request
    // ... your logic here ...
    
    // Return result
    result := Result{
        Modified: true,
        Continue: true,
    }
    
    return returnResult(&result)
}

func main() {}
```

**Compile**:
```bash
tinygo build -o plugin.wasm -target=wasi plugin.go
```

### Example: Rust Plugin

```rust
use serde::{Deserialize, Serialize};

#[derive(Deserialize)]
struct Input {
    context: Context,
    input: serde_json::Value,
}

#[derive(Serialize)]
struct Result {
    modified: bool,
    continue_: bool,
    response: Option<serde_json::Value>,
    error: String,
}

#[no_mangle]
pub extern "C" fn execute(input_ptr: *const u8, input_len: usize) -> u64 {
    // Read input
    let input_data = unsafe { 
        std::slice::from_raw_parts(input_ptr, input_len)
    };
    
    let input: Input = serde_json::from_slice(input_data).unwrap();
    
    // Process request
    // ... your logic here ...
    
    // Return result
    let result = Result {
        modified: true,
        continue_: true,
        response: None,
        error: String::new(),
    };
    
    return_result(&result)
}
```

**Compile**:
```bash
cargo build --target wasm32-wasi --release
```

### Example: AssemblyScript Plugin

```typescript
import { JSON } from "assemblyscript-json";

@external("env", "log")
declare function hostLog(ptr: usize, len: i32): void;

export function execute(inputPtr: usize, inputSize: i32): u64 {
  // Read input
  const inputData = String.UTF8.decode(
    memory.data(inputPtr, inputSize)
  );
  
  const input = JSON.parse(inputData);
  
  // Process request
  // ... your logic here ...
  
  // Return result
  const result = {
    modified: true,
    continue: true
  };
  
  return returnResult(result);
}
```

**Compile**:
```bash
asc plugin.ts -o plugin.wasm --target wasi
```

## Host Functions

The gateway provides these functions to WASM plugins:

### log(message)
```c
void log(char* message, int length)
```
Write a log message to the gateway log.

### get_time()
```c
int64 get_time()
```
Get current Unix timestamp.

### get_config(key)
```c
char* get_config(char* key)
```
Retrieve configuration value.

### http_call(method, url, headers, body)
```c
response* http_call(char* method, char* url, headers* h, char* body)
```
Make an HTTP call to external services.

## Use Cases

### 1. Custom Authentication

```yaml
plugins:
  - name: api-key-auth
    path: auth-apikey.wasm
    type: auth
    priority: 1
    config:
      headerName: X-API-Key
      validKeys:
        - key1
        - key2
```

### 2. Request Validation

```yaml
plugins:
  - name: schema-validator
    path: validator.wasm
    type: request
    priority: 3
    services:
      - users-service
    config:
      schemaPath: /etc/odin/schemas/user.json
      strict: true
```

### 3. Response Caching

```yaml
plugins:
  - name: cache-plugin
    path: cache.wasm
    type: response
    priority: 10
    allowedUrls:
      - /api/products.*
    config:
      ttl: 300
      backend: redis
```

### 4. Data Enrichment

```yaml
plugins:
  - name: enrich-response
    path: enrichment.wasm
    type: response
    priority: 8
    config:
      addUserDetails: true
      userServiceUrl: http://users-service:8080
```

### 5. Advanced Rate Limiting

```yaml
plugins:
  - name: intelligent-ratelimit
    path: ratelimit-advanced.wasm
    type: ratelimit
    priority: 2
    config:
      algorithm: sliding-window
      redis:
        host: redis:6379
      limits:
        anonymous: 10
        authenticated: 100
        premium: 1000
```

## Performance Considerations

### Memory Management

- Each plugin has its own isolated memory
- Configure `maxMemoryPages` based on plugin needs
- Default: 100 pages = 6.4MB per plugin
- Monitor memory usage in production

### Execution Timeout

```yaml
plugins:
  - name: my-plugin
    timeout: 5s  # Kill plugin if execution exceeds 5 seconds
```

### Instance Pooling

```yaml
wasm:
  maxInstances: 50  # Max concurrent plugin executions
```

### Compilation Caching

```yaml
wasm:
  cacheEnabled: true  # Cache compiled modules for faster startup
```

## Deployment

### Docker

```dockerfile
FROM your-gateway-image

# Copy WASM plugins
COPY plugins/*.wasm /etc/odin/plugins/

# Copy configuration
COPY config/wasm.yaml /etc/odin/config.yaml
```

### Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: wasm-plugins
binaryData:
  auth.wasm: <base64-encoded-wasm>
  transform.wasm: <base64-encoded-wasm>
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: odin
        volumeMounts:
        - name: plugins
          mountPath: /etc/odin/plugins
      volumes:
      - name: plugins
        configMap:
          name: wasm-plugins
```

### Hot Reload

Plugins can be dynamically loaded/unloaded:

```bash
# Via admin API
curl -X POST http://admin:8081/api/plugins/load \
  -H "Content-Type: application/json" \
  -d '{"name": "new-plugin", "path": "/plugins/new.wasm"}'

curl -X DELETE http://admin:8081/api/plugins/unload/old-plugin
```

## Security

### Sandboxing

- WASM plugins run in a sandboxed environment
- No access to filesystem (except via host functions)
- No network access (except via host functions)
- No system calls

### Resource Limits

```yaml
wasm:
  maxMemoryPages: 100      # Limit memory usage
  maxInstances: 50         # Limit concurrent executions
  timeout: 5s              # Per-execution timeout
```

### Plugin Verification

```bash
# Verify WASM module before loading
wasm-validate plugin.wasm

# Check for malicious code
wasm-security-scanner plugin.wasm
```

## Monitoring

### Plugin Metrics

The gateway exposes these metrics:

- `wasm_plugin_executions_total` - Total plugin executions
- `wasm_plugin_execution_duration_seconds` - Execution duration
- `wasm_plugin_errors_total` - Plugin errors
- `wasm_plugin_memory_bytes` - Memory usage
- `wasm_plugin_instances_active` - Active instances

### Prometheus Example

```promql
# Average plugin execution time
rate(wasm_plugin_execution_duration_seconds_sum[5m]) 
  / rate(wasm_plugin_executions_total[5m])

# Plugin error rate
rate(wasm_plugin_errors_total[5m]) 
  / rate(wasm_plugin_executions_total[5m])
```

### Logging

```yaml
logging:
  level: debug  # Enable plugin debug logs
```

## Troubleshooting

### Common Issues

#### 1. Plugin Fails to Load

**Error**: `failed to compile plugin`

**Solution**:
```bash
# Verify WASM file is valid
file plugin.wasm
wasm-objdump -h plugin.wasm

# Check target architecture
wasm-objdump -x plugin.wasm | grep target
```

#### 2. Execution Timeout

**Error**: `plugin execution timeout`

**Solution**:
- Increase timeout in config
- Optimize plugin logic
- Use async operations

#### 3. Memory Exceeded

**Error**: `failed to allocate memory`

**Solution**:
```yaml
wasm:
  maxMemoryPages: 200  # Increase memory limit
```

#### 4. Plugin Not Applied

**Solution**:
- Check `allowedUrls` patterns
- Verify `services` list
- Check `enabled: true`
- Verify plugin priority

### Debug Mode

```yaml
wasm:
  enabled: true
  plugins:
    - name: debug-plugin
      path: plugin.wasm
      config:
        debug: true
        logLevel: trace
```

## Best Practices

### 1. Keep Plugins Small

- Focus on single responsibility
- Minimize dependencies
- Optimize for fast execution

### 2. Error Handling

```go
result := Result{
    Modified: false,
    Continue: true,  // Continue even on error
    Error: "descriptive error message",
}
```

### 3. Configuration

```yaml
plugins:
  - name: my-plugin
    config:
      # Use environment variables
      apiKey: ${API_KEY}
      # Use reasonable defaults
      timeout: 5s
      retries: 3
```

### 4. Testing

```bash
# Test plugin locally
odin-cli test-plugin plugin.wasm input.json

# Unit test plugin code
go test ./plugin/...
cargo test
```

### 5. Versioning

```yaml
plugins:
  - name: my-plugin-v1
    path: my-plugin-v1.0.0.wasm
```

## Examples

See `examples/wasm-plugins/` for complete examples:

- `header-injection/` - Add custom headers
- `jwt-auth/` - JWT authentication
- `rate-limiter/` - Advanced rate limiting
- `transformer/` - Request/response transformation
- `cache/` - Response caching
- `validator/` - Schema validation

## See Also

- [Middleware Documentation](./middleware.md)
- [Authentication](./auth.md)
- [Performance Tuning](./performance.md)
- [Plugin Development Guide](./plugin-development.md)
