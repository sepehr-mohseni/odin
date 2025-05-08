# Deployment Guide

This guide covers different deployment options for Odin API Gateway.

## Docker Deployment

### Single Container

The simplest way to deploy Odin is using Docker:

```bash
# Build the Docker image
docker build -t odin-gateway:latest .

# Run the container
docker run -p 8080:8080 -p 8081:8081 -v $(pwd)/config:/app/config odin-gateway:latest
```

Environment variables can be passed to override configuration:

```bash
docker run -p 8080:8080 \
  -e LOG_LEVEL=debug \
  -e GATEWAY_PORT=8080 \
  -e ODIN_JWT_SECRET=your-secret-key \
  -v $(pwd)/config:/app/config \
  odin-gateway:latest
```

### Docker Compose

For a complete environment with Redis, Prometheus, and Grafana:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f odin
```

The `docker-compose.yml` file in the root directory includes:

- Odin API Gateway
- Redis (for caching and rate limiting)
- Prometheus (for metrics collection)
- Grafana (for metrics visualization)

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured to connect to your cluster
- Optional: Helm for chart-based deployment

### Deploying with kubectl

1. Create ConfigMap and Secret:

```bash
# Create a ConfigMap for configuration
kubectl create configmap odin-config \
  --from-file=config.yaml=./config/config.yaml \
  --from-file=services.yaml=./config/services.yaml

# Create Secret for sensitive data
kubectl create secret generic odin-secrets \
  --from-literal=jwt-secret=your-jwt-secret-here
```

2. Apply Kubernetes manifests:

```bash
kubectl apply -f deployments/kubernetes/deployment.yaml
kubectl apply -f deployments/kubernetes/service.yaml
```

3. Create Ingress (optional):

```bash
kubectl apply -f deployments/kubernetes/ingress.yaml
```

### Deploying with Helm (Coming Soon)

We're working on a Helm chart to simplify Kubernetes deployments. Check back soon!

## Production Considerations

### High Availability

For production deployments, consider:

1. Running multiple replicas (at least 2-3 for high availability)
2. Using a load balancer for distributing traffic
3. Implementing health checks and auto-healing
4. Setting appropriate resource limits and requests

### Persistent Configuration

For production, store your configuration in a persistent medium:

1. ConfigMaps and Secrets in Kubernetes
2. Environment variables for critical settings
3. Redis or a database for dynamic configuration

### TLS/SSL Configuration

In production, always use TLS:

1. Configure TLS certificates (Let's Encrypt or your own certs)
2. Use a secure ingress or load balancer with TLS termination
3. Ensure internal communication is also encrypted when necessary

### Resource Requirements

Minimum recommended resources:

- 1 CPU core
- 512MB RAM
- 1GB disk space

For high-traffic environments:

- 2-4 CPU cores
- 1-2GB RAM
- Separate Redis instance with appropriate resources

## Troubleshooting Deployments

Common issues and solutions:

1. **Gateway not starting**: Check logs with `docker logs` or `kubectl logs`
2. **Cannot connect to services**: Verify network connectivity and service discovery
3. **High latency**: Check resource utilization and consider scaling
4. **Authentication failures**: Verify JWT secret configuration

For more help, check the logs or open an issue on GitHub.
