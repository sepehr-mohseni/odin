package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"odin/pkg/aggregator"
	"odin/pkg/config"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// No need to seed the random number generator in Go 1.20+
// The math/rand package is automatically initialized with a secure seed

// LoadBalancer defines an interface for load balancing strategies
type LoadBalancer interface {
	NextTarget() *url.URL
}

// RoundRobinBalancer implements round-robin load balancing
type RoundRobinBalancer struct {
	targets []*url.URL
	current int
	mu      sync.Mutex
}

// RandomBalancer implements random target selection
type RandomBalancer struct {
	targets []*url.URL
}

// Handler manages HTTP request proxying to backend services
type Handler struct {
	service      config.ServiceConfig
	targets      []*url.URL
	logger       *logrus.Logger
	client       *http.Client
	loadBalancer LoadBalancer
}

// NewHandler creates a new proxy handler for a service
func NewHandler(service config.ServiceConfig, logger *logrus.Logger) (echo.HandlerFunc, error) {
	targets := make([]*url.URL, 0, len(service.Targets))
	for _, target := range service.Targets {
		u, err := url.Parse(target)
		if err != nil {
			return nil, err
		}
		targets = append(targets, u)
	}

	client := &http.Client{
		Timeout: service.Timeout,
	}

	// Create the appropriate load balancer based on the service configuration
	var lb LoadBalancer
	switch service.LoadBalancing {
	case "round-robin":
		lb = &RoundRobinBalancer{targets: targets}
	case "random":
		lb = &RandomBalancer{targets: targets}
	default:
		lb = &RoundRobinBalancer{targets: targets}
	}

	handler := &Handler{
		service:      service,
		targets:      targets,
		logger:       logger,
		client:       client,
		loadBalancer: lb,
	}

	return handler.Handle, nil
}

