package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Alerter handles sending alerts for detected anomalies
type Alerter struct {
	config     *AIConfig
	repository Repository
	logger     *logrus.Logger
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAlerter creates a new alerter
func NewAlerter(config *AIConfig, repository Repository, logger *logrus.Logger) *Alerter {
	ctx, cancel := context.WithCancel(context.Background())

	return &Alerter{
		config:     config,
		repository: repository,
		logger:     logger,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the alerter loop
func (a *Alerter) Start() {
	if !a.config.EnableAlerts {
		a.logger.Info("Alerts disabled, alerter not started")
		return
	}

	go a.alertLoop()
	a.logger.Info("Alerter started")
}

// alertLoop processes pending alerts
func (a *Alerter) alertLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			if err := a.processPendingAlerts(); err != nil {
				a.logger.WithError(err).Error("Failed to process pending alerts")
			}
		}
	}
}

// processPendingAlerts sends all pending alerts
func (a *Alerter) processPendingAlerts() error {
	alerts, err := a.repository.GetPendingAlerts(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending alerts: %w", err)
	}

	if len(alerts) == 0 {
		return nil
	}

	a.logger.WithField("count", len(alerts)).Info("Processing pending alerts")

	for _, alert := range alerts {
		if err := a.sendAlert(alert); err != nil {
			a.logger.WithError(err).WithField("alert_id", alert.ID).Error("Failed to send alert")
			continue
		}

		// Mark as sent
		if err := a.repository.MarkAlertSent(a.ctx, alert.ID); err != nil {
			a.logger.WithError(err).WithField("alert_id", alert.ID).Error("Failed to mark alert as sent")
		}
	}

	return nil
}

// sendAlert sends an alert via configured channels
func (a *Alerter) sendAlert(alert *Alert) error {
	for _, channel := range alert.Channels {
		switch channel {
		case "webhook":
			if err := a.sendWebhookAlert(alert); err != nil {
				return fmt.Errorf("webhook failed: %w", err)
			}
		case "email":
			// Email implementation would go here
			a.logger.WithField("alert_id", alert.ID).Warn("Email alerts not yet implemented")
		case "slack":
			// Slack implementation would go here
			a.logger.WithField("alert_id", alert.ID).Warn("Slack alerts not yet implemented")
		default:
			a.logger.WithField("channel", channel).Warn("Unknown alert channel")
		}
	}

	return nil
}

// sendWebhookAlert sends an alert via webhook
func (a *Alerter) sendWebhookAlert(alert *Alert) error {
	if a.config.AlertWebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	payload := map[string]interface{}{
		"id":           alert.ID,
		"timestamp":    alert.Timestamp,
		"severity":     alert.Severity,
		"title":        alert.Title,
		"message":      alert.Message,
		"service_name": alert.ServiceName,
		"endpoint":     alert.Endpoint,
		"anomaly_id":   alert.AnomalyID,
		"tags":         alert.Tags,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(a.ctx, "POST", a.config.AlertWebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	a.logger.WithFields(logrus.Fields{
		"alert_id": alert.ID,
		"severity": alert.Severity,
	}).Info("Alert sent successfully via webhook")

	return nil
}

// Stop stops the alerter
func (a *Alerter) Stop() {
	a.cancel()
	a.logger.Info("Alerter stopped")
}
