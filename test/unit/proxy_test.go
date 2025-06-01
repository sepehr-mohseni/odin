package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"odin/pkg/config"
	"odin/pkg/proxy"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyHandlerSimpleForwarding(t *testing.T) {
	// Create a mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Hello from backend"}`))
	}))
	defer backend.Close()

	// Create service config
	serviceConfig := config.ServiceConfig{
		Name:          "test-service",
		BasePath:      "/api/test",
		Targets:       []string{backend.URL},
		Timeout:       30 * time.Second,
		StripBasePath: true,
	}

	// Create proxy handler
	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)

	// Create Echo context
	e := echo.New()
	req := httptest.NewRequest("GET", "/api/test/resource", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute handler
	err = handler(c)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Hello from backend")
}

func TestLoadBalancing(t *testing.T) {
	// Create multiple backend servers
	backend1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"server": "backend1"}`))
	}))
	defer backend1.Close()

	backend2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"server": "backend2"}`))
	}))
	defer backend2.Close()

	// Create service config with multiple targets
	serviceConfig := config.ServiceConfig{
		Name:          "test-lb-service",
		BasePath:      "/api/test-lb",
		Targets:       []string{backend1.URL, backend2.URL},
		Timeout:       30 * time.Second,
		LoadBalancing: "round-robin",
		StripBasePath: true,
	}

	// Create proxy handler
	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)

	e := echo.New()

	// Make multiple requests to test load balancing
	responses := make(map[string]int)
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("GET", "/api/test-lb/resource", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = handler(c)
		require.NoError(t, err)

		body := rec.Body.String()
		responses[body]++
	}

	// Both backends should have received requests
	assert.Len(t, responses, 2)
}

func TestProxyHandlerBasic(t *testing.T) {
	serviceConfig := config.ServiceConfig{
		Name:     "test",
		BasePath: "/api/test",
		Targets:  []string{"http://localhost:9999"},
		Timeout:  1 * time.Second,
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestNewHandler(t *testing.T) {
	serviceConfig := config.ServiceConfig{
		Name:     "test",
		BasePath: "/api/test",
		Targets:  []string{"http://localhost:8081"},
		Timeout:  30 * time.Second,
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestNewHandlerInvalidTarget(t *testing.T) {
	serviceConfig := config.ServiceConfig{
		Name:     "test",
		BasePath: "/api/test",
		Targets:  []string{"invalid-url"}, // This is not a valid URL with scheme
		Timeout:  30 * time.Second,
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	assert.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "invalid target URL")
}

func TestRoundRobinBalancer(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer backend.Close()

	serviceConfig := config.ServiceConfig{
		Name:          "test-service",
		BasePath:      "/api/test",
		Targets:       []string{backend.URL},
		Timeout:       30 * time.Second,
		LoadBalancing: "round-robin",
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest("GET", "/api/test/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPathStripping(t *testing.T) {
	tests := []struct {
		name          string
		stripBasePath bool
		requestPath   string
		expectedPath  string
	}{
		{
			name:          "strip base path",
			stripBasePath: true,
			requestPath:   "/api/users/123",
			expectedPath:  "/123",
		},
		{
			name:          "keep base path",
			stripBasePath: false,
			requestPath:   "/api/users/123",
			expectedPath:  "/api/users/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.expectedPath, r.URL.Path)
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			serviceConfig := config.ServiceConfig{
				Name:          "test-service",
				BasePath:      "/api/users",
				Targets:       []string{backend.URL},
				Timeout:       30 * time.Second,
				StripBasePath: tt.stripBasePath,
			}

			handler, err := proxy.NewHandler(serviceConfig, logrus.New())
			require.NoError(t, err)

			e := echo.New()
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err = handler(c)
			require.NoError(t, err)
		})
	}
}

func TestRetryMechanism(t *testing.T) {
	attempts := 0
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer backend.Close()

	serviceConfig := config.ServiceConfig{
		Name:       "test-service",
		BasePath:   "/api/test",
		Targets:    []string{backend.URL},
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 100 * time.Millisecond,
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, attempts >= 2, "Should have retried at least once")
}
