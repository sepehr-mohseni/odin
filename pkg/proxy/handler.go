package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"odin/pkg/config"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	service      config.ServiceConfig
	logger       *logrus.Logger
	client       *http.Client
	targets      []*url.URL
	loadBalancer LoadBalancer
}

type LoadBalancer interface {
	NextTarget() *url.URL
}

type RoundRobinBalancer struct {
	targets []*url.URL
	current int
	mu      sync.Mutex
}

type RandomBalancer struct {
	targets []*url.URL
}

// NewHandler creates a new proxy handler for a service
func NewHandler(service config.ServiceConfig, logger *logrus.Logger) (echo.HandlerFunc, error) {
	if len(service.Targets) == 0 {
		return nil, fmt.Errorf("service %s has no targets", service.Name)
	}

	var targets []*url.URL
	for _, target := range service.Targets {
		parsedURL, err := url.Parse(target)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL %s: %w", target, err)
		}

		// Additional validation for proper URL format
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return nil, fmt.Errorf("invalid target URL %s: missing scheme or host", target)
		}

		targets = append(targets, parsedURL)
	}

	handler := &Handler{
		service: service,
		logger:  logger,
		client: &http.Client{
			Timeout: service.Timeout,
		},
		targets: targets,
	}

	// Initialize load balancer
	switch service.LoadBalancing {
	case "random":
		handler.loadBalancer = &RandomBalancer{targets: targets}
	default: // round-robin
		handler.loadBalancer = &RoundRobinBalancer{targets: targets}
	}

	return handler.Handle, nil
}

// Handle processes HTTP requests and forwards them to backend services
func (h *Handler) Handle(c echo.Context) error {
	// Get target URL
	targetURL := h.loadBalancer.NextTarget()
	if targetURL == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "No available targets")
	}

	// Build the request path
	path := c.Request().URL.Path
	if h.service.StripBasePath && strings.HasPrefix(path, h.service.BasePath) {
		path = strings.TrimPrefix(path, h.service.BasePath)
		if path == "" {
			path = "/"
		}
	}

	// Create target URL
	target := fmt.Sprintf("%s%s", targetURL.String(), path)
	if c.Request().URL.RawQuery != "" {
		target += "?" + c.Request().URL.RawQuery
	}

	h.logger.WithFields(logrus.Fields{
		"service":       h.service.Name,
		"target_url":    target,
		"original_path": c.Request().URL.Path,
	}).Info("Forwarding to target")

	// Create proxy request
	var body io.Reader
	if c.Request().Body != nil {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body")
		}
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(c.Request().Context(), c.Request().Method, target, body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create proxy request")
	}

	// Copy headers
	for k, v := range c.Request().Header {
		req.Header[k] = v
	}

	// Add custom headers
	for k, v := range h.service.Headers {
		req.Header.Set(k, v)
	}

	// Make request with retries
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= h.service.RetryCount; attempt++ {
		resp, lastErr = h.client.Do(req)
		if lastErr == nil && resp.StatusCode < 500 {
			break
		}
		if attempt < h.service.RetryCount {
			time.Sleep(h.service.RetryDelay)
		}
	}

	if lastErr != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "Service unavailable")
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		c.Response().Header()[k] = v
	}

	// Copy response body
	c.Response().WriteHeader(resp.StatusCode)
	_, err = io.Copy(c.Response().Writer, resp.Body)
	return err
}

// NextTarget returns the next target for round-robin balancing
func (rr *RoundRobinBalancer) NextTarget() *url.URL {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.targets) == 0 {
		return nil
	}

	target := rr.targets[rr.current]
	rr.current = (rr.current + 1) % len(rr.targets)
	return target
}

// NextTarget returns a random target
func (rb *RandomBalancer) NextTarget() *url.URL {
	if len(rb.targets) == 0 {
		return nil
	}
	return rb.targets[rand.Intn(len(rb.targets))]
}

