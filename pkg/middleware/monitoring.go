package middleware

import (
	"time"

	"odin/pkg/admin"

	"github.com/labstack/echo/v4"
)

// MonitoringMiddleware creates middleware that records metrics for the monitoring dashboard
func MonitoringMiddleware() echo.MiddlewareFunc {
	collector := admin.GetCollector()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip monitoring for admin routes to avoid recursive calls
			requestPath := c.Request().URL.Path
			if requestPath == "/admin/ws/monitoring" || requestPath == "/admin/api/monitoring/metrics" {
				return next(c)
			}

			start := time.Now()

			// Execute the next handler
			err := next(c)

			// Record the metrics
			duration := time.Since(start)
			method := c.Request().Method
			path := requestPath
			status := c.Response().Status

			// Determine service name based on path
			service := "gateway"
			if path != "" {
				// Extract service name from path (e.g., /api/users -> users-service)
				// This is a simple implementation, you might want to make it more sophisticated
				parts := splitPath(path)
				if len(parts) > 2 && parts[1] == "api" {
					service = parts[2] + "-service"
				}
			}

			collector.RecordRequest(method, path, duration, status, service)

			return err
		}
	}
}

// Helper function to split path
func splitPath(path string) []string {
	if path == "" || path == "/" {
		return []string{}
	}

	// Remove leading slash and split
	if path[0] == '/' {
		path = path[1:]
	}

	parts := make([]string, 0)
	current := ""

	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}
