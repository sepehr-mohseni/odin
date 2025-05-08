package aggregator

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"odin/pkg/config"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Aggregator struct {
	logger         *logrus.Logger
	serviceConfigs map[string]config.ServiceConfig
	client         *http.Client
}

type ServiceResponse struct {
	Service string                 `json:"service"`
	Status  int                    `json:"status"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error,omitempty"`
}

type AggregateResponse struct {
	Success   bool                        `json:"success"`
	Timestamp string                      `json:"timestamp"`
	Results   map[string]*ServiceResponse `json:"results"`
}

func New(logger *logrus.Logger, services []config.ServiceConfig) *Aggregator {
	serviceMap := make(map[string]config.ServiceConfig)
	for _, svc := range services {
		serviceMap[svc.Name] = svc
	}

	return &Aggregator{
		logger:         logger,
		serviceConfigs: serviceMap,
		client:         &http.Client{},
	}
}

func (a *Aggregator) RegisterRoutes(e *echo.Echo) {
	e.GET("/aggregate", a.AggregateHandler)
	e.POST("/aggregate", a.AggregateHandler)
}

func (a *Aggregator) AggregateHandler(c echo.Context) error {
	var services []string
	if servicesParam := c.QueryParam("services"); servicesParam != "" {
		if err := json.Unmarshal([]byte(servicesParam), &services); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid services parameter")
		}
	}

	if len(services) == 0 {
		for serviceName := range a.serviceConfigs {
			services = append(services, serviceName)
		}
	}

	response := AggregateResponse{
		Success:   true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Results:   make(map[string]*ServiceResponse),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	timeout := 10 * time.Second
	if timeoutParam := c.QueryParam("timeout"); timeoutParam != "" {
		if t, err := time.ParseDuration(timeoutParam); err == nil && t > 0 {
			timeout = t
		}
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), timeout)
	defer cancel()

	for _, serviceName := range services {
		serviceConfig, exists := a.serviceConfigs[serviceName]
		if !exists {
			response.Results[serviceName] = &ServiceResponse{
				Service: serviceName,
				Status:  http.StatusNotFound,
				Error:   "Service not found",
			}
			continue
		}

		wg.Add(1)
		go func(svc config.ServiceConfig) {
			defer wg.Done()
			serviceResp := a.callService(ctx, c, &svc)
			mu.Lock()
			response.Results[svc.Name] = serviceResp
			mu.Unlock()
		}(serviceConfig)
	}

	wg.Wait()
	return c.JSON(http.StatusOK, response)
}

func (a *Aggregator) callService(ctx context.Context, c echo.Context, svc *config.ServiceConfig) *ServiceResponse {
	if len(svc.Targets) == 0 {
		return &ServiceResponse{
			Service: svc.Name,
			Status:  http.StatusServiceUnavailable,
			Error:   "No targets available",
		}
	}

	targetURL := svc.Targets[0]
	endpoint := c.QueryParam("endpoint")
	if endpoint == "" {
		endpoint = "/"
	}
	url := targetURL + endpoint

	req, err := http.NewRequestWithContext(ctx, c.Request().Method, url, nil)
	if err != nil {
		return &ServiceResponse{
			Service: svc.Name,
			Status:  http.StatusInternalServerError,
			Error:   "Failed to create request",
		}
	}

	for k, vals := range c.Request().Header {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	for k, v := range svc.Headers {
		req.Header.Set(k, v)
	}

	if (c.Request().Method == http.MethodPost || c.Request().Method == http.MethodPut) && c.Request().Body != nil {
		bodyBytes, _ := io.ReadAll(c.Request().Body)
		c.Request().Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
		req.Header.Set("Content-Type", c.Request().Header.Get("Content-Type"))
	}

	client := &http.Client{
		Timeout: svc.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return &ServiceResponse{
			Service: svc.Name,
			Status:  http.StatusServiceUnavailable,
			Error:   "Request failed: " + err.Error(),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ServiceResponse{
			Service: svc.Name,
			Status:  http.StatusInternalServerError,
			Error:   "Failed to read response",
		}
	}

	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		return &ServiceResponse{
			Service: svc.Name,
			Status:  resp.StatusCode,
			Data:    map[string]interface{}{"raw": string(respBody)},
		}
	}

	return &ServiceResponse{
		Service: svc.Name,
		Status:  resp.StatusCode,
		Data:    respData,
	}
}
