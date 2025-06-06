package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"odin/pkg/cache"
	"odin/pkg/middleware"
	"odin/pkg/ratelimit"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheMiddleware(t *testing.T) {
	store := cache.NewMemoryStore()
	logger := logrus.New()

	middlewareFunc := middleware.CacheMiddleware(store, logger)

	e := echo.New()

	// Test cache miss
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	next := func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "test response")
	}

	err := middlewareFunc(next)(c)
	assert.NoError(t, err)
	assert.True(t, nextCalled)

	// Test non-GET request (should skip caching)
	req = httptest.NewRequest("POST", "/api/test", strings.NewReader(`{"test": true}`))
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	nextCalled = false
	err = middlewareFunc(next)(c)
	assert.NoError(t, err)
	assert.True(t, nextCalled)

	// Test with registered route
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "cached response"})
	})

	// First request - should cache the response
	req = httptest.NewRequest("GET", "/test", nil)
	rec = httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "cached response")
}

func TestRateLimiterMiddleware(t *testing.T) {
	config := ratelimit.Config{
		Enabled:       true,
		DefaultLimit:  10,
		DefaultWindow: 60,
		Algorithm:     ratelimit.AlgorithmFixedWindow,
	}

	limiter, err := ratelimit.NewLimiter(config, logrus.New())
	require.NoError(t, err)

	middlewareFunc := middleware.RateLimiterMiddleware(limiter, logrus.New())

	e := echo.New()

	// Test handler
	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	}

	// Test the middleware
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = middlewareFunc(handler)(c)
	assert.NoError(t, err)
}
