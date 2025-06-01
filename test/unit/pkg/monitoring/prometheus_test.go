package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"odin/pkg/monitoring"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	e := echo.New()

	monitoring.Register(e, "/metrics")

	// Test that metrics endpoint is registered

	// Find the route
	found := false
	for _, route := range e.Routes() {
		if route.Path == "/metrics" && route.Method == "GET" {
			found = true
			break
		}
	}

	assert.True(t, found, "Metrics endpoint should be registered")
}

func TestMetricsMiddleware(t *testing.T) {
	e := echo.New()

	// Add the metrics middleware
	e.Use(monitoring.MetricsMiddleware)

	// Create a test handler
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "test"})
	})

	// Make a request
	req := httptest.NewRequest("GET", "/api/users/test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// The middleware should process the request without errors
	assert.Equal(t, http.StatusNotFound, rec.Code) // 404 because /api/users/test doesn't exist
}
