package admin

import (
	"odin/pkg/plugins"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

// PluginUploadHandler handles binary plugin uploads and management
type PluginUploadHandler struct {
	uploader      *plugins.PluginUploader
	pluginManager *plugins.PluginManager
	logger        *logrus.Logger
}

// NewPluginUploadHandler creates a new plugin upload handler
func NewPluginUploadHandler(db *mongo.Database, pluginManager *plugins.PluginManager, logger *logrus.Logger) (*PluginUploadHandler, error) {
	uploader, err := plugins.NewPluginUploader(db, logger)
	if err != nil {
		return nil, err
	}

	return &PluginUploadHandler{
		uploader:      uploader,
		pluginManager: pluginManager,
		logger:        logger,
	}, nil
}

// RegisterRoutes registers plugin upload routes
func (h *PluginUploadHandler) RegisterRoutes(adminGroup *echo.Group) {
	apiGroup := adminGroup.Group("/api/plugin-binaries")

	// Binary plugin upload and management
	apiGroup.POST("/upload", h.uploader.UploadPlugin)
	apiGroup.GET("", h.uploader.ListPlugins)
	apiGroup.GET("/stats", h.uploader.GetPluginStats)
	apiGroup.GET("/:id", h.uploader.GetPlugin)
	apiGroup.DELETE("/:id", h.wrapDelete)
	apiGroup.POST("/:id/enable", h.wrapEnable)
	apiGroup.POST("/:id/disable", h.wrapDisable)
	apiGroup.PUT("/:id/config", h.uploader.UpdatePluginConfig)

	// UI pages
	adminGroup.GET("/plugin-binaries/upload", h.uploadPage)
	adminGroup.GET("/plugin-binaries", h.listPage)
}

// Wrapper functions to pass plugin manager
func (h *PluginUploadHandler) wrapEnable(c echo.Context) error {
	return h.uploader.EnablePlugin(c, h.pluginManager)
}

func (h *PluginUploadHandler) wrapDisable(c echo.Context) error {
	return h.uploader.DisablePlugin(c, h.pluginManager)
}

func (h *PluginUploadHandler) wrapDelete(c echo.Context) error {
	return h.uploader.DeletePlugin(c, h.pluginManager)
}

// UI Handlers

func (h *PluginUploadHandler) uploadPage(c echo.Context) error {
	return c.Render(200, "plugin-binary-upload.html", map[string]interface{}{
		"Title": "Upload Plugin Binary",
	})
}

func (h *PluginUploadHandler) listPage(c echo.Context) error {
	return c.Render(200, "plugin-binaries.html", map[string]interface{}{
		"Title": "Plugin Binaries",
	})
}
