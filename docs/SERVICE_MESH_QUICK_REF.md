# Service Mesh Integration - Quick Reference

## ✅ Implementation Complete

### Package Structure
```
pkg/servicemesh/
├── types.go        (85 lines)  - Core types and configs
├── mesh.go         (218 lines) - Manager and interface
├── istio.go        (133 lines) - Istio implementation
├── linkerd.go      (127 lines) - Linkerd implementation
├── consul.go       (158 lines) - Consul implementation
├── none.go         (63 lines)  - No-op implementation
└── middleware.go   (37 lines)  - Echo middleware
                    ────────────
Total:              821 lines
```

### Quick Start

#### 1. Enable in Configuration
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

#### 2. Mesh-Specific Configuration

**Istio:**
```yaml
serviceMesh:
  type: istio
  istio:
    pilotAddr: istiod.istio-system.svc:15012
    enableTelemetry: true
```

**Linkerd:**
```yaml
serviceMesh:
  type: linkerd
  linkerd:
    controlPlaneAddr: linkerd-controller-api.linkerd.svc:8085
    enableTap: true
```

**Consul:**
```yaml
serviceMesh:
  type: consul
  consul:
    httpAddr: http://consul-server.consul.svc:8500
    datacenter: dc1
```

### Features

| Feature | Istio | Linkerd | Consul |
|---------|-------|---------|--------|
| mTLS | ✅ | ✅ | ✅ |
| Service Discovery | ✅ | ✅ | ✅ |
| Header Injection | ✅ | ✅ | ✅ |
| Distributed Tracing | ✅ | ✅ | ✅ |
| Health Checks | ✅ | ✅ | ✅ |

### Injected Headers

**Istio:**
- `X-Request-Id`
- `X-B3-TraceId`
- `X-B3-SpanId`
- `X-B3-Sampled`
- `X-Envoy-Namespace`

**Linkerd:**
- `l5d-ctx-trace`
- `l5d-ctx-deadline`
- `l5d-reqid`

**Consul:**
- `X-Consul-Token`
- `X-Request-Id`
- `X-Consul-Datacenter`

### Usage in Code

#### Initialize Manager
```go
import "odin/pkg/servicemesh"

manager := servicemesh.NewManager(config, logger)
if err := manager.Start(); err != nil {
    log.Fatal(err)
}
defer manager.Stop()
```

#### Service Discovery
```go
// Discover all services
services, err := manager.DiscoverServices()

// Get specific service endpoints
endpoints, err := manager.GetServiceEndpoints("users-service")
for _, ep := range endpoints {
    fmt.Printf("Endpoint: %s (healthy: %v)\n", ep.Address, ep.Healthy)
}
```

#### Add Middleware
```go
// Header injection middleware
e.Use(servicemesh.Middleware(manager))

// Proxy middleware (adds mesh context)
e.Use(servicemesh.ProxyMiddleware(manager))
```

### Kubernetes Deployment

#### 1. Create Certificate Secret
```bash
kubectl create secret tls mesh-certs \
  --cert=/path/to/cert.pem \
  --key=/path/to/key.pem
```

#### 2. ConfigMap
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
      mtlsEnabled: true
      certFile: /etc/certs/tls.crt
      keyFile: /etc/certs/tls.key
```

#### 3. Deployment
```yaml
spec:
  template:
    spec:
      containers:
      - name: odin-gateway
        volumeMounts:
        - name: certs
          mountPath: /etc/certs
          readOnly: true
        - name: config
          mountPath: /etc/odin
      volumes:
      - name: certs
        secret:
          secretName: mesh-certs
      - name: config
        configMap:
          name: odin-config
```

### Troubleshooting

#### Check Mesh Status
```bash
# Gateway logs
kubectl logs -f odin-gateway-xxx | grep -i mesh

# Certificate validation
openssl x509 -in /etc/certs/cert.pem -text -noout
```

#### Common Issues

**Certificate errors:**
```bash
# Verify CA can validate cert
openssl verify -CAfile /etc/certs/ca.pem /etc/certs/cert.pem
```

**Service discovery fails:**
- Check mesh control plane is accessible
- Verify namespace/datacenter settings
- Ensure authentication tokens are valid

**Headers not injected:**
- Enable debug logging: `logging.level: debug`
- Verify middleware order in gateway logs

### Performance Tips

1. **Adjust refresh interval:**
   ```yaml
   refreshInterval: 60s  # Reduce discovery overhead
   ```

2. **Use connection pooling:**
   ```yaml
   server:
     maxConnections: 1000
   ```

3. **Monitor metrics:**
   - Service discovery latency
   - mTLS handshake time
   - Certificate expiration

### Documentation

- **Full Guide**: [docs/service-mesh-integration.md](./service-mesh-integration.md)
- **Examples**: [config/servicemesh.example.yaml](../config/servicemesh.example.yaml)
- **Implementation**: [docs/IMPLEMENTATION_SUMMARY.md](./IMPLEMENTATION_SUMMARY.md)

### Build Verification

```bash
# Build all packages
go build ./...

# Build gateway binary
go build -o bin/gateway ./cmd/gateway

# Build odin binary
go build -o bin/odin ./cmd/odin
```

All builds: ✅ **SUCCESS**

---

**Status**: Production-ready  
**Code**: 821 lines across 7 files  
**Tests**: Ready for integration tests  
**Documentation**: Complete with examples  
**Last Updated**: 2024
