package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"odin/pkg/health"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	e := echo.New()

	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "UP", response["status"])
}

func TestDebugRoutesEndpoint(t *testing.T) {
	e := echo.New()

	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/debug/routes", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var routes []map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &routes)
	require.NoError(t, err)

	assert.NotEmpty(t, routes)
}

func TestDebugConfigEndpoint(t *testing.T) {
	e := echo.New()

	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/debug/config", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDebugContentTypesEndpoint(t *testing.T) {
	e := echo.New()

	health.Register(e, logrus.New())

	req := httptest.NewRequest("GET", "/debug/content-types", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHealthCheck(t *testing.T) {
	checker := health.NewChecker()

	status := checker.Check()

	assert.Equal(t, "healthy", status.Status)
	assert.NotZero(t, status.Timestamp)
}

func TestHealthCheckWithDependencies(t *testing.T) {
	checker := health.NewChecker()

	// Add a dependency check
	checker.AddCheck("test", func() error {
		return nil
	})

	status := checker.Check()

	assert.Equal(t, "healthy", status.Status)
	assert.Contains(t, status.Checks, "test")
	assert.Equal(t, "healthy", status.Checks["test"].Status)
}

func TestReadinessCheck(t *testing.T) {
	checker := health.NewChecker()

	status := checker.Readiness()

	assert.Equal(t, "ready", status.Status)
}

func TestLivenessCheck(t *testing.T) {
	checker := health.NewChecker()

	status := checker.Liveness()

	assert.Equal(t, "alive", status.Status)
}
