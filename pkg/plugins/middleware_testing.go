package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// MiddlewareTestResult contains the results of a middleware test
type MiddlewareTestResult struct {
	MiddlewareName string                 `json:"middlewareName"`
	Success        bool                   `json:"success"`
	ExecutionTime  time.Duration          `json:"executionTime"`
	RequestData    map[string]interface{} `json:"requestData"`
	ResponseData   map[string]interface{} `json:"responseData"`
	Error          string                 `json:"error,omitempty"`
	Logs           []string               `json:"logs"`
	Timestamp      time.Time              `json:"timestamp"`
}

// MiddlewareHealthStatus represents the health status of a middleware
type MiddlewareHealthStatus struct {
	MiddlewareName    string        `json:"middlewareName"`
	Status            string        `json:"status"` // healthy, degraded, unhealthy
	LastCheck         time.Time     `json:"lastCheck"`
	ResponseTime      time.Duration `json:"responseTime"`
	ErrorRate         float64       `json:"errorRate"`
	TotalRequests     int64         `json:"totalRequests"`
	FailedRequests    int64         `json:"failedRequests"`
	AverageLatency    time.Duration `json:"averageLatency"`
	ConsecutiveErrors int           `json:"consecutiveErrors"`
	Message           string        `json:"message,omitempty"`
}

// MiddlewareMetrics tracks performance metrics for middleware
type MiddlewareMetrics struct {
	Name              string
	TotalRequests     int64
	FailedRequests    int64
	TotalLatency      time.Duration
	MinLatency        time.Duration
	MaxLatency        time.Duration
	LastError         error
	LastErrorTime     time.Time
	ConsecutiveErrors int
	mu                sync.RWMutex
}

// GetStats returns a snapshot of metrics (thread-safe)
func (mm *MiddlewareMetrics) GetStats() map[string]interface{} {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	stats := map[string]interface{}{
		"name":              mm.Name,
		"totalRequests":     mm.TotalRequests,
		"failedRequests":    mm.FailedRequests,
		"totalLatency":      mm.TotalLatency.String(),
		"minLatency":        mm.MinLatency.String(),
		"maxLatency":        mm.MaxLatency.String(),
		"consecutiveErrors": mm.ConsecutiveErrors,
	}

	if mm.LastError != nil {
		stats["lastError"] = mm.LastError.Error()
		stats["lastErrorTime"] = mm.LastErrorTime
	}

	if mm.TotalRequests > 0 {
		avgLatency := time.Duration(int64(mm.TotalLatency) / mm.TotalRequests)
		stats["averageLatency"] = avgLatency.String()
		stats["errorRate"] = float64(mm.FailedRequests) / float64(mm.TotalRequests)
	}

	return stats
}

// MiddlewareTester provides testing and health check capabilities for middleware
type MiddlewareTester struct {
	manager       *PluginManager
	metrics       map[string]*MiddlewareMetrics
	healthChecks  map[string]*MiddlewareHealthStatus
	logger        *logrus.Logger
	mu            sync.RWMutex
	checkInterval time.Duration
	stopChan      chan struct{}
}

// NewMiddlewareTester creates a new middleware tester
func NewMiddlewareTester(manager *PluginManager, logger *logrus.Logger) *MiddlewareTester {
	if logger == nil {
		logger = logrus.New()
	}

	return &MiddlewareTester{
		manager:       manager,
		metrics:       make(map[string]*MiddlewareMetrics),
		healthChecks:  make(map[string]*MiddlewareHealthStatus),
		logger:        logger,
		checkInterval: 30 * time.Second,
		stopChan:      make(chan struct{}),
	}
}

// StartHealthChecks begins periodic health checking for all middleware
func (mt *MiddlewareTester) StartHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(mt.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mt.performHealthChecks()
		case <-mt.stopChan:
			mt.logger.Info("Stopping middleware health checks")
			return
		case <-ctx.Done():
			mt.logger.Info("Context cancelled, stopping health checks")
			return
		}
	}
}

// StopHealthChecks stops the health check routine
func (mt *MiddlewareTester) StopHealthChecks() {
	close(mt.stopChan)
}

// performHealthChecks checks health of all registered middleware
func (mt *MiddlewareTester) performHealthChecks() {
	chain := mt.manager.GetMiddlewareChain()

	for _, entry := range chain {
		status := mt.checkMiddlewareHealth(entry.Name)
		mt.mu.Lock()
		mt.healthChecks[entry.Name] = status
		mt.mu.Unlock()

		// Log if unhealthy
		if status.Status == "unhealthy" {
			mt.logger.WithFields(logrus.Fields{
				"middleware":        entry.Name,
				"consecutiveErrors": status.ConsecutiveErrors,
				"errorRate":         status.ErrorRate,
			}).Warn("Middleware health check failed")
		}
	}
}

