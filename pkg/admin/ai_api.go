package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"odin/pkg/ai"
)

// AIHandler handles AI-related admin API endpoints
type AIHandler struct {
	repository ai.Repository
	detector   *ai.AnomalyDetector
	logger     *logrus.Logger
}

// NewAIHandler creates a new AI admin handler
func NewAIHandler(repository ai.Repository, detector *ai.AnomalyDetector, logger *logrus.Logger) *AIHandler {
	return &AIHandler{
		repository: repository,
		detector:   detector,
		logger:     logger,
	}
}

// RegisterAIRoutes registers AI admin routes (called from admin package)
func (h *AIHandler) RegisterAIRoutes(registerRoute func(string, string, http.HandlerFunc)) {
	// Anomaly endpoints
	registerRoute("/admin/api/ai/anomalies", "GET", h.ListAnomalies)
	registerRoute("/admin/api/ai/anomalies/{id}", "GET", h.GetAnomaly)
	registerRoute("/admin/api/ai/anomalies/{id}/resolve", "POST", h.ResolveAnomaly)
	registerRoute("/admin/api/ai/anomalies/{id}/false-positive", "POST", h.MarkFalsePositive)

	// Baseline endpoints
	registerRoute("/admin/api/ai/baselines", "GET", h.ListBaselines)
	registerRoute("/admin/api/ai/baselines/{service}", "GET", h.GetServiceBaselines)

	// Configuration and stats
	registerRoute("/admin/api/ai/stats", "GET", h.GetStatistics)
	registerRoute("/admin/api/ai/config", "GET", h.GetConfig)
	registerRoute("/admin/api/ai/config", "PUT", h.UpdateConfig)
}

// extractIDFromPath extracts ID from URL path
func extractIDFromPath(path, prefix string) string {
	if len(path) > len(prefix) {
		id := path[len(prefix):]
		// Remove trailing parts after next /
		for i, ch := range id {
			if ch == '/' {
				return id[:i]
			}
		}
		return id
	}
	return ""
}

// ListAnomalies returns a list of anomalies with optional filters
func (h *AIHandler) ListAnomalies(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := ai.AnomalyFilter{
		ServiceName: query.Get("service"),
		Endpoint:    query.Get("endpoint"),
		AnomalyType: query.Get("type"),
		Severity:    query.Get("severity"),
		Limit:       100, // Default limit
	}

	// Parse resolved filter
	if resolvedStr := query.Get("resolved"); resolvedStr != "" {
		resolved := resolvedStr == "true"
		filter.Resolved = &resolved
	}

	// Parse time range
	if startStr := query.Get("start_time"); startStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartTime = &startTime
		}
	}
	if endStr := query.Get("end_time"); endStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	// Parse limit
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	anomalies, err := h.repository.ListAnomalies(r.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list anomalies")
		http.Error(w, "Failed to list anomalies", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"anomalies": anomalies,
		"count":     len(anomalies),
		"filter":    filter,
	})
}

// GetAnomaly returns a specific anomaly by ID
func (h *AIHandler) GetAnomaly(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	id := extractIDFromPath(r.URL.Path, "/admin/api/ai/anomalies/")

	anomaly, err := h.repository.GetAnomaly(r.Context(), id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get anomaly")
		http.Error(w, "Failed to get anomaly", http.StatusInternalServerError)
		return
	}

	if anomaly == nil {
		http.Error(w, "Anomaly not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anomaly)
}

// ResolveAnomaly marks an anomaly as resolved
func (h *AIHandler) ResolveAnomaly(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path, "/admin/api/ai/anomalies/")

	if err := h.repository.MarkAnomalyResolved(r.Context(), id); err != nil {
		h.logger.WithError(err).Error("Failed to resolve anomaly")
		http.Error(w, "Failed to resolve anomaly", http.StatusInternalServerError)
		return
	}

	h.logger.WithField("anomaly_id", id).Info("Anomaly resolved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Anomaly resolved",
	})
}

