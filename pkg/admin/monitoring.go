package admin

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

// MetricsData represents the structure of monitoring data
type MetricsData struct {
	TotalRequests     int64                   `json:"totalRequests"`
	AvgResponseTime   float64                 `json:"avgResponseTime"`
	ActiveConnections int                     `json:"activeConnections"`
	SuccessRate       float64                 `json:"successRate"`
	RequestRate       float64                 `json:"requestRate"`
	ResponseTime      ResponseTimePercentiles `json:"responseTime"`
	StatusCodes       StatusCodeDistribution  `json:"statusCodes"`
	Services          []ServiceStatus         `json:"services"`
	Traces            []TraceInfo             `json:"traces"`
}

type ResponseTimePercentiles struct {
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
	P99 float64 `json:"p99"`
}

type StatusCodeDistribution struct {
	Success     int `json:"success"`     // 2xx
	Redirect    int `json:"redirect"`    // 3xx
	ClientError int `json:"clientError"` // 4xx
	ServerError int `json:"serverError"` // 5xx
}

type ServiceStatus struct {
	Name         string  `json:"name"`
	Protocol     string  `json:"protocol"`
	Healthy      bool    `json:"healthy"`
	Warning      bool    `json:"warning"`
	ResponseTime float64 `json:"responseTime"`
	LastCheck    string  `json:"lastCheck"`
}

type TraceInfo struct {
	TraceID   string    `json:"traceId"`
	SpanID    string    `json:"spanId"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Service   string    `json:"service"`
	Duration  float64   `json:"duration"`
	Status    int       `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// MonitoringCollector collects and aggregates monitoring metrics
type MonitoringCollector struct {
	mu             sync.RWMutex
	requestCount   int64
	responseTimes  []float64
	statusCodes    map[int]int
	activeConns    int
	traces         []TraceInfo
	services       map[string]ServiceStatus
	wsClients      map[*websocket.Conn]bool
	wsClientsMutex sync.RWMutex
}

// NewMonitoringCollector creates a new monitoring collector
func NewMonitoringCollector() *MonitoringCollector {
	return &MonitoringCollector{
		responseTimes: make([]float64, 0),
		statusCodes:   make(map[int]int),
		traces:        make([]TraceInfo, 0),
		services:      make(map[string]ServiceStatus),
		wsClients:     make(map[*websocket.Conn]bool),
	}
}

var (
	collector *MonitoringCollector
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin in development
		},
	}
)

func init() {
	collector = NewMonitoringCollector()
}

// GetCollector returns the global monitoring collector instance
func GetCollector() *MonitoringCollector {
	return collector
}

// RecordRequest records a new request metric
func (mc *MonitoringCollector) RecordRequest(method, path string, duration time.Duration, statusCode int, service string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.requestCount++
	durationMs := float64(duration.Nanoseconds()) / 1e6
	mc.responseTimes = append(mc.responseTimes, durationMs) // Keep only last 1000 response times for memory efficiency
	if len(mc.responseTimes) > 1000 {
		mc.responseTimes = mc.responseTimes[len(mc.responseTimes)-1000:]
	}

	mc.statusCodes[statusCode]++

	// Add trace info
	trace := TraceInfo{
		TraceID:   generateTraceID(),
		SpanID:    generateSpanID(),
		Method:    method,
		Path:      path,
		Service:   service,
		Duration:  durationMs,
		Status:    statusCode,
		Timestamp: time.Now(),
	}

	mc.traces = append(mc.traces, trace)
	// Keep only last 100 traces
	if len(mc.traces) > 100 {
		mc.traces = mc.traces[len(mc.traces)-100:]
	}
}

// UpdateActiveConnections updates the active connections count
func (mc *MonitoringCollector) UpdateActiveConnections(count int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.activeConns = count
}

// UpdateServiceStatus updates the status of a service
func (mc *MonitoringCollector) UpdateServiceStatus(name, protocol string, healthy bool, responseTime float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.services[name] = ServiceStatus{
		Name:         name,
		Protocol:     protocol,
		Healthy:      healthy,
		Warning:      responseTime > 1000, // Consider >1s as warning
		ResponseTime: responseTime,
		LastCheck:    time.Now().Format("15:04:05"),
	}
}

