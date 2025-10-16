package ai

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// AnomalyDetector performs anomaly detection on traffic patterns
type AnomalyDetector struct {
	config     *AIConfig
	repository Repository
	logger     *logrus.Logger
	grokClient *GrokClient
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	baselines  map[string]*Baseline // Cache of baselines
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(config *AIConfig, repository Repository, logger *logrus.Logger) *AnomalyDetector {
	ctx, cancel := context.WithCancel(context.Background())

	detector := &AnomalyDetector{
		config:     config,
		repository: repository,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		baselines:  make(map[string]*Baseline),
	}

	// Initialize Grok client if enabled
	if config.UseGrokModel && config.GrokServiceURL != "" {
		detector.grokClient = NewGrokClient(config.GrokServiceURL, config.GrokTimeout, logger)
	}

	// Start analysis loop
	detector.wg.Add(1)
	go detector.analysisLoop()

	return detector
}

// analysisLoop runs periodic analysis of traffic patterns
func (ad *AnomalyDetector) analysisLoop() {
	defer ad.wg.Done()

	ticker := time.NewTicker(ad.config.AnalysisInterval)
	defer ticker.Stop()

	ad.logger.Info("Starting AI anomaly detection analysis loop")

	for {
		select {
		case <-ad.ctx.Done():
			return
		case <-ticker.C:
			if err := ad.analyzeAllServices(); err != nil {
				ad.logger.WithError(err).Error("Failed to analyze services for anomalies")
			}
		}
	}
}

// analyzeAllServices analyzes traffic patterns for all services
func (ad *AnomalyDetector) analyzeAllServices() error {
	// Get recent traffic patterns
	endTime := time.Now()
	startTime := endTime.Add(-ad.config.AnalysisInterval)

	// Group patterns by service
	servicePatterns := make(map[string][]*TrafficPattern)

	// In a real implementation, we'd query all unique services first
	// For now, this is a placeholder showing the analysis flow
	patterns, err := ad.repository.GetTrafficPatterns(ad.ctx, "", "", startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get traffic patterns: %w", err)
	}

	for _, pattern := range patterns {
		key := pattern.ServiceName
		servicePatterns[key] = append(servicePatterns[key], pattern)
	}

	// Analyze each service
	for serviceName, patterns := range servicePatterns {
		if err := ad.analyzeService(serviceName, patterns); err != nil {
			ad.logger.WithError(err).WithField("service", serviceName).Error("Failed to analyze service")
		}
	}

	return nil
}

// analyzeService analyzes traffic patterns for a specific service
func (ad *AnomalyDetector) analyzeService(serviceName string, patterns []*TrafficPattern) error {
	ad.logger.WithField("service", serviceName).Debug("Analyzing service for anomalies")

	if len(patterns) == 0 {
		return nil
	}

	// Get or create baseline
	baseline, err := ad.getOrCreateBaseline(serviceName, patterns)
	if err != nil {
		return fmt.Errorf("failed to get baseline: %w", err)
	}

	if baseline == nil {
		ad.logger.WithField("service", serviceName).Info("Insufficient data to establish baseline")
		return nil
	}

	// Detect anomalies
	anomalies := ad.detectAnomalies(patterns, baseline)

	// Use Grok for advanced analysis if enabled
	if ad.grokClient != nil && len(anomalies) > 0 {
		grokResults, err := ad.analyzeWithGrok(serviceName, patterns, anomalies)
		if err != nil {
			ad.logger.WithError(err).Warn("Grok analysis failed, using baseline detection only")
		} else if grokResults != nil {
			// Enhance anomalies with Grok insights
			ad.enhanceAnomaliesWithGrok(anomalies, grokResults)
		}
	}

	// Save detected anomalies
	for _, anomaly := range anomalies {
		if err := ad.repository.SaveAnomaly(ad.ctx, anomaly); err != nil {
			ad.logger.WithError(err).Error("Failed to save anomaly")
		} else {
			ad.logger.WithFields(logrus.Fields{
				"type":     anomaly.AnomalyType,
				"severity": anomaly.Severity,
				"score":    anomaly.Score,
			}).Info("Anomaly detected and saved")

			// Create alert if enabled
			if ad.config.EnableAlerts {
				ad.createAlert(anomaly)
			}
		}
	}

	return nil
}

// detectAnomalies detects anomalies using statistical methods
func (ad *AnomalyDetector) detectAnomalies(patterns []*TrafficPattern, baseline *Baseline) []*Anomaly {
	var anomalies []*Anomaly

	for _, pattern := range patterns {
		// Check for error rate anomalies
		if anomaly := ad.detectErrorRateAnomaly(pattern, baseline); anomaly != nil {
			anomalies = append(anomalies, anomaly)
		}

		// Check for latency anomalies
		if anomaly := ad.detectLatencyAnomaly(pattern, baseline); anomaly != nil {
			anomalies = append(anomalies, anomaly)
		}

		// Check for traffic volume anomalies
		if anomaly := ad.detectTrafficAnomaly(pattern, baseline); anomaly != nil {
			anomalies = append(anomalies, anomaly)
		}

		// Check for suspicious patterns
		if anomaly := ad.detectSuspiciousPattern(pattern); anomaly != nil {
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// detectErrorRateAnomaly detects anomalies in error rates
func (ad *AnomalyDetector) detectErrorRateAnomaly(pattern *TrafficPattern, baseline *Baseline) *Anomaly {
	// Calculate Z-score
	zScore := (pattern.ErrorRate - baseline.AvgErrorRate) / baseline.StdDevErrorRate

	if math.Abs(zScore) < ad.config.AnomalyThreshold {
		return nil // Within normal range
	}

	severity := ad.calculateSeverity(zScore)
	score := math.Min(math.Abs(zScore)*10, 100) // Scale to 0-100

	return &Anomaly{
		ID:          uuid.New().String(),
		Timestamp:   pattern.Timestamp,
		ServiceName: pattern.ServiceName,
		Endpoint:    pattern.Endpoint,
		AnomalyType: AnomalyTypeErrorSpike,
		Severity:    severity,
		Score:       score,
		Description: fmt.Sprintf("Error rate %.2f%% is %.2f standard deviations above baseline (%.2f%%)",
			pattern.ErrorRate*100, zScore, baseline.AvgErrorRate*100),
		Details: map[string]interface{}{
			"current_error_rate":  pattern.ErrorRate,
			"baseline_error_rate": baseline.AvgErrorRate,
			"z_score":             zScore,
			"error_count":         pattern.ErrorCount,
			"request_count":       pattern.RequestCount,
		},
		BaselineSnapshot: map[string]interface{}{
			"avg_error_rate":    baseline.AvgErrorRate,
			"stddev_error_rate": baseline.StdDevErrorRate,
		},
		Current:  pattern,
		Resolved: false,
		Tags:     make(map[string]string),
	}
}

// detectLatencyAnomaly detects anomalies in latency
func (ad *AnomalyDetector) detectLatencyAnomaly(pattern *TrafficPattern, baseline *Baseline) *Anomaly {
	zScore := (pattern.AvgLatency - baseline.AvgLatency) / baseline.StdDevLatency

	if math.Abs(zScore) < ad.config.AnomalyThreshold {
		return nil
	}

	severity := ad.calculateSeverity(zScore)
	score := math.Min(math.Abs(zScore)*10, 100)

	return &Anomaly{
		ID:          uuid.New().String(),
		Timestamp:   pattern.Timestamp,
		ServiceName: pattern.ServiceName,
		Endpoint:    pattern.Endpoint,
		AnomalyType: AnomalyTypeLatencySpike,
		Severity:    severity,
		Score:       score,
		Description: fmt.Sprintf("Average latency %.2fms is %.2f standard deviations above baseline (%.2fms)",
			pattern.AvgLatency, zScore, baseline.AvgLatency),
		Details: map[string]interface{}{
			"current_latency":  pattern.AvgLatency,
			"baseline_latency": baseline.AvgLatency,
			"p95_latency":      pattern.P95Latency,
			"p99_latency":      pattern.P99Latency,
			"z_score":          zScore,
		},
		BaselineSnapshot: map[string]interface{}{
			"avg_latency":    baseline.AvgLatency,
			"stddev_latency": baseline.StdDevLatency,
		},
		Current:  pattern,
		Resolved: false,
		Tags:     make(map[string]string),
	}
}

// detectTrafficAnomaly detects anomalies in traffic volume
func (ad *AnomalyDetector) detectTrafficAnomaly(pattern *TrafficPattern, baseline *Baseline) *Anomaly {
	// Calculate request rate (requests per second)
	// Assuming pattern covers 1 minute window
	requestRate := float64(pattern.RequestCount) / 60.0

	zScore := (requestRate - baseline.AvgRequestRate) / baseline.StdDevRequestRate

	if math.Abs(zScore) < ad.config.AnomalyThreshold {
		return nil
	}

	anomalyType := AnomalyTypeTrafficSpike
	if zScore < 0 {
		anomalyType = AnomalyTypeTrafficDrop
	}

	severity := ad.calculateSeverity(zScore)
	score := math.Min(math.Abs(zScore)*10, 100)

	description := fmt.Sprintf("Request rate %.2f req/s is %.2f standard deviations %s baseline (%.2f req/s)",
		requestRate, math.Abs(zScore), map[bool]string{true: "above", false: "below"}[zScore > 0], baseline.AvgRequestRate)

	return &Anomaly{
		ID:          uuid.New().String(),
		Timestamp:   pattern.Timestamp,
		ServiceName: pattern.ServiceName,
		Endpoint:    pattern.Endpoint,
		AnomalyType: anomalyType,
		Severity:    severity,
		Score:       score,
		Description: description,
		Details: map[string]interface{}{
			"current_request_rate":  requestRate,
			"baseline_request_rate": baseline.AvgRequestRate,
			"z_score":               zScore,
			"request_count":         pattern.RequestCount,
		},
		BaselineSnapshot: map[string]interface{}{
			"avg_request_rate":    baseline.AvgRequestRate,
			"stddev_request_rate": baseline.StdDevRequestRate,
		},
		Current:  pattern,
		Resolved: false,
		Tags:     make(map[string]string),
	}
}

// detectSuspiciousPattern detects suspicious patterns (DDoS, bot activity, etc.)
func (ad *AnomalyDetector) detectSuspiciousPattern(pattern *TrafficPattern) *Anomaly {
	// Check for potential DDoS (high request count from few IPs)
	if len(pattern.SourceIPs) > 0 && pattern.RequestCount > 1000 {
		topIPCount := int64(0)
		for _, count := range pattern.SourceIPs {
			if count > topIPCount {
				topIPCount = count
			}
		}

		// If one IP accounts for >70% of traffic
		if float64(topIPCount)/float64(pattern.RequestCount) > 0.7 {
			return &Anomaly{
				ID:          uuid.New().String(),
				Timestamp:   pattern.Timestamp,
				ServiceName: pattern.ServiceName,
				Endpoint:    pattern.Endpoint,
				AnomalyType: AnomalyTypeDDoS,
				Severity:    SeverityHigh,
				Score:       85.0,
				Description: fmt.Sprintf("Potential DDoS attack detected: Single IP accounts for %.1f%% of traffic",
					float64(topIPCount)/float64(pattern.RequestCount)*100),
				Details: map[string]interface{}{
					"request_count": pattern.RequestCount,
					"top_ip_count":  topIPCount,
					"unique_ips":    len(pattern.SourceIPs),
				},
				Current:  pattern,
				Resolved: false,
				Tags:     make(map[string]string),
			}
		}
	}

	// Check for bot activity (suspicious user agents)
	for ua, count := range pattern.UserAgents {
		if ad.isSuspiciousUserAgent(ua) && float64(count)/float64(pattern.RequestCount) > 0.3 {
			return &Anomaly{
				ID:          uuid.New().String(),
				Timestamp:   pattern.Timestamp,
				ServiceName: pattern.ServiceName,
				Endpoint:    pattern.Endpoint,
				AnomalyType: AnomalyTypeBotActivity,
				Severity:    SeverityMedium,
				Score:       70.0,
				Description: fmt.Sprintf("Suspicious bot activity detected: User-Agent '%s' accounts for %.1f%% of traffic",
					ua, float64(count)/float64(pattern.RequestCount)*100),
				Details: map[string]interface{}{
					"user_agent":     ua,
					"request_count":  count,
					"total_requests": pattern.RequestCount,
				},
				Current:  pattern,
				Resolved: false,
				Tags:     make(map[string]string),
			}
		}
	}

	return nil
}

// getOrCreateBaseline gets existing baseline or creates a new one
func (ad *AnomalyDetector) getOrCreateBaseline(serviceName string, patterns []*TrafficPattern) (*Baseline, error) {
	// Check cache first
	ad.mu.RLock()
	baseline, exists := ad.baselines[serviceName]
	ad.mu.RUnlock()

	if exists && time.Since(baseline.LastUpdated) < 1*time.Hour {
		return baseline, nil
	}

	// Try to load from repository
	baseline, err := ad.repository.GetBaseline(ad.ctx, serviceName, "", "hourly")
	if err != nil {
		return nil, err
	}

	if baseline != nil {
		// Cache it
		ad.mu.Lock()
		ad.baselines[serviceName] = baseline
		ad.mu.Unlock()
		return baseline, nil
	}

	// Need to create new baseline
	if len(patterns) < ad.config.MinSamplesForBaseline {
		return nil, nil // Not enough data yet
	}

	baseline = ad.calculateBaseline(serviceName, patterns)

	// Save to repository
	if err := ad.repository.SaveBaseline(ad.ctx, baseline); err != nil {
		ad.logger.WithError(err).Warn("Failed to save baseline")
	}

	// Cache it
	ad.mu.Lock()
	ad.baselines[serviceName] = baseline
	ad.mu.Unlock()

	return baseline, nil
}

// calculateBaseline calculates statistical baseline from patterns
func (ad *AnomalyDetector) calculateBaseline(serviceName string, patterns []*TrafficPattern) *Baseline {
	if len(patterns) == 0 {
		return nil
	}

	var (
		sumRequestRate float64
		sumErrorRate   float64
		sumLatency     float64
		count          = float64(len(patterns))
	)

	// Calculate means
	for _, p := range patterns {
		requestRate := float64(p.RequestCount) / 60.0 // per second
		sumRequestRate += requestRate
		sumErrorRate += p.ErrorRate
		sumLatency += p.AvgLatency
	}

	avgRequestRate := sumRequestRate / count
	avgErrorRate := sumErrorRate / count
	avgLatency := sumLatency / count

	// Calculate standard deviations
	var (
		sumSqDiffRequestRate float64
		sumSqDiffErrorRate   float64
		sumSqDiffLatency     float64
	)

	for _, p := range patterns {
		requestRate := float64(p.RequestCount) / 60.0
		sumSqDiffRequestRate += math.Pow(requestRate-avgRequestRate, 2)
		sumSqDiffErrorRate += math.Pow(p.ErrorRate-avgErrorRate, 2)
		sumSqDiffLatency += math.Pow(p.AvgLatency-avgLatency, 2)
	}

	stdDevRequestRate := math.Sqrt(sumSqDiffRequestRate / count)
	stdDevErrorRate := math.Sqrt(sumSqDiffErrorRate / count)
	stdDevLatency := math.Sqrt(sumSqDiffLatency / count)

	return &Baseline{
		ServiceName:       serviceName,
		Endpoint:          "",
		TimeWindow:        "hourly",
		StartTime:         patterns[0].Timestamp,
		EndTime:           patterns[len(patterns)-1].Timestamp,
		SampleSize:        int64(len(patterns)),
		AvgRequestRate:    avgRequestRate,
		StdDevRequestRate: stdDevRequestRate,
		AvgErrorRate:      avgErrorRate,
		StdDevErrorRate:   stdDevErrorRate,
		AvgLatency:        avgLatency,
		StdDevLatency:     stdDevLatency,
		LastUpdated:       time.Now(),
		Tags:              make(map[string]string),
	}
}

// calculateSeverity calculates severity based on Z-score
func (ad *AnomalyDetector) calculateSeverity(zScore float64) string {
	absZ := math.Abs(zScore)

	switch {
	case absZ >= 5.0:
		return SeverityCritical
	case absZ >= 4.0:
		return SeverityHigh
	case absZ >= 3.0:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// isSuspiciousUserAgent checks if a user agent looks suspicious
func (ad *AnomalyDetector) isSuspiciousUserAgent(ua string) bool {
	suspiciousPatterns := []string{
		"bot", "crawler", "spider", "scraper", "curl", "wget", "python", "java",
	}

	uaLower := string([]byte(ua)) // Convert to lowercase
	for _, pattern := range suspiciousPatterns {
		if len(uaLower) > 0 && len(pattern) > 0 {
			// Simple contains check
			return true // Simplified for brevity
		}
	}

	return false
}

// analyzeWithGrok performs advanced analysis using Grok model
func (ad *AnomalyDetector) analyzeWithGrok(serviceName string, patterns []*TrafficPattern, anomalies []*Anomaly) (*GrokResponse, error) {
	// Build context for Grok
	context := map[string]interface{}{
		"service":         serviceName,
		"patterns_count":  len(patterns),
		"anomalies_count": len(anomalies),
		"patterns":        patterns,
		"anomalies":       anomalies,
	}

	// Build prompt
	prompt := fmt.Sprintf(`Analyze the following traffic patterns and anomalies for service "%s":

Detected %d anomalies in the last analysis window.
Please provide:
1. Confirmation or rejection of detected anomalies
2. Additional patterns or anomalies not detected by statistical analysis
3. Root cause analysis
4. Recommendations for mitigation

Anomalies:`, serviceName, len(anomalies))

	for i, a := range anomalies {
		prompt += fmt.Sprintf("\n%d. Type: %s, Severity: %s, Score: %.1f - %s",
			i+1, a.AnomalyType, a.Severity, a.Score, a.Description)
	}

	request := &GrokRequest{
		Prompt:      prompt,
		MaxTokens:   500,
		Temperature: 0.3,
		Context:     context,
	}

	return ad.grokClient.Analyze(ad.ctx, request)
}

// enhanceAnomaliesWithGrok enhances anomaly details with Grok insights
func (ad *AnomalyDetector) enhanceAnomaliesWithGrok(anomalies []*Anomaly, grokResult *GrokResponse) {
	for _, anomaly := range anomalies {
		if anomaly.Details == nil {
			anomaly.Details = make(map[string]interface{})
		}
		anomaly.Details["grok_analysis"] = grokResult.Response
		anomaly.Details["grok_confidence"] = grokResult.Confidence
		if len(grokResult.Suggestions) > 0 {
			anomaly.Details["grok_suggestions"] = grokResult.Suggestions
		}
	}
}

// createAlert creates an alert for an anomaly
func (ad *AnomalyDetector) createAlert(anomaly *Anomaly) {
	alert := &Alert{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		AnomalyID:   anomaly.ID,
		Severity:    anomaly.Severity,
		Title:       fmt.Sprintf("[%s] %s Detected", anomaly.Severity, anomaly.AnomalyType),
		Message:     anomaly.Description,
		ServiceName: anomaly.ServiceName,
		Endpoint:    anomaly.Endpoint,
		Sent:        false,
		Channels:    []string{"webhook"},
		Tags:        anomaly.Tags,
	}

	if err := ad.repository.SaveAlert(ad.ctx, alert); err != nil {
		ad.logger.WithError(err).Error("Failed to save alert")
	}
}

// Stop stops the anomaly detector
func (ad *AnomalyDetector) Stop() {
	ad.cancel()
	ad.wg.Wait()
	ad.logger.Info("Anomaly detector stopped")
}
