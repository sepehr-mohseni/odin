# AI-Powered Traffic Analysis and Anomaly Detection

## Overview

Odin API Gateway now includes AI-powered traffic analysis and anomaly detection capabilities using a combination of statistical methods and optional Grok-1 model integration. This system continuously monitors traffic patterns, establishes baselines, detects anomalies, and generates alerts in real-time.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      API Gateway                             │
│                                                              │
│  ┌──────────────┐    ┌────────────────┐    ┌─────────────┐ │
│  │   Traffic    │───▶│  Collector     │───▶│  MongoDB    │ │
│  │  Middleware  │    │  (Aggregation) │    │   Storage   │ │
│  └──────────────┘    └────────────────┘    └─────────────┘ │
│                                                  │          │
│  ┌──────────────┐    ┌────────────────┐        │          │
│  │  Anomaly     │◀───│  Baseline      │◀───────┘          │
│  │  Detector    │    │  Calculator    │                    │
│  └──────┬───────┘    └────────────────┘                    │
│         │                                                   │
│         │  ┌──────────────────────────┐                    │
│         └─▶│     Grok-1 Service        │                    │
│            │  (Optional AI Analysis)   │                    │
│            └──────────┬───────────────┘                    │
│                       │                                     │
│         ┌─────────────▼────────────┐                       │
│         │     Alert Manager        │                       │
│         │  (Webhooks/Notifications)│                       │
│         └────────────────────────────┘                     │
└─────────────────────────────────────────────────────────────┘
```

## Key Features

### 1. Real-Time Traffic Monitoring
- Collects metrics for every request (latency, errors, status codes, IPs, user agents)
- Aggregates data in configurable time windows (default: 1 minute)
- Minimal performance impact (<1ms overhead per request)
- Automatic data retention management

### 2. Statistical Anomaly Detection
- **Error Rate Anomalies**: Detects spikes in 4xx/5xx responses
- **Latency Anomalies**: Identifies unusual response times
- **Traffic Volume Anomalies**: Detects traffic spikes or drops
- **DDoS Detection**: Recognizes distributed denial-of-service patterns
- **Bot Activity**: Identifies suspicious user agents and bot traffic
- **Data Exfiltration**: Detects unusual data transfer patterns

### 3. Dynamic Baseline Establishment
- Automatically builds statistical baselines from historical data
- Time-window based (hourly, daily, weekly patterns)
- Adapts to changing traffic patterns
- Requires minimum sample size for accuracy

### 4. AI-Enhanced Analysis (Optional)
- Integration with Grok-1 model for advanced pattern recognition
- Three operating modes:
  - **Lightweight**: Rule-based detection (recommended for production)
  - **Full**: Complete Grok-1 model (requires significant GPU resources)
  - **Proxy**: Forwards to external Grok API

### 5. Intelligent Alerting
- Severity-based classification (Low, Medium, High, Critical)
- Configurable alert channels (Webhook, Email, Slack)
- False positive management
- Alert deduplication and throttling

### 6. Admin Dashboard Integration
- Real-time anomaly visualization
- Baseline management and tuning
- Configuration updates without restart
- Historical analysis and reporting

## Installation

### Prerequisites

1. **MongoDB** (for data storage)
   ```bash
   # Already configured if you completed MongoDB integration
   ```

2. **Grok Service** (optional, for AI analysis)
   ```bash
   cd deployments/grok
   docker build -t odin-grok:latest .
   ```

### Configuration

Add to your `config.yaml`:

```yaml
ai:
  # Enable AI-powered traffic analysis
  enabled: true
  
  # How often to run anomaly detection analysis
  analysisInterval: 5m
  
  # Time window for baseline calculation
  baselineWindow: 24h
  
  # Z-score threshold for anomaly detection (3.0 = 3 standard deviations)
  anomalyThreshold: 3.0
  
  # Minimum number of samples required to establish baseline
  minSamplesForBaseline: 100
  
  # Use Grok model for enhanced analysis
  useGrokModel: false  # Set to true if Grok service is available
  
  # Grok service URL (if enabled)
  grokServiceUrl: "http://localhost:8000"
  
  # Timeout for Grok API calls
  grokTimeout: 30s
  
  # Enable alert notifications
  enableAlerts: true
  
  # Webhook URL for alerts
  alertWebhookUrl: "https://your-webhook-endpoint.com/alerts"
  
  # Data retention in days
  retentionDays: 90
  
  # How often to flush collected traffic data to MongoDB
  flushInterval: 1m
  
  # Optional tags
  tags:
    environment: "production"
    team: "platform"
