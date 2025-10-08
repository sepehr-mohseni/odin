# GraphQL Proxy Support

Odin API Gateway now supports GraphQL proxying with advanced features for query validation, caching, and security.

## Features

- **Query Validation**: Depth and complexity limits to prevent expensive queries
- **Introspection Control**: Enable/disable GraphQL introspection for security
- **Query Caching**: Cache GraphQL responses for improved performance
- **Error Handling**: Proper GraphQL error formatting and HTTP status mapping

## Configuration

Add a GraphQL service to your `config.yaml`:

```yaml
services:
  - name: graphql-api
    basePath: /graphql
    protocol: graphql
    targets:
      - http://localhost:4000/graphql
    authentication: false
    timeout: 30s
    graphql:
      maxQueryDepth: 10           # Maximum query nesting depth
      maxQueryComplexity: 1000    # Maximum query complexity score
      enableIntrospection: true   # Allow introspection queries
      enableQueryCaching: true    # Cache GraphQL responses
      cacheTTL: 5m               # Cache time-to-live
```

## Query Validation

The gateway validates incoming GraphQL queries for:

- **Depth Limits**: Prevents deeply nested queries that could cause performance issues
- **Complexity Limits**: Basic complexity scoring to prevent expensive operations
- **Introspection Checks**: Blocks introspection queries if disabled

## Usage

Send GraphQL queries via POST to the configured endpoint:

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "{ users { id name email } }",
    "variables": {},
    "operationName": null
  }'
```

## Security Considerations

- **Disable introspection** in production environments
- **Set appropriate depth/complexity limits** based on your schema
- **Use authentication** for sensitive GraphQL endpoints
- **Monitor query patterns** to detect abuse

## Error Handling

The proxy returns standard GraphQL error responses:

```json
{
  "errors": [
    {
      "message": "Query depth 15 exceeds maximum allowed depth 10",
      "locations": [],
      "path": [],
      "extensions": {}
    }
  ]
}
```