package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"odin/pkg/plugins"

	"github.com/labstack/echo/v4"
)

// PluginHandler handles plugin-related admin API requests
type PluginHandler struct {
	manager *plugins.PluginManager
	repo    *plugins.PluginRepository
}

// NewPluginHandler creates a new plugin handler
func NewPluginHandler(manager *plugins.PluginManager, repo *plugins.PluginRepository) *PluginHandler {
	return &PluginHandler{
		manager: manager,
		repo:    repo,
	}
}

// RegisterPluginRoutes registers plugin admin routes
func (h *PluginHandler) RegisterPluginRoutes(adminGroup *echo.Group) {
	// API routes
	apiGroup := adminGroup.Group("/api/plugins")
	apiGroup.GET("", h.listPlugins)
	apiGroup.POST("", h.createPlugin)
	apiGroup.POST("/upload", h.uploadPlugin)
	apiGroup.POST("/build", h.buildPlugin)
	apiGroup.POST("/test/:name", h.testPlugin)
	apiGroup.GET("/:name", h.getPlugin)
	apiGroup.PUT("/:name", h.updatePlugin)
	apiGroup.DELETE("/:name", h.deletePlugin)
	apiGroup.POST("/:name/enable", h.enablePlugin)
	apiGroup.POST("/:name/disable", h.disablePlugin)
	apiGroup.POST("/:name/load", h.loadPlugin)
	apiGroup.POST("/:name/unload", h.unloadPlugin)

	// UI routes
	adminGroup.GET("/plugins", h.pluginsPage)
	adminGroup.GET("/plugins/new", h.newPluginPage)
	adminGroup.GET("/plugins/:name", h.pluginDetailPage)
}

// List Plugins API
func (h *PluginHandler) listPlugins(c echo.Context) error {
	pluginList, err := h.repo.ListPlugins(context.Background(), nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to list plugins: %v", err),
		})
	}

	// Add loaded status
	loadedPlugins := h.manager.ListPlugins()
	loadedMap := make(map[string]bool)
	for _, name := range loadedPlugins {
		loadedMap[name] = true
	}

	type PluginResponse struct {
		plugins.PluginRecord
		Loaded bool `json:"loaded"`
	}

	response := make([]PluginResponse, len(pluginList))
	for i, p := range pluginList {
		response[i] = PluginResponse{
			PluginRecord: *p,
			Loaded:       loadedMap[p.Name],
		}
	}

	return c.JSON(http.StatusOK, response)
}

// Get Plugin API
func (h *PluginHandler) getPlugin(c echo.Context) error {
	name := c.Param("name")

	plugin, err := h.repo.GetPlugin(context.Background(), name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Plugin not found: %v", err),
		})
	}

	// Check if loaded
	_, loaded := h.manager.GetPlugin(name)

	type PluginResponse struct {
		*plugins.PluginRecord
		Loaded bool `json:"loaded"`
	}

	return c.JSON(http.StatusOK, PluginResponse{
		PluginRecord: plugin,
		Loaded:       loaded,
	})
}

// Create Plugin API
func (h *PluginHandler) createPlugin(c echo.Context) error {
	var plugin plugins.PluginRecord

	if err := c.Bind(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Validate required fields
	if plugin.Name == "" || plugin.BinaryPath == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name and BinaryPath are required",
		})
	}

	// Save to database
	if err := h.repo.SavePlugin(context.Background(), &plugin); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to save plugin: %v", err),
		})
	}

	// Auto-load if enabled
	if plugin.Enabled {
		if err := h.manager.LoadPlugin(plugin.Name, plugin.BinaryPath, plugin.Config, plugin.Hooks); err != nil {
			// Log but don't fail
			c.Logger().Errorf("Failed to load plugin %s: %v", plugin.Name, err)
		}
	}

	return c.JSON(http.StatusCreated, plugin)
}

// Update Plugin API
func (h *PluginHandler) updatePlugin(c echo.Context) error {
	name := c.Param("name")

	var plugin plugins.PluginRecord
	if err := c.Bind(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Ensure name matches
	plugin.Name = name

	// Update in database
	if err := h.repo.UpdatePlugin(context.Background(), &plugin); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to update plugin: %v", err),
		})
	}

	// Reload if currently loaded
	if _, loaded := h.manager.GetPlugin(name); loaded {
		h.manager.UnloadPlugin(name)
		if plugin.Enabled {
			if err := h.manager.LoadPlugin(plugin.Name, plugin.BinaryPath, plugin.Config, plugin.Hooks); err != nil {
				c.Logger().Errorf("Failed to reload plugin %s: %v", plugin.Name, err)
			}
		}
	}

	return c.JSON(http.StatusOK, plugin)
}

// Delete Plugin API
func (h *PluginHandler) deletePlugin(c echo.Context) error {
	name := c.Param("name")

	// Unload if loaded
	if _, loaded := h.manager.GetPlugin(name); loaded {
		if err := h.manager.UnloadPlugin(name); err != nil {
			c.Logger().Errorf("Failed to unload plugin %s: %v", name, err)
		}
	}

	// Delete from database
	if err := h.repo.DeletePlugin(context.Background(), name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to delete plugin: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Plugin deleted successfully",
	})
}

