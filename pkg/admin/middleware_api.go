package admin

import (
	"context"
	"fmt"
	"net/http"

	"odin/pkg/plugins"

	"github.com/labstack/echo/v4"
)

// MiddlewareAPIHandler handles middleware chain management API requests
type MiddlewareAPIHandler struct {
	manager *plugins.PluginManager
	repo    *plugins.PluginRepository
}

// NewMiddlewareAPIHandler creates a new middleware API handler
func NewMiddlewareAPIHandler(manager *plugins.PluginManager, repo *plugins.PluginRepository) *MiddlewareAPIHandler {
	return &MiddlewareAPIHandler{
		manager: manager,
		repo:    repo,
	}
}

// RegisterMiddlewareAPIRoutes registers middleware management API routes
func (h *MiddlewareAPIHandler) RegisterMiddlewareAPIRoutes(adminGroup *echo.Group) {
	// Middleware chain management
	adminGroup.GET("/api/middleware/chain", h.getMiddlewareChain)
	adminGroup.POST("/api/middleware/chain/reorder", h.reorderMiddlewareChain)
	adminGroup.GET("/api/middleware/chain/stats", h.getMiddlewareStats)

	// Individual middleware management
	adminGroup.POST("/api/middleware/:name/register", h.registerMiddleware)
	adminGroup.DELETE("/api/middleware/:name/unregister", h.unregisterMiddleware)
	adminGroup.PUT("/api/middleware/:name/priority", h.updateMiddlewarePriority)
	adminGroup.PUT("/api/middleware/:name/routes", h.updateMiddlewareRoutes)
	adminGroup.PUT("/api/middleware/:name/phase", h.updateMiddlewarePhase)

	// Plugin compilation (for future enhancement)
	adminGroup.POST("/api/middleware/compile", h.compilePlugin)

	// Middleware testing
	adminGroup.POST("/api/middleware/:name/test", h.testMiddleware)
	adminGroup.GET("/api/middleware/:name/health", h.getMiddlewareHealth)

	// Bulk operations
	adminGroup.POST("/api/middleware/reload-all", h.reloadAllMiddleware)
	adminGroup.POST("/api/middleware/enable-all", h.enableAllMiddleware)
	adminGroup.POST("/api/middleware/disable-all", h.disableAllMiddleware)

	// Testing and metrics
	adminGroup.GET("/api/middleware/metrics", h.getAllMetrics)
	adminGroup.GET("/api/middleware/:name/metrics", h.getMiddlewareMetrics)
	adminGroup.POST("/api/middleware/:name/metrics/reset", h.resetMiddlewareMetrics)

	// Rollback operations
	adminGroup.POST("/api/middleware/:name/snapshot", h.createSnapshot)
	adminGroup.POST("/api/middleware/:name/rollback", h.rollbackMiddleware)
	adminGroup.GET("/api/middleware/:name/snapshots", h.getSnapshots)
}

// GetMiddlewareChain returns the current middleware execution chain
func (h *MiddlewareAPIHandler) getMiddlewareChain(c echo.Context) error {
	chain := h.manager.GetMiddlewareChain()

	// Enrich with database information
	type EnrichedMiddlewareEntry struct {
		Name        string   `json:"name"`
		Priority    int      `json:"priority"`
		Routes      []string `json:"routes"`
		Phase       string   `json:"phase"`
		Version     string   `json:"version,omitempty"`
		Description string   `json:"description,omitempty"`
		Enabled     bool     `json:"enabled"`
		Loaded      bool     `json:"loaded"`
	}

	enrichedChain := make([]EnrichedMiddlewareEntry, 0, len(chain))
	for _, entry := range chain {
		// Get plugin info from database
		plugin, err := h.repo.GetPlugin(context.Background(), entry.Name)

		enriched := EnrichedMiddlewareEntry{
			Name:     entry.Name,
			Priority: entry.Priority,
			Routes:   entry.Routes,
			Phase:    entry.Phase,
			Enabled:  true,
			Loaded:   true,
		}

		if err == nil {
			enriched.Version = plugin.Version
			enriched.Description = plugin.Description
			enriched.Enabled = plugin.Enabled
		}

		enrichedChain = append(enrichedChain, enriched)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"chain":      enrichedChain,
		"totalCount": len(enrichedChain),
		"message":    "Middleware chain retrieved successfully",
	})
}

