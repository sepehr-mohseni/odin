package admin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"odin/pkg/config"
	"odin/pkg/integrations/postman"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// IntegrationHandler handles Postman integration API endpoints
type IntegrationHandler struct {
	client     *postman.Client
	syncEngine *postman.SyncEngine
	newman     *postman.NewmanRunner
	repository *postman.MongoDBRepository
	logger     *logrus.Logger
	config     *postman.IntegrationConfig
}

// NewIntegrationHandler creates a new integration handler
func NewIntegrationHandler(
	repository *postman.MongoDBRepository,
	logger *logrus.Logger,
) *IntegrationHandler {
	return &IntegrationHandler{
		repository: repository,
		logger:     logger,
	}
}

// Initialize initializes the handler with configuration from database
func (h *IntegrationHandler) Initialize(ctx context.Context) error {
	// Load config from database
	config, err := h.repository.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if config == nil || !config.Enabled {
		h.logger.Info("Postman integration not enabled")
		return nil
	}

	h.config = config

	// Initialize Postman client
	h.client = postman.NewClient(config.APIKey, h.logger)

	// Test connection
	if err := h.client.TestConnection(ctx); err != nil {
		h.logger.WithError(err).Warn("Failed to connect to Postman API")
		return fmt.Errorf("postman connection failed: %w", err)
	}

	// Initialize sync engine
	h.syncEngine = postman.NewSyncEngine(h.client, config, h.repository, h.logger)

	// Initialize Newman runner
	h.newman = postman.NewNewmanRunner(h.repository, h.logger)

	// Start auto-sync if enabled
	if config.AutoSync {
		if err := h.syncEngine.Start(ctx); err != nil {
			h.logger.WithError(err).Warn("Failed to start sync engine")
		} else {
			h.logger.Info("Postman sync engine started")
		}
	}

	h.logger.Info("Postman integration initialized successfully")
	return nil
}

// Shutdown gracefully shuts down the integration
func (h *IntegrationHandler) Shutdown() error {
	if h.syncEngine != nil && h.syncEngine.IsRunning() {
		return h.syncEngine.Stop()
	}
	return nil
}

// RegisterRoutes registers integration API routes
func (h *IntegrationHandler) RegisterRoutes(g *echo.Group) {
	integration := g.Group("/integrations/postman")

	// Status and configuration
	integration.GET("/status", h.GetStatus)
	integration.GET("/config", h.GetConfig)
	integration.POST("/config", h.SaveConfig)
	integration.POST("/connect", h.TestConnection)
	integration.DELETE("/config", h.DeleteConfig)

	// Collections
	integration.GET("/collections", h.ListCollections)
	integration.GET("/collections/:id", h.GetCollection)
	integration.POST("/collections/:id/import", h.ImportCollection)
	integration.POST("/collections/export/:service", h.ExportService)

	// Environments
	integration.GET("/environments", h.ListEnvironments)
	integration.GET("/environments/:id", h.GetEnvironment)

	// Workspaces
	integration.GET("/workspaces", h.ListWorkspaces)

	// Sync operations
	integration.POST("/sync", h.SyncAll)
	integration.POST("/sync/:collectionId", h.SyncCollection)
	integration.GET("/sync/history", h.GetSyncHistory)
	integration.GET("/sync/history/:collectionId", h.GetCollectionSyncHistory)
	integration.DELETE("/sync/stop", h.StopSync)
	integration.POST("/sync/start", h.StartSync)

	// Testing
	integration.POST("/test/:collectionId", h.RunTests)
	integration.GET("/test/results", h.GetTestResults)
	integration.GET("/test/results/:collectionId", h.GetCollectionTestResults)
	integration.GET("/test/stats/:collectionId", h.GetTestStats)
}

// Response types

type StatusResponse struct {
	Enabled       bool      `json:"enabled"`
	Connected     bool      `json:"connected"`
	SyncRunning   bool      `json:"syncRunning"`
	WorkspaceID   string    `json:"workspaceId,omitempty"`
	AutoSync      bool      `json:"autoSync"`
	SyncInterval  string    `json:"syncInterval,omitempty"`
	LastSync      time.Time `json:"lastSync,omitempty"`
	Mappings      int       `json:"mappings"`
	NewmanEnabled bool      `json:"newmanEnabled"`
}

type CollectionSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type ImportRequest struct {
	ServiceName string `json:"serviceName"`
	AutoSync    bool   `json:"autoSync"`
	AutoTest    bool   `json:"autoTest"`
}

type ExportRequest struct {
	CollectionID   string `json:"collectionId,omitempty"`
	CollectionName string `json:"collectionName"`
}

type SyncRequest struct {
	CollectionID string `json:"collectionId"`
	ServiceName  string `json:"serviceName"`
	Direction    string `json:"direction"` // import, export, bidirectional
}

type TestRequest struct {
	Iterations int `json:"iterations"`
	Timeout    int `json:"timeout"`
}

// Handler implementations

