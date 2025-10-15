# Canary Deployments

Odin supports canary deployments, allowing you to gradually roll out new versions of your services with configurable traffic splitting.

## Features

- **Weight-based routing**: Route a percentage of traffic to canary versions
- **Header-based routing**: Route traffic based on custom headers
- **Cookie-based routing**: Route traffic based on cookies
- **Sticky sessions**: Consistent routing per client IP using MD5 hashing

## Configuration

Add a `canary` section to your service configuration:

```yaml
services:
  - name: my-service
    basePath: /api/service
    targets:
      - http://prod-service:8080
    canary:
      enabled: true
      targets:
        - http://canary-service:8080
      weight: 10  # Route 10% of traffic to canary
      # Optional: Header-based routing
      header: X-Beta-User
      headerValue: "true"
      # Optional: Cookie-based routing
      cookieName: beta_user
      cookieValue: "1"
```

## Routing Strategies

### Weight-based Routing

Routes a percentage of traffic to canary based on the `weight` field (0-100):

```yaml
canary:
  enabled: true
  targets:
    - http://canary-service:8080
  weight: 20  # 20% of traffic goes to canary
```

The routing is sticky per client IP address, ensuring consistent user experience.

### Header-based Routing

Route specific users to canary based on HTTP headers:

```yaml
canary:
  enabled: true
  targets:
    - http://canary-service:8080
  header: X-Beta-User
  headerValue: "true"
```

Users with the header `X-Beta-User: true` will always hit the canary version.

### Cookie-based Routing

Route users to canary based on cookies:

```yaml
canary:
  enabled: true
  targets:
    - http://canary-service:8080
  cookieName: beta_user
  cookieValue: "1"
```

## Testing Canary Deployments

### Test weight-based routing:

```bash
# Generate multiple requests to see distribution
for i in {1..10}; do
  curl http://localhost:8080/api/service
done
```

### Test header-based routing:

```bash
# This will hit the canary version
curl -H "X-Beta-User: true" http://localhost:8080/api/service

# This will hit the production version
curl http://localhost:8080/api/service
```

### Test cookie-based routing:

```bash
# This will hit the canary version
curl -b "beta_user=1" http://localhost:8080/api/service
```

## Best Practices

1. **Start Small**: Begin with a low weight (5-10%) and gradually increase
2. **Monitor Metrics**: Watch error rates and performance in the monitoring dashboard
3. **Use Headers for Internal Testing**: Give your team access to canary via headers
4. **Gradual Rollout**: 
   - Week 1: 10% traffic to canary
   - Week 2: 25% traffic to canary
   - Week 3: 50% traffic to canary
   - Week 4: 100% traffic, promote canary to production
5. **Rollback Strategy**: Keep production version running until canary is proven stable

## Monitoring Canary Deployments

The monitoring dashboard shows which target (production or canary) handled each request. Look for:

- Error rate differences between production and canary
- Response time differences
- Success rate variations

If canary shows issues, simply set `enabled: false` or reduce the weight.
