package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLLocation      `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLLocation represents a location in a GraphQL query
type GraphQLLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// ProxyConfig holds GraphQL proxy configuration
type ProxyConfig struct {
	Endpoint            string        `yaml:"endpoint"`
	MaxQueryDepth       int           `yaml:"maxQueryDepth"`
	MaxQueryComplexity  int           `yaml:"maxQueryComplexity"`
	EnableIntrospection bool          `yaml:"enableIntrospection"`
	Timeout             time.Duration `yaml:"timeout"`
	EnableQueryCaching  bool          `yaml:"enableQueryCaching"`
	CacheTTL            time.Duration `yaml:"cacheTTL"`
}

// Proxy handles GraphQL requests and forwards them to backend services
type Proxy struct {
	config *ProxyConfig
	logger *logrus.Logger
	client *http.Client
}

// NewProxy creates a new GraphQL proxy
func NewProxy(config *ProxyConfig, logger *logrus.Logger) *Proxy {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxQueryDepth == 0 {
		config.MaxQueryDepth = 10
	}
	if config.MaxQueryComplexity == 0 {
		config.MaxQueryComplexity = 1000
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	return &Proxy{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Handle processes GraphQL requests
func (p *Proxy) Handle(c echo.Context) error {
	// Only handle POST requests for GraphQL
	if c.Request().Method != http.MethodPost {
		return c.JSON(http.StatusMethodNotAllowed, map[string]string{
			"error": "GraphQL endpoint only accepts POST requests",
		})
	}

	// Parse GraphQL request
	var req GraphQLRequest
	if err := c.Bind(&req); err != nil {
		p.logger.WithError(err).Error("Failed to parse GraphQL request")
		return c.JSON(http.StatusBadRequest, GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Invalid GraphQL request format",
			}},
		})
	}

	// Validate query
	if req.Query == "" {
		return c.JSON(http.StatusBadRequest, GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Query is required",
			}},
		})
	}

	// Check if introspection query and if it's allowed
	if !p.config.EnableIntrospection && p.isIntrospectionQuery(req.Query) {
		return c.JSON(http.StatusForbidden, GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Introspection is disabled",
			}},
		})
	}

	// Validate query complexity and depth
	if err := p.validateQuery(req.Query); err != nil {
		return c.JSON(http.StatusBadRequest, GraphQLResponse{
			Errors: []GraphQLError{{
				Message: err.Error(),
			}},
		})
	}

	// Forward request to backend
	resp, err := p.forwardRequest(c.Request().Context(), &req)
	if err != nil {
		p.logger.WithError(err).Error("Failed to forward GraphQL request")
		return c.JSON(http.StatusInternalServerError, GraphQLResponse{
			Errors: []GraphQLError{{
				Message: "Internal server error",
			}},
		})
	}

	// Return the response
	return c.JSON(http.StatusOK, resp)
}

// forwardRequest forwards the GraphQL request to the backend service
func (p *Proxy) forwardRequest(ctx context.Context, req *GraphQLRequest) (*GraphQLResponse, error) {
	// Serialize request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Execute request
	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse GraphQL response
	var resp GraphQLResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	return &resp, nil
}

// isIntrospectionQuery checks if the query is an introspection query
func (p *Proxy) isIntrospectionQuery(query string) bool {
	// Simple check for common introspection fields
	introspectionFields := []string{
		"__schema",
		"__type",
		"__typename",
		"__Field",
		"__Directive",
		"__EnumValue",
		"__InputValue",
	}

	queryLower := strings.ToLower(query)
	for _, field := range introspectionFields {
		if strings.Contains(queryLower, strings.ToLower(field)) {
			return true
		}
	}
	return false
}

// validateQuery performs basic validation on the GraphQL query
func (p *Proxy) validateQuery(query string) error {
	// Simple depth check by counting nested braces
	depth := 0
	maxDepth := 0

	for _, char := range query {
		switch char {
		case '{':
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		case '}':
			depth--
		}
	}

	if maxDepth > p.config.MaxQueryDepth {
		return fmt.Errorf("query depth %d exceeds maximum allowed depth %d", maxDepth, p.config.MaxQueryDepth)
	}

	// Simple complexity check by counting fields (approximate)
	fieldCount := strings.Count(query, "{") + strings.Count(query, "}")
	if fieldCount > p.config.MaxQueryComplexity {
		return fmt.Errorf("query complexity %d exceeds maximum allowed complexity %d", fieldCount, p.config.MaxQueryComplexity)
	}

	return nil
}

// RegisterRoutes registers GraphQL proxy routes
func (p *Proxy) RegisterRoutes(e *echo.Echo, basePath string) {
	e.POST(basePath, p.Handle)
	e.GET(basePath, func(c echo.Context) error {
		// GraphQL Playground or similar can be served here
		return c.JSON(http.StatusOK, map[string]string{
			"message": "GraphQL endpoint is available at POST " + basePath,
		})
	})
}
