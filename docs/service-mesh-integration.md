# Service Mesh Integration

The Odin API Gateway provides seamless integration with popular service mesh solutions, enabling advanced features like mutual TLS (mTLS), service discovery, intelligent routing, and enhanced observability.

## Supported Service Meshes

- **Istio** - The most popular service mesh with advanced traffic management
- **Linkerd** - Lightweight and simple service mesh focused on reliability
- **Consul Connect** - HashiCorp's service mesh solution with service discovery
- **None** - No service mesh integration (default)

## Why Use Service Mesh Integration?

Service mesh integration provides several benefits:

### 1. **Mutual TLS (mTLS)**
- Automatic encryption of all service-to-service communication
- Certificate-based authentication between services
- Secure communication without code changes

### 2. **Service Discovery**
- Automatic discovery of backend services
- Dynamic endpoint updates
- Health-aware load balancing

### 3. **Traffic Management**
- Advanced routing rules
- Traffic splitting for A/B testing
- Circuit breaking and retries

### 4. **Observability**
- Distributed tracing integration
- Metrics collection
- Traffic visualization

### 5. **Security**
- Zero-trust networking
- Service-to-service authorization
- Traffic encryption

## Configuration

### Basic Configuration

Add the `serviceMesh` section to your `config.yaml`:

```yaml
serviceMesh:
  enabled: true
  type: istio  # or linkerd, consul, none
  namespace: default
  refreshInterval: 30s
  mtlsEnabled: true
  certFile: /etc/certs/cert.pem
  keyFile: /etc/certs/key.pem
  caFile: /etc/certs/ca.pem
```

### Istio Integration

```yaml
serviceMesh:
  enabled: true
  type: istio
  namespace: default
  trustDomain: cluster.local
  
  mtlsEnabled: true
  certFile: /etc/certs/cert-chain.pem
  keyFile: /etc/certs/key.pem
  caFile: /etc/certs/root-cert.pem
  
  istio:
    pilotAddr: istiod.istio-system.svc.cluster.local:15012
    mixerAddr: istio-telemetry.istio-system.svc.cluster.local:15004
    enableTelemetry: true
    enablePolicyCheck: true
    customHeaders:
      - X-Request-ID
      - X-B3-TraceId
    injectSidecar: true
```

#### Istio Features

- **Pilot Integration**: Connect to Istio's control plane for service discovery
- **Mixer Integration**: Send telemetry data to Istio's mixer
- **Header Injection**: Automatically inject tracing and context headers
- **Policy Checks**: Enforce Istio policies
- **Sidecar Injection**: Configure sidecar injection behavior

### Linkerd Integration

```yaml
serviceMesh:
  enabled: true
  type: linkerd
  namespace: default
  
  mtlsEnabled: true
  certFile: /var/run/linkerd/tls/cert.pem
  keyFile: /var/run/linkerd/tls/key.pem
  caFile: /var/run/linkerd/tls/ca.pem
  
  linkerd:
    controlPlaneAddr: linkerd-controller-api.linkerd.svc.cluster.local:8085
    tapAddr: linkerd-tap.linkerd.svc.cluster.local:8088
    enableTap: true
    profileNamespace: linkerd
```

#### Linkerd Features

- **Control Plane Connection**: Integrate with Linkerd's control plane
- **Tap API**: Enable real-time traffic inspection
- **Service Profiles**: Use Linkerd service profiles for routing
- **Automatic mTLS**: Leverage Linkerd's automatic TLS

### Consul Connect Integration

```yaml
serviceMesh:
  enabled: true
  type: consul
  namespace: default
  
  mtlsEnabled: true
  certFile: /etc/consul/tls/cert.pem
  keyFile: /etc/consul/tls/key.pem
  caFile: /etc/consul/tls/ca.pem
  
  consul:
    httpAddr: http://consul-server.consul.svc.cluster.local:8500
    datacenter: dc1
    token: ${CONSUL_HTTP_TOKEN}
    enableConnect: true
```

#### Consul Features

- **Service Discovery**: Automatic discovery via Consul catalog
- **Health Checks**: Integration with Consul health checks
- **Connect Integration**: Use Consul Connect for mTLS
- **Datacenter Awareness**: Multi-datacenter support

## How It Works

### 1. Initialization

When the gateway starts with service mesh enabled:

1. **Mesh Manager**: Initializes the appropriate mesh client (Istio/Linkerd/Consul)
2. **TLS Setup**: Configures mTLS certificates if enabled
3. **Connection**: Establishes connection to mesh control plane
4. **Discovery**: Begins periodic service discovery

### 2. Request Flow

For each incoming request:

