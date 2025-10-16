package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// GrokClient handles communication with Grok inference service
type GrokClient struct {
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewGrokClient creates a new Grok client
func NewGrokClient(baseURL string, timeout time.Duration, logger *logrus.Logger) *GrokClient {
	return &GrokClient{
		baseURL: baseURL,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// Analyze sends a request to Grok for traffic analysis
func (gc *GrokClient) Analyze(ctx context.Context, request *GrokRequest) (*GrokResponse, error) {
	gc.logger.WithField("prompt_length", len(request.Prompt)).Debug("Sending request to Grok")

	// Marshal request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := gc.baseURL + "/analyze"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	startTime := time.Now()
	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)
	gc.logger.WithFields(logrus.Fields{
		"status":   resp.StatusCode,
		"duration": duration,
	}).Debug("Received response from Grok")

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("grok returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var grokResp GrokResponse
	if err := json.NewDecoder(resp.Body).Decode(&grokResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &grokResp, nil
}

// HealthCheck checks if Grok service is available
func (gc *GrokClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", gc.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("grok service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