```

## Usage

### Starting the Gateway with AI

```bash
# Build with AI support
make build-all-tools

# Start gateway
./bin/odin --config config.yaml

# Optionally start Grok service
docker run -d \
  -p 8000:8000 \
  -e GROK_MODE=lightweight \
  --name odin-grok \
  odin-grok:latest
```

### Using the Admin API

#### List Anomalies

```bash
# Get all anomalies
curl http://localhost:8080/admin/api/ai/anomalies \
  -u admin:password

# Filter by service
curl "http://localhost:8080/admin/api/ai/anomalies?service=users-service" \
  -u admin:password

# Filter by severity
curl "http://localhost:8080/admin/api/ai/anomalies?severity=high" \
  -u admin:password

# Get unresolved anomalies only
curl "http://localhost:8080/admin/api/ai/anomalies?resolved=false" \
  -u admin:password
```

Response:
```json
{
  "anomalies": [
    {
      "id": "anom-123",
      "timestamp": "2025-10-16T12:00:00Z",
      "service_name": "users-service",
      "endpoint": "/api/users",
      "anomaly_type": "error_spike",
      "severity": "high",
      "score": 85.5,
      "description": "Error rate 15.00% is 4.23 standard deviations above baseline (2.50%)",
      "details": {
        "current_error_rate": 0.15,
        "baseline_error_rate": 0.025,
        "z_score": 4.23,
        "error_count": 150,
        "request_count": 1000
      },
      "resolved": false
    }
  ],
  "count": 1
}
```

#### Get Specific Anomaly

```bash
curl http://localhost:8080/admin/api/ai/anomalies/anom-123 \
  -u admin:password
```

#### Resolve Anomaly

```bash
curl -X POST http://localhost:8080/admin/api/ai/anomalies/anom-123/resolve \
  -u admin:password
```

#### Mark as False Positive

```bash
curl -X POST http://localhost:8080/admin/api/ai/anomalies/anom-123/false-positive \
  -u admin:password
```

#### Get Statistics

```bash
curl http://localhost:8080/admin/api/ai/stats \
  -u admin:password
```

Response:
```json
{
  "total_anomalies": 25,
  "by_severity": {
    "critical": 2,
    "high": 8,
    "medium": 10,
    "low": 5
  },
  "by_type": {
    "error_spike": 10,
    "latency_spike": 8,
    "traffic_spike": 5,
    "ddos_attack": 2
  },
  "unresolved_count": 15,
  "analysis_status": "active",
  "last_analysis": "2025-10-16T12:05:00Z"
}
```

#### Get Baselines

```bash
# List all baselines for a service
curl http://localhost:8080/admin/api/ai/baselines/users-service \
  -u admin:password
