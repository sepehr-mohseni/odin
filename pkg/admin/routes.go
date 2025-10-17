package admin

import (
	"github.com/labstack/echo/v4"
)

func (h *AdminHandler) Register(e *echo.Echo) {
	// Serve static assets (old assets directory)
	e.Static("/admin/assets", "pkg/admin/assets")

	// Serve new static files (CSS, JS)
	e.Static("/static", "pkg/admin/static")

	h.initTemplates()

	adminGroup := e.Group("/admin")

	adminGroup.GET("", h.handleLogin)
	adminGroup.GET("/login", h.handleLogin)
	adminGroup.POST("/login", h.handleLoginPost)

	protected := adminGroup.Group("")
	protected.Use(h.basicAuthMiddleware)

	protected.GET("/dashboard", h.handleDashboard)

	// Monitoring routes
	protected.GET("/monitoring", h.handleMonitoring)
	protected.GET("/api/monitoring/metrics", GetMetricsAPI)
	protected.GET("/ws/monitoring", WebSocketMonitoring)

	// Traces routes
	protected.GET("/traces", h.handleTraces)

	// Middleware chain management
	protected.GET("/middleware-chain", h.handleMiddlewareChain)

	// Debug endpoint for testing (remove in production)
	adminGroup.GET("/debug/metrics", GetMetricsAPI)

	protected.GET("/services", h.handleListServices)
	protected.GET("/services/new", h.handleNewService)
	protected.POST("/services", h.handleAddService)
	protected.GET("/services/:name", h.handleEditService)
	protected.POST("/services/:name", h.handleUpdateService)
	protected.DELETE("/services/:name", h.handleDeleteService)

	// Settings API routes
	settingsHandler := NewSettingsHandler(h.configPath, h.config)

	protected.GET("/settings", h.handleSettings)
	protected.GET("/api/settings", settingsHandler.GetAllSettings)
	protected.GET("/api/settings/info", settingsHandler.GetGatewayInfo)
	protected.GET("/api/settings/stats", settingsHandler.GetSystemStats)

	// Server settings
	protected.GET("/api/settings/server", settingsHandler.GetServerSettings)
	protected.PUT("/api/settings/server", settingsHandler.UpdateServerSettings)

	// Logging settings
	protected.GET("/api/settings/logging", settingsHandler.GetLoggingSettings)
	protected.PUT("/api/settings/logging", settingsHandler.UpdateLoggingSettings)

	// Rate limit settings
	protected.GET("/api/settings/ratelimit", settingsHandler.GetRateLimitSettings)
	protected.PUT("/api/settings/ratelimit", settingsHandler.UpdateRateLimitSettings)

	// Cache settings
	protected.GET("/api/settings/cache", settingsHandler.GetCacheSettings)
	protected.PUT("/api/settings/cache", settingsHandler.UpdateCacheSettings)

	// Monitoring settings
	protected.GET("/api/settings/monitoring", settingsHandler.GetMonitoringSettings)
	protected.PUT("/api/settings/monitoring", settingsHandler.UpdateMonitoringSettings)

	// Tracing settings
	protected.GET("/api/settings/tracing", settingsHandler.GetTracingSettings)
	protected.PUT("/api/settings/tracing", settingsHandler.UpdateTracingSettings)

	// Config management
	protected.GET("/api/settings/export", settingsHandler.ExportConfig)
	protected.POST("/api/settings/validate", settingsHandler.ValidateConfig)
	protected.GET("/api/settings/backups", settingsHandler.GetConfigBackups)
	protected.POST("/api/settings/backups/:name/restore", settingsHandler.RestoreConfigBackup)
	protected.POST("/api/settings/cache/clear", settingsHandler.ClearCache)
	protected.POST("/api/settings/reload", settingsHandler.ReloadConfig)
	protected.GET("/api/settings/json", settingsHandler.GetConfigAsJSON)

	// Register plugin routes if plugin handler is available
	if h.pluginHandler != nil {
		h.pluginHandler.RegisterPluginRoutes(protected)
	}

	// Register middleware API routes if middleware handler is available
	if h.middlewareAPIHandler != nil {
		h.middlewareAPIHandler.RegisterMiddlewareAPIRoutes(protected)
	}

	// Register plugin upload routes if plugin upload handler is available
	if h.pluginUploadHandler != nil {
		h.pluginUploadHandler.RegisterRoutes(protected)
	}

	// Register integration routes if integration handler is available
	if h.integrationHandler != nil {
		protected.GET("/integrations/postman", h.handleIntegrationsPostman)
		h.integrationHandler.RegisterRoutes(protected)
	}

	h.logger.Info("Admin routes registered")
}

func (h *AdminHandler) handleDashboard(c echo.Context) error {
	return h.renderTemplate(c, "dashboard.html", nil)
}

func (h *AdminHandler) handleMonitoring(c echo.Context) error {
	return h.renderTemplate(c, "monitoring.html", nil)
}

func (h *AdminHandler) handleTraces(c echo.Context) error {
	return h.renderTemplate(c, "traces.html", nil)
}

func (h *AdminHandler) handleMiddlewareChain(c echo.Context) error {
	return h.renderTemplate(c, "middleware_chain.html", nil)
}

func (h *AdminHandler) handleSettings(c echo.Context) error {
	return h.renderTemplate(c, "settings.html", nil)
}
