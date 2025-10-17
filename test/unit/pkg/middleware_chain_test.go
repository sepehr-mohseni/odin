package pkg

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"odin/pkg/plugins"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMiddleware implements the Middleware interface for testing
type MockMiddleware struct {
	name            string
	version         string
	config          map[string]interface{}
	executionCount  int
	shouldFail      bool
	executionDelay  time.Duration
	beforeExecution func()
	afterExecution  func()
}

func NewMockMiddleware(name, version string) *MockMiddleware {
	return &MockMiddleware{
		name:    name,
		version: version,
	}
}

func (m *MockMiddleware) Name() string {
	return m.name
}

func (m *MockMiddleware) Version() string {
	return m.version
}

func (m *MockMiddleware) Initialize(config map[string]interface{}) error {
	m.config = config
	return nil
}

func (m *MockMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if m.beforeExecution != nil {
			m.beforeExecution()
		}

		m.executionCount++

		if m.executionDelay > 0 {
			time.Sleep(m.executionDelay)
		}

		if m.shouldFail {
			if m.afterExecution != nil {
				m.afterExecution()
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Mock middleware error")
		}

		// Add header to track execution
		c.Response().Header().Set("X-"+m.name, "executed")

		err := next(c)

		if m.afterExecution != nil {
			m.afterExecution()
		}

		return err
	}
}

func (m *MockMiddleware) Cleanup() error {
	return nil
}

func (m *MockMiddleware) GetExecutionCount() int {
	return m.executionCount
}

func (m *MockMiddleware) SetShouldFail(fail bool) {
	m.shouldFail = fail
}

func (m *MockMiddleware) SetExecutionDelay(delay time.Duration) {
	m.executionDelay = delay
}