// ReorderMiddlewareChain updates the priority of multiple middleware at once
func (h *MiddlewareAPIHandler) reorderMiddlewareChain(c echo.Context) error {
	type ReorderRequest struct {
		Order []struct {
			Name     string `json:"name"`
			Priority int    `json:"priority"`
		} `json:"order"`
	}

	var req ReorderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Update priorities in manager
	for _, item := range req.Order {
		if err := h.manager.UpdateMiddlewarePriority(item.Name, item.Priority); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to update %s priority: %v", item.Name, err),
			})
		}

		// Update in database
		if err := h.repo.UpdatePluginPriority(context.Background(), item.Name, item.Priority); err != nil {
			c.Logger().Errorf("Failed to persist priority for %s: %v", item.Name, err)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Middleware chain reordered successfully",
	})
}

// GetMiddlewareStats returns statistics about the middleware chain
func (h *MiddlewareAPIHandler) getMiddlewareStats(c echo.Context) error {
	chain := h.manager.GetMiddlewareChain()
	allMiddlewares := h.manager.ListMiddlewares()

	// Count by phase
	phaseCount := make(map[string]int)
	for _, entry := range chain {
		if entry.Phase != "" {
			phaseCount[entry.Phase]++
		} else {
			phaseCount["unassigned"]++
		}
	}

	// Count by route targeting
	globalCount := 0
	specificCount := 0
	for _, entry := range chain {
		if len(entry.Routes) == 0 || (len(entry.Routes) == 1 && entry.Routes[0] == "*") {
			globalCount++
		} else {
			specificCount++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"totalMiddlewares":  len(allMiddlewares),
		"activeInChain":     len(chain),
		"globalMiddlewares": globalCount,
		"routeSpecific":     specificCount,
		"byPhase":           phaseCount,
		"averagePriority":   calculateAveragePriority(chain),
	})
}

// RegisterMiddleware registers an existing middleware plugin in the chain
func (h *MiddlewareAPIHandler) registerMiddleware(c echo.Context) error {
	name := c.Param("name")

	type RegisterRequest struct {
		Priority int      `json:"priority"`
		Routes   []string `json:"routes"`
		Phase    string   `json:"phase"`
	}

	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Get the middleware (must already be loaded)
	middleware, exists := h.manager.GetMiddleware(name)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Middleware %s not loaded. Load it first using the plugin API.", name),
		})
	}

	// Register in chain
	if err := h.manager.RegisterMiddleware(name, middleware, req.Priority, req.Routes, req.Phase); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to register middleware: %v", err),
		})
	}

	// Update database
	plugin, err := h.repo.GetPlugin(context.Background(), name)
	if err == nil {
		plugin.Priority = req.Priority
		plugin.AppliedTo = req.Routes
		plugin.Phase = req.Phase
		if err := h.repo.UpdatePlugin(context.Background(), plugin); err != nil {
			c.Logger().Errorf("Failed to persist middleware registration for %s: %v", name, err)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Middleware %s registered successfully", name),
	})
}

// UnregisterMiddleware removes a middleware from the chain
func (h *MiddlewareAPIHandler) unregisterMiddleware(c echo.Context) error {
	name := c.Param("name")

	if err := h.manager.UnregisterMiddleware(name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to unregister middleware: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Middleware %s unregistered successfully", name),
	})
}

// UpdateMiddlewarePriority updates the priority of a middleware
func (h *MiddlewareAPIHandler) updateMiddlewarePriority(c echo.Context) error {
	name := c.Param("name")

	type PriorityRequest struct {
		Priority int `json:"priority"`
	}

	var req PriorityRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Validate priority range
	if req.Priority < 0 || req.Priority > 1000 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Priority must be between 0 and 1000",
		})
	}

	// Update in manager
	if err := h.manager.UpdateMiddlewarePriority(name, req.Priority); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to update priority: %v", err),
		})
	}

	// Update in database
	if err := h.repo.UpdatePluginPriority(context.Background(), name, req.Priority); err != nil {
		c.Logger().Errorf("Failed to persist priority for %s: %v", name, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Priority updated to %d", req.Priority),
	})
}

// UpdateMiddlewareRoutes updates the routes a middleware applies to
func (h *MiddlewareAPIHandler) updateMiddlewareRoutes(c echo.Context) error {
	name := c.Param("name")

	type RoutesRequest struct {
		Routes []string `json:"routes"`
	}

	var req RoutesRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Update in manager
	if err := h.manager.UpdateMiddlewareRoutes(name, req.Routes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to update routes: %v", err),
		})
	}

	// Update in database
	if err := h.repo.UpdatePluginRoutes(context.Background(), name, req.Routes); err != nil {
		c.Logger().Errorf("Failed to persist routes for %s: %v", name, err)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Routes updated successfully",
	})
}

