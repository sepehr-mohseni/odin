package proxy

import (
	"bytes"
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
