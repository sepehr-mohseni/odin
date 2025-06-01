package aggregator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"odin/pkg/config"
	"strings"
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
	ctx := c.Request().Context()

	// Get query parameters for services to aggregate
	serviceNames := c.QueryParam("services")
	if serviceNames == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "services parameter is required")
	}

	services := strings.Split(serviceNames, ",")
	responses := make(map[string]*ServiceResponse)

	// Use sync.WaitGroup for concurrent requests
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, serviceName := range services {
		serviceName = strings.TrimSpace(serviceName)
		if serviceConfig, exists := a.serviceConfigs[serviceName]; exists {
			wg.Add(1)
			go func(svcName string, svcConfig *config.ServiceConfig) {
				defer wg.Done()

				response := a.callService(ctx, c, svcConfig)

				mu.Lock()
				responses[svcName] = response
				mu.Unlock()
			}(serviceName, &serviceConfig) // Fix: take address of serviceConfig
		}
	}

	wg.Wait()

	// Build aggregated response
	aggregatedResponse := map[string]interface{}{
		"services":  responses,
		"timestamp": time.Now().Unix(),
	}

	return c.JSON(http.StatusOK, aggregatedResponse)
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

func (a *Aggregator) EnrichResponse(ctx context.Context, serviceName string, responseBody []byte, headers http.Header, authToken string) ([]byte, error) {
	serviceConfig, exists := a.serviceConfigs[serviceName]
	if !exists || serviceConfig.Aggregation == nil {
		return responseBody, nil
	}

	var originalResponse map[string]interface{}
	if err := json.Unmarshal(responseBody, &originalResponse); err != nil {
		return responseBody, nil
	}

	enrichedResponse := make(map[string]interface{})

	// Copy original response
	for k, v := range originalResponse {
		enrichedResponse[k] = v
	}

	// Fetch and merge dependency data
	for _, dep := range serviceConfig.Aggregation.Dependencies {
		depData, err := a.fetchDependencyDataForEnrichment(ctx, dep, originalResponse, authToken)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to fetch dependency data from %s", dep.Service)
			continue
		}

		// Apply result mappings
		if len(dep.ResultMapping) > 0 {
			for _, mapping := range dep.ResultMapping {
				toPath := strings.TrimPrefix(mapping.To, "$.")
				if toPath != "" {
					setNestedValue(enrichedResponse, strings.Split(toPath, "."), depData)
				}
			}
		} else {
			// If no mapping specified, use service name as key
			enrichedResponse[dep.Service] = depData
		}
	}

	return json.Marshal(enrichedResponse)
}

func (a *Aggregator) fetchDependencyDataForEnrichment(ctx context.Context, dep config.DependencyConfig, originalResponse map[string]interface{}, authToken string) (interface{}, error) {
	targetURL := dep.Path

	// If we have a service configuration, use its first target as base
	if serviceConfig, exists := a.serviceConfigs[dep.Service]; exists && len(serviceConfig.Targets) > 0 {
		baseURL := serviceConfig.Targets[0]
		if !strings.HasSuffix(baseURL, "/") && !strings.HasPrefix(dep.Path, "/") {
			targetURL = baseURL + "/" + dep.Path
		} else {
			targetURL = baseURL + dep.Path
		}
	}

	// Replace parameters in the URL using original response data
	for _, mapping := range dep.ParameterMapping {
		paramName := strings.TrimPrefix(mapping.To, "{")
		paramName = strings.TrimSuffix(paramName, "}")

		// Extract parameter value from original response
		fromPath := strings.TrimPrefix(mapping.From, "$.")
		paramValue, _ := getNestedValue(originalResponse, strings.Split(fromPath, "."))

		if paramValue != nil {
			targetURL = strings.ReplaceAll(targetURL, "{"+paramName+"}", fmt.Sprintf("%v", paramValue))
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

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

	return a.mapResponseData(data, dep.ResultMapping), nil
}

func (a *Aggregator) mapResponseData(data interface{}, mappings []config.MappingConfig) interface{} {
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
