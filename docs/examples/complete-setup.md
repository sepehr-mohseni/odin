# Complete Setup Examples

This document provides comprehensive examples for setting up Odin API Gateway in various scenarios.

## Basic Setup

### 1. Simple API Gateway

```yaml
# config/config.yaml
gateway:
  port: 8080
  cors:
    enabled: true
    allowOrigins: ['*']

auth:
  jwtSecret: 'your-super-secret-key-here'
  accessTokenTTL: 1h

services:
  - name: users-api
    basePath: /api/users
    targets:
      - http://users-service:8081
    loadBalancer:
      strategy: round_robin
    authentication: true
```

### 2. With Circuit Breaker

```yaml
# config/config.yaml
circuitBreaker:
  enabled: true
  maxRequests: 10
  interval: 60s
  timeout: 60s
  failureRatio: 0.5
  minRequests: 5

services:
  - name: payment-api
    basePath: /api/payments
    targets:
      - http://payment-service:8082
    circuitBreaker:
      enabled: true
      maxRequests: 5
      failureRatio: 0.3
```

### 3. OAuth2 Integration

```yaml
# config/config.yaml
auth:
  oauth2:
    enabled: true
    providers:
      google:
        clientId: 'your-google-client-id'
        clientSecret: 'your-google-client-secret'
        authUrl: 'https://accounts.google.com/o/oauth2/v2/auth'
        tokenUrl: 'https://oauth2.googleapis.com/token'
        userInfoUrl: 'https://www.googleapis.com/oauth2/v1/userinfo'
        scopes: ['openid', 'profile', 'email']
        redirectUrl: 'http://localhost:8080/auth/google/callback'

      github:
        clientId: 'your-github-client-id'
        clientSecret: 'your-github-client-secret'
        authUrl: 'https://github.com/login/oauth/authorize'
        tokenUrl: 'https://github.com/login/oauth/access_token'
        userInfoUrl: 'https://api.github.com/user'
        scopes: ['user:email']
        redirectUrl: 'http://localhost:8080/auth/github/callback'
```

## Advanced Configurations

### 4. WebSocket Proxying

```yaml
# config/config.yaml
websocket:
  enabled: true
  readBufferSize: 4096
  writeBufferSize: 4096
  handshakeTimeout: 10s
  readTimeout: 60s
  writeTimeout: 10s
  maxMessageSize: 524288

services:
  - name: chat-api
    basePath: /api/chat
    targets:
      - ws://chat-service:8083
    websocket: true
    authentication: true
```

### 5. Response Aggregation

```yaml
# config/config.yaml
services:
  - name: user-profile-aggregate
    basePath: /api/profile
    aggregation:
      enabled: true
      requests:
        - name: user
          url: '/api/users/{user_id}'
          method: GET
        - name: preferences
          url: '/api/preferences/{user_id}'
          method: GET
        - name: activity
          url: '/api/activity/{user_id}'
          method: GET
      responseMapping:
        user: '$.user'
        preferences: '$.preferences'
        recentActivity: '$.activity.recent[0:5]'
```

### 6. Request Transformation

```yaml
# config/config.yaml
services:
  - name: legacy-api
    basePath: /api/v2/orders
    targets:
      - http://legacy-service:8084
    transformation:
      request:
        headers:
          add:
            X-API-Version: 'v1'
          remove:
            - Authorization
        body:
          mapping:
            customerId: '$.customer.id'
            items: '$.orderItems[*].{productId: product_id, quantity: qty}'
      response:
        headers:
          add:
            X-Response-Version: 'v2'
        body:
          mapping:
            orderId: '$.id'
            customer: '$.customer_details'
            status: '$.order_status'
```

## Production Setup Examples

### 7. High Availability with Load Balancing

```yaml
# config/config.yaml
gateway:
  port: 8080
  readTimeout: 30s
  writeTimeout: 30s
  gracefulShutdownTimeout: 30s

services:
  - name: api-cluster
    basePath: /api
    targets:
      - http://api-server-1:8081
      - http://api-server-2:8081
      - http://api-server-3:8081
    loadBalancer:
      strategy: weighted_round_robin
      weights:
        - 30
        - 35
        - 35
    healthCheck:
      enabled: true
      path: /health
      interval: 30s
      timeout: 5s
      unhealthyThreshold: 3
      healthyThreshold: 2
```

### 8. Complete Microservices Setup