// Enable Plugin API
func (h *PluginHandler) enablePlugin(c echo.Context) error {
	name := c.Param("name")

	if err := h.repo.EnablePlugin(context.Background(), name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to enable plugin: %v", err),
		})
	}

	// Load the plugin
	plugin, err := h.repo.GetPlugin(context.Background(), name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get plugin: %v", err),
		})
	}

	// Load based on plugin type
	if plugin.PluginType == "middleware" {
		if err := h.manager.LoadMiddleware(plugin.Name, plugin.BinaryPath, plugin.Config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to load middleware: %v", err),
			})
		}
	} else {
		if err := h.manager.LoadPlugin(plugin.Name, plugin.BinaryPath, plugin.Config, plugin.Hooks); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to load plugin: %v", err),
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Plugin enabled successfully",
	})
}

// Disable Plugin API
func (h *PluginHandler) disablePlugin(c echo.Context) error {
	name := c.Param("name")

	if err := h.repo.DisablePlugin(context.Background(), name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to disable plugin: %v", err),
		})
	}

	// Get plugin to check type
	plugin, err := h.repo.GetPlugin(context.Background(), name)
	if err == nil {
		// Unload based on type
		if plugin.PluginType == "middleware" {
			if err := h.manager.UnloadMiddleware(name); err != nil {
				c.Logger().Errorf("Failed to unload middleware %s: %v", name, err)
			}
		} else {
			if err := h.manager.UnloadPlugin(name); err != nil {
				c.Logger().Errorf("Failed to unload plugin %s: %v", name, err)
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Plugin disabled successfully",
	})
}

// Load Plugin API
func (h *PluginHandler) loadPlugin(c echo.Context) error {
	name := c.Param("name")

	plugin, err := h.repo.GetPlugin(context.Background(), name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Plugin not found: %v", err),
		})
	}

	if err := h.manager.LoadPlugin(plugin.Name, plugin.BinaryPath, plugin.Config, plugin.Hooks); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to load plugin: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Plugin loaded successfully",
	})
}

// Unload Plugin API
func (h *PluginHandler) unloadPlugin(c echo.Context) error {
	name := c.Param("name")

	if err := h.manager.UnloadPlugin(name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to unload plugin: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Plugin unloaded successfully",
	})
}

// UI Handlers

func (h *PluginHandler) pluginsPage(c echo.Context) error {
	pluginList, err := h.repo.ListPlugins(context.Background(), nil)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load plugins")
	}

	// Add loaded status
	loadedPlugins := h.manager.ListPlugins()
	loadedMap := make(map[string]bool)
	for _, name := range loadedPlugins {
		loadedMap[name] = true
	}

	type PluginView struct {
		plugins.PluginRecord
		Loaded       bool
		EnabledBadge template.HTML
		LoadedBadge  template.HTML
	}

	pluginViews := make([]PluginView, len(pluginList))
	for i, p := range pluginList {
		enabledBadge := template.HTML(`<span class="badge bg-success">Enabled</span>`)
		if !p.Enabled {
			enabledBadge = template.HTML(`<span class="badge bg-secondary">Disabled</span>`)
		}

		loadedBadge := template.HTML(`<span class="badge bg-success">Loaded</span>`)
		if !loadedMap[p.Name] {
			loadedBadge = template.HTML(`<span class="badge bg-secondary">Not Loaded</span>`)
		}

		pluginViews[i] = PluginView{
			PluginRecord: *p,
			Loaded:       loadedMap[p.Name],
			EnabledBadge: enabledBadge,
			LoadedBadge:  loadedBadge,
		}
	}

	data := map[string]interface{}{
		"Title":   "Plugins",
		"Plugins": pluginViews,
	}

	// Render template (you'll need to create this template)
	return c.Render(http.StatusOK, "plugins.html", data)
}

func (h *PluginHandler) newPluginPage(c echo.Context) error {
	data := map[string]interface{}{
		"Title": "Add New Plugin",
	}

	return c.Render(http.StatusOK, "plugin_new.html", data)
}

func (h *PluginHandler) pluginDetailPage(c echo.Context) error {
	name := c.Param("name")

	plugin, err := h.repo.GetPlugin(context.Background(), name)
	if err != nil {
		return c.String(http.StatusNotFound, "Plugin not found")
	}

	// Check if loaded
	_, loaded := h.manager.GetPlugin(name)

	// Convert config to JSON for display
	configJSON, _ := json.MarshalIndent(plugin.Config, "", "  ")

	// Create template data with helper methods
	type PluginDetailView struct {
		*plugins.PluginRecord
		Loaded       bool
		LoadedBadge  template.HTML
		EnabledBadge template.HTML
		ConfigJSON   string
		CreatedAt    string
		UpdatedAt    string
	}

	enabledBadge := template.HTML(`<span class="badge bg-success">Enabled</span>`)
	if !plugin.Enabled {
		enabledBadge = template.HTML(`<span class="badge bg-secondary">Disabled</span>`)
	}

	loadedBadge := template.HTML(`<span class="badge bg-success">Loaded</span>`)
	if !loaded {
		loadedBadge = template.HTML(`<span class="badge bg-secondary">Not Loaded</span>`)
	}

	pluginView := PluginDetailView{
		PluginRecord: plugin,
		Loaded:       loaded,
		LoadedBadge:  loadedBadge,
		EnabledBadge: enabledBadge,
		ConfigJSON:   string(configJSON),
		CreatedAt:    plugin.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    plugin.UpdatedAt.Format(time.RFC3339),
	}

	data := map[string]interface{}{
		"Title":  fmt.Sprintf("Plugin: %s", plugin.Name),
		"Plugin": pluginView,
	}

	return c.Render(http.StatusOK, "plugin_detail.html", data)
}
