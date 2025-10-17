package admin

import (
	"odin/pkg/config"
	"odin/pkg/plugins"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type AdminCredentials struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type AdminHandler struct {
	config               *config.Config
	configPath           string
	logger               *logrus.Logger
	username             string
	password             string
	enabled              bool
	pluginHandler        *PluginHandler
	middlewareAPIHandler *MiddlewareAPIHandler
	integrationHandler   *IntegrationHandler
	pluginUploadHandler  *PluginUploadHandler
}

func loadAdminCredentials(logger *logrus.Logger) (*AdminCredentials, error) {
	credsPaths := []string{
		"config/admin_creds.yaml",
		"/etc/odin/admin_creds.yaml",
		filepath.Join(os.Getenv("HOME"), ".odin", "admin_creds.yaml"),
	}

	var credentials AdminCredentials
	var loaded bool

	for _, path := range credsPaths {
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				logger.WithError(err).Warnf("Failed to read admin credentials file: %s", path)
				continue
			}

			err = yaml.Unmarshal(data, &credentials)
			if err != nil {
				logger.WithError(err).Warnf("Failed to parse admin credentials file: %s", path)
				continue
			}

			loaded = true
			break
		}
	}

	if !loaded {
		return nil, nil
	}

	return &credentials, nil
}

func New(cfg *config.Config, configPath string, logger *logrus.Logger) *AdminHandler {
	creds, _ := loadAdminCredentials(logger)

	username := ""
	if creds != nil && creds.Username != "" {
		username = creds.Username
	} else {
		username = cfg.Admin.Username
	}

	if username == "" {
		username = "admin"
	}

	password := ""
	if creds != nil && creds.Password != "" {
		password = creds.Password
	} else {
		password = cfg.Admin.Password
	}

	if password == "" {
		password = "admin1"
	}

	return &AdminHandler{
		config:               cfg,
		configPath:           configPath,
		logger:               logger,
		username:             username,
		password:             password,
		enabled:              cfg.Admin.Enabled,
		pluginHandler:        nil, // Will be set later via SetPluginHandler
		middlewareAPIHandler: nil, // Will be set later via SetMiddlewareAPIHandler
		integrationHandler:   nil, // Will be set later via SetIntegrationHandler
		pluginUploadHandler:  nil, // Will be set later via SetPluginUploadHandler
	}
}

// SetPluginHandler sets the plugin handler for admin
func (h *AdminHandler) SetPluginHandler(pluginManager *plugins.PluginManager, pluginRepo *plugins.PluginRepository) {
	h.pluginHandler = NewPluginHandler(pluginManager, pluginRepo)
}

// SetMiddlewareAPIHandler sets the middleware API handler for admin
func (h *AdminHandler) SetMiddlewareAPIHandler(pluginManager *plugins.PluginManager, pluginRepo *plugins.PluginRepository) {
	h.middlewareAPIHandler = NewMiddlewareAPIHandler(pluginManager, pluginRepo)
}

// SetIntegrationHandler sets the integration handler for admin
func (h *AdminHandler) SetIntegrationHandler(handler *IntegrationHandler) {
	h.integrationHandler = handler
}

// SetPluginUploadHandler sets the plugin upload handler for admin
func (h *AdminHandler) SetPluginUploadHandler(handler *PluginUploadHandler) {
	h.pluginUploadHandler = handler
}

// GetIntegrationHandler returns the integration handler
func (h *AdminHandler) GetIntegrationHandler() *IntegrationHandler {
	return h.integrationHandler
}

func (h *AdminHandler) saveConfig() error {
	data, err := yaml.Marshal(h.config)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal configuration")
		return err
	}

	if err := os.WriteFile(h.configPath, data, 0644); err != nil {
		h.logger.WithError(err).Error("Failed to write configuration file")
		return err
	}

	h.logger.Info("Configuration saved successfully")
	return nil
}
