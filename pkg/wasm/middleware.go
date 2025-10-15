package wasm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Middleware creates an Echo middleware for WASM plugin execution
func Middleware(runtime Runtime, logger *logrus.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get all plugins
			plugins := runtime.ListPlugins()
			if len(plugins) == 0 {
				return next(c)
			}

			// Filter plugins for this request
			applicablePlugins := filterPluginsForRequest(plugins, c)
			if len(applicablePlugins) == 0 {
				return next(c)
			}

			// Create plugin context
			pluginCtx := &PluginContext{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				ServiceID: c.Get("service_id").(string),
				RoutePath: c.Path(),
				Metadata:  make(map[string]string),
			}

			// Execute request plugins
			for _, plugin := range applicablePlugins {
				if plugin.Type() == PluginTypeRequest || plugin.Type() == PluginTypeMiddleware || plugin.Type() == PluginTypeAuth {
					req, err := requestToHTTPRequest(c.Request())
					if err != nil {
						logger.WithError(err).Error("Failed to convert request")
						continue
					}

					result, err := plugin.Execute(c.Request().Context(), pluginCtx, req)
					if err != nil {
						logger.WithError(err).WithField("plugin", plugin.Name()).Error("Plugin execution failed")
						continue
					}

					// Check if plugin wants to stop processing
					if !result.Continue {
						if result.Response != nil {
							return writeResponse(c, result.Response)
						}
						return echo.NewHTTPError(http.StatusForbidden, result.Error)
					}

					// Update metadata
					if result.Metadata != nil {
						for k, v := range result.Metadata {
							pluginCtx.Metadata[k] = v
						}
					}

					// Apply modifications if any
					if result.Modified && result.Response != nil {
						if err := applyRequestModifications(c, result.Response); err != nil {
							logger.WithError(err).Error("Failed to apply request modifications")
						}
					}
				}
			}

			// Call next handler
			if err := next(c); err != nil {
				return err
			}

			// Execute response plugins
			for _, plugin := range applicablePlugins {
				if plugin.Type() == PluginTypeResponse || plugin.Type() == PluginTypeMiddleware {
					resp := &HTTPResponse{
						StatusCode: c.Response().Status,
						Headers:    c.Response().Header(),
						Body:       c.Response().Writer.(interface{ Bytes() []byte }).Bytes(),
					}

					result, err := plugin.Execute(c.Request().Context(), pluginCtx, resp)
					if err != nil {
						logger.WithError(err).WithField("plugin", plugin.Name()).Error("Plugin execution failed")
						continue
					}

					// Apply response modifications
					if result.Modified && result.Response != nil {
						if err := applyResponseModifications(c, result.Response); err != nil {
							logger.WithError(err).Error("Failed to apply response modifications")
						}
					}
				}
			}

			return nil
		}
	}
}

// filterPluginsForRequest filters plugins applicable to the current request
func filterPluginsForRequest(plugins []Plugin, c echo.Context) []Plugin {
	var applicable []Plugin

	for _, plugin := range plugins {
		// Check if plugin applies to this service
		if config := getPluginConfig(plugin); config != nil {
			if len(config.Services) > 0 {
				serviceID, ok := c.Get("service_id").(string)
				if !ok || !contains(config.Services, serviceID) {
					continue
				}
			}

			// Check if plugin applies to this URL
			if len(config.AllowedURLs) > 0 {
				matched := false
				for _, pattern := range config.AllowedURLs {
					if matched, _ := regexp.MatchString(pattern, c.Request().URL.Path); matched {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
		}

		applicable = append(applicable, plugin)
	}

	return applicable
}

// requestToHTTPRequest converts echo.Context request to HTTPRequest
func requestToHTTPRequest(req *http.Request) (*HTTPRequest, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewBuffer(body)) // Restore body

	return &HTTPRequest{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: req.Header,
		Body:    body,
		Query:   req.URL.Query(),
	}, nil
}

// applyRequestModifications applies plugin modifications to the request
func applyRequestModifications(c echo.Context, resp *HTTPResponse) error {
	// Update headers
	for k, values := range resp.Headers {
		c.Request().Header.Del(k)
		for _, v := range values {
			c.Request().Header.Add(k, v)
		}
	}

	// Update body
	if len(resp.Body) > 0 {
		c.Request().Body = io.NopCloser(bytes.NewBuffer(resp.Body))
	}

	return nil
}

// applyResponseModifications applies plugin modifications to the response
func applyResponseModifications(c echo.Context, resp *HTTPResponse) error {
	// Update status code
	c.Response().Status = resp.StatusCode

	// Update headers
	for k, values := range resp.Headers {
		c.Response().Header().Del(k)
		for _, v := range values {
			c.Response().Header().Add(k, v)
		}
	}

	// Update body (if possible)
	if len(resp.Body) > 0 {
		return c.Blob(resp.StatusCode, c.Response().Header().Get(echo.HeaderContentType), resp.Body)
	}

	return nil
}

// writeResponse writes an HTTP response
func writeResponse(c echo.Context, resp *HTTPResponse) error {
	// Set headers
	for k, values := range resp.Headers {
		for _, v := range values {
			c.Response().Header().Add(k, v)
		}
	}

	// Set content type if not set
	if c.Response().Header().Get(echo.HeaderContentType) == "" {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}

	return c.Blob(resp.StatusCode, c.Response().Header().Get(echo.HeaderContentType), resp.Body)
}

// getPluginConfig extracts PluginConfig from Plugin (helper)
func getPluginConfig(plugin Plugin) *PluginConfig {
	if wp, ok := plugin.(*wasmPlugin); ok {
		return wp.config
	}
	return nil
}

// contains checks if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// PluginInfo contains information about a loaded plugin
type PluginInfo struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Enabled  bool                   `json:"enabled"`
	Priority int                    `json:"priority"`
	Config   map[string]interface{} `json:"config"`
}

// GetPluginInfo returns information about all loaded plugins
func GetPluginInfo(runtime Runtime) []PluginInfo {
	plugins := runtime.ListPlugins()
	info := make([]PluginInfo, 0, len(plugins))

	for _, plugin := range plugins {
		config := getPluginConfig(plugin)
		if config != nil {
			info = append(info, PluginInfo{
				Name:     plugin.Name(),
				Type:     string(plugin.Type()),
				Enabled:  config.Enabled,
				Priority: config.Priority,
				Config:   config.Config,
			})
		}
	}

	return info
}

// ExecutePluginHandler creates a handler for manually executing plugins (for testing)
func ExecutePluginHandler(runtime Runtime, logger *logrus.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		pluginName := c.Param("name")

		plugin, err := runtime.GetPlugin(pluginName)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "Plugin not found")
		}

		// Parse request body as input
		var input interface{}
		if err := json.NewDecoder(c.Request().Body).Decode(&input); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid input")
		}

		// Create context
		pluginCtx := &PluginContext{
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			Metadata:  make(map[string]string),
		}

		// Execute plugin
		result, err := plugin.Execute(c.Request().Context(), pluginCtx, input)
		if err != nil {
			logger.WithError(err).WithField("plugin", pluginName).Error("Plugin execution failed")
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, result)
	}
}