```
Client Request
    ↓
API Gateway
    ↓
Service Mesh Middleware (injects headers)
    ↓
Proxy Middleware (adds mesh context)
    ↓
Backend Service (via mesh)
```

### 3. Header Injection

The gateway automatically injects mesh-specific headers:

**Istio Headers:**
- `X-Request-Id`: Unique request identifier
- `X-B3-TraceId`: Zipkin trace ID
- `X-B3-SpanId`: Zipkin span ID
- `X-B3-Sampled`: Sampling decision
- `X-Envoy-Namespace`: Target namespace

**Linkerd Headers:**
- `l5d-ctx-trace`: Linkerd trace context
- `l5d-ctx-deadline`: Request deadline
- `l5d-reqid`: Linkerd request ID

**Consul Headers:**
- `X-Consul-Token`: Consul ACL token
- `X-Request-Id`: Request identifier
- `X-Consul-Datacenter`: Target datacenter

### 4. Service Discovery

The gateway periodically queries the service mesh for updated service endpoints:

```go
// Automatic refresh every 30s (configurable)
services, err := meshManager.DiscoverServices()

// Get endpoints for specific service
endpoints, err := meshManager.GetServiceEndpoints("users-service")
```

### 5. mTLS Communication

When mTLS is enabled:

1. Gateway loads certificates from configured paths
2. Creates TLS-enabled HTTP client
3. All backend communication uses mTLS
4. Certificates are validated against mesh CA

## Deployment

### Kubernetes with Istio

1. **Install Istio**:
   ```bash
   istioctl install --set profile=default
   ```

2. **Label namespace for sidecar injection**:
   ```bash
   kubectl label namespace default istio-injection=enabled
   ```

3. **Deploy gateway with mesh config**:
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: odin-config
   data:
     config.yaml: |
       serviceMesh:
         enabled: true
         type: istio
         namespace: default
         mtlsEnabled: true
         certFile: /etc/istio/certs/cert-chain.pem
         keyFile: /etc/istio/certs/key.pem
         caFile: /etc/istio/certs/root-cert.pem
   ```

4. **Mount certificates in deployment**:
   ```yaml
   volumeMounts:
     - name: istio-certs
       mountPath: /etc/istio/certs
       readOnly: true
   volumes:
     - name: istio-certs
       secret:
         secretName: istio.default
   ```

### Kubernetes with Linkerd

1. **Install Linkerd**:
   ```bash
   linkerd install | kubectl apply -f -
   ```

2. **Inject Linkerd proxy**:
   ```bash
   kubectl get deploy odin-gateway -o yaml | linkerd inject - | kubectl apply -f -
   ```

3. **Configure gateway**:
   ```yaml
   serviceMesh:
     enabled: true
     type: linkerd
     namespace: default
     mtlsEnabled: true
     certFile: /var/run/linkerd/tls/cert.pem
     keyFile: /var/run/linkerd/tls/key.pem
     caFile: /var/run/linkerd/tls/ca.pem
   ```

### Kubernetes with Consul

1. **Install Consul**:
   ```bash
   helm install consul hashicorp/consul --set global.name=consul
   ```

2. **Configure gateway**:
   ```yaml
   serviceMesh:
     enabled: true
     type: consul
     consul:
       httpAddr: http://consul-server.default.svc:8500
       datacenter: dc1
       enableConnect: true
   ```

## Best Practices

### 1. Certificate Management

- **Use cert-manager**: Automate certificate rotation
- **Short-lived certificates**: Rotate certificates frequently (e.g., every 24h)
- **Separate CA**: Use dedicated CA for service mesh

### 2. Namespace Isolation

```yaml
serviceMesh:
  namespace: production  # Isolate production traffic
```

### 3. Gradual Rollout

Start with mesh disabled, then enable gradually:

1. Deploy without mesh integration
2. Enable mesh with `mtlsEnabled: false`
3. Enable mTLS for non-production
4. Enable mTLS for production

### 4. Monitoring

Monitor mesh integration:

```yaml
monitoring:
  enabled: true
  path: /metrics
```

Watch for:
- Certificate expiration
- Service discovery failures
- mTLS handshake errors
- Connection pool exhaustion

### 5. Performance Tuning

```yaml
serviceMesh:
  refreshInterval: 30s  # Balance freshness vs overhead
  
server:
  timeout: 30s
  readTimeout: 10s
  writeTimeout: 10s
```

## Troubleshooting

### Common Issues

#### 1. Certificate Errors

**Symptom**: `x509: certificate signed by unknown authority`

**Solution**:
```bash
# Verify certificate paths
ls -la /etc/certs/

# Check certificate validity
openssl x509 -in /etc/certs/cert.pem -text -noout

