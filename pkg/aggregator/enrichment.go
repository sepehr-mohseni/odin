package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"odin/pkg/config"
	"strings"
	"time"
)

// EnrichmentService handles response enrichment with dependency data
type EnrichmentService struct {
	aggregator *Aggregator
}

// NewEnrichmentService creates a new enrichment service
func NewEnrichmentService(aggregator *Aggregator) *EnrichmentService {
	return &EnrichmentService{
		aggregator: aggregator,
	}
}

// EnrichResponse enriches a response with data from dependencies
func (e *EnrichmentService) EnrichResponse(ctx context.Context, serviceName string, responseBody []byte, headers http.Header, authToken string) ([]byte, error) {
	serviceConfig, exists := e.aggregator.serviceConfigs[serviceName]
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
		depData, err := e.fetchDependencyData(ctx, dep, originalResponse, authToken)
		if err != nil {
			e.aggregator.logger.WithError(err).Warnf("Failed to fetch dependency data from %s", dep.Service)
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

// fetchDependencyData fetches data from a dependency service
func (e *EnrichmentService) fetchDependencyData(ctx context.Context, dep config.DependencyConfig, originalResponse map[string]interface{}, authToken string) (interface{}, error) {
	targetURL := dep.Path

	// If we have a service configuration, use its first target as base
	if serviceConfig, exists := e.aggregator.serviceConfigs[dep.Service]; exists && len(serviceConfig.Targets) > 0 {
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
		paramValue, found := getNestedValue(originalResponse, strings.Split(fromPath, "."))

		if found && paramValue != nil {
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

	return e.aggregator.mapResponseData(data, dep.ResultMapping), nil
}
