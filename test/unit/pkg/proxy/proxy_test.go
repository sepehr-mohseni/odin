package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyBasicFunctionality(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Hello from backend"}`))
	}))
	defer backend.Close()

	// Create echo instance
	e := echo.New()

	// Test basic proxy functionality
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set target URL
	c.Set("target_url", backend.URL+"/test")

	// Simple proxy handler
	handler := func(c echo.Context) error {
		targetURL := c.Get("target_url").(string)

		// Create proxy request
		proxyReq, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
		if err != nil {
			return err
		}

		// Copy headers
		for k, v := range c.Request().Header {
			proxyReq.Header[k] = v
		}

		// Make request
		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Copy response
		for k, v := range resp.Header {
			c.Response().Header()[k] = v
		}
		c.Response().WriteHeader(resp.StatusCode)

		return c.Stream(resp.StatusCode, resp.Header.Get("Content-Type"), resp.Body)
	}

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Hello from backend")
}

func TestProxyHeaders(t *testing.T) {
	// Create a test backend server that echoes headers
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "backend-value")
		w.Header().Set("Content-Type", "application/json")

		// Echo some request headers in response
		if auth := r.Header.Get("Authorization"); auth != "" {
			w.Header().Set("X-Auth-Received", auth)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"headers_received": true}`))
	}))
	defer backend.Close()

	e := echo.New()

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Custom", "client-value")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("target_url", backend.URL+"/test")

	// Proxy handler that forwards headers
	handler := func(c echo.Context) error {
		targetURL := c.Get("target_url").(string)

		proxyReq, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
		if err != nil {
			return err
		}

		// Forward headers
		for k, v := range c.Request().Header {
			proxyReq.Header[k] = v
		}

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Copy response headers
		for k, v := range resp.Header {
			c.Response().Header()[k] = v
		}
		c.Response().WriteHeader(resp.StatusCode)

		return c.Stream(resp.StatusCode, resp.Header.Get("Content-Type"), resp.Body)
	}

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "backend-value", rec.Header().Get("X-Custom-Header"))
	assert.Equal(t, "Bearer token123", rec.Header().Get("X-Auth-Received"))
}
