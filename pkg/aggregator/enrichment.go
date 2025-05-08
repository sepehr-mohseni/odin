package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"odin/pkg/config"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

func (a *Aggregator) EnrichResponse(ctx context.Context, serviceName string, responseBody []byte, headers http.Header, authToken string) ([]byte, error) {
	svcConfig, exists := a.serviceConfigs[serviceName]
	if !exists {
		a.logger.WithField("service", serviceName).Info("No service config found for enrichment")
		return responseBody, nil
	}

	if svcConfig.Aggregation == nil || len(svcConfig.Aggregation.Dependencies) == 0 {
		a.logger.WithField("service", serviceName).Debug("No aggregation config found")
		return responseBody, nil
	}

	if len(responseBody) > 0 && responseBody[0] == '<' {
		a.logger.WithFields(logrus.Fields{
			"service":      serviceName,
			"body_preview": string(responseBody[:min(100, len(responseBody))]),
		}).Warn("Received HTML instead of JSON")
		return responseBody, fmt.Errorf("received HTML instead of JSON from service %s", serviceName)
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		a.logger.WithError(err).WithFields(logrus.Fields{
			"service":      serviceName,
			"body_length":  len(responseBody),
			"body_preview": string(responseBody[:min(100, len(responseBody))]),
		}).Error("Failed to parse response for aggregation")
		return responseBody, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var enrichmentErrors []error

	for _, dep := range svcConfig.Aggregation.Dependencies {
		depSvcConfig, exists := a.serviceConfigs[dep.Service]
		if !exists {
			a.logger.Warnf("Dependent service %s not found in config", dep.Service)
			continue
		}

		pathParams := make(map[string][]string)
		for _, mapping := range dep.ParameterMapping {
			values := a.extractValues(responseData, mapping.From)
			if len(values) > 0 {
				pathParams[mapping.To] = values
				a.logger.WithFields(logrus.Fields{
					"from":   mapping.From,
					"to":     mapping.To,
					"values": values,
				}).Debug("Extracted parameter values")
			} else {
				a.logger.WithFields(logrus.Fields{
					"from":    mapping.From,
					"service": serviceName,
				}).Warn("No parameter values extracted")
			}
		}

		if len(pathParams) == 0 {
			a.logger.WithFields(logrus.Fields{
				"service":    serviceName,
				"dependency": dep.Service,
			}).Warn("No parameters extracted for dependency")
			continue
		}

		for paramName, paramValues := range pathParams {
			for _, paramValue := range paramValues {
				wg.Add(1)

				go func(paramName, paramValue string, dependency *config.DependencyConfig) {
					defer wg.Done()

					depPath := dependency.Path
					paramPattern := fmt.Sprintf("{%s}", paramName)
					depPath = strings.Replace(depPath, paramPattern, paramValue, -1)

					target := depSvcConfig.Targets[0]
					depURL := fmt.Sprintf("%s%s", target, depPath)

					a.logger.WithFields(logrus.Fields{
						"url":        depURL,
						"dependency": dependency.Service,
						"param":      fmt.Sprintf("%s=%s", paramName, paramValue),
					}).Debug("Making dependency request")

					depResp, err := a.makeRequest(ctx, depURL, authToken)
					if err != nil {
						mu.Lock()
						enrichmentErrors = append(enrichmentErrors, fmt.Errorf("aggregation request to %s failed: %w", depURL, err))
						mu.Unlock()
						a.logger.WithError(err).Errorf("Aggregation request to %s failed", depURL)
						return
					}

					if len(depResp) > 0 && depResp[0] == '<' {
						mu.Lock()
						enrichmentErrors = append(enrichmentErrors, fmt.Errorf("received HTML from %s", depURL))
						mu.Unlock()
						a.logger.WithFields(logrus.Fields{
							"url":          depURL,
							"body_preview": string(depResp[:min(100, len(depResp))]),
						}).Error("Dependency returned HTML instead of JSON")
						return
					}

					var depData map[string]interface{}
					if err := json.Unmarshal(depResp, &depData); err != nil {
						mu.Lock()
						enrichmentErrors = append(enrichmentErrors, fmt.Errorf("failed to parse response from %s: %w", depURL, err))
						mu.Unlock()
						a.logger.WithError(err).WithFields(logrus.Fields{
							"url":          depURL,
							"body_preview": string(depResp[:min(100, len(depResp))]),
						}).Error("Failed to parse dependency response")
						return
					}

					mu.Lock()
					for _, resultMapping := range dependency.ResultMapping {
						mappingTo := resultMapping.To
						mappingTo = strings.Replace(mappingTo, "{"+paramName+"}", paramValue, -1)

						a.logger.WithFields(logrus.Fields{
							"from":        resultMapping.From,
							"to":          mappingTo,
							"param_value": paramValue,
						}).Debug("Applying result mapping")
						a.mapData(responseData, depData, resultMapping.From, mappingTo, paramValue, paramName)
					}
					mu.Unlock()
				}(paramName, paramValue, &dep)
			}
		}
	}

	wg.Wait()

	keysToDelete := []string{}
	for k := range responseData {
		if strings.Contains(k, "?(") || strings.Contains(k, "items[?(@") {
			keysToDelete = append(keysToDelete, k)
		}
	}

	for _, k := range keysToDelete {
		a.logger.WithField("key", k).Info("Removing invalid JSONPath key")
		delete(responseData, k)
	}

	if len(enrichmentErrors) > 0 {
		a.logger.WithField("error_count", len(enrichmentErrors)).Warn("Encountered errors during enrichment")
	}

	enrichedResponse, err := json.Marshal(responseData)
	if err != nil {
		a.logger.WithError(err).Error("Failed to marshal enriched response")
		return responseBody, err
	}

	return enrichedResponse, nil
}
