package main

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

// RequestTransformer modifies requests before they reach backend services
type RequestTransformer struct {
	addHeaders     map[string]string
	removeHeaders  []string
	replaceHeaders map[string]string
}

// Export the middleware instance
var Middleware RequestTransformer

// Name returns the middleware name
func (m *RequestTransformer) Name() string {
	return "request-transformer"
}

// Version returns the middleware version
func (m *RequestTransformer) Version() string {
	return "1.0.0"
}

// Initialize sets up the middleware
func (m *RequestTransformer) Initialize(config map[string]interface{}) error {
	m.addHeaders = make(map[string]string)
	m.removeHeaders = []string{}
	m.replaceHeaders = make(map[string]string)

	// Parse addHeaders
	if addHeaders, ok := config["addHeaders"].(map[string]interface{}); ok {
		for key, value := range addHeaders {
			if valueStr, ok := value.(string); ok {
				m.addHeaders[key] = valueStr
			}
		}
	}

	// Parse removeHeaders
	if removeHeaders, ok := config["removeHeaders"].([]interface{}); ok {
		for _, header := range removeHeaders {
			if headerStr, ok := header.(string); ok {
				m.removeHeaders = append(m.removeHeaders, headerStr)
			}
		}
	}

	// Parse replaceHeaders
	if replaceHeaders, ok := config["replaceHeaders"].(map[string]interface{}); ok {
		for key, value := range replaceHeaders {
			if valueStr, ok := value.(string); ok {
				m.replaceHeaders[key] = valueStr
			}
		}
	}

	return nil
}

// Handle transforms the request
func (m *RequestTransformer) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()

		// Add headers
		for key, value := range m.addHeaders {
			// Support variable replacement
			finalValue := m.interpolateValue(value, c)
			req.Header.Add(key, finalValue)
		}

		// Remove headers
		for _, key := range m.removeHeaders {
			req.Header.Del(key)
		}

		// Replace headers
		for key, value := range m.replaceHeaders {
			finalValue := m.interpolateValue(value, c)
			req.Header.Set(key, finalValue)
		}

		return next(c)
	}
}

// interpolateValue replaces variables in configuration values
func (m *RequestTransformer) interpolateValue(value string, c echo.Context) string {
	// Simple variable replacement (can be extended)
	if value == "${timestamp}" {
		return fmt.Sprintf("%d", c.Request().Context().Value("timestamp"))
	}
	if value == "${request_id}" {
		return c.Response().Header().Get(echo.HeaderXRequestID)
	}
	return value
}

// Cleanup releases resources
func (m *RequestTransformer) Cleanup() error {
	return nil
}

// Required for Go plugins
func main() {}