```

Response:
```json
{
  "service": "users-service",
  "baselines": [
    {
      "service_name": "users-service",
      "endpoint": "",
      "time_window": "hourly",
      "sample_size": 1440,
      "avg_request_rate": 100.5,
      "stddev_request_rate": 15.2,
      "avg_error_rate": 0.025,
      "stddev_error_rate": 0.008,
      "avg_latency": 45.3,
      "stddev_latency": 12.1,
      "last_updated": "2025-10-16T12:00:00Z"
    }
  ],
  "count": 1
}
```

## Anomaly Types

### 1. Error Spike
**Trigger**: Error rate exceeds baseline by configured threshold  
**Typical Causes**:
- Backend service failures
- Database connectivity issues
- Invalid input validation
- Recent deployment bugs

**Recommended Actions**:
- Check backend service health
- Review recent deployments
- Examine error logs
- Increase timeout values if needed

### 2. Latency Spike
**Trigger**: Average response time significantly higher than baseline  
**Typical Causes**:
- Database query performance degradation
- Insufficient backend capacity
- Network issues
- Resource contention

**Recommended Actions**:
- Scale backend instances
- Optimize database queries
- Enable caching
- Check resource utilization

### 3. Traffic Spike
**Trigger**: Request rate significantly above baseline  
**Typical Causes**:
- Legitimate traffic increase (marketing campaign, viral content)
- Bot activity
- DDoS attack
- Misconfigured client retries

**Recommended Actions**:
- Verify legitimacy of traffic
- Scale infrastructure if needed
- Enable rate limiting
- Activate CDN caching

### 4. Traffic Drop
**Trigger**: Request rate significantly below baseline  
**Typical Causes**:
- Upstream service failures
- DNS issues
- Network problems
- Planned maintenance

**Recommended Actions**:
- Check upstream dependencies
- Verify DNS resolution
- Review firewall rules
- Check for maintenance windows

### 5. DDoS Attack
**Trigger**: High request volume from few IPs  
**Detection Criteria**: Single IP accounts for >70% of traffic with >1000 requests  
**Recommended Actions**:
- Enable aggressive rate limiting
- Block suspicious IP ranges
- Contact DDoS mitigation provider
- Enable CDN protection

### 6. Bot Activity
**Trigger**: Suspicious user agents with high request volume  
**Detection Patterns**: Known bot signatures, scrapers, automated tools  
**Recommended Actions**:
- Implement CAPTCHA challenges
- Update bot detection rules
- Block or throttle bot traffic
- Review robots.txt

## Tuning and Optimization

### Anomaly Threshold

The `anomalyThreshold` parameter controls sensitivity:

- **2.0**: Very sensitive (more anomalies, more false positives)
- **3.0**: Balanced (recommended for most use cases)
- **4.0**: Conservative (fewer anomalies, may miss some issues)
- **5.0**: Very conservative (only extreme outliers)

### Baseline Window

- **Hourly**: Captures short-term patterns, quick adaptation
- **Daily**: Balances adaptation speed with stability (recommended)
- **Weekly**: Captures weekly patterns (weekend vs weekday)

### Analysis Interval

- **1m**: Real-time detection, higher CPU usage
- **5m**: Good balance (recommended)
- **15m**: Lower resource usage, slower detection

### Flush Interval

- **30s**: More frequent updates, higher MongoDB load
- **1m**: Balanced (recommended)
- **5m**: Lower MongoDB load, less granular data

## Performance Impact

### Resource Usage (per 1000 req/s)

| Component | CPU | Memory | MongoDB Storage |
|-----------|-----|--------|-----------------|
| Middleware | ~5% | ~50MB | - |
| Collector | ~3% | ~100MB | - |
| Detector | ~10% | ~200MB | - |
| Total | ~18% | ~350MB | ~1GB/day |

### Optimization Tips

1. **Disable Grok for production**: Use lightweight mode instead
2. **Adjust flush interval**: Longer intervals reduce MongoDB writes
3. **Limit baseline samples**: Don't exceed 10,000 samples
4. **Use indexes**: MongoDB indexes are auto-created
5. **Enable TTL**: Traffic patterns auto-expire after 7 days

## Monitoring

### Prometheus Metrics

The following metrics are exported:

```
# Anomaly detection
odin_ai_anomalies_total{service,type,severity}
odin_ai_analysis_duration_seconds
odin_ai_baselines_total

# Traffic collection
odin_ai_patterns_collected_total
odin_ai_patterns_flushed_total
odin_ai_flush_errors_total

# Alerts
odin_ai_alerts_sent_total{channel,severity}
odin_ai_alert_errors_total{channel}
```

### Health Checks

```bash
# Check AI system status
curl http://localhost:8080/admin/api/ai/stats