# Ensure CA certificate is correct
openssl verify -CAfile /etc/certs/ca.pem /etc/certs/cert.pem
```

#### 2. Service Discovery Failures

**Symptom**: `failed to discover services`

**Solution**:
- Verify mesh control plane is accessible
- Check network policies
- Ensure correct namespace/datacenter
- Validate authentication tokens

#### 3. Header Injection Not Working

**Symptom**: Mesh headers not appearing in backend

**Solution**:
```yaml
# Enable debug logging
logging:
  level: debug

# Check middleware order in logs
```

#### 4. mTLS Handshake Failures

**Symptom**: `tls: bad certificate`

**Solution**:
- Ensure certificates are not expired
- Verify trust domain matches
- Check certificate subject alternative names (SANs)

### Debug Mode

Enable debug logging for mesh integration:

```yaml
logging:
  level: debug
```

Check logs for mesh-related messages:
```bash
kubectl logs -f odin-gateway-xxx | grep -i mesh
kubectl logs -f odin-gateway-xxx | grep -i tls
```

## Examples

### Example 1: Istio with mTLS

```yaml
serviceMesh:
  enabled: true
  type: istio
  namespace: production
  trustDomain: cluster.local
  mtlsEnabled: true
  certFile: /etc/certs/cert-chain.pem
  keyFile: /etc/certs/key.pem
  caFile: /etc/certs/root-cert.pem
  
  istio:
    pilotAddr: istiod.istio-system.svc:15012
    enableTelemetry: true
    enablePolicyCheck: true

services:
  - name: users-service
    basePath: /api/users
    # Targets discovered automatically from Istio
    targets:
      - http://users-service.production.svc.cluster.local:8080
```

### Example 2: Linkerd with Tap

```yaml
serviceMesh:
  enabled: true
  type: linkerd
  namespace: default
  
  linkerd:
    controlPlaneAddr: linkerd-controller-api.linkerd.svc:8085
    tapAddr: linkerd-tap.linkerd.svc:8088
    enableTap: true

services:
  - name: products-service
    basePath: /api/products
    targets:
      - http://products-service.default.svc.cluster.local:8080
```

### Example 3: Multi-Datacenter Consul

```yaml
serviceMesh:
  enabled: true
  type: consul
  
  consul:
    httpAddr: http://consul-server.consul.svc:8500
    datacenter: us-east-1
    token: ${CONSUL_TOKEN}
    enableConnect: true

services:
  - name: orders-service
    basePath: /api/orders
    # Service discovered from Consul catalog
    targets:
      - http://orders-service.service.consul:8080
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Odin API Gateway                         │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Service Mesh Integration                             │  │
│  │                                                      │  │
│  │  ┌────────────┐  ┌────────────┐  ┌──────────────┐  │  │
│  │  │  Istio     │  │  Linkerd   │  │   Consul     │  │  │
│  │  │  Client    │  │  Client    │  │   Client     │  │  │
│  │  └────────────┘  └────────────┘  └──────────────┘  │  │
│  │         │               │                │          │  │
│  │         └───────────────┴────────────────┘          │  │
│  │                        │                            │  │
│  │                 ┌──────▼─────────┐                  │  │
│  │                 │  Mesh Manager  │                  │  │
│  │                 │                │                  │  │
│  │                 │ • mTLS Setup   │                  │  │
│  │                 │ • Discovery    │                  │  │
│  │                 │ • Headers      │                  │  │
│  │                 └────────────────┘                  │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                │
│                           ▼                                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ HTTP Client with mTLS                                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                 Service Mesh Control Plane                  │
│  ┌──────────┐    ┌───────────┐    ┌──────────────┐        │
│  │  Istiod  │    │  Linkerd  │    │    Consul    │        │
│  │ (Pilot)  │    │Controller │    │    Server    │        │
│  └──────────┘    └───────────┘    └──────────────┘        │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   Backend Services                          │
│  ┌──────────┐    ┌───────────┐    ┌──────────────┐        │
│  │  Users   │    │ Products  │    │    Orders    │        │
│  │ Service  │    │  Service  │    │   Service    │        │
│  └──────────┘    └───────────┘    └──────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## Future Enhancements

- [ ] Automatic service endpoint updates from mesh
- [ ] Circuit breaker integration with mesh policies
- [ ] Rate limiting based on mesh quotas
- [ ] Custom mesh adapters/plugins
- [ ] WebAssembly filter support
- [ ] Service mesh dashboard in admin UI

## See Also

- [Distributed Tracing](./tracing.md)
- [Health Monitoring](./health-monitoring.md)
- [mTLS Configuration](./mtls.md)
- [Kubernetes Deployment](./deployment.md)
