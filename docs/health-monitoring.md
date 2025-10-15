# Service Health Monitoring

The Odin API Gateway includes a comprehensive health monitoring system that actively checks the health of backend service targets and sends alerts when issues are detected.

## Features

### Active Health Checking
- Periodic health checks to backend `/health` endpoints
- Configurable check intervals, timeouts, and thresholds
- Support for multiple expected HTTP status codes
- Automatic retry logic with consecutive failure tracking

### Target Status Management
- **Healthy**: Target is responding correctly
- **Unhealthy**: Target has failed the configured number of consecutive checks
- **Degraded**: (Future feature) Partial functionality

### Alert System
- **Multi-channel alerts**: Send alerts to multiple destinations
  - Log channel: Logs alerts via standard logging
  - Webhook channel: POST alerts to external systems (Slack, PagerDuty, etc.)
- **Alert types**:
  - `target_down`: Critical alert when a target becomes unhealthy
  - `target_recovered`: Info alert when a target recovers
  - `high_error_rate`: (Future) Warning for elevated error rates
  - `slow_response`: (Future) Warning for degraded performance
- **Alert throttling**: Prevents alert spam (5-minute minimum between duplicates)

### Performance Tracking
- Response time measurement for each health check
- Average response time calculation
- Success/failure rate tracking
- Total checks counter

## Configuration

### Global Configuration

Configure health monitoring in your `config.yaml`:

```yaml
monitoring:
  enabled: true
  path: /metrics
  webhookUrl: "https://your-webhook-endpoint.com/alerts"  # Optional
```

### Service-Specific Configuration

Add health check configuration to each service in `services.yaml`:

```yaml
services:
  - name: users-service
    basePath: /api/users
    targets:
      - http://localhost:3001
      - http://localhost:3002
    
    healthCheck:
      enabled: true                 # Enable health monitoring
      interval: 30s                 # Check every 30 seconds
      timeout: 5s                   # Timeout for each check
      unhealthyThreshold: 3         # Failures before marking unhealthy
      healthyThreshold: 2           # Successes before marking healthy
      expectedStatus: [200, 204]    # Expected HTTP status codes
      insecureSkipVerify: false     # Skip TLS verification (dev only)
```

### Default Values

If not specified, the following defaults are used:

| Setting | Default |
|---------|---------|
| interval | 30s |
| timeout | 5s |
| unhealthyThreshold | 3 |
| healthyThreshold | 2 |
| expectedStatus | [200, 204] |
| insecureSkipVerify | false |

## How It Works

### Health Check Process

1. **Initialization**: When the gateway starts, health checkers are created for services with `healthCheck.enabled: true`

2. **Target Registration**: Each backend target is registered with the health checker

3. **Periodic Checks**: 
   - Health checker sends GET requests to `<target>/health`
   - Measures response time
   - Checks if status code matches `expectedStatus`

4. **Status Evaluation**:
   - Consecutive failures increment fail counter
   - Consecutive successes increment pass counter
   - Target marked **unhealthy** after reaching `unhealthyThreshold` failures
   - Target marked **healthy** after reaching `healthyThreshold` successes

5. **Alert Generation**:
   - Status changes trigger alerts
   - Alerts sent to all configured channels
   - Throttling prevents duplicate alerts within 5 minutes

### Alert Payload Example

Webhook alerts are sent as JSON POST requests:

```json
{
  "type": "target_down",
  "severity": "critical",
  "target": "http://localhost:3001",
  "message": "Target http://localhost:3001 is down",
  "timestamp": "2024-01-15T10:30:00Z",
  "metadata": {
    "error": "health check failed: connection refused"
  }
}
```

## Best Practices

### 1. Choose Appropriate Intervals
- **Critical services**: 15-30s intervals for fast detection
- **Standard services**: 30-60s intervals for balance
- **Non-critical services**: 60-120s intervals to reduce overhead

### 2. Set Reasonable Thresholds
- **Production**: Higher thresholds (3-5 failures) to avoid false positives
- **Development**: Lower thresholds (2-3 failures) for faster feedback

