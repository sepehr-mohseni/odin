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

	// Test metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "# HELP")
}

func TestMetricsMiddleware(t *testing.T) {
	middleware := monitoring.MetricsMiddleware

	e := echo.New()

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "test response")
	}

	err := middleware(next)(c)
	assert.NoError(t, err)
	assert.True(t, nextCalled)
}