```yaml
# config/config.yaml
gateway:
  port: 8080
  metrics:
    enabled: true
    path: /metrics

auth:
  jwtSecret: '${JWT_SECRET}'
  accessTokenTTL: 15m
  refreshTokenTTL: 24h
  oauth2:
    enabled: true
    providers:
      internal:
        clientId: '${OAUTH2_CLIENT_ID}'
        clientSecret: '${OAUTH2_CLIENT_SECRET}'

circuitBreaker:
  enabled: true
  defaultConfig:
    maxRequests: 10
    interval: 60s
    timeout: 60s
    failureRatio: 0.5

websocket:
  enabled: true
  maxMessageSize: 1048576 # 1MB

cache:
  enabled: true
  redis:
    address: 'redis:6379'
    db: 0

rateLimit:
  enabled: true
  redis:
    address: 'redis:6379'
  rules:
    - path: '/api/public/*'
      limit: 100
      window: 1m
    - path: '/api/users/*'
      limit: 1000
      window: 1h
      authenticated: true

services:
  - name: auth-service
    basePath: /api/auth
    targets:
      - http://auth-service:8081
    rateLimit:
      limit: 50
      window: 1m

  - name: users-service
    basePath: /api/users
    targets:
      - http://users-service-1:8082
      - http://users-service-2:8082
    authentication: true
    circuitBreaker:
      enabled: true
    cache:
      enabled: true
      ttl: 5m
      keyPattern: 'users:{path}'

  - name: orders-service
    basePath: /api/orders
    targets:
      - http://orders-service:8083
    authentication: true
    circuitBreaker:
      enabled: true
      maxRequests: 5
    transformation:
      response:
        headers:
          add:
            X-Service-Version: '1.0'

  - name: notifications-ws
    basePath: /ws/notifications
    targets:
      - ws://notifications-service:8084
    websocket: true
    authentication: true

  - name: file-upload
    basePath: /api/files
    targets:
      - http://file-service:8085
    authentication: true
    timeout: 5m
    maxRequestSize: 100MB
```

### 9. Docker Compose Production Setup

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  odin:
    image: odin-gateway:latest
    ports:
      - '80:8080'
      - '8081:8081'
    environment:
      - JWT_SECRET=${JWT_SECRET}
      - OAUTH2_CLIENT_ID=${OAUTH2_CLIENT_ID}
      - OAUTH2_CLIENT_SECRET=${OAUTH2_CLIENT_SECRET}
      - REDIS_URL=redis:6379
      - LOG_LEVEL=info
    volumes:
      - ./config:/app/config:ro
    depends_on:
      - redis
      - prometheus
    networks:
      - odin-network
    deploy:
      replicas: 3
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M
    healthcheck:
      test: ['CMD', 'wget', '-q', '-O-', 'http://localhost:8080/health']
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    networks:
      - odin-network
    deploy:
      resources:
        limits:
          memory: 256M

  prometheus:
    image: prom/prometheus:latest
    ports:
      - '9090:9090'
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - odin-network

networks:
  odin-network:
    driver: overlay

volumes:
  redis-data:
  prometheus-data:
```

### 10. Kubernetes Production Deployment

```yaml
# k8s-production.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: odin-system

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: odin-config
  namespace: odin-system
data:
  config.yaml: |
    gateway:
      port: 8080
      metrics:
        enabled: true
    auth:
      jwtSecret: "${JWT_SECRET}"
    circuitBreaker:
      enabled: true
    websocket:
      enabled: true
    # ... rest of configuration

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: odin-gateway
  namespace: odin-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: odin-gateway
  template:
    metadata:
      labels:
        app: odin-gateway
    spec:
      containers:
        - name: odin
          image: odin-gateway:latest
          ports:
            - containerPort: 8080
            - containerPort: 8081
          env:
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: odin-secrets
                  key: jwt-secret
          resources:
            requests:
              memory: '256Mi'
              cpu: '250m'
            limits:
              memory: '512Mi'
              cpu: '500m'
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: odin-gateway-service
  namespace: odin-system
spec:
  selector:
    app: odin-gateway
  ports:
    - name: http
      port: 80
      targetPort: 8080
    - name: admin
      port: 8081
      targetPort: 8081
  type: LoadBalancer
```

## Testing Examples

### 11. Authentication Testing

```bash
# Get OAuth2 authorization URL
curl "http://localhost:8080/auth/google/login"

# Test JWT authentication
TOKEN=$(curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"user","password":"pass"}' | jq -r .token)

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/users/profile
```

### 12. Circuit Breaker Testing

```bash
# Test circuit breaker with load
for i in {1..20}; do
  curl -w "%{http_code}\n" -o /dev/null -s \
    http://localhost:8080/api/unstable-service/test
  sleep 0.1
done

# Check circuit breaker status
curl http://localhost:8081/admin/circuit-breakers
```

### 13. WebSocket Testing

```javascript
// WebSocket client test
const ws = new WebSocket('ws://localhost:8080/ws/notifications', [], {
  headers: {
    Authorization: 'Bearer ' + token,
  },
});

ws.on('open', () => {
  console.log('Connected to WebSocket');
  ws.send(JSON.stringify({ type: 'subscribe', channel: 'user-123' }));
});

ws.on('message', (data) => {
  console.log('Received:', JSON.parse(data));
});
```

These examples demonstrate the comprehensive capabilities of Odin API Gateway and provide practical starting points for various deployment scenarios.
