package wasm

import (
	"context"
	"time"
)

// PluginType defines the type of WASM plugin
type PluginType string

const (
	PluginTypeRequest     PluginType = "request"     // Request transformation
	PluginTypeResponse    PluginType = "response"    // Response transformation
	PluginTypeAuth        PluginType = "auth"        // Authentication
	PluginTypeRateLimit   PluginType = "ratelimit"   // Rate limiting
	PluginTypeMiddleware  PluginType = "middleware"  // General middleware
	PluginTypeAggregation PluginType = "aggregation" // Response aggregation
)

// PluginConfig defines configuration for a WASM plugin
type PluginConfig struct {
	Name        string                 `yaml:"name" json:"name"`
	Path        string                 `yaml:"path" json:"path"` // Path to .wasm file
	Type        PluginType             `yaml:"type" json:"type"`
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	Priority    int                    `yaml:"priority" json:"priority"`       // Execution order (lower runs first)
	Config      map[string]interface{} `yaml:"config" json:"config"`           // Plugin-specific config
	Timeout     time.Duration          `yaml:"timeout" json:"timeout"`         // Execution timeout
	AllowedURLs []string               `yaml:"allowedUrls" json:"allowedUrls"` // URL patterns this plugin applies to
	Services    []string               `yaml:"services" json:"services"`       // Service names this plugin applies to
}

// Config defines the WASM extension configuration
type Config struct {
	Enabled        bool           `yaml:"enabled" json:"enabled"`
	PluginDir      string         `yaml:"pluginDir" json:"pluginDir"` // Directory containing WASM plugins
	Plugins        []PluginConfig `yaml:"plugins" json:"plugins"`
	MaxMemoryPages int            `yaml:"maxMemoryPages" json:"maxMemoryPages"` // Max memory pages (64KB each)
	MaxInstances   int            `yaml:"maxInstances" json:"maxInstances"`     // Max plugin instances
	CacheEnabled   bool           `yaml:"cacheEnabled" json:"cacheEnabled"`     // Cache compiled modules
}

// HTTPRequest represents an HTTP request for WASM plugins
type HTTPRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
	Query   map[string][]string `json:"query"`
	Params  map[string]string   `json:"params"`
}

// HTTPResponse represents an HTTP response for WASM plugins
type HTTPResponse struct {
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers"`
	Body       []byte              `json:"body"`
}

// PluginContext provides context to WASM plugins
type PluginContext struct {
	RequestID string                 `json:"requestId"`
	ServiceID string                 `json:"serviceId"`
	RoutePath string                 `json:"routePath"`
	Timestamp time.Time              `json:"timestamp"`
	Config    map[string]interface{} `json:"config"`
	Metadata  map[string]string      `json:"metadata"`
}

// PluginResult represents the result of plugin execution
type PluginResult struct {
	Modified bool              `json:"modified"` // Whether the request/response was modified
	Continue bool              `json:"continue"` // Whether to continue processing
	Response *HTTPResponse     `json:"response"` // Optional response to return immediately
	Error    string            `json:"error"`    // Error message if any
	Metadata map[string]string `json:"metadata"` // Metadata to pass to next plugin
}

// Plugin interface defines operations for WASM plugins
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Type returns the plugin type
	Type() PluginType

	// Execute runs the plugin with the given context and input
	Execute(ctx context.Context, pluginCtx *PluginContext, input interface{}) (*PluginResult, error)

	// Close cleans up plugin resources
	Close() error
}

// Runtime manages WASM plugin lifecycle
type Runtime interface {
	// LoadPlugin loads a WASM plugin from file
	LoadPlugin(config *PluginConfig) (Plugin, error)

	// GetPlugin retrieves a loaded plugin by name
	GetPlugin(name string) (Plugin, error)

	// ListPlugins returns all loaded plugins
	ListPlugins() []Plugin

	// UnloadPlugin unloads a plugin
	UnloadPlugin(name string) error

	// Close shuts down the runtime
	Close() error
}

// HostFunctions defines functions available to WASM plugins
type HostFunctions interface {
	// Log writes a log message
	Log(level, message string)

	// GetConfig retrieves configuration value
	GetConfig(key string) (string, error)

	// SetMetadata sets metadata value
	SetMetadata(key, value string)

	// GetMetadata retrieves metadata value
	GetMetadata(key string) (string, error)

	// HTTPCall makes an HTTP call (for service composition)
	HTTPCall(method, url string, headers map[string]string, body []byte) (*HTTPResponse, error)
}
