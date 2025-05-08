# Request & Response Transformation

This document explains how to use Odin's transformation capabilities to modify API requests and responses.

## Overview

Odin API Gateway allows you to transform both requests and responses as they pass through the gateway, enabling you to:

- Rename, restructure, or filter fields
- Add default values
- Convert between different data formats
- Normalize API responses from different services

## Basic Configuration

Transformations are defined in your service configuration using JSONPath-like syntax:

```yaml
services:
  - name: users-service
    basePath: /api/users
    targets:
      - http://users-service:8081
    transform:
      request: # Transforms applied to incoming requests
        - from: $.user.id
          to: $.userId
          default: 'anonymous'
      response: # Transforms applied to service responses
        - from: $.data
          to: $.users
          default: []
```

## Request Transformation

Request transformations are applied before the request is sent to the backend service.

### Examples

#### Basic Field Mapping

```yaml
transform:
  request:
    - from: $.username
      to: $.user.name
```

This moves the value from `{"username": "john"}` to `{"user": {"name": "john"}}`.

#### Adding Default Values

```yaml
transform:
  request:
    - from: $.includeInactive
      to: $.filters.inactive
      default: false
```

If `includeInactive` is not in the original request, `filters.inactive` will be set to `false`.

#### Multiple Transformations

```yaml
transform:
  request:
    - from: $.name
      to: $.user.name
    - from: $.email
      to: $.user.email
    - from: $.filters
      to: $.queryOptions.filters
```

Transformations are applied in order, so later transformations can use the results of earlier ones.

## Response Transformation

Response transformations are applied after receiving the response from the backend service.

### Examples

#### Renaming Fields

```yaml
transform:
  response:
    - from: $.usersList
      to: $.users
```

This changes `{"usersList": [...]}` to `{"users": [...]}`.

#### Restructuring Data

```yaml
transform:
  response:
    - from: $.data.items
      to: $.results
    - from: $.data.pagination
      to: $.meta.pagination
```

This transforms:

```json
{
  "data": {
    "items": [...],
    "pagination": { "page": 1 }
  }
}
```

Into:

```json
{
  "results": [...],
  "meta": {
    "pagination": { "page": 1 }
  }
}
```

#### Working with Arrays

```yaml
transform:
  response:
    - from: $.data[*].id
      to: $.ids
```

This extracts all IDs from an array of objects into a separate array.

## Advanced Transformations

### Array Manipulation

```yaml
transform:
  response:
    - from: $.data[?(@.active==true)]
      to: $.activeUsers
```

This filters the array to include only objects where `active` is `true`.

### Conditional Transformations

```yaml
transform:
  response:
    - from: $.error
      to: $.apiError
      condition: $.statusCode >= 400
```

This transformation only applies when the status code is 400 or higher.

## Testing Transformations

You can test your transformations using the `/debug/transform` endpoint:

```bash
curl -X POST http://localhost:8080/debug/transform \
  -H "Content-Type: application/json" \
  -d '{
    "input": {"username": "john", "age": 30},
    "transforms": [
      {"from": "$.username", "to": "$.user.name"},
      {"from": "$.age", "to": "$.user.age"}
    ]
  }'
```

## Error Handling in Transformations

If a source path isn't found:

1. If a default value is specified, it will be used
2. If no default is specified, the field is skipped
3. If `failOnMissingField: true` is set, an error is returned

## Performance Considerations

Transformations add some processing overhead. To minimize this:

1. Only transform what you need
2. Use simpler paths when possible
3. Consider caching transformed responses
4. For high-performance APIs, apply transformations selectively

## Limitations

1. Transformations work only with JSON data
2. Complex logic beyond path-based mapping requires custom middleware
3. Binary data cannot be transformed
