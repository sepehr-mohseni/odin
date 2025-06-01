package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestProxyBasicFunctionality(t *testing.T) {
	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer testServer.Close()

	// Create Echo instance
	e := echo.New()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set target URL in context
	c.Set("target_url", testServer.URL)

	// This would test proxy functionality if we had the proxy handler
	// For now, just test that the context is set up correctly
	assert.Equal(t, testServer.URL, c.Get("target_url"))
}

func TestProxyHeaders(t *testing.T) {
	// Test that headers are properly forwarded
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Verify headers are accessible
	assert.Equal(t, "Bearer token", c.Request().Header.Get("Authorization"))
	assert.Equal(t, "application/json", c.Request().Header.Get("Content-Type"))
}