// checkMiddlewareHealth performs a health check on a specific middleware
func (mt *MiddlewareTester) checkMiddlewareHealth(name string) *MiddlewareHealthStatus {
	start := time.Now()

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health-check", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	status := &MiddlewareHealthStatus{
		MiddlewareName: name,
		LastCheck:      time.Now(),
		Status:         "healthy",
	}

	// Get metrics (safely)
	mt.mu.RLock()
	metrics, metricsExist := mt.metrics[name]
	mt.mu.RUnlock()

	if metricsExist {
		metrics.mu.RLock()
		status.TotalRequests = metrics.TotalRequests
		status.FailedRequests = metrics.FailedRequests
		status.ConsecutiveErrors = metrics.ConsecutiveErrors

		if metrics.TotalRequests > 0 {
			status.ErrorRate = float64(metrics.FailedRequests) / float64(metrics.TotalRequests)
			status.AverageLatency = time.Duration(int64(metrics.TotalLatency) / metrics.TotalRequests)
		}
		metrics.mu.RUnlock()
	}

	// Get middleware entry
	chain := mt.manager.GetMiddlewareChain()
	var middleware Middleware
	for _, entry := range chain {
		if entry.Name == name {
			middleware = entry.Middleware
			break
		}
	}

	if middleware == nil {
		status.Status = "unhealthy"
		status.Message = "Middleware not found in chain"
		return status
	}

	// Test middleware execution
	handler := middleware.Handle(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	status.ResponseTime = time.Since(start)

	if err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("Execution error: %v", err)
		return status
	}

	// Determine health status based on metrics
	if status.ErrorRate > 0.5 {
		status.Status = "unhealthy"
		status.Message = "High error rate (>50%)"
	} else if status.ErrorRate > 0.1 {
		status.Status = "degraded"
		status.Message = "Elevated error rate (>10%)"
	} else if status.ConsecutiveErrors >= 5 {
		status.Status = "degraded"
		status.Message = fmt.Sprintf("Consecutive errors: %d", status.ConsecutiveErrors)
	} else if status.ResponseTime > 1*time.Second {
		status.Status = "degraded"
		status.Message = "Slow response time"
	}

	return status
}

// GetHealthStatus returns the current health status of a middleware
func (mt *MiddlewareTester) GetHealthStatus(name string) *MiddlewareHealthStatus {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if status, exists := mt.healthChecks[name]; exists {
		return status
	}

	// If no cached status, perform check now
	return mt.checkMiddlewareHealth(name)
}

// GetAllHealthStatuses returns health statuses for all middleware
func (mt *MiddlewareTester) GetAllHealthStatuses() map[string]*MiddlewareHealthStatus {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	statuses := make(map[string]*MiddlewareHealthStatus, len(mt.healthChecks))
	for name, status := range mt.healthChecks {
		statuses[name] = status
	}

	return statuses
}

// TestMiddleware executes a middleware in a sandboxed environment
func (mt *MiddlewareTester) TestMiddleware(name string, testData map[string]interface{}) (*MiddlewareTestResult, error) {
	result := &MiddlewareTestResult{
		MiddlewareName: name,
		RequestData:    testData,
		ResponseData:   make(map[string]interface{}),
		Logs:           []string{},
		Timestamp:      time.Now(),
	}

	// Get middleware from chain
	chain := mt.manager.GetMiddlewareChain()
	var middleware Middleware
	for _, entry := range chain {
		if entry.Name == name {
			middleware = entry.Middleware
			break
		}
	}

	if middleware == nil {
		return nil, fmt.Errorf("middleware %s not found in chain", name)
	}

	// Create test request
	method := "GET"
	path := "/"
	var body io.Reader

	if methodVal, ok := testData["method"].(string); ok {
		method = methodVal
	}
	if pathVal, ok := testData["path"].(string); ok {
		path = pathVal
	}
	if bodyVal, ok := testData["body"]; ok {
		bodyBytes, _ := json.Marshal(bodyVal)
		body = bytes.NewReader(bodyBytes)
	}

	req := httptest.NewRequest(method, path, body)

	// Add headers from test data
	if headers, ok := testData["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			if valueStr, ok := value.(string); ok {
				req.Header.Set(key, valueStr)
			}
		}
	}

	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	// Capture logs
	logBuffer := &bytes.Buffer{}
	testLogger := logrus.New()
	testLogger.SetOutput(logBuffer)

	// Execute middleware
	start := time.Now()
	handler := middleware.Handle(func(c echo.Context) error {
		return c.String(http.StatusOK, "Test passed")
	})

	err := handler(c)
	result.ExecutionTime = time.Since(start)

	// Capture response
	result.ResponseData["status"] = rec.Code
	result.ResponseData["headers"] = rec.Header()
	result.ResponseData["body"] = rec.Body.String()
	result.Logs = append(result.Logs, logBuffer.String())

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		mt.recordError(name, err)
	} else {
		result.Success = true
		mt.recordSuccess(name, result.ExecutionTime)
	}

	return result, nil
}