// Handle processes HTTP requests and forwards them to backend services
func (h *Handler) Handle(c echo.Context) error {
	// Get target using the pre-configured load balancer
	targetURL := h.loadBalancer.NextTarget()

	reqPath := c.Request().URL.Path
	h.logger.WithFields(logrus.Fields{
		"service":       h.service.Name,
		"reqPath":       reqPath,
		"basePath":      h.service.BasePath,
		"stripBase":     h.service.StripBasePath,
		"original_path": c.Request().URL.Path,
	}).Debug("Processing request path")

	// Path handling
	if h.service.StripBasePath {
		if strings.HasPrefix(reqPath, h.service.BasePath) {
			if reqPath == h.service.BasePath || reqPath == h.service.BasePath+"/" {
				reqPath = "/"
			} else {
				reqPath = strings.TrimPrefix(reqPath, h.service.BasePath)
				if !strings.HasPrefix(reqPath, "/") {
					reqPath = "/" + reqPath
				}
			}
			h.logger.WithField("modified_path", reqPath).Debug("Path after stripping base path")
		}
	}

	// Handle target URL path construction
	targetURLCopy := *targetURL
	if strings.HasSuffix(targetURLCopy.Path, "/") {
		targetURLCopy.Path = targetURLCopy.Path + strings.TrimPrefix(reqPath, "/")
	} else if !strings.HasSuffix(targetURLCopy.Path, "/") && !strings.HasPrefix(reqPath, "/") {
		targetURLCopy.Path = targetURLCopy.Path + "/" + reqPath
	} else {
		targetURLCopy.Path = targetURLCopy.Path + reqPath
	}

	// Copy query parameters
	targetURLCopy.RawQuery = c.Request().URL.RawQuery

	h.logger.WithFields(logrus.Fields{
		"service":       h.service.Name,
		"original_path": c.Request().URL.Path,
		"target_url":    targetURLCopy.String(),
	}).Info("Forwarding to target")

	// Read request body if present
	var reqBody []byte
	var err error
	if c.Request().Body != nil {
		reqBody, err = io.ReadAll(c.Request().Body)
		if err != nil {
			h.logger.WithError(err).Error("Failed to read request body")
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read request body")
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	// Apply request transformations if configured
	if len(h.service.Transform.Request) > 0 && reqBody != nil {
		var requestData map[string]interface{}
		if err := json.Unmarshal(reqBody, &requestData); err == nil {
			for _, rule := range h.service.Transform.Request {
				applyTransformation(requestData, rule)
			}
			reqBody, _ = json.Marshal(requestData)
		}
	}

	// Set up retries
	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt <= h.service.RetryCount; attempt++ {
		if attempt > 0 {
			h.logger.WithFields(logrus.Fields{
				"service": h.service.Name,
				"attempt": attempt,
				"url":     targetURLCopy.String(),
			}).Debug("Retrying request")
			time.Sleep(h.service.RetryDelay)

			// Get a new target for retry using the load balancer
			targetURL = h.loadBalancer.NextTarget()
			targetURLCopy.Host = targetURL.Host
			targetURLCopy.Scheme = targetURL.Scheme
		}

		req, err := http.NewRequest(c.Request().Method, targetURLCopy.String(), bytes.NewReader(reqBody))
		if err != nil {
			lastErr = err
			continue
		}

		// Copy headers from original request
		for k, vals := range c.Request().Header {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}

		// Add service-specific headers
		for k, v := range h.service.Headers {
			req.Header.Set(k, v)
		}

		h.logger.WithFields(logrus.Fields{
			"service":         h.service.Name,
			"method":          req.Method,
			"target_url":      req.URL.String(),
			"strip_base_path": h.service.StripBasePath,
			"original_path":   c.Request().URL.Path,
		}).Debug("Forwarding request to target")

		ctx, cancel := context.WithTimeout(c.Request().Context(), h.service.Timeout)
		defer cancel()

		req = req.WithContext(ctx)

		resp, err = h.client.Do(req)
		if err != nil {
			lastErr = err
			h.logger.WithError(err).WithFields(logrus.Fields{
				"service": h.service.Name,
				"url":     targetURLCopy.String(),
			}).Error("Request failed")
			continue
		}

		break
	}

	// Handle case where all retries failed
	if resp == nil {
		h.logger.WithError(lastErr).Error("All retry attempts failed")

		acceptHeader := c.Request().Header.Get("Accept")
		if strings.Contains(acceptHeader, "text/html") {
			return echo.NewHTTPError(http.StatusBadGateway, "Service unavailable")
		}

		return c.JSON(http.StatusBadGateway, map[string]string{
			"error": "Service unavailable",
		})
	}

	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read response body")
		return echo.NewHTTPError(http.StatusInternalServerError, "Error reading response")
	}

	h.logger.WithFields(logrus.Fields{
		"service":        h.service.Name,
		"status":         resp.StatusCode,
		"content_type":   resp.Header.Get("Content-Type"),
		"content_length": resp.ContentLength,
	}).Debug("Received response from target")

	contentType := resp.Header.Get("Content-Type")
	isJSON := strings.Contains(contentType, "application/json") || isJSONContent(respBody)

	// Apply response transformations if configured
	if len(h.service.Transform.Response) > 0 && isJSON {
		var responseData map[string]interface{}
		if err := json.Unmarshal(respBody, &responseData); err == nil {
			for _, rule := range h.service.Transform.Response {
				applyTransformation(responseData, rule)
			}
			respBody, _ = json.Marshal(responseData)
		}
	}

	// Handle response aggregation if configured
	if isJSON && h.service.Aggregation != nil && len(h.service.Aggregation.Dependencies) > 0 {
		if agg, ok := c.Get("aggregator").(*aggregator.Aggregator); ok {
			authToken := ""
			if authHeader := c.Request().Header.Get("Authorization"); authHeader != "" {
				authToken = authHeader
			}

			h.logger.WithField("service", h.service.Name).Debug("Enriching response with aggregation")
			enrichedBody, err := agg.EnrichResponse(c.Request().Context(), h.service.Name, respBody, resp.Header, authToken)
			if err == nil {
				respBody = enrichedBody
				contentType = "application/json"
			} else {
				h.logger.WithError(err).Error("Failed to enrich response")
			}
		} else {
			h.logger.Warn("Aggregator not found in context")
		}
	}

	// Copy headers from upstream response
	for k, vals := range resp.Header {
		if strings.ToLower(k) != "content-type" {
			for _, v := range vals {
				c.Response().Header().Add(k, v)
			}
		}
	}

	// Set content type
	if isJSON {
		c.Response().Header().Set("Content-Type", "application/json; charset=utf-8")
	} else {
		if contentType == "" {
			contentType = http.DetectContentType(respBody)
		}
		c.Response().Header().Set("Content-Type", contentType)
	}

	return c.Blob(resp.StatusCode, c.Response().Header().Get("Content-Type"), respBody)
}

// Helper function to detect JSON content
func isJSONContent(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	i := 0
	for i < len(data) && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
		i++
	}
	if i >= len(data) {
		return false
	}

	return data[i] == '{' || data[i] == '[' || data[i] == '"'
}

