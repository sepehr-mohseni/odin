package ai

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// TrafficCollector collects and aggregates traffic data for AI analysis
type TrafficCollector struct {
	patterns      map[string]*TrafficPattern
	mu            sync.RWMutex
	logger        *logrus.Logger
	flushInterval time.Duration
	repository    Repository
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// TrafficData represents a single traffic event
type TrafficData struct {
	Timestamp    time.Time
	ServiceName  string
	Endpoint     string
	Method       string
	StatusCode   int
	Latency      float64
	RequestSize  int64
	ResponseSize int64
	SourceIP     string
	UserAgent    string
	UserID       string
	Error        string
}

// NewTrafficCollector creates a new traffic collector
func NewTrafficCollector(logger *logrus.Logger, repository Repository, flushInterval time.Duration) *TrafficCollector {
	ctx, cancel := context.WithCancel(context.Background())

	tc := &TrafficCollector{
		patterns:      make(map[string]*TrafficPattern),
		logger:        logger,
		flushInterval: flushInterval,
		repository:    repository,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start background flushing
	tc.wg.Add(1)
	go tc.flushLoop()

	return tc
}

// Collect adds a traffic data point to the collector
func (tc *TrafficCollector) Collect(data *TrafficData) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	key := tc.makeKey(data.ServiceName, data.Endpoint)
	pattern, exists := tc.patterns[key]

	if !exists {
		pattern = &TrafficPattern{
			Timestamp:     time.Now(),
			ServiceName:   data.ServiceName,
			Endpoint:      data.Endpoint,
			Method:        data.Method,
			StatusCodes:   make(map[string]int64),
			UserAgents:    make(map[string]int64),
			SourceIPs:     make(map[string]int64),
			RequestSizes:  make([]int64, 0, 100),
			ResponseSizes: make([]int64, 0, 100),
			Tags:          make(map[string]string),
		}
		tc.patterns[key] = pattern
	}

	// Update counters
	pattern.RequestCount++
	if data.Error != "" || data.StatusCode >= 400 {
		pattern.ErrorCount++
	}
	pattern.ErrorRate = float64(pattern.ErrorCount) / float64(pattern.RequestCount)

	// Update latency metrics
	pattern.AvgLatency = (pattern.AvgLatency*float64(pattern.RequestCount-1) + data.Latency) / float64(pattern.RequestCount)

	// Track status codes
	statusKey := string(rune(data.StatusCode/100)) + "xx"
	pattern.StatusCodes[statusKey]++

	// Track user agents (limit to top 10)
	if data.UserAgent != "" && len(pattern.UserAgents) < 10 {
		pattern.UserAgents[data.UserAgent]++
	}

	// Track source IPs (limit to top 20)
	if data.SourceIP != "" && len(pattern.SourceIPs) < 20 {
		pattern.SourceIPs[data.SourceIP]++
	}

	// Track sizes
	if len(pattern.RequestSizes) < 1000 {
		pattern.RequestSizes = append(pattern.RequestSizes, data.RequestSize)
	}
	if len(pattern.ResponseSizes) < 1000 {
		pattern.ResponseSizes = append(pattern.ResponseSizes, data.ResponseSize)
	}

	// Update unique users
	if data.UserID != "" {
		pattern.UniqueUsers++
	}

	// Calculate average sizes
	pattern.AvgRequestSize = tc.calculateAverage(pattern.RequestSizes)
	pattern.AvgResponseSize = tc.calculateAverage(pattern.ResponseSizes)

	// Calculate percentiles
	if len(pattern.RequestSizes) > 10 {
		pattern.P95Latency = tc.calculatePercentile(pattern.RequestSizes, 0.95)
		pattern.P99Latency = tc.calculatePercentile(pattern.RequestSizes, 0.99)
	}
}

// flushLoop periodically flushes collected patterns to the repository
func (tc *TrafficCollector) flushLoop() {
	defer tc.wg.Done()

	ticker := time.NewTicker(tc.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tc.ctx.Done():
			tc.flush() // Final flush
			return
		case <-ticker.C:
			tc.flush()
		}
	}
}

// flush saves collected patterns to the repository
func (tc *TrafficCollector) flush() {
	tc.mu.Lock()
	patterns := make([]*TrafficPattern, 0, len(tc.patterns))
	for _, pattern := range tc.patterns {
		patterns = append(patterns, pattern)
	}
	tc.patterns = make(map[string]*TrafficPattern) // Reset
	tc.mu.Unlock()

	if len(patterns) == 0 {
		return
	}

	tc.logger.WithField("count", len(patterns)).Debug("Flushing traffic patterns to repository")

	for _, pattern := range patterns {
		if err := tc.repository.SaveTrafficPattern(tc.ctx, pattern); err != nil {
			tc.logger.WithError(err).Error("Failed to save traffic pattern")
		}
	}
}

// GetPatterns returns current collected patterns (for testing/debugging)
func (tc *TrafficCollector) GetPatterns() []*TrafficPattern {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	patterns := make([]*TrafficPattern, 0, len(tc.patterns))
	for _, pattern := range tc.patterns {
		patterns = append(patterns, pattern)
	}
	return patterns
}

// Stop stops the collector and flushes remaining data
func (tc *TrafficCollector) Stop() {
	tc.cancel()
	tc.wg.Wait()
	tc.logger.Info("Traffic collector stopped")
}

// makeKey creates a unique key for pattern grouping
func (tc *TrafficCollector) makeKey(serviceName, endpoint string) string {
	return serviceName + ":" + endpoint
}

// calculateAverage calculates average of int64 slice
func (tc *TrafficCollector) calculateAverage(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}

	var sum int64
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}

// calculatePercentile calculates the nth percentile (0-1) of values
func (tc *TrafficCollector) calculatePercentile(values []int64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple implementation - in production, use a proper percentile algorithm
	sorted := make([]int64, len(values))
	copy(sorted, values)

	// Quick sort implementation would go here
	// For now, return approximate value
	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return float64(sorted[index])
}
