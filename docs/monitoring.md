# Monitoring and Observability

This guide covers monitoring, logging, and observability features of Odin API Gateway.

## Overview

Odin provides comprehensive monitoring capabilities:

1. **Prometheus metrics** for real-time performance monitoring
2. **Structured logging** for troubleshooting and audit trails
3. **Health checks** for system status monitoring
4. **Tracing** for request flow visualization

## Prometheus Metrics

### Configuration

Enable and configure Prometheus metrics in your `config.yaml`:

```yaml
monitoring:
  enabled: true
  path: /metrics
```

### Available Metrics

Odin exposes the following metrics:

#### Request Metrics

- `api_gateway_requests_total` - Total requests processed (labels: `service`, `method`, `status`)
- `api_gateway_request_duration_seconds` - Request duration histogram (labels: `service`, `method`)
- `api_gateway_response_size_bytes` - Response size histogram (labels: `service`)
- `api_gateway_active_requests` - Currently active requests gauge (labels: `service`)

#### Cache Metrics

- `api_gateway_cache_hits_total` - Cache hit counter (labels: `service`)
- `api_gateway_cache_misses_total` - Cache miss counter (labels: `service`)

#### Rate Limiting Metrics

- `api_gateway_rate_limited_total` - Rate limited requests counter (labels: `service`)

#### System Metrics

- `api_gateway_uptime_seconds` - Gateway uptime
- `api_gateway_goroutines` - Number of active goroutines
- `go_*` - Standard Go runtime metrics

### Prometheus Integration

To scrape metrics:

1. Add the Odin endpoint to your Prometheus config:

```yaml
scrape_configs:
  - job_name: 'odin-api-gateway'
    scrape_interval: 5s
    static_configs:
      - targets: ['odin:8080']
```

2. Access metrics directly:

```bash
curl http://localhost:8080/metrics
```

### Grafana Dashboards

We provide pre-built Grafana dashboards in the `monitoring/grafana/dashboards` directory. To use them:

1. Import the dashboards into your Grafana instance
2. Ensure your Prometheus data source is configured correctly

## Logging

### Configuration

Configure logging in your `config.yaml`:

```yaml
logging:
  level: info # debug, info, warn, error
  json: true # true for structured JSON logs, false for human-readable
```

You can override the log level with the `LOG_LEVEL` environment variable.

### Log Levels

- **Debug**: Verbose development information
- **Info**: General operational information
- **Warn**: Warning conditions
- **Error**: Error conditions

### Request Logging

Each request generates a log entry with:

- Request method and URL
- Response status code
- Response time
- Client IP
- User-Agent
- Error details (if any)

Example:

```json
{
  "level": "info",
  "method": "GET",
  "uri": "/api/users/123",
  "status": 200,
  "latency_ms": 45,
  "ip": "192.168.1.1",
  "user_agent": "curl/7.64.1",
  "time": "2023-05-15T12:34:56Z"
}
```

### Structured Logging

When `json: true` is set, logs are output in JSON format for easy ingestion into log management systems like:

- ELK Stack (Elasticsearch, Logstash, Kibana)
- Graylog
- Splunk
- Loki

### Log Rotation

For production deployments, we recommend using a log rotation tool like `logrotate` or container/orchestrator logging features.

## Health Checks

Odin provides a health check endpoint at `/health` that returns:

```json
{
  "status": "UP",
  "version": "1.0.0",
  "uptime": "24h12m5s"
}
```

Use this endpoint for:

- Kubernetes liveness/readiness probes
- Load balancer health checks
- Monitoring system integration

## Distributed Tracing

### Tracing Configuration

Enable tracing in your `config.yaml`:

```yaml
tracing:
  enabled: true
  type: jaeger # jaeger, zipkin, or otlp
  endpoint: http://jaeger:14268/api/traces
  serviceName: odin-api-gateway
  sampleRate: 0.1 # Sample 10% of requests
```

### Trace Propagation

Odin propagates trace headers to backend services:

- B3 Headers (Zipkin)
- Jaeger Headers
- W3C Trace Context

### Viewing Traces

Use your tracing system's UI (like Jaeger UI) to view:

- End-to-end request flow
- Service dependencies
- Performance bottlenecks
- Error points

## Monitoring Stack

We recommend the following monitoring stack:

1. **Prometheus** for metrics collection
2. **Grafana** for metrics visualization
3. **Loki** or **ELK** for log aggregation
4. **Jaeger** or **Zipkin** for distributed tracing
5. **Alertmanager** for alerts

Our `docker-compose.yml` includes these components for easy setup.

## Alerting

Configure alerts for important conditions:

1. High error rate
2. Elevated latency
3. High rate of 5xx responses
4. High CPU or memory usage

Example Prometheus alert rule:

```yaml
groups:
  - name: odin-alerts
    rules:
      - alert: HighErrorRate
        expr: sum(rate(api_gateway_requests_total{status=~"5.."}[5m])) / sum(rate(api_gateway_requests_total[5m])) > 0.05
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: 'High error rate detected'
          description: 'Error rate is above 5% for more than 1 minute'
```

## Troubleshooting with Monitoring

Common troubleshooting scenarios:

1. **Slow API Response**: Check `api_gateway_request_duration_seconds` metrics and tracing data
2. **Service Unavailable**: Check logs for connection errors
3. **Memory Leaks**: Monitor `go_memstats_alloc_bytes` for unusual growth
4. **Rate Limiting Issues**: Check `api_gateway_rate_limited_total` metrics
5. **Cache Problems**: Look at hit/miss ratios in cache metrics
