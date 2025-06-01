package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"odin/pkg/health"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	e := echo.New()
	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "UP")
}

func TestDebugRoutesEndpoint(t *testing.T) {
	e := echo.New()
	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/debug/routes", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "method")
	assert.Contains(t, rec.Body.String(), "path")
}

func TestDebugConfigEndpoint(t *testing.T) {
	e := echo.New()
	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/debug/config", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Config not available")
}

func TestDebugContentTypesEndpoint(t *testing.T) {
	e := echo.New()
	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/debug/content-types", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "message")
	assert.Contains(t, rec.Body.String(), "timestamp")
}

func TestHealthCheck(t *testing.T) {
	e := echo.New()
	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "status")
	assert.Contains(t, rec.Body.String(), "UP")
}

func TestHealthCheckWithDependencies(t *testing.T) {
	// Test would include checking external dependencies
	// For now, basic health check test
	e := echo.New()
	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestReadinessCheck(t *testing.T) {
	e := echo.New()
	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Readiness check should return OK when services are ready
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ready")
}

func TestLivenessCheck(t *testing.T) {
	e := echo.New()
	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/live", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Liveness check should always return OK if the service is running
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "alive")
}
