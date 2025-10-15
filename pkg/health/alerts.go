package health

import (
"bytes"
"encoding/json"
"fmt"
"net/http"
"sync"
"time"

"github.com/sirupsen/logrus"
)

// AlertType represents the type of alert
type AlertType string

const (
AlertTypeTargetDown       AlertType = "target_down"
AlertTypeTargetRecovered  AlertType = "target_recovered"
AlertTypeHighErrorRate    AlertType = "high_error_rate"
AlertTypeSlowResponse     AlertType = "slow_response"
)

// Severity represents alert severity
type Severity string

const (
SeverityInfo     Severity = "info"
SeverityWarning  Severity = "warning"
SeverityCritical Severity = "critical"
)

// Alert represents an alert to be sent
type Alert struct {
Type      AlertType              `json:"type"`
Severity  Severity               `json:"severity"`
Target    string                 `json:"target"`
Message   string                 `json:"message"`
Timestamp time.Time              `json:"timestamp"`
Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertChannel defines how to send alerts
type AlertChannel interface {
Send(alert Alert) error
Name() string
}

// WebhookChannel sends alerts via HTTP webhook
type WebhookChannel struct {
URL    string
client *http.Client
logger *logrus.Logger
}

// NewWebhookChannel creates a new webhook alert channel
func NewWebhookChannel(url string, logger *logrus.Logger) *WebhookChannel {
return &WebhookChannel{
URL: url,
client: &http.Client{
Timeout: 10 * time.Second,
},
logger: logger,
}
}

func (w *WebhookChannel) Name() string {
return "webhook"
}

func (w *WebhookChannel) Send(alert Alert) error {
payload, err := json.Marshal(alert)
if err != nil {
return fmt.Errorf("failed to marshal alert: %w", err)
}

resp, err := w.client.Post(w.URL, "application/json", bytes.NewBuffer(payload))
if err != nil {
return fmt.Errorf("failed to send webhook: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode < 200 || resp.StatusCode >= 300 {
return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

w.logger.WithFields(logrus.Fields{
"url":      w.URL,
"type":     alert.Type,
"severity": alert.Severity,
}).Debug("Alert sent via webhook")

return nil
}

// LogChannel logs alerts to the logger
type LogChannel struct {
logger *logrus.Logger
}

// NewLogChannel creates a new log alert channel
func NewLogChannel(logger *logrus.Logger) *LogChannel {
return &LogChannel{
logger: logger,
}
}

func (l *LogChannel) Name() string {
return "log"
}

func (l *LogChannel) Send(alert Alert) error {
fields := logrus.Fields{
"type":     alert.Type,
"severity": alert.Severity,
"target":   alert.Target,
}

if alert.Metadata != nil {
for k, v := range alert.Metadata {
fields[k] = v
}
}

switch alert.Severity {
case SeverityCritical:
l.logger.WithFields(fields).Error(alert.Message)
case SeverityWarning:
l.logger.WithFields(fields).Warn(alert.Message)
default:
l.logger.WithFields(fields).Info(alert.Message)
}

return nil
}

// AlertManager manages alert channels and sends alerts
type AlertManager struct {
mu            sync.RWMutex
channels      []AlertChannel
logger        *logrus.Logger
alertQueue    chan Alert
stopChan      chan struct{}
wg            sync.WaitGroup
minInterval   time.Duration
lastAlerts    map[string]time.Time
lastAlertsMu  sync.RWMutex
}

// NewAlertManager creates a new alert manager
func NewAlertManager(logger *logrus.Logger) *AlertManager {
return &AlertManager{
channels:    make([]AlertChannel, 0),
logger:      logger,
alertQueue:  make(chan Alert, 100),
stopChan:    make(chan struct{}),
minInterval: 5 * time.Minute, // Minimum 5 minutes between same alerts
lastAlerts:  make(map[string]time.Time),
}
}

// AddChannel adds an alert channel
func (am *AlertManager) AddChannel(channel AlertChannel) {
am.mu.Lock()
defer am.mu.Unlock()
am.channels = append(am.channels, channel)
am.logger.WithField("channel", channel.Name()).Info("Added alert channel")
}

// Start begins processing alerts
func (am *AlertManager) Start() {
am.wg.Add(1)
go am.processAlerts()
am.logger.Info("Alert manager started")
}

// Stop stops processing alerts
func (am *AlertManager) Stop() {
close(am.stopChan)
close(am.alertQueue)
am.wg.Wait()
am.logger.Info("Alert manager stopped")
}

// SendAlert queues an alert to be sent
func (am *AlertManager) SendAlert(alert Alert) {
// Check if we should throttle this alert
if am.shouldThrottle(alert) {
am.logger.WithFields(logrus.Fields{
"type":   alert.Type,
"target": alert.Target,
}).Debug("Alert throttled")
return
}

select {
case am.alertQueue <- alert:
default:
am.logger.Warn("Alert queue full, dropping alert")
}
}

// shouldThrottle checks if an alert should be throttled
func (am *AlertManager) shouldThrottle(alert Alert) bool {
am.lastAlertsMu.Lock()
defer am.lastAlertsMu.Unlock()

key := fmt.Sprintf("%s:%s", alert.Type, alert.Target)
lastSent, exists := am.lastAlerts[key]

if !exists {
am.lastAlerts[key] = time.Now()
return false
}

if time.Since(lastSent) < am.minInterval {
return true
}

am.lastAlerts[key] = time.Now()
return false
}

// processAlerts processes alerts from the queue
func (am *AlertManager) processAlerts() {
defer am.wg.Done()

for {
select {
case alert, ok := <-am.alertQueue:
if !ok {
return
}
am.sendToChannels(alert)
case <-am.stopChan:
return
}
}
}

// sendToChannels sends an alert to all channels
func (am *AlertManager) sendToChannels(alert Alert) {
am.mu.RLock()
channels := make([]AlertChannel, len(am.channels))
copy(channels, am.channels)
am.mu.RUnlock()

for _, channel := range channels {
go func(ch AlertChannel) {
if err := ch.Send(alert); err != nil {
am.logger.WithFields(logrus.Fields{
"channel": ch.Name(),
"error":   err,
}).Error("Failed to send alert")
}
}(channel)
}
}