// GetStatus returns the integration status
func (h *IntegrationHandler) GetStatus(c echo.Context) error {
	ctx := c.Request().Context()

	config, _ := h.repository.GetConfig(ctx)

	status := StatusResponse{
		Enabled:     config != nil && config.Enabled,
		Connected:   h.client != nil,
		SyncRunning: h.syncEngine != nil && h.syncEngine.IsRunning(),
	}

	if config != nil {
		status.WorkspaceID = config.WorkspaceID
		status.AutoSync = config.AutoSync
		status.SyncInterval = config.SyncInterval
		status.Mappings = len(config.Mappings)
		status.NewmanEnabled = config.Newman.Enabled
	}

	return c.JSON(http.StatusOK, status)
}

// GetConfig returns the integration configuration
func (h *IntegrationHandler) GetConfig(c echo.Context) error {
	ctx := c.Request().Context()

	config, err := h.repository.GetConfig(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	if config == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "configuration not found",
		})
	}

	// Mask API key
	config.APIKey = "****" + config.APIKey[len(config.APIKey)-4:]

	return c.JSON(http.StatusOK, config)
}

// SaveConfig saves the integration configuration
func (h *IntegrationHandler) SaveConfig(c echo.Context) error {
	ctx := c.Request().Context()

	var config postman.IntegrationConfig
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	config.Provider = "postman"

	if err := h.repository.SaveConfig(ctx, &config); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// Reinitialize with new config
	if err := h.Initialize(ctx); err != nil {
		h.logger.WithError(err).Error("Failed to reinitialize after config save")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "configuration saved successfully",
	})
}

// TestConnection tests connection to Postman API
func (h *IntegrationHandler) TestConnection(c echo.Context) error {
	ctx := c.Request().Context()

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	if err := h.client.TestConnection(ctx); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"connected": false,
			"error":     err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]bool{
		"connected": true,
	})
}

// DeleteConfig deletes the integration configuration
func (h *IntegrationHandler) DeleteConfig(c echo.Context) error {
	ctx := c.Request().Context()

	// Stop sync engine first
	if h.syncEngine != nil && h.syncEngine.IsRunning() {
		_ = h.syncEngine.Stop()
	}

	if err := h.repository.DeleteConfig(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	h.client = nil
	h.syncEngine = nil
	h.config = nil

	return c.JSON(http.StatusOK, map[string]string{
		"message": "configuration deleted successfully",
	})
}

// ListCollections lists all Postman collections
func (h *IntegrationHandler) ListCollections(c echo.Context) error {
	ctx := c.Request().Context()

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	collections, err := h.client.ListCollections(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"collections": collections,
		"count":       len(collections),
	})
}

// GetCollection retrieves a specific collection
func (h *IntegrationHandler) GetCollection(c echo.Context) error {
	ctx := c.Request().Context()
	collectionID := c.Param("id")

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	collection, err := h.client.GetCollection(ctx, collectionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, collection)
}

// ImportCollection imports a Postman collection to Odin
func (h *IntegrationHandler) ImportCollection(c echo.Context) error {
	ctx := c.Request().Context()
	collectionID := c.Param("id")

	var req ImportRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	// Fetch collection
	collection, err := h.client.GetCollection(ctx, collectionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to fetch collection: %v", err),
		})
	}

	// Transform to Odin service
	transformer := postman.NewTransformer()
	service, routes, err := transformer.EnhancedTransform(collection, req.ServiceName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to transform collection: %v", err),
		})
	}

	// Add mapping if auto-sync enabled
	if req.AutoSync && h.config != nil {
		mapping := postman.CollectionMapping{
			PostmanCollectionID: collectionID,
			OdinServiceName:     req.ServiceName,
			SyncDirection:       "bidirectional",
			AutoTest:            req.AutoTest,
			AutoSync:            true,
		}
		h.config.Mappings = append(h.config.Mappings, mapping)
		_ = h.repository.SaveConfig(ctx, h.config)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "collection imported successfully",
		"service":    service,
		"routes":     routes,
		"routeCount": len(routes),
	})
}

// ExportService exports an Odin service to Postman
func (h *IntegrationHandler) ExportService(c echo.Context) error {
	ctx := c.Request().Context()
	serviceName := c.Param("service")

	var req ExportRequest
	if err := c.Bind(&req); err != nil {
		req.CollectionName = serviceName
	}

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	// TODO: Load actual service config from Odin
	// For now, create a placeholder
	serviceConfig := &config.ServiceConfig{
		Name:     serviceName,
		BasePath: "/api/v1/" + serviceName,
	}

	// Transform to Postman collection
	transformer := postman.NewTransformer()
	collection, err := transformer.OdinServiceToPostman(serviceConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to transform service: %v", err),
		})
	}

	if req.CollectionName != "" {
		collection.Info.Name = req.CollectionName
	}

	// Create or update in Postman
	var result interface{}
	if req.CollectionID != "" {
		result, err = h.client.UpdateCollection(ctx, req.CollectionID, collection)
	} else {
		result, err = h.client.CreateCollection(ctx, collection)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to save collection: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "service exported successfully",
		"collection": result,
	})
}