// UpdateMiddlewarePhase updates the execution phase of a middleware
func (h *MiddlewareAPIHandler) updateMiddlewarePhase(c echo.Context) error {
	name := c.Param("name")

	type PhaseRequest struct {
		Phase string `json:"phase"`
	}

	var req PhaseRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Validate phase
	validPhases := []string{"pre-auth", "post-auth", "pre-route", "post-route", ""}
	isValid := false
	for _, phase := range validPhases {
		if req.Phase == phase {
			isValid = true
			break
		}
	}

	if !isValid {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid phase. Must be one of: pre-auth, post-auth, pre-route, post-route",
		})
	}

	// Update in database
	plugin, err := h.repo.GetPlugin(context.Background(), name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Plugin not found: %v", err),
		})
	}

	plugin.Phase = req.Phase
	if err := h.repo.UpdatePlugin(context.Background(), plugin); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to update phase: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Phase updated to %s", req.Phase),
	})
}

// CompilePlugin compiles a Go plugin from source code (placeholder for future)
func (h *MiddlewareAPIHandler) compilePlugin(c echo.Context) error {
	type CompileRequest struct {
		Name       string                 `json:"name"`
		SourceCode string                 `json:"sourceCode"`
		BuildMode  string                 `json:"buildMode"` // "plugin" or "wasm"
		Config     map[string]interface{} `json:"config"`
	}

	var req CompileRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// TODO: Implement plugin compilation
	// This would involve:
	// 1. Validate source code
	// 2. Create temporary directory
	// 3. Write source to file
	// 4. Run `go build -buildmode=plugin`
	// 5. Store compiled binary
	// 6. Register plugin

	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error":   "Plugin compilation not yet implemented",
		"message": "Upload pre-compiled .so files for now",
	})
}

// TestMiddleware tests a middleware with sample data
func (h *MiddlewareAPIHandler) testMiddleware(c echo.Context) error {
	name := c.Param("name")

	type TestRequest struct {
		Method  string                 `json:"method"`
		Path    string                 `json:"path"`
		Headers map[string]interface{} `json:"headers"`
		Body    interface{}            `json:"body"`
	}

	var req TestRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Check if middleware is loaded
	_, exists := h.manager.GetMiddleware(name)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Middleware %s not loaded", name),
		})
	}

	// Use the testing framework
	tester := h.manager.GetTester()
	if tester == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Middleware tester not initialized",
		})
	}

	// Prepare test data
	testData := map[string]interface{}{
		"method":  req.Method,
		"path":    req.Path,
		"headers": req.Headers,
		"body":    req.Body,
	}

	// Execute test
	result, err := tester.TestMiddleware(name, testData)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Test execution failed: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Middleware test completed",
		"result":  result,
	})
}

// GetMiddlewareHealth returns health status of a middleware
func (h *MiddlewareAPIHandler) getMiddlewareHealth(c echo.Context) error {
	name := c.Param("name")

	middleware, exists := h.manager.GetMiddleware(name)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Middleware %s not loaded", name),
		})
	}

	// Use the testing framework for health check
	tester := h.manager.GetTester()
	if tester == nil {
		// Fallback to basic health info
		plugin, err := h.repo.GetPlugin(context.Background(), name)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to get plugin info: %v", err),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"name":    middleware.Name(),
			"version": middleware.Version(),
			"enabled": plugin.Enabled,
			"loaded":  true,
			"healthy": true,
			"message": "Basic health check (tester unavailable)",
		})
	}

	// Get comprehensive health status
	health := tester.GetHealthStatus(name)
	metrics := tester.GetMetrics(name)

	response := map[string]interface{}{
		"name":              middleware.Name(),
		"version":           middleware.Version(),
		"status":            health.Status,
		"lastCheck":         health.LastCheck,
		"responseTime":      health.ResponseTime.String(),
		"errorRate":         health.ErrorRate,
		"consecutiveErrors": health.ConsecutiveErrors,
		"message":           health.Message,
	}

	if metrics != nil {
		response["metrics"] = metrics.GetStats()
	}

	return c.JSON(http.StatusOK, response)
}

// ReloadAllMiddleware reloads all middleware from database
func (h *MiddlewareAPIHandler) reloadAllMiddleware(c echo.Context) error {
	// Get all enabled middleware plugins from database
	plugins, err := h.repo.GetMiddlewarePlugins(context.Background(), true)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to load middleware plugins: %v", err),
		})
	}

	reloadCount := 0
	errorCount := 0

	for _, plugin := range plugins {
		// Unload if already loaded
		h.manager.UnregisterMiddleware(plugin.Name)
		h.manager.UnloadMiddleware(plugin.Name)

		// Load and register
		err := h.manager.LoadMiddlewareWithChain(
			plugin.Name,
			plugin.BinaryPath,
			plugin.Config,
			plugin.Priority,
			plugin.AppliedTo,
			plugin.Phase,
		)

		if err != nil {
			c.Logger().Errorf("Failed to reload middleware %s: %v", plugin.Name, err)
			errorCount++
		} else {
			reloadCount++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Middleware reload completed",
		"reloaded":     reloadCount,
		"errors":       errorCount,
		"totalPlugins": len(plugins),
	})
}

