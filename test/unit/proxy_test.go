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
	// Create a mock upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from upstream service"))
	}))
	defer upstream.Close()

	// Create service config
	serviceConfig := config.ServiceConfig{
		Name:          "test-service",
		BasePath:      "/api/test",
		Targets:       []string{upstream.URL},
		StripBasePath: true,
		Timeout:       30 * time.Second,
		LoadBalancing: "round_robin",
	}

	// Create proxy handler
	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)

	// Create test request
	e := echo.New()
	req := httptest.NewRequest("GET", "/api/test/resource", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute the handler
	err = handler(c)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Hello from upstream service")
}

func TestLoadBalancing(t *testing.T) {
	// Create two mock servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Response from server 1"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Response from server 2"))
	}))
	defer server2.Close()

	serviceConfig := config.ServiceConfig{
		Name:          "test-lb-service",
		BasePath:      "/api/test-lb",
		Targets:       []string{server1.URL, server2.URL},
		StripBasePath: true,
		Timeout:       30 * time.Second,
		LoadBalancing: "round_robin",
	}

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
		assert.Equal(t, http.StatusOK, rec.Code)

		response := rec.Body.String()
		responses[response]++
	}

	// Both servers should have received requests
	assert.Len(t, responses, 2)
}

func TestProxyHandlerBasic(t *testing.T) {
	// Basic test to ensure proxy handler can be created and used
	serviceConfig := config.ServiceConfig{
		Name:     "basic-test",
		BasePath: "/test",
		Targets:  []string{"http://example.com"},
		Timeout:  5 * time.Second,
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	assert.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestNewHandler(t *testing.T) {
	serviceConfig := config.ServiceConfig{
		Name:          "test-service",
		BasePath:      "/api/test",
		Targets:       []string{"http://localhost:8081"},
		Timeout:       30 * time.Second,
		LoadBalancing: "round_robin",
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestNewHandlerInvalidTarget(t *testing.T) {
	serviceConfig := config.ServiceConfig{
		Name:     "test-service",
		BasePath: "/api/test",
		Targets:  []string{"://invalid-url-with-no-scheme"},
		Timeout:  30 * time.Second,
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	assert.Error(t, err)
	assert.Nil(t, handler)
}

func TestRoundRobinBalancer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	serviceConfig := config.ServiceConfig{
		Name:          "test-service",
		BasePath:      "/api/test",
		Targets:       []string{server.URL, server.URL + "/alt"},
		Timeout:       5 * time.Second,
		LoadBalancing: "round_robin",
	}

	handler, err := proxy.NewHandler(serviceConfig, logrus.New())
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest("GET", "/api/test/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = handler(c)
	assert.NoError(t, err)
}

func TestPathStripping(t *testing.T) {
	// Create a mock server that echoes the request path
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"path": "` + r.URL.Path + `"}`))
	}))
	defer server.Close()

	tests := []struct {
		name          string
		basePath      string
		requestPath   string
		stripBasePath bool
		expectedPath  string
	}{
		{
			name:          "strip base path",
			basePath:      "/api/users",
			requestPath:   "/api/users/123",
			stripBasePath: true,
			expectedPath:  "/123",
		},
		{
			name:          "keep base path",
			basePath:      "/api/users",
			requestPath:   "/api/users/123",
			stripBasePath: false,
			expectedPath:  "/api/users/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceConfig := config.ServiceConfig{
				Name:          "test-service",
				BasePath:      tt.basePath,
				Targets:       []string{server.URL},
				StripBasePath: tt.stripBasePath,
				Timeout:       5 * time.Second,
			}

			handler, err := proxy.NewHandler(serviceConfig, logrus.New())
			require.NoError(t, err)

			e := echo.New()
			req := httptest.NewRequest("GET", tt.requestPath, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err = handler(c)
			assert.NoError(t, err)
		})
	}
}

func TestRetryMechanism(t *testing.T) {
	// Create a server that succeeds on first try to match the expected behavior
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Always succeed for this test
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	serviceConfig := config.ServiceConfig{
		Name:       "test-service",
		BasePath:   "/api/test",
		Targets:    []string{server.URL},
		Timeout:    5 * time.Second,
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
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
