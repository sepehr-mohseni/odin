package routing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"odin/pkg/cache"
	"odin/pkg/service"
	"strings"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type ServiceHandler struct {
	service    *service.Config
	logger     *logrus.Logger
	cacheStore cache.Store
	client     *http.Client
	nextTarget uint64
}

func NewServiceHandler(svc *service.Config, logger *logrus.Logger, cacheStore cache.Store) (*ServiceHandler, error) {
	if len(svc.Targets) == 0 {
		return nil, fmt.Errorf("service %s has no targets", svc.Name)
	}

	client := &http.Client{
		Timeout: svc.Timeout,
	}

	return &ServiceHandler{
		service:    svc,
		logger:     logger,
		cacheStore: cacheStore,
		client:     client,
		nextTarget: 0,
	}, nil
}

func (h *ServiceHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	target := h.getTargetURL()
	path := c.Request().URL.Path

	if h.service.StripBasePath && strings.HasPrefix(path, h.service.BasePath) {
		path = strings.TrimPrefix(path, h.service.BasePath)
		if path == "" {
			path = "/"
		}
	}

	targetURL := target + path
	if c.Request().URL.RawQuery != "" {
		targetURL += "?" + c.Request().URL.RawQuery
	}

	h.logger.WithFields(logrus.Fields{
		"service": h.service.Name,
		"target":  targetURL,
		"method":  c.Request().Method,
	}).Debug("Forwarding request")

	req, err := h.createProxyRequest(c, targetURL)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create proxy request")
	}

	resp, err := h.doRequestWithRetries(ctx, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "Service unavailable")
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		for _, v := range vals {
			c.Response().Header().Add(k, v)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read response body")
	}

	if h.service.Aggregation != nil {
		// Initialize aggregation handler
		h.logger.Debug("Aggregation config found but not processed yet")
	}

	c.Response().WriteHeader(resp.StatusCode)
	_, err = c.Response().Write(body)
	return err
}

func (h *ServiceHandler) getTargetURL() string {
	if len(h.service.Targets) == 1 {
		return h.service.Targets[0]
	}

	switch h.service.LoadBalancing {
	case "random":
		idx := time.Now().UnixNano() % int64(len(h.service.Targets))
		return h.service.Targets[idx]
	default:
		idx := atomic.AddUint64(&h.nextTarget, 1) % uint64(len(h.service.Targets))
		return h.service.Targets[idx]
	}
}

func (h *ServiceHandler) createProxyRequest(c echo.Context, targetURL string) (*http.Request, error) {
	var body io.Reader = nil

	if c.Request().Body != nil {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		body = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(c.Request().Context(), c.Request().Method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	for k, v := range c.Request().Header {
		req.Header[k] = v
	}

	return req, nil
}

func (h *ServiceHandler) doRequestWithRetries(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i <= h.service.RetryCount; i++ {
		resp, err = h.client.Do(req.WithContext(ctx))
		if err == nil {
			return resp, nil
		}

		if i < h.service.RetryCount {
			h.logger.WithError(err).Warnf("Request to %s failed, retrying (%d/%d)",
				req.URL.String(), i+1, h.service.RetryCount)
			time.Sleep(h.service.RetryDelay)

			body, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
	}

	return nil, err
}