func (h *Handler) handleAggregation(c echo.Context, responseBody []byte) ([]byte, error) {
	if h.service.Aggregation == nil {
		return responseBody, nil
	}

	aggregatedData := make(map[string]interface{})

	var originalResponse interface{}
	if err := json.Unmarshal(responseBody, &originalResponse); err != nil {
		h.logger.WithError(err).Error("Failed to unmarshal original response")
		return responseBody, nil
	}

	aggregatedData["original"] = originalResponse

	for _, dep := range h.service.Aggregation.Dependencies {
		depResponse, err := h.fetchDependencyData(c, dep)
		if err != nil {
			h.logger.WithError(err).Warnf("Failed to fetch dependency data from %s", dep.Service)
			continue
		}
		aggregatedData[dep.Service] = depResponse
	}

	result, err := json.Marshal(aggregatedData)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal aggregated data")
		return responseBody, nil
	}

	return result, nil
}

func (h *Handler) fetchDependencyData(c echo.Context, dep config.DependencyConfig) (interface{}, error) {
	targetURL := h.buildDependencyURL(c, dep)

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	// Copy relevant headers from original request
	h.copyHeaders(c.Request(), req, dep.ParameterMapping)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dependency service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return h.mapDependencyResponse(data, dep.ResultMapping), nil
}

func (h *Handler) buildDependencyURL(c echo.Context, dep config.DependencyConfig) string {
	targetURL := dep.Path

	// Replace parameters in the URL using request context
	for _, mapping := range dep.ParameterMapping {
		paramName := strings.TrimPrefix(mapping.To, "{")
		paramName = strings.TrimSuffix(paramName, "}")

		// Get parameter value from request
		paramValue := c.Param(paramName)
		if paramValue == "" {
			paramValue = c.QueryParam(paramName)
		}

		if paramValue != "" {
			targetURL = strings.ReplaceAll(targetURL, "{"+paramName+"}", paramValue)
		}
	}

	return targetURL
}

func (h *Handler) copyHeaders(from *http.Request, to *http.Request, mappings []config.MappingConfig) {
	// Copy relevant headers from original request
	for k, vals := range from.Header {
		for _, v := range vals {
			to.Header.Add(k, v)
		}
	}

	// Apply header mappings if specified
	for _, mapping := range mappings {
		fromHeader := strings.TrimPrefix(mapping.From, "header:")
		toHeader := strings.TrimPrefix(mapping.To, "header:")

		if fromValue := from.Header.Get(fromHeader); fromValue != "" {
			to.Header.Set(toHeader, fromValue)
		}
	}
}

func (h *Handler) mapDependencyResponse(data interface{}, mappings []config.MappingConfig) interface{} {
	if len(mappings) == 0 {
		return data
	}

	// Convert data to map for processing
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})

	// Apply mappings
	for _, mapping := range mappings {
		fromPath := strings.TrimPrefix(mapping.From, "$.")
		toPath := strings.TrimPrefix(mapping.To, "$.")

		// Extract value from source
		if fromPath == "" || fromPath == "$" {
			// Map entire object
			if toPath == "" || toPath == "$" {
				return dataMap
			}
			setNestedValue(result, strings.Split(toPath, "."), dataMap)
		} else {
			// Extract specific field
			value, found := getNestedValue(dataMap, strings.Split(fromPath, "."))
			if found {
				if toPath == "" || toPath == "$" {
					return value
				}
				setNestedValue(result, strings.Split(toPath, "."), value)
			}
		}
	}

	if len(result) == 0 {
		return dataMap
	}

	return result
}

// getNestedValue extracts a value from nested map using dot notation
func getNestedValue(data map[string]interface{}, path []string) (interface{}, bool) {
	current := data

	for i, key := range path {
		if i == len(path)-1 {
			value, exists := current[key]
			return value, exists
		}

		next, exists := current[key]
		if !exists {
			return nil, false
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return nil, false
		}

		current = nextMap
	}

	return nil, false
}

// setNestedValue sets a value in nested map using dot notation
func setNestedValue(data map[string]interface{}, path []string, value interface{}) {
	current := data

	for i, key := range path {
		if i == len(path)-1 {
			current[key] = value
			return
		}

		next, exists := current[key]
		if !exists {
			next = make(map[string]interface{})
			current[key] = next
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			nextMap = make(map[string]interface{})
			current[key] = nextMap
		}

		current = nextMap
	}
}