// RecordRequest records metrics for a middleware request
func (mt *MiddlewareTester) RecordRequest(name string, duration time.Duration, err error) {
	if err != nil {
		mt.recordError(name, err)
	} else {
		mt.recordSuccess(name, duration)
	}
}

// recordSuccess records a successful middleware execution
func (mt *MiddlewareTester) recordSuccess(name string, duration time.Duration) {
	metrics := mt.getMetrics(name)

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.TotalRequests++
	metrics.TotalLatency += duration
	metrics.ConsecutiveErrors = 0

	if metrics.MinLatency == 0 || duration < metrics.MinLatency {
		metrics.MinLatency = duration
	}
	if duration > metrics.MaxLatency {
		metrics.MaxLatency = duration
	}
}

// recordError records a failed middleware execution
func (mt *MiddlewareTester) recordError(name string, err error) {
	metrics := mt.getMetrics(name)

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.TotalRequests++
	metrics.FailedRequests++
	metrics.LastError = err
	metrics.LastErrorTime = time.Now()
	metrics.ConsecutiveErrors++
}

// getMetrics returns or creates metrics for a middleware
func (mt *MiddlewareTester) getMetrics(name string) *MiddlewareMetrics {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if metrics, exists := mt.metrics[name]; exists {
		return metrics
	}

	metrics := &MiddlewareMetrics{
		Name: name,
	}
	mt.metrics[name] = metrics
	return metrics
}

// GetMetrics returns metrics for a middleware
func (mt *MiddlewareTester) GetMetrics(name string) *MiddlewareMetrics {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if metrics, exists := mt.metrics[name]; exists {
		return metrics
	}

	return nil
}

// GetAllMetrics returns all middleware metrics
func (mt *MiddlewareTester) GetAllMetrics() map[string]*MiddlewareMetrics {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	allMetrics := make(map[string]*MiddlewareMetrics, len(mt.metrics))
	for name, metrics := range mt.metrics {
		allMetrics[name] = metrics
	}

	return allMetrics
}

// ResetMetrics resets metrics for a specific middleware
func (mt *MiddlewareTester) ResetMetrics(name string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.metrics[name] = &MiddlewareMetrics{Name: name}
}

// ResetAllMetrics resets all middleware metrics
func (mt *MiddlewareTester) ResetAllMetrics() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	for name := range mt.metrics {
		mt.metrics[name] = &MiddlewareMetrics{Name: name}
	}
}

// CreateMiddlewareWrapper creates a middleware wrapper that records metrics
func (mt *MiddlewareTester) CreateMiddlewareWrapper(middleware Middleware) Middleware {
	return &metricsMiddleware{
		wrapped: middleware,
		tester:  mt,
	}
}

// metricsMiddleware wraps a middleware and records metrics
type metricsMiddleware struct {
	wrapped Middleware
	tester  *MiddlewareTester
}

func (m *metricsMiddleware) Name() string {
	return m.wrapped.Name()
}

func (m *metricsMiddleware) Version() string {
	return m.wrapped.Version()
}

func (m *metricsMiddleware) Initialize(config map[string]interface{}) error {
	return m.wrapped.Initialize(config)
}

func (m *metricsMiddleware) Handle(next echo.HandlerFunc) echo.HandlerFunc {
	wrappedHandler := m.wrapped.Handle(next)

	return func(c echo.Context) error {
		start := time.Now()
		err := wrappedHandler(c)
		duration := time.Since(start)

		m.tester.RecordRequest(m.wrapped.Name(), duration, err)

		return err
	}
}

func (m *metricsMiddleware) Cleanup() error {
	return m.wrapped.Cleanup()
}