// Get next target using round-robin strategy
func (b *RoundRobinBalancer) NextTarget() *url.URL {
	b.mu.Lock()
	defer b.mu.Unlock()

	target := b.targets[b.current]
	b.current = (b.current + 1) % len(b.targets)
	return target
}

// Get next target using random strategy
func (b *RandomBalancer) NextTarget() *url.URL {
	return b.targets[rand.Intn(len(b.targets))]
}

// Apply transformation rule to data
func applyTransformation(data map[string]interface{}, rule config.TransformRule) {
	fromParts := strings.Split(strings.TrimPrefix(rule.From, "$."), ".")
	toParts := strings.Split(strings.TrimPrefix(rule.To, "$."), ".")

	sourceValue := getNestedValue(data, fromParts)

	if sourceValue == nil && rule.Default != "" {
		sourceValue = rule.Default
	}

	if sourceValue != nil {
		setNestedValue(data, toParts, sourceValue)
	}
}

// Get a nested value from a nested map using path
func getNestedValue(data map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return nil
	}

	currentMap := data
	for i, key := range path {
		if i == len(path)-1 {
			return currentMap[key]
		}

		nextMap, ok := currentMap[key].(map[string]interface{})
		if !ok {
			return nil
		}
		currentMap = nextMap
	}

	return nil
}

// Set a nested value in a nested map using path
func setNestedValue(data map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 {
		return
	}

	currentMap := data
	for i, key := range path {
		if i == len(path)-1 {
			currentMap[key] = value
			return
		}

		if nextMap, ok := currentMap[key].(map[string]interface{}); ok {
			currentMap = nextMap
		} else {
			newMap := make(map[string]interface{})
			currentMap[key] = newMap
			currentMap = newMap
		}
	}
}

func (h *Handler) createProxyRequest(c echo.Context, targetURL string) (*http.Request, error) {
	var body io.Reader = nil

	if c.Request().Body != nil {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return nil, err
		}
		c.Request().Body = io.NopCloser(bytes.NewReader(bodyBytes))
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(c.Request().Method, targetURL, body)
	if err != nil {
		return nil, err
	}

	for k, vals := range c.Request().Header {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	for k, v := range h.service.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
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
	// Start with the dependency path
	targetURL := dep.Path

	// Replace parameters in the URL using request context
	for _, mapping := range dep.ParameterMapping {
		paramName := strings.TrimPrefix(mapping.To, "{")
		paramName = strings.TrimSuffix(paramName, "}")

		// Get parameter value from request context
		paramValue := c.QueryParam(paramName)
		if paramValue == "" {
			paramValue = c.Param(paramName)
		}

		if paramValue != "" {
			targetURL = strings.ReplaceAll(targetURL, "{"+paramName+"}", paramValue)
		}
	}

	return targetURL
}

func (h *Handler) copyHeaders(srcReq *http.Request, destReq *http.Request, paramMappings []config.MappingConfig) {
	// Copy standard headers
	headersToForward := []string{
		"Authorization",
		"Content-Type",
		"Accept",
		"User-Agent",
		"X-Forwarded-For",
		"X-Real-IP",
	}

	for _, header := range headersToForward {
		if value := srcReq.Header.Get(header); value != "" {
			destReq.Header.Set(header, value)
		}
	}

	// Apply header mappings from parameter configuration
	for _, mapping := range paramMappings {
		if strings.HasPrefix(mapping.From, "$.headers.") {
			headerName := strings.TrimPrefix(mapping.From, "$.headers.")
			if value := srcReq.Header.Get(headerName); value != "" {
				destReq.Header.Set(mapping.To, value)
			}
		}
	}
}

func (h *Handler) mapDependencyResponse(data interface{}, mappings []config.MappingConfig) interface{} {
	if len(mappings) == 0 {
		return data
	}

	// Convert to map for easier manipulation
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return data
	}

	result := make(map[string]interface{})

	// Apply each mapping
	for _, mapping := range mappings {
		fromPath := strings.TrimPrefix(mapping.From, "$.")
		toPath := strings.TrimPrefix(mapping.To, "$.")

		// If from path is root ($), copy entire object
		if mapping.From == "$" {
			if toPath == "" {
				return dataMap
			}
			setNestedValue(result, strings.Split(toPath, "."), dataMap)
		} else {
			// Extract value from source path
			value := getNestedValue(dataMap, strings.Split(fromPath, "."))
			if value != nil {
				if toPath == "" {
					return value
				}
				setNestedValue(result, strings.Split(toPath, "."), value)
			}
		}
	}

	// If no mappings resulted in data, return original
	if len(result) == 0 {
		return dataMap
	}

	return result
}
