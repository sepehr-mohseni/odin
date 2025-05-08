# Response Aggregation

This guide explains how to use Odin's response aggregation features to combine data from multiple backend services.

## Overview

Response aggregation allows you to:

1. Call multiple backend services in a single client request
2. Enrich API responses with related data from other services
3. Return consolidated data to clients, reducing the number of API calls needed

This is particularly useful for:

- Mobile applications with limited bandwidth
- Microservices architectures where related data is split across services
- Dashboards that need data from multiple sources

## Basic Configuration

Response aggregation is configured in your service definitions:

```yaml
services:
  - name: orders
    basePath: /api/orders
    targets:
      - http://orders-service:8084
    aggregation:
      dependencies:
        - service: users
          path: /api/users/{user_id}
          parameterMapping:
            - from: $.userId
              to: user_id
          resultMapping:
            - from: $
              to: $.user
```

## Configuration Components

### Dependencies

Each dependency specifies:

1. Which service to call for additional data
2. What path to call on that service
3. How to extract parameters for the path
4. How to map the response into the original response

### Parameter Mapping

Parameter mappings extract values from the original service response to use in the dependent service path:

```yaml
parameterMapping:
  - from: $.userId # Source field in the main response
    to: user_id # Target parameter in the dependency path
```

In this example, the value from `userId` in the main response replaces `{user_id}` in the dependency path.

### Result Mapping

Result mappings determine where to place the dependency response in the main response:

```yaml
resultMapping:
  - from: $ # Source field in the dependency response
    to: $.user # Target location in the main response
```

## Advanced Aggregation Examples

### Multiple Dependencies

```yaml
aggregation:
  dependencies:
    - service: users
      path: /api/users/{user_id}
      parameterMapping:
        - from: $.userId
          to: user_id
      resultMapping:
        - from: $
          to: $.user
    - service: products
      path: /api/products/{product_id}
      parameterMapping:
        - from: $.productId
          to: product_id
      resultMapping:
        - from: $
          to: $.product
```

### Array Item Enrichment

You can enrich items in an array with related data:

```yaml
aggregation:
  dependencies:
    - service: products
      path: /api/products/{product_id}
      parameterMapping:
        - from: $.items[*].productId
          to: product_id
      resultMapping:
        - from: $
          to: $.items[*].product
```

This will call the products service for each item in the array and add the product details to each item.

## Conditional Aggregation

You can conditionally perform aggregation:

```yaml
aggregation:
  dependencies:
    - service: details
      path: /api/details/{id}
      condition: $.includeDetails == true
      parameterMapping:
        - from: $.id
          to: id
      resultMapping:
        - from: $
          to: $.details
```

This will only call the details service if the main response has `includeDetails: true`.

## Error Handling

By default, if a dependency call fails:

1. The error is logged
2. The aggregation for that dependency is skipped
3. The original response is returned without the additional data

You can configure stricter behavior:

```yaml
aggregation:
  failOnError: true # Fail the entire request if any dependency fails
  dependencies:
    # ...
```

## Performance Considerations

Aggregation involves additional HTTP requests, which can impact performance:

1. **Parallel Processing**: Odin calls dependencies in parallel when possible
2. **Caching**: Consider enabling caching for frequently accessed dependency data
3. **Timeouts**: Configure appropriate timeouts for dependencies
4. **Selective Aggregation**: Use query parameters to enable/disable aggregation based on client needs

## Debugging Aggregation

Enable debug logging to see detailed information about aggregation:

```yaml
logging:
  level: debug
```

This will show:

- Parameters extracted from the main response
- URLs called for dependencies
- Timing information
- Any errors encountered

You can also use the `/debug/aggregation` endpoint to test aggregation configurations:

```bash
curl -X POST http://localhost:8080/debug/aggregation \
  -d '{
    "service": "orders",
    "response": {"userId": "user123", "orderId": "order456"}
  }'
```