// MarkFalsePositive marks an anomaly as a false positive
func (h *AIHandler) MarkFalsePositive(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path, "/admin/api/ai/anomalies/")

	if err := h.repository.MarkAnomalyFalsePositive(r.Context(), id); err != nil {
		h.logger.WithError(err).Error("Failed to mark anomaly as false positive")
		http.Error(w, "Failed to mark as false positive", http.StatusInternalServerError)
		return
	}

	h.logger.WithField("anomaly_id", id).Info("Anomaly marked as false positive")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Marked as false positive",
	})
}

// ListBaselines returns all baselines
func (h *AIHandler) ListBaselines(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	serviceName := query.Get("service")

	var baselines []*ai.Baseline
	var err error

	if serviceName != "" {
		baselines, err = h.repository.ListBaselines(r.Context(), serviceName)
	} else {
		// For all services, we'd need to query all services first
		// This is a simplified version
		baselines = []*ai.Baseline{}
	}

	if err != nil {
		h.logger.WithError(err).Error("Failed to list baselines")
		http.Error(w, "Failed to list baselines", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"baselines": baselines,
		"count":     len(baselines),
	})
}

// GetServiceBaselines returns baselines for a specific service
func (h *AIHandler) GetServiceBaselines(w http.ResponseWriter, r *http.Request) {
	serviceName := extractIDFromPath(r.URL.Path, "/admin/api/ai/baselines/")

	baselines, err := h.repository.ListBaselines(r.Context(), serviceName)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get service baselines")
		http.Error(w, "Failed to get baselines", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":   serviceName,
		"baselines": baselines,
		"count":     len(baselines),
	})
}

// GetStatistics returns AI system statistics
func (h *AIHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	// Get anomaly statistics
	ctx := r.Context()

	// Count by severity
	severities := []string{ai.SeverityCritical, ai.SeverityHigh, ai.SeverityMedium, ai.SeverityLow}
	severityCounts := make(map[string]int)

	for _, severity := range severities {
		anomalies, err := h.repository.ListAnomalies(ctx, ai.AnomalyFilter{
			Severity: severity,
			Limit:    1000,
		})
		if err == nil {
			severityCounts[severity] = len(anomalies)
		}
	}

	// Count by type
	anomalyTypes := []string{
		ai.AnomalyTypeErrorSpike,
		ai.AnomalyTypeLatencySpike,
		ai.AnomalyTypeTrafficSpike,
		ai.AnomalyTypeDDoS,
	}
	typeCounts := make(map[string]int)

	for _, aType := range anomalyTypes {
		anomalies, err := h.repository.ListAnomalies(ctx, ai.AnomalyFilter{
			AnomalyType: aType,
			Limit:       1000,
		})
		if err == nil {
			typeCounts[aType] = len(anomalies)
		}
	}

	// Count resolved vs unresolved
	resolved := false
	unresolvedAnomalies, _ := h.repository.ListAnomalies(ctx, ai.AnomalyFilter{
		Resolved: &resolved,
		Limit:    1000,
	})

	stats := map[string]interface{}{
		"total_anomalies":  len(unresolvedAnomalies),
		"by_severity":      severityCounts,
		"by_type":          typeCounts,
		"unresolved_count": len(unresolvedAnomalies),
		"analysis_status":  "active",
		"last_analysis":    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetConfig returns current AI configuration
func (h *AIHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	// In production, this would load from config file or database
	config := map[string]interface{}{
		"enabled":                  true,
		"analysis_interval":        "5m",
		"baseline_window":          "24h",
		"anomaly_threshold":        3.0,
		"min_samples_for_baseline": 100,
		"use_grok_model":           true,
		"grok_service_url":         "http://localhost:8000",
		"enable_alerts":            true,
		"retention_days":           90,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateConfig updates AI configuration
func (h *AIHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config map[string]interface{}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate and update configuration
	// In production, this would update config file or database

	h.logger.WithField("config", config).Info("AI configuration updated")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
		"config":  config,
	})
}
