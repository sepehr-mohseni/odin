package aggregator

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

func (a *Aggregator) makeRequest(ctx context.Context, url, authToken string) ([]byte, error) {
	a.logger.WithField("url", url).Info("Making HTTP request")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		a.logger.WithError(err).WithField("url", url).Error("Request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	a.logger.WithFields(logrus.Fields{
		"url":          url,
		"status":       resp.StatusCode,
		"body_length":  len(body),
		"body_preview": string(body[:min(100, len(body))]),
	}).Info("Received response")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dependency request failed with status %d: %s",
			resp.StatusCode, string(body[:min(100, len(body))]))
	}

	return body, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