// ListEnvironments lists all Postman environments
func (h *IntegrationHandler) ListEnvironments(c echo.Context) error {
	ctx := c.Request().Context()

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	environments, err := h.client.ListEnvironments(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"environments": environments,
		"count":        len(environments),
	})
}

// GetEnvironment retrieves a specific environment
func (h *IntegrationHandler) GetEnvironment(c echo.Context) error {
	ctx := c.Request().Context()
	envID := c.Param("id")

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	environment, err := h.client.GetEnvironment(ctx, envID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, environment)
}

// ListWorkspaces lists all Postman workspaces
func (h *IntegrationHandler) ListWorkspaces(c echo.Context) error {
	ctx := c.Request().Context()

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	workspaces, err := h.client.ListWorkspaces(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"workspaces": workspaces,
		"count":      len(workspaces),
	})
}

// SyncAll syncs all configured collections
func (h *IntegrationHandler) SyncAll(c echo.Context) error {
	if h.syncEngine == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "sync engine not initialized",
		})
	}

	go func() {
		if err := h.syncEngine.SyncAll(context.Background()); err != nil {
			h.logger.WithError(err).Error("Sync all failed")
		}
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "sync started",
	})
}

// SyncCollection syncs a specific collection
func (h *IntegrationHandler) SyncCollection(c echo.Context) error {
	collectionID := c.Param("collectionId")

	var req SyncRequest
	if err := c.Bind(&req); err != nil {
		req.Direction = "import"
	}

	if req.ServiceName == "" {
		req.ServiceName = "imported-service"
	}

	if h.syncEngine == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "sync engine not initialized",
		})
	}

	direction := postman.SyncDirectionImport
	switch req.Direction {
	case "export":
		direction = postman.SyncDirectionExport
	case "bidirectional":
		direction = postman.SyncDirectionBidirection
	}

	go func() {
		if err := h.syncEngine.ForceSync(context.Background(), collectionID, req.ServiceName, direction); err != nil {
			h.logger.WithError(err).Error("Collection sync failed")
		}
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "sync started",
	})
}

// GetSyncHistory retrieves sync history
func (h *IntegrationHandler) GetSyncHistory(c echo.Context) error {
	ctx := c.Request().Context()

	limit := 50
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	history, err := h.repository.GetAllSyncs(ctx, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

// GetCollectionSyncHistory retrieves sync history for a collection
func (h *IntegrationHandler) GetCollectionSyncHistory(c echo.Context) error {
	ctx := c.Request().Context()
	collectionID := c.Param("collectionId")

	limit := 50
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	history, err := h.repository.GetSyncHistory(ctx, collectionID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

// StopSync stops the sync engine
func (h *IntegrationHandler) StopSync(c echo.Context) error {
	if h.syncEngine == nil || !h.syncEngine.IsRunning() {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "sync engine not running",
		})
	}

	if err := h.syncEngine.Stop(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "sync engine stopped",
	})
}

// StartSync starts the sync engine
func (h *IntegrationHandler) StartSync(c echo.Context) error {
	if h.syncEngine == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "sync engine not initialized",
		})
	}

	if h.syncEngine.IsRunning() {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "sync engine already running",
		})
	}

	if err := h.syncEngine.Start(context.Background()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "sync engine started",
	})
}

// RunTests runs Newman tests for a collection
func (h *IntegrationHandler) RunTests(c echo.Context) error {
	collectionID := c.Param("collectionId")

	var req TestRequest
	if err := c.Bind(&req); err != nil {
		req.Iterations = 1
		req.Timeout = 30000
	}

	if h.newman == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "newman not initialized",
		})
	}

	if h.client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "client not initialized",
		})
	}

	// Run tests in background
	go func() {
		options := &postman.NewmanRunOptions{
			IterationCount: req.Iterations,
			Timeout:        req.Timeout,
			Reporters:      []string{"json"},
		}

		_, err := h.newman.RunCollectionByID(context.Background(), h.client, collectionID, "test-service", options)
		if err != nil {
			h.logger.WithError(err).Error("Newman test failed")
		}
	}()

	return c.JSON(http.StatusAccepted, map[string]string{
		"message": "tests started",
	})
}

// GetTestResults retrieves all test results
func (h *IntegrationHandler) GetTestResults(c echo.Context) error {
	// Get all test results (would need a repository method for this)
	// For now, return empty array
	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": []interface{}{},
		"count":   0,
	})
}

// GetCollectionTestResults retrieves test results for a collection
func (h *IntegrationHandler) GetCollectionTestResults(c echo.Context) error {
	ctx := c.Request().Context()
	collectionID := c.Param("collectionId")

	limit := 50
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	results, err := h.repository.GetTestHistory(ctx, collectionID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"count":   len(results),
	})
}

// GetTestStats retrieves test statistics for a collection
func (h *IntegrationHandler) GetTestStats(c echo.Context) error {
	ctx := c.Request().Context()
	collectionID := c.Param("collectionId")

	stats, err := h.repository.GetTestStats(ctx, collectionID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, stats)
}