# Check Grok service (if enabled)
curl http://localhost:8000/health
```

## Troubleshooting

### No Anomalies Detected

**Possible Causes**:
1. Insufficient baseline data (need minimum samples)
2. Threshold too high
3. Traffic too consistent
4. AI disabled in config

**Solutions**:
- Wait for baseline establishment (`minSamplesForBaseline` samples)
- Lower `anomalyThreshold` to 2.5 or 2.0
- Verify `ai.enabled: true` in config

### Too Many False Positives

**Possible Causes**:
1. Threshold too low
2. High traffic variability
3. Recent pattern changes

**Solutions**:
- Increase `anomalyThreshold` to 4.0 or 5.0
- Increase `baselineWindow` to capture more patterns
- Mark false positives via API to help tuning

### High Resource Usage

**Possible Causes**:
1. Analysis interval too frequent
2. Grok model enabled
3. Too many services monitored

**Solutions**:
- Increase `analysisInterval` to 10m or 15m
- Disable Grok (`useGrokModel: false`)
- Increase `flushInterval` to 5m

### Alerts Not Sending

**Possible Causes**:
1. `enableAlerts: false`
2. Invalid webhook URL
3. Network connectivity issues

**Solutions**:
- Enable alerts in config
- Test webhook URL manually
- Check alerter logs for errors

## Integration Examples

### Webhook Alert Format

When an alert is triggered, a POST request is sent to `alertWebhookUrl`:

```json
{
  "id": "alert-456",
  "timestamp": "2025-10-16T12:00:00Z",
  "severity": "high",
  "title": "[high] error_spike Detected",
  "message": "Error rate 15.00% is 4.23 standard deviations above baseline (2.50%)",
  "service_name": "users-service",
  "endpoint": "/api/users",
  "anomaly_id": "anom-123",
  "tags": {
    "environment": "production",
    "team": "platform"
  }
}
```

### Slack Integration

Create a Slack webhook and configure:

```yaml
ai:
  alertWebhookUrl: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
```

### PagerDuty Integration

Use PagerDuty's Events API v2:

```yaml
ai:
  alertWebhookUrl: "https://events.pagerduty.com/v2/enqueue"
```

## Production Deployment

### docker-compose.yml

```yaml
version: '3.8'

services:
  odin-gateway:
    image: odin:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/etc/odin/config.yaml
    environment:
      - MONGODB_URI=${MONGODB_URI}
      - ALERT_WEBHOOK_URL=${ALERT_WEBHOOK_URL}
    depends_on:
      - mongodb
      - odin-grok

  odin-grok:
    image: odin-grok:lightweight
    ports:
      - "8000:8000"
    environment:
      - GROK_MODE=lightweight
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  mongodb:
    image: mongo:7
    volumes:
      - mongodb_data:/data/db
    environment:
      - MONGO_INITDB_ROOT_USERNAME=${MONGO_USERNAME}
      - MONGO_INITDB_ROOT_PASSWORD=${MONGO_PASSWORD}

volumes:
  mongodb_data:
```

### Kubernetes Deployment

See `deployments/helm/odin/values.yaml` for AI configuration:

```yaml
ai:
  enabled: true
  analysisInterval: "5m"
  useGrokModel: false
  
grok:
  enabled: true
  mode: lightweight
  replicas: 2
  resources:
    requests:
      cpu: "500m"
      memory: "1Gi"
    limits:
      cpu: "2000m"
      memory: "2Gi"
```

## Security Considerations

1. **Webhook URLs**: Use HTTPS and authenticate webhook endpoints
2. **MongoDB Access**: Enable authentication and TLS
3. **Grok Service**: Run in private network, not publicly exposed
4. **Admin API**: Protect with authentication
5. **Data Retention**: Configure appropriate retention for compliance

## Best Practices

1. **Start Conservative**: Begin with high threshold (4.0) and lower gradually
2. **Monitor False Positives**: Track and mark false positives to improve accuracy
3. **Regular Baseline Updates**: Baselines auto-update, but review periodically
4. **Alert Fatigue**: Configure severity appropriately to avoid alert overload
5. **Test Before Production**: Run in staging environment first
6. **Document Baselines**: Keep notes on expected traffic patterns
7. **Tune Per Service**: Different services may need different thresholds

## Roadmap

- [ ] Machine learning model training from historical data
- [ ] Predictive anomaly detection
- [ ] Automatic threshold tuning
- [ ] Multi-variate anomaly detection
- [ ] Custom anomaly rules via admin UI
- [ ] Integration with APM tools (Datadog, New Relic)
- [ ] Advanced visualization and dashboards

## Support

For issues and questions:
- GitHub Issues: https://github.com/your-org/odin/issues
- Documentation: docs/ai-analysis.md
- Slack: #odin-support

## License

Apache 2.0 (same as main Odin project and Grok-1)
