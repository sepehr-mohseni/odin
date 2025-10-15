package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"odin/pkg/config"
	"odin/pkg/mongodb"
	"odin/pkg/service"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// MongoDBServiceHandler handles MongoDB service management
type MongoDBServiceHandler struct {
	adapter  *mongodb.ServiceAdapter
	logger   *logrus.Logger
	registry *service.Registry
}

// NewMongoDBServiceHandler creates a new handler
func NewMongoDBServiceHandler(adapter *mongodb.ServiceAdapter, registry *service.Registry, logger *logrus.Logger) *MongoDBServiceHandler {
	return &MongoDBServiceHandler{
		adapter:  adapter,
		logger:   logger,
		registry: registry,
	}
}

// ServiceRequest represents a service create/update request
type ServiceRequest struct {
	Name           string            `json:"name"`
	BasePath       string            `json:"basePath"`
	Targets        []string          `json:"targets"`
	StripBasePath  bool              `json:"stripBasePath"`
	Timeout        string            `json:"timeout"` // duration string like "30s"
	RetryCount     int               `json:"retryCount"`
	RetryDelay     string            `json:"retryDelay"` // duration string like "1s"
	Authentication bool              `json:"authentication"`
	LoadBalancing  string            `json:"loadBalancing"`
	Headers        map[string]string `json:"headers"`
	Protocol       string            `json:"protocol"`
}

