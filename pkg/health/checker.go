package health

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TargetStatus represents the health status of a backend target
type TargetStatus string

const (
	TargetStatusHealthy   TargetStatus = "healthy"
	TargetStatusUnhealthy TargetStatus = "unhealthy"
	TargetStatusDegraded  TargetStatus = "degraded"
)

// TargetHealth tracks the health status of a single backend target
type TargetHealth struct {
	URL                 string
	Status              TargetStatus
	LastCheck           time.Time
	ConsecutiveFails    int
	ConsecutivePasses   int
	LastError           string
	ResponseTime        time.Duration
	TotalChecks         int64
	SuccessfulChecks    int64
	FailedChecks        int64
	AverageResponseTime time.Duration
}

// Config holds health checker configuration
type Config struct {
	Interval           time.Duration // How often to check health
	Timeout            time.Duration // Timeout for each health check
	UnhealthyThreshold int           // Number of consecutive failures before marking unhealthy
	HealthyThreshold   int           // Number of consecutive successes before marking healthy
	ExpectedStatus     []int         // Expected HTTP status codes (default: 200)
	InsecureSkipVerify bool          // Skip TLS verification
}

// TargetChecker performs active health checks on backend targets
type TargetChecker struct {
	config   Config
	targets  map[string]*TargetHealth
	mu       sync.RWMutex
	logger   *logrus.Logger
	alerts   *AlertManager
	stopChan chan struct{}
	wg       sync.WaitGroup
	client   *http.Client
}

// NewTargetChecker creates a new health checker for backend targets
func NewTargetChecker(config Config, logger *logrus.Logger, alerts *AlertManager) *TargetChecker {
	// Set defaults
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.UnhealthyThreshold == 0 {
		config.UnhealthyThreshold = 3
	}
	if config.HealthyThreshold == 0 {
		config.HealthyThreshold = 2
	}
	if len(config.ExpectedStatus) == 0 {
		config.ExpectedStatus = []int{200, 204}
	}

	return &TargetChecker{
		config:   config,
		targets:  make(map[string]*TargetHealth),
		logger:   logger,
		alerts:   alerts,
		stopChan: make(chan struct{}),
		client: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: config.InsecureSkipVerify,
				},
			},
		},
	}
}

// AddTarget adds a new target to monitor
func (c *TargetChecker) AddTarget(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.targets[url]; !exists {
		c.targets[url] = &TargetHealth{
			URL:    url,
			Status: TargetStatusHealthy, // Start optimistic
		}
		c.logger.WithField("url", url).Info("Added target for health monitoring")
	}
}

// RemoveTarget removes a target from monitoring
func (c *TargetChecker) RemoveTarget(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.targets, url)
	c.logger.WithField("url", url).Info("Removed target from health monitoring")
}

// Start begins the health checking loop
func (c *TargetChecker) Start() {
	c.logger.Info("Starting health checker")

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		ticker := time.NewTicker(c.config.Interval)
		defer ticker.Stop()

		// Do initial check immediately
		c.checkAll()

		for {
			select {
			case <-ticker.C:
				c.checkAll()
			case <-c.stopChan:
				return
			}
		}
	}()
}

// Stop halts the health checking loop
func (c *TargetChecker) Stop() {
	c.logger.Info("Stopping health checker")
	close(c.stopChan)
	c.wg.Wait()
}

// checkAll performs health checks on all targets concurrently
func (c *TargetChecker) checkAll() {
	c.mu.RLock()
	urls := make([]string, 0, len(c.targets))
	for url := range c.targets {
		urls = append(urls, url)
	}
	c.mu.RUnlock()

	// Check all targets concurrently
	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()

			success, responseTime, err := c.checkTarget(targetURL)
			c.updateTargetHealth(targetURL, success, responseTime, err)
		}(url)
	}
	wg.Wait()
}

// checkTarget performs a health check on a single target
func (c *TargetChecker) checkTarget(url string) (bool, time.Duration, error) {
	start := time.Now()

	// Build health check URL (append /health if not present)
	healthURL := url
	if healthURL[len(healthURL)-1] != '/' {
		healthURL += "/"
	}
	healthURL += "health"

	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		return false, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, time.Since(start), fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	responseTime := time.Since(start)

	// Check if status code is expected
	for _, expectedStatus := range c.config.ExpectedStatus {
		if resp.StatusCode == expectedStatus {
			return true, responseTime, nil
		}
	}

	return false, responseTime, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// updateTargetHealth updates the health status of a target based on check result
func (c *TargetChecker) updateTargetHealth(url string, success bool, responseTime time.Duration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	target, exists := c.targets[url]
	if !exists {
		return
	}

	// Update metrics
	target.LastCheck = time.Now()
	target.ResponseTime = responseTime
	target.TotalChecks++

	// Update average response time
	if target.TotalChecks > 1 {
		target.AverageResponseTime = time.Duration(
			(int64(target.AverageResponseTime)*(target.TotalChecks-1) + int64(responseTime)) / target.TotalChecks,
		)
	} else {
		target.AverageResponseTime = responseTime
	}

	// Update status based on consecutive failures/passes
	oldStatus := target.Status

	if success {
		target.ConsecutiveFails = 0
		target.ConsecutivePasses++
		target.SuccessfulChecks++

		if target.Status != TargetStatusHealthy && target.ConsecutivePasses >= c.config.HealthyThreshold {
			target.Status = TargetStatusHealthy
			c.logger.WithFields(logrus.Fields{
				"url":    url,
				"passes": target.ConsecutivePasses,
			}).Info("Target recovered to healthy")
		}
	} else {
		target.ConsecutivePasses = 0
		target.ConsecutiveFails++
		target.FailedChecks++
		target.LastError = err.Error()

		if target.Status == TargetStatusHealthy && target.ConsecutiveFails >= c.config.UnhealthyThreshold {
			target.Status = TargetStatusUnhealthy
			c.logger.WithFields(logrus.Fields{
				"url":   url,
				"fails": target.ConsecutiveFails,
				"error": err.Error(),
			}).Warn("Target marked as unhealthy")
		}
	}

	// Send alerts on status changes
	if oldStatus != target.Status {
		if target.Status == TargetStatusUnhealthy {
			c.alerts.SendAlert(Alert{
				Type:      AlertTypeTargetDown,
				Severity:  SeverityCritical,
				Target:    url,
				Message:   fmt.Sprintf("Target %s is down", url),
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"error": target.LastError,
				},
			})
		} else if target.Status == TargetStatusHealthy && oldStatus == TargetStatusUnhealthy {
			c.alerts.SendAlert(Alert{
				Type:      AlertTypeTargetRecovered,
				Severity:  SeverityInfo,
				Target:    url,
				Message:   fmt.Sprintf("Target %s has recovered", url),
				Timestamp: time.Now(),
			})
		}
	}
}

// IsHealthy returns whether a specific target is healthy
func (c *TargetChecker) IsHealthy(url string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if target, exists := c.targets[url]; exists {
		return target.Status == TargetStatusHealthy
	}
	return true // Assume healthy if not monitored
}

// GetTargetHealth returns the health status of a specific target
func (c *TargetChecker) GetTargetHealth(url string) *TargetHealth {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if target, exists := c.targets[url]; exists {
		// Return a copy to avoid race conditions
		health := *target
		return &health
	}
	return nil
}

// GetAllTargetsHealth returns health status for all monitored targets
func (c *TargetChecker) GetAllTargetsHealth() map[string]*TargetHealth {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*TargetHealth)
	for url, target := range c.targets {
		health := *target
		result[url] = &health
	}
	return result
}