// TestMiddlewareChainOrdering tests that middleware executes in priority order
func TestMiddlewareChainOrdering(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs

	manager := plugins.NewPluginManager(logger)

	// Create middlewares with different priorities
	middleware1 := NewMockMiddleware("mw1", "1.0.0")
	middleware2 := NewMockMiddleware("mw2", "1.0.0")
	middleware3 := NewMockMiddleware("mw3", "1.0.0")

	// Register with different priorities (lower = earlier execution)
	err := manager.RegisterMiddleware("mw1", middleware1, 100, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	err = manager.RegisterMiddleware("mw2", middleware2, 50, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	err = manager.RegisterMiddleware("mw3", middleware3, 75, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	// Get chain and verify ordering
	chain := manager.GetMiddlewareChain()
	require.Len(t, chain, 3)

	// Verify order: mw2 (50) < mw3 (75) < mw1 (100)
	assert.Equal(t, "mw2", chain[0].Name)
	assert.Equal(t, 50, chain[0].Priority)

	assert.Equal(t, "mw3", chain[1].Name)
	assert.Equal(t, 75, chain[1].Priority)

	assert.Equal(t, "mw1", chain[2].Name)
	assert.Equal(t, 100, chain[2].Priority)

	// Execute request to verify execution order
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create handler with dynamic middleware
	handler := manager.DynamicMiddleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err = handler(c)
	require.NoError(t, err)

	// Verify all middleware executed
	assert.Equal(t, 1, middleware1.GetExecutionCount())
	assert.Equal(t, 1, middleware2.GetExecutionCount())
	assert.Equal(t, 1, middleware3.GetExecutionCount())

	// Verify response headers (all should be set)
	assert.Equal(t, "executed", rec.Header().Get("X-mw1"))
	assert.Equal(t, "executed", rec.Header().Get("X-mw2"))
	assert.Equal(t, "executed", rec.Header().Get("X-mw3"))
}

// TestMiddlewarePriorityUpdate tests dynamic priority updates
func TestMiddlewarePriorityUpdate(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	manager := plugins.NewPluginManager(logger)

	mw1 := NewMockMiddleware("mw1", "1.0.0")
	mw2 := NewMockMiddleware("mw2", "1.0.0")

	err := manager.RegisterMiddleware("mw1", mw1, 100, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	err = manager.RegisterMiddleware("mw2", mw2, 200, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	// Initial order: mw1 (100), mw2 (200)
	chain := manager.GetMiddlewareChain()
	assert.Equal(t, "mw1", chain[0].Name)
	assert.Equal(t, "mw2", chain[1].Name)

	// Update mw1 priority to 300 (should move after mw2)
	err = manager.UpdateMiddlewarePriority("mw1", 300)
	require.NoError(t, err)

	// Verify new order: mw2 (200), mw1 (300)
	chain = manager.GetMiddlewareChain()
	assert.Equal(t, "mw2", chain[0].Name)
	assert.Equal(t, 200, chain[0].Priority)
	assert.Equal(t, "mw1", chain[1].Name)
	assert.Equal(t, 300, chain[1].Priority)
}

// TestRouteSpecificMiddleware tests route pattern matching
func TestRouteSpecificMiddleware(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	manager := plugins.NewPluginManager(logger)

	// Middleware for all routes
	mwAll := NewMockMiddleware("mw-all", "1.0.0")
	err := manager.RegisterMiddleware("mw-all", mwAll, 100, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	// Middleware for /api/* only
	mwAPI := NewMockMiddleware("mw-api", "1.0.0")
	err = manager.RegisterMiddleware("mw-api", mwAPI, 110, []string{"/api/*"}, "pre-auth")
	require.NoError(t, err)

	// Middleware for /admin/* only
	mwAdmin := NewMockMiddleware("mw-admin", "1.0.0")
	err = manager.RegisterMiddleware("mw-admin", mwAdmin, 110, []string{"/admin/*"}, "pre-auth")
	require.NoError(t, err)

	e := echo.New()

	// Test /api/users - should execute mw-all and mw-api
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := manager.DynamicMiddleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err = handler(c)
	require.NoError(t, err)

	assert.Equal(t, 1, mwAll.GetExecutionCount())
	assert.Equal(t, 1, mwAPI.GetExecutionCount())
	assert.Equal(t, 0, mwAdmin.GetExecutionCount())

	// Test /admin/dashboard - should execute mw-all and mw-admin
	req = httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = handler(c)
	require.NoError(t, err)

	assert.Equal(t, 2, mwAll.GetExecutionCount())
	assert.Equal(t, 1, mwAPI.GetExecutionCount())
	assert.Equal(t, 1, mwAdmin.GetExecutionCount())

	// Test /public/home - should only execute mw-all
	req = httptest.NewRequest(http.MethodGet, "/public/home", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = handler(c)
	require.NoError(t, err)

	assert.Equal(t, 3, mwAll.GetExecutionCount())
	assert.Equal(t, 1, mwAPI.GetExecutionCount())
	assert.Equal(t, 1, mwAdmin.GetExecutionCount())
}

// TestMiddlewareRouteUpdate tests dynamic route updates
func TestMiddlewareRouteUpdate(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	manager := plugins.NewPluginManager(logger)

	mw := NewMockMiddleware("test-mw", "1.0.0")
	err := manager.RegisterMiddleware("test-mw", mw, 100, []string{"/api/*"}, "pre-auth")
	require.NoError(t, err)

	e := echo.New()

	// Test /api/users - should execute
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := manager.DynamicMiddleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, 1, mw.GetExecutionCount())

	// Update routes to /admin/*
	err = manager.UpdateMiddlewareRoutes("test-mw", []string{"/admin/*"})
	require.NoError(t, err)

	// Test /api/users again - should NOT execute
	req = httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, 1, mw.GetExecutionCount()) // Count unchanged

	// Test /admin/dashboard - should execute
	req = httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, 2, mw.GetExecutionCount()) // Count increased
}

// TestMiddlewareUnregister tests middleware removal from chain
func TestMiddlewareUnregister(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	manager := plugins.NewPluginManager(logger)

	mw1 := NewMockMiddleware("mw1", "1.0.0")
	mw2 := NewMockMiddleware("mw2", "1.0.0")

	err := manager.RegisterMiddleware("mw1", mw1, 100, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	err = manager.RegisterMiddleware("mw2", mw2, 200, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	chain := manager.GetMiddlewareChain()
	assert.Len(t, chain, 2)

	// Unregister mw1
	err = manager.UnregisterMiddleware("mw1")
	require.NoError(t, err)

	chain = manager.GetMiddlewareChain()
	assert.Len(t, chain, 1)
	assert.Equal(t, "mw2", chain[0].Name)

	// Execute request - only mw2 should execute
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := manager.DynamicMiddleware()(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err = handler(c)
	require.NoError(t, err)

	assert.Equal(t, 0, mw1.GetExecutionCount())
	assert.Equal(t, 1, mw2.GetExecutionCount())
}

// TestMiddlewareTester tests the testing framework
func TestMiddlewareTester(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	manager := plugins.NewPluginManager(logger)
	tester := manager.GetTester()
	require.NotNil(t, tester)

	mw := NewMockMiddleware("test-mw", "1.0.0")
	err := manager.RegisterMiddleware("test-mw", mw, 100, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	// Test middleware with test data
	testData := map[string]interface{}{
		"method": "POST",
		"path":   "/api/test",
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
		},
		"body": map[string]interface{}{
			"key": "value",
		},
	}

	result, err := tester.TestMiddleware("test-mw", testData)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "test-mw", result.MiddlewareName)
	assert.Greater(t, result.ExecutionTime, time.Duration(0))

	// Verify metrics were recorded
	metrics := tester.GetMetrics("test-mw")
	require.NotNil(t, metrics)
	stats := metrics.GetStats()
	assert.Equal(t, int64(1), stats["totalRequests"])
	assert.Equal(t, int64(0), stats["failedRequests"])
}

// TestMiddlewareHealthCheck tests health checking functionality
func TestMiddlewareHealthCheck(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	manager := plugins.NewPluginManager(logger)
	tester := manager.GetTester()

	mw := NewMockMiddleware("health-test", "1.0.0")
	err := manager.RegisterMiddleware("health-test", mw, 100, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	// Perform health check
	health := tester.GetHealthStatus("health-test")
	require.NotNil(t, health)
	assert.Equal(t, "health-test", health.MiddlewareName)
	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, 0, health.ConsecutiveErrors)
}

// TestMiddlewareMetricsRecording tests metrics recording
func TestMiddlewareMetricsRecording(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	manager := plugins.NewPluginManager(logger)
	tester := manager.GetTester()

	mw := NewMockMiddleware("metrics-test", "1.0.0")
	err := manager.RegisterMiddleware("metrics-test", mw, 100, []string{"*"}, "pre-auth")
	require.NoError(t, err)

	// Execute requests
	for i := 0; i < 10; i++ {
		testData := map[string]interface{}{
			"method": "GET",
			"path":   "/test",
		}

		if i%3 == 0 {
			// Simulate some failures
			mw.SetShouldFail(true)
		} else {
			mw.SetShouldFail(false)
		}

		_, _ = tester.TestMiddleware("metrics-test", testData)
	}

	// Verify metrics
	metrics := tester.GetMetrics("metrics-test")
	require.NotNil(t, metrics)

	stats := metrics.GetStats()
	assert.Equal(t, int64(10), stats["totalRequests"])
	assert.GreaterOrEqual(t, stats["failedRequests"].(int64), int64(3))

	// Reset metrics
	tester.ResetMetrics("metrics-test")
	metrics = tester.GetMetrics("metrics-test")
	stats = metrics.GetStats()
	assert.Equal(t, int64(0), stats["totalRequests"])
}