### 3. Backend Health Endpoints
Ensure your backend services implement `/health` endpoints:

```go
// Example health endpoint (Go)
func healthHandler(w http.ResponseWriter, r *http.Request) {
    // Check database connection, external dependencies, etc.
    if everythingOK {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "healthy",
        })
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
        })
    }
}
```

### 4. Monitor Alert Channels
- Set up webhook receivers (Slack, PagerDuty, email, etc.)
- Test alerts regularly
- Review logs for health check patterns

### 5. TLS Verification
- **Production**: Always use `insecureSkipVerify: false`
- **Development/Internal**: Only use `insecureSkipVerify: true` when necessary

## Integration Examples

### Slack Webhook

```yaml
monitoring:
  webhookUrl: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

The gateway will POST alerts to Slack in the format above.

### PagerDuty Integration

Use PagerDuty's Events API v2:

```yaml
monitoring:
  webhookUrl: "https://events.pagerduty.com/v2/enqueue"
```

You may need to create a middleware or adapter to transform the alert format.

### Custom Webhook Handler

Create a simple webhook receiver:

```go
func alertHandler(w http.ResponseWriter, r *http.Request) {
    var alert struct {
        Type      string                 `json:"type"`
        Severity  string                 `json:"severity"`
        Target    string                 `json:"target"`
        Message   string                 `json:"message"`
        Timestamp time.Time              `json:"timestamp"`
        Metadata  map[string]interface{} `json:"metadata"`
    }
    
    json.NewDecoder(r.Body).Decode(&alert)
    
    // Handle alert (send email, create ticket, etc.)
    log.Printf("Alert received: %s - %s", alert.Type, alert.Message)
    
    w.WriteHeader(http.StatusOK)
}
```

## Monitoring Health Status

### Via Logs

Health status changes are automatically logged:

```
INFO[0000] Starting health checker
INFO[0001] Added target for health monitoring target=http://localhost:3001 service=users-service
INFO[0032] Target recovered to healthy passes=2 url=http://localhost:3001
WARN[0125] Target marked as unhealthy error="connection refused" fails=3 url=http://localhost:3002
```

### Future: Admin UI Integration

A future enhancement will add health status visualization to the admin panel:
- Real-time health status dashboard
- Historical health trends
- Manual target enable/disable
- Alert history

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway                              │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Health Monitoring System                             │  │
│  │                                                      │  │
│  │  ┌────────────────┐    ┌─────────────────┐         │  │
│  │  │ TargetChecker  │───▶│  AlertManager   │         │  │
│  │  │                │    │                 │         │  │
│  │  │ • Periodic     │    │ • Webhook       │         │  │
│  │  │   health checks│    │   Channel       │         │  │
│  │  │ • Status       │    │ • Log Channel   │         │  │
│  │  │   tracking     │    │ • Throttling    │         │  │
│  │  │ • Metrics      │    └─────────────────┘         │  │
│  │  └────────────────┘                                │  │
│  │         │                                           │  │
│  │         ▼                                           │  │
│  │  ┌────────────────┐                                │  │
│  │  │ Backend        │                                │  │
│  │  │ Targets        │                                │  │
│  │  │                │                                │  │
│  │  │ • Health status│                                │  │
│  │  │ • Response time│                                │  │
│  │  │ • Success rate │                                │  │
│  │  └────────────────┘                                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Code Structure

```
pkg/health/
├── endpoints.go      # Health/readiness/liveness endpoints
├── checker.go        # TargetChecker - Active health monitoring
├── alerts.go         # AlertManager - Alert distribution system
└── types.go         # (Future) Shared types
```

## Future Enhancements

- [ ] Health status UI in admin panel
- [ ] Historical health data storage
- [ ] Advanced metrics (error rates, latency percentiles)
- [ ] Custom health check scripts
- [ ] Circuit breaker integration
- [ ] Automatic traffic shifting based on health
- [ ] Health-based load balancing weights

## See Also

- [Configuration Guide](./configuration.md)
- [Monitoring with Prometheus](./monitoring.md)
- [Admin Dashboard](./admin.md)
