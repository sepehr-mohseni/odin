package main

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// APIKeyAuth validates API keys from request headers
type APIKeyAuth struct {
	validKeys    map[string]bool
	headerName   string
	errorMessage string
}

// Export the middleware instance
var Middleware APIKeyAuth

// Name returns the middleware name
func (m *APIKeyAuth) Name() string {
	return "api-key-auth"
}

// Version returns the middleware version
func (m *APIKeyAuth) Version() string {
	return "1.0.0"
}

// Initialize sets up the middleware
func (m *APIKeyAuth) Initialize(config map[string]interface{}) error {
	// Get API keys list
	keys, ok := config["keys"].([]interface{})
	if !ok || len(keys) == 0 {
		return fmt.Errorf("'keys' configuration is required and must be a non-empty array")
	}

	// Build hash map for fast lookup
	m.validKeys = make(map[string]bool)
	for _, key := range keys {
		if keyStr, ok := key.(string); ok {
			m.validKeys[keyStr] = true
		}
	}

	if len(m.validKeys) == 0 {
		return fmt.Errorf("no valid API keys configured")
	}

	// Get header name (default: X-API-Key)
	if headerName, ok := config["headerName"].(string); ok {
		m.headerName = headerName
	} else {
		m.headerName = "X-API-Key"
	}

	// Get custom error message
	if errMsg, ok := config["errorMessage"].(string); ok {
		m.errorMessage = errMsg
	} else {
		m.errorMessage = "Invalid or missing API key"
	}

	return nil
}

// Handle validates the API key
func (m *APIKeyAuth) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get API key from header
		apiKey := c.Request().Header.Get(m.headerName)

		// Validate key
		if apiKey == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, m.errorMessage)
		}

		if !m.validKeys[apiKey] {
			return echo.NewHTTPError(http.StatusUnauthorized, m.errorMessage)
		}

		// Key is valid, set context value for downstream middleware
		c.Set("apiKey", apiKey)
		c.Set("authenticated", true)

		return next(c)
	}
}

// Cleanup releases resources
func (m *APIKeyAuth) Cleanup() error {
	return nil
}

// Required for Go plugins
func main() {}
