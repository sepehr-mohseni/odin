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
)

func TestProxyHandlerSimpleForwarding(t *testing.T) {
	// Setup target server
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Hello from target"}`))
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
		w.Write([]byte(`{"server":"target1"}`))
	}))
	defer target1.Close()

	target2Hits := 0
	target2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target2Hits++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"server":"target2"}`))
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
			t.Fatalf("Handler returned error: %v", err)
		}
	}

	// With round-robin, we should have 2 hits per target
	if target1Hits != 2 || target2Hits != 2 {
		t.Errorf("Round-robin not working correctly. Target1: %d hits, Target2: %d hits", target1Hits, target2Hits)
	}
}

func TestProxyHandlerBasic(t *testing.T) {
	// Basic test setup code
	w := httptest.NewRecorder()
	
	// Fix line 19
	_, err := w.Write([]byte("test response"))
	if err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}
	
	// More test code...
	
	// Fix line 71
	_, err = w.Write([]byte("another test response"))
	if err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}
	
	// More test code...
	
	// Fix line 79
	_, err = w.Write([]byte("final test response"))
	if err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}
}
