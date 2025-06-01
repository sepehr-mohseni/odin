# Odin API Gateway Helm Chart

This Helm chart deploys Odin API Gateway on a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installation

### Add Helm Repository (if available)

```bash
helm repo add odin https://charts.odin-gateway.io
helm repo update
```

### Install from Local Chart

```bash
# Clone the repository
git clone https://github.com/sepehr-mohseni/odin.git
cd odin/deployments/helm

# Install the chart
helm install my-odin odin/
```

### Install with Custom Values

```bash
helm install my-odin odin/ -f values-production.yaml
```

## Configuration

The following table lists the configurable parameters and their default values.

| Parameter                       | Description              | Default        |
| ------------------------------- | ------------------------ | -------------- |
| `replicaCount`                  | Number of replicas       | `2`            |
| `image.repository`              | Image repository         | `odin-gateway` |
| `image.tag`                     | Image tag                | `latest`       |
| `service.type`                  | Kubernetes service type  | `ClusterIP`    |
| `service.port`                  | Service port             | `8080`         |
| `ingress.enabled`               | Enable ingress           | `false`        |
| `config.auth.jwtSecret`         | JWT secret key           | `""`           |
| `config.circuitBreaker.enabled` | Enable circuit breaker   | `true`         |
| `config.websocket.enabled`      | Enable WebSocket support | `true`         |
| `redis.enabled`                 | Enable Redis dependency  | `true`         |

## Examples

### Basic Installation

```bash
helm install odin odin/ \
  --set config.auth.jwtSecret="your-secret-key" \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host="api.example.com"
```

### Production Installation with OAuth2

```bash
helm install odin odin/ \
  --set replicaCount=3 \
  --set config.auth.jwtSecret="your-secret-key" \
  --set config.auth.oauth2.enabled=true \
  --set secrets.oauth2ClientSecrets.google="google-client-secret" \
  --set ingress.enabled=true \
  --set autoscaling.enabled=true
```

### With Monitoring

```bash
helm install odin odin/ \
  --set monitoring.serviceMonitor.enabled=true \
  --set prometheus.enabled=true
```

## Upgrading

```bash
helm upgrade my-odin odin/
```

## Uninstalling

```bash
helm uninstall my-odin
```