// ServiceResponse represents a service response
type ServiceResponse struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	BasePath       string            `json:"basePath"`
	Targets        []string          `json:"targets"`
	StripBasePath  bool              `json:"stripBasePath"`
	Timeout        string            `json:"timeout"`
	RetryCount     int               `json:"retryCount"`
	RetryDelay     string            `json:"retryDelay"`
	Authentication bool              `json:"authentication"`
	LoadBalancing  string            `json:"loadBalancing"`
	Headers        map[string]string `json:"headers"`
	Protocol       string            `json:"protocol"`
	Enabled        bool              `json:"enabled"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

// RegisterMongoDBRoutes registers MongoDB API routes
func (h *AdminHandler) RegisterMongoDBRoutes(e *echo.Echo, adapter *mongodb.ServiceAdapter) {
	handler := NewMongoDBServiceHandler(adapter, nil, h.logger)

	api := e.Group("/admin/api/mongodb")
	api.Use(h.basicAuthMiddleware)

	// Service endpoints
	api.GET("/services", handler.ListServices)
	api.GET("/services/:name", handler.GetService)
	api.POST("/services", handler.CreateService)
	api.PUT("/services/:name", handler.UpdateService)
	api.DELETE("/services/:name", handler.DeleteService)

	// Health check
	api.GET("/health", handler.CheckHealth)

	// Statistics
	api.GET("/stats", handler.GetStatistics)
}

// ListServices lists all services from MongoDB
func (h *MongoDBServiceHandler) ListServices(c echo.Context) error {
	ctx := context.Background()

	services, err := h.adapter.LoadServices(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to load services from MongoDB")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load services",
		})
	}

	// Convert to response format
	responses := make([]ServiceResponse, 0, len(services))
	for _, svc := range services {
		responses = append(responses, ServiceResponse{
			Name:           svc.Name,
			BasePath:       svc.BasePath,
			Targets:        svc.Targets,
			StripBasePath:  svc.StripBasePath,
			Timeout:        svc.Timeout.String(),
			RetryCount:     svc.RetryCount,
			RetryDelay:     svc.RetryDelay.String(),
			Authentication: svc.Authentication,
			LoadBalancing:  svc.LoadBalancing,
			Headers:        svc.Headers,
			Protocol:       svc.Protocol,
			Enabled:        true,
		})
	}

	return c.JSON(http.StatusOK, responses)
}

// GetService retrieves a specific service
func (h *MongoDBServiceHandler) GetService(c echo.Context) error {
	ctx := context.Background()
	name := c.Param("name")

	svc, err := h.adapter.GetService(ctx, name)
	if err != nil {
		h.logger.WithError(err).WithField("service", name).Error("Failed to get service")
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Service not found",
		})
	}

	response := ServiceResponse{
		Name:           svc.Name,
		BasePath:       svc.BasePath,
		Targets:        svc.Targets,
		StripBasePath:  svc.StripBasePath,
		Timeout:        svc.Timeout.String(),
		RetryCount:     svc.RetryCount,
		RetryDelay:     svc.RetryDelay.String(),
		Authentication: svc.Authentication,
		LoadBalancing:  svc.LoadBalancing,
		Headers:        svc.Headers,
		Protocol:       svc.Protocol,
		Enabled:        true,
	}

	return c.JSON(http.StatusOK, response)
}

// CreateService creates a new service
func (h *MongoDBServiceHandler) CreateService(c echo.Context) error {
	ctx := context.Background()

	var req ServiceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Service name is required",
		})
	}
	if req.BasePath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Base path is required",
		})
	}
	if len(req.Targets) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "At least one target is required",
		})
	}

	// Parse durations
	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil || timeout == 0 {
		timeout = 30 * time.Second
	}

	retryDelay, err := time.ParseDuration(req.RetryDelay)
	if err != nil || retryDelay == 0 {
		retryDelay = 1 * time.Second
	}

	// Set defaults
	if req.LoadBalancing == "" {
		req.LoadBalancing = "round_robin"
	}
	if req.RetryCount == 0 {
		req.RetryCount = 3
	}
	if req.Protocol == "" {
		req.Protocol = "http"
	}

	// Create service config
	svc := &config.ServiceConfig{
		Name:           req.Name,
		BasePath:       req.BasePath,
		Targets:        req.Targets,
		StripBasePath:  req.StripBasePath,
		Timeout:        timeout,
		RetryCount:     req.RetryCount,
		RetryDelay:     retryDelay,
		Authentication: req.Authentication,
		LoadBalancing:  req.LoadBalancing,
		Headers:        req.Headers,
		Protocol:       req.Protocol,
	}

	// Save to MongoDB
	if err := h.adapter.SaveService(ctx, svc); err != nil {
		h.logger.WithError(err).WithField("service", req.Name).Error("Failed to create service")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create service",
		})
	}

	h.logger.WithField("service", req.Name).Info("Service created successfully")

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Service created successfully",
		"service": req.Name,
	})
}

// UpdateService updates an existing service
func (h *MongoDBServiceHandler) UpdateService(c echo.Context) error {
	ctx := context.Background()
	name := c.Param("name")

	var req ServiceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Ensure name matches
	if req.Name != "" && req.Name != name {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Service name in URL and body must match",
		})
	}
	req.Name = name

	// Parse durations
	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil || timeout == 0 {
		timeout = 30 * time.Second
	}

	retryDelay, err := time.ParseDuration(req.RetryDelay)
	if err != nil || retryDelay == 0 {
		retryDelay = 1 * time.Second
	}

	// Create service config
	svc := &config.ServiceConfig{
		Name:           req.Name,
		BasePath:       req.BasePath,
		Targets:        req.Targets,
		StripBasePath:  req.StripBasePath,
		Timeout:        timeout,
		RetryCount:     req.RetryCount,
		RetryDelay:     retryDelay,
		Authentication: req.Authentication,
		LoadBalancing:  req.LoadBalancing,
		Headers:        req.Headers,
		Protocol:       req.Protocol,
	}

	// Update in MongoDB
	if err := h.adapter.UpdateService(ctx, name, svc); err != nil {
		h.logger.WithError(err).WithField("service", name).Error("Failed to update service")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update service",
		})
	}

	h.logger.WithField("service", name).Info("Service updated successfully")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Service updated successfully",
		"service": name,
	})
}

// DeleteService deletes a service
func (h *MongoDBServiceHandler) DeleteService(c echo.Context) error {
	ctx := context.Background()
	name := c.Param("name")

	if err := h.adapter.DeleteService(ctx, name); err != nil {
		h.logger.WithError(err).WithField("service", name).Error("Failed to delete service")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete service",
		})
	}

	h.logger.WithField("service", name).Info("Service deleted successfully")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Service deleted successfully",
		"service": name,
	})
}

// CheckHealth checks MongoDB connection health
func (h *MongoDBServiceHandler) CheckHealth(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// We don't have direct access to repo, so we'll try to list services
	_, err := h.adapter.LoadServices(ctx)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":  "unhealthy",
			"message": "MongoDB connection failed",
			"error":   err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"message": "MongoDB connection is healthy",
	})
}

// GetStatistics returns MongoDB statistics
func (h *MongoDBServiceHandler) GetStatistics(c echo.Context) error {
	ctx := context.Background()

	services, err := h.adapter.LoadServices(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to load services for statistics")
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load statistics",
		})
	}

	enabledCount := 0
	protocols := make(map[string]int)
	loadBalancers := make(map[string]int)

	for _, svc := range services {
		enabledCount++

		if svc.Protocol != "" {
			protocols[svc.Protocol]++
		} else {
			protocols["http"]++
		}

		if svc.LoadBalancing != "" {
			loadBalancers[svc.LoadBalancing]++
		} else {
			loadBalancers["round_robin"]++
		}
	}

	stats := map[string]interface{}{
		"total_services":   len(services),
		"enabled_services": enabledCount,
		"protocols":        protocols,
		"load_balancers":   loadBalancers,
		"timestamp":        time.Now(),
	}

	return c.JSON(http.StatusOK, stats)
}

// Helper to convert response to JSON bytes
func toJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
