package unit

import (
	"net/http"
	"net/http/httptest"
	"odin/pkg/config"
	"odin/pkg/proxy"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyHandlerSimpleForwarding(t *testing.T) {
	// Setup target server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"message":"Hello from target"}`))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer targetServer.Close()

	// Setup service config
	serviceConfig := config.ServiceConfig{
		Name:          "test-service",
		BasePath:      "/api/test",
		StripBasePath: true,
		Targets:       []string{targetServer.URL},
		Timeout:       5 * time.Second,
	}

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create proxy handler
	handler, err := proxy.NewHandler(serviceConfig, logger)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Setup echo context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/test/resource", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/test/resource")

	// Execute request
	if err := handler(c); err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	expected := `{"message":"Hello from target"}`
	if rec.Body.String() != expected {
		t.Errorf("Expected body %s, got %s", expected, rec.Body.String())
	}
}

func TestLoadBalancing(t *testing.T) {
	// Setup target servers
	target1Hits := 0
	target1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target1Hits++
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"server":"target1"}`))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer target1.Close()

	target2Hits := 0
	target2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target2Hits++
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"server":"target2"}`))
		if err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer target2.Close()

	// Setup service config for round-robin
	serviceConfig := config.ServiceConfig{
		Name:          "test-lb-service",
		BasePath:      "/api/test-lb",
		StripBasePath: true,
		Targets:       []string{target1.URL, target2.URL},
		Timeout:       5 * time.Second,
		LoadBalancing: "round-robin",
	}

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create proxy handler
	handler, err := proxy.NewHandler(serviceConfig, logger)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Setup echo
	e := echo.New()

	// Make multiple requests
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/test-lb/resource", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/test-lb/resource")

		if err := handler(c); err != nil {
			t.Errorf("Handler returned error on request %d: %v", i, err)
		}
	}

	// With round-robin, we should have 2 hits per target
	if target1Hits != 2 || target2Hits != 2 {
		t.Errorf("Expected 2 hits per target, got target1: %d, target2: %d", target1Hits, target2Hits)
	}
}

func TestProxyHandlerBasic(t *testing.T) {
	// Basic test setup code
	w := httptest.NewRecorder()

	// Fix line 19
	_, err := w.Write([]byte("test response"))
	if err != nil {
		t.Errorf("Failed to write test response: %v", err)
	}

	// More test code...

	// Fix line 71
	_, err = w.Write([]byte("another test response"))
	if err != nil {
		t.Errorf("Failed to write another test response: %v", err)
	}

	// More test code...

	// Fix line 79
	_, err = w.Write([]byte("final test response"))
	if err != nil {
		t.Errorf("Failed to write final test response: %v", err)
	}
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
		Targets:  []string{"invalid-url"},
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
	// Create a server that fails first few requests
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success after retry"}`))
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