// GetMetrics returns current aggregated metrics
func (mc *MonitoringCollector) GetMetrics() MetricsData {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Calculate response time percentiles
	percentiles := mc.calculatePercentiles()

	// Calculate success rate
	successRate := mc.calculateSuccessRate()

	// Calculate request rate (requests per minute)
	requestRate := mc.calculateRequestRate()

	// Convert services map to slice
	services := make([]ServiceStatus, 0, len(mc.services))
	for _, service := range mc.services {
		services = append(services, service)
	}

	// Sort services by name
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	// Calculate status code distribution
	statusDist := StatusCodeDistribution{
		Success:     mc.statusCodes[200] + mc.statusCodes[201] + mc.statusCodes[204],
		Redirect:    mc.statusCodes[301] + mc.statusCodes[302] + mc.statusCodes[304],
		ClientError: mc.statusCodes[400] + mc.statusCodes[401] + mc.statusCodes[403] + mc.statusCodes[404],
		ServerError: mc.statusCodes[500] + mc.statusCodes[502] + mc.statusCodes[503] + mc.statusCodes[504],
	}

	// Add up all other status codes
	for code, count := range mc.statusCodes {
		if code >= 200 && code < 300 && code != 200 && code != 201 && code != 204 {
			statusDist.Success += count
		} else if code >= 300 && code < 400 && code != 301 && code != 302 && code != 304 {
			statusDist.Redirect += count
		} else if code >= 400 && code < 500 && code != 400 && code != 401 && code != 403 && code != 404 {
			statusDist.ClientError += count
		} else if code >= 500 && code != 500 && code != 502 && code != 503 && code != 504 {
			statusDist.ServerError += count
		}
	}

	avgResponseTime := 0.0
	if len(mc.responseTimes) > 0 {
		sum := 0.0
		for _, rt := range mc.responseTimes {
			sum += rt
		}
		avgResponseTime = sum / float64(len(mc.responseTimes))
	}

	return MetricsData{
		TotalRequests:     mc.requestCount,
		AvgResponseTime:   avgResponseTime,
		ActiveConnections: mc.activeConns,
		SuccessRate:       successRate,
		RequestRate:       requestRate,
		ResponseTime:      percentiles,
		StatusCodes:       statusDist,
		Services:          services,
		Traces:            mc.traces,
	}
}

// Calculate percentiles for response times
func (mc *MonitoringCollector) calculatePercentiles() ResponseTimePercentiles {
	if len(mc.responseTimes) == 0 {
		return ResponseTimePercentiles{}
	}

	// Copy and sort response times
	sorted := make([]float64, len(mc.responseTimes))
	copy(sorted, mc.responseTimes)
	sort.Float64s(sorted)

	n := len(sorted)
	return ResponseTimePercentiles{
		P50: sorted[int(float64(n)*0.5)],
		P90: sorted[int(float64(n)*0.9)],
		P99: sorted[int(float64(n)*0.99)],
	}
}

// Calculate success rate (2xx status codes)
func (mc *MonitoringCollector) calculateSuccessRate() float64 {
	total := 0
	success := 0

	for code, count := range mc.statusCodes {
		total += count
		if code >= 200 && code < 300 {
			success += count
		}
	}

	if total == 0 {
		return 0
	}

	return float64(success) / float64(total)
}

// Calculate request rate (simplified - in real implementation would use time windows)
func (mc *MonitoringCollector) calculateRequestRate() float64 {
	// For demo purposes, return a mock rate
	// In real implementation, this would calculate requests per minute
	return float64(mc.requestCount) / 60.0 // Rough estimate
}

// BroadcastMetrics sends metrics to all connected WebSocket clients
func (mc *MonitoringCollector) BroadcastMetrics() {
	metrics := mc.GetMetrics()
	data, err := json.Marshal(metrics)
	if err != nil {
		log.Printf("Error marshaling metrics: %v", err)
		return
	}

	mc.wsClientsMutex.RLock()
	defer mc.wsClientsMutex.RUnlock()

	for client := range mc.wsClients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Error writing to websocket: %v", err)
			client.Close()
			delete(mc.wsClients, client)
		}
	}
}

// StartMetricsBroadcaster starts broadcasting metrics to WebSocket clients
func (mc *MonitoringCollector) StartMetricsBroadcaster() {
	ticker := time.NewTicker(5 * time.Second) // Broadcast every 5 seconds
	go func() {
		for range ticker.C {
			mc.BroadcastMetrics()
		}
	}()
}

// HTTP Handlers

// GetMonitoringPage serves the monitoring dashboard HTML
func GetMonitoringPage(c echo.Context) error {
	return c.Render(http.StatusOK, "monitoring.html", nil)
}

// GetMetricsAPI returns metrics as JSON
func GetMetricsAPI(c echo.Context) error {
	metrics := collector.GetMetrics()
	return c.JSON(http.StatusOK, metrics)
}

// WebSocketMonitoring handles WebSocket connections for real-time monitoring
func WebSocketMonitoring(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return err
	}
	defer ws.Close()

	// Add client to the list
	collector.wsClientsMutex.Lock()
	collector.wsClients[ws] = true
	collector.wsClientsMutex.Unlock()

	// Remove client when connection closes
	defer func() {
		collector.wsClientsMutex.Lock()
		delete(collector.wsClients, ws)
		collector.wsClientsMutex.Unlock()
	}()

	// Send initial metrics
	metrics := collector.GetMetrics()
	if data, err := json.Marshal(metrics); err == nil {
		ws.WriteMessage(websocket.TextMessage, data)
	}

	// Keep connection alive and handle ping/pong
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}

	return nil
}

// Utility functions for generating trace/span IDs
func generateTraceID() string {
	return time.Now().Format("20060102150405") + "000000"
}

func generateSpanID() string {
	return time.Now().Format("150405") + "00"
}
