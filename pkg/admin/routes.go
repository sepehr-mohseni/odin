package admin

import (
	"github.com/labstack/echo/v4"
)

func (h *AdminHandler) Register(e *echo.Echo) {
	e.Static("/admin/assets", "pkg/admin/assets")

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

	// Debug endpoint for testing (remove in production)
	adminGroup.GET("/debug/metrics", GetMetricsAPI)

	protected.GET("/services", h.handleListServices)
	protected.GET("/services/new", h.handleNewService)
	protected.POST("/services", h.handleAddService)
	protected.GET("/services/:name", h.handleEditService)
	protected.POST("/services/:name", h.handleUpdateService)
	protected.DELETE("/services/:name", h.handleDeleteService)

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