// EnableAllMiddleware enables all middleware plugins
func (h *MiddlewareAPIHandler) enableAllMiddleware(c echo.Context) error {
	plugins, err := h.repo.GetMiddlewarePlugins(context.Background(), false)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get middleware plugins: %v", err),
		})
	}

	enabledCount := 0
	for _, plugin := range plugins {
		if !plugin.Enabled {
			if err := h.repo.EnablePlugin(context.Background(), plugin.Name); err != nil {
				c.Logger().Errorf("Failed to enable %s: %v", plugin.Name, err)
			} else {
				enabledCount++
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "All middleware enabled",
		"enabled": enabledCount,
	})
}

// DisableAllMiddleware disables all middleware plugins
func (h *MiddlewareAPIHandler) disableAllMiddleware(c echo.Context) error {
	plugins, err := h.repo.GetMiddlewarePlugins(context.Background(), false)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get middleware plugins: %v", err),
		})
	}

	disabledCount := 0
	for _, plugin := range plugins {
		if plugin.Enabled {
			if err := h.repo.DisablePlugin(context.Background(), plugin.Name); err != nil {
				c.Logger().Errorf("Failed to disable %s: %v", plugin.Name, err)
			} else {
				// Unregister from chain
				h.manager.UnregisterMiddleware(plugin.Name)
				disabledCount++
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "All middleware disabled",
		"disabled": disabledCount,
	})
}

// Metrics API Handlers

// getAllMetrics returns metrics for all middleware
func (h *MiddlewareAPIHandler) getAllMetrics(c echo.Context) error {
	tester := h.manager.GetTester()
	if tester == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Middleware tester not available",
		})
	}

	allMetrics := tester.GetAllMetrics()

	result := make(map[string]interface{})
	for name, metrics := range allMetrics {
		result[name] = metrics.GetStats()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"metrics": result,
		"count":   len(result),
	})
}

// getMiddlewareMetrics returns metrics for a specific middleware
func (h *MiddlewareAPIHandler) getMiddlewareMetrics(c echo.Context) error {
	name := c.Param("name")

	tester := h.manager.GetTester()
	if tester == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Middleware tester not available",
		})
	}

	metrics := tester.GetMetrics(name)
	if metrics == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("No metrics found for middleware %s", name),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"middleware": name,
		"metrics":    metrics.GetStats(),
	})
}

// resetMiddlewareMetrics resets metrics for a specific middleware
func (h *MiddlewareAPIHandler) resetMiddlewareMetrics(c echo.Context) error {
	name := c.Param("name")

	tester := h.manager.GetTester()
	if tester == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Middleware tester not available",
		})
	}

	tester.ResetMetrics(name)

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Metrics reset for middleware %s", name),
	})
}

// Rollback API Handlers

// createSnapshot creates a snapshot of middleware state for rollback
func (h *MiddlewareAPIHandler) createSnapshot(c echo.Context) error {
	name := c.Param("name")

	rollback := h.manager.GetRollback()
	if rollback == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Rollback manager not available",
		})
	}

	if err := rollback.CreateSnapshot(context.Background(), name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create snapshot: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Snapshot created for middleware %s", name),
	})
}

// rollbackMiddleware rolls back middleware to previous snapshot
func (h *MiddlewareAPIHandler) rollbackMiddleware(c echo.Context) error {
	name := c.Param("name")

	rollback := h.manager.GetRollback()
	if rollback == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Rollback manager not available",
		})
	}

	if err := rollback.Rollback(context.Background(), name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Rollback failed: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Middleware %s rolled back successfully", name),
	})
}

// getSnapshots returns snapshot history for a middleware
func (h *MiddlewareAPIHandler) getSnapshots(c echo.Context) error {
	name := c.Param("name")

	rollback := h.manager.GetRollback()
	if rollback == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "Rollback manager not available",
		})
	}

	snapshots := rollback.GetSnapshots(name)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"middleware": name,
		"snapshots":  snapshots,
		"count":      len(snapshots),
	})
}

// Helper functions

func calculateAveragePriority(chain []plugins.MiddlewareEntry) float64 {
	if len(chain) == 0 {
		return 0
	}

	sum := 0
	for _, entry := range chain {
		sum += entry.Priority
	}

	return float64(sum) / float64(len(chain))
}
