package plugins

import (
	"context"
	"fmt"
	"plugin"
	"reflect"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// HookType represents the type of plugin hook
type HookType string

const (
	PreRequestHook   HookType = "pre-request"
	PostRequestHook  HookType = "post-request"
	PreResponseHook  HookType = "pre-response"
	PostResponseHook HookType = "post-response"
)

// PluginContext contains request/response context for plugins
type PluginContext struct {
	RequestID   string
	ServiceName string
	Path        string
	Method      string
	Headers     map[string][]string
	Body        []byte
	UserID      string
	Metadata    map[string]interface{}
}

// Plugin interface that all plugins must implement
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Version returns the plugin version
	Version() string

	// Initialize initializes the plugin with configuration
	Initialize(config map[string]interface{}) error

	// PreRequest is called before forwarding the request
	PreRequest(ctx context.Context, pluginCtx *PluginContext) error

	// PostRequest is called after receiving the response
	PostRequest(ctx context.Context, pluginCtx *PluginContext) error

	// PreResponse is called before sending response to client
	PreResponse(ctx context.Context, pluginCtx *PluginContext) error

	// PostResponse is called after sending response to client
	PostResponse(ctx context.Context, pluginCtx *PluginContext) error

	// Cleanup is called when the plugin is being unloaded
	Cleanup() error
}

// Middleware interface for Traefik-style middleware plugins
type Middleware interface {
	// Name returns the middleware name
	Name() string

	// Version returns the middleware version
	Version() string

	// Initialize initializes the middleware with configuration
	Initialize(config map[string]interface{}) error

	// Handle wraps the next handler in the chain
	Handle(next echo.HandlerFunc) echo.HandlerFunc

	// Cleanup is called when the middleware is being unloaded
	Cleanup() error
}

// MiddlewareChain represents an ordered list of middleware
type MiddlewareChain struct {
	Middlewares []MiddlewareEntry
	mu          sync.RWMutex
}

// MiddlewareEntry represents a middleware with its metadata
type MiddlewareEntry struct {
	Name       string
	Middleware Middleware
	Priority   int
	Routes     []string // Routes this middleware applies to (supports wildcards)
	Phase      string   // Middleware execution phase
}

// PluginManager manages all loaded plugins
type PluginManager struct {
	plugins         map[string]Plugin
	middlewares     map[string]Middleware
	hooks           map[HookType][]Plugin
	middlewareChain *MiddlewareChain
	tester          *MiddlewareTester
	rollback        *MiddlewareRollback
	logger          *logrus.Logger
	mu              sync.RWMutex
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(logger *logrus.Logger) *PluginManager {
	pm := &PluginManager{
		plugins:     make(map[string]Plugin),
		middlewares: make(map[string]Middleware),
		hooks: map[HookType][]Plugin{
			PreRequestHook:   {},
			PostRequestHook:  {},
			PreResponseHook:  {},
			PostResponseHook: {},
		},
		middlewareChain: &MiddlewareChain{
			Middlewares: []MiddlewareEntry{},
		},
		logger: logger,
	}

	// Initialize tester with the plugin manager
	pm.tester = NewMiddlewareTester(pm, logger)

	return pm
}

// createPluginWrapper creates a wrapper for plugins that may have interface compatibility issues
func (pm *PluginManager) createPluginWrapper(sym interface{}) Plugin {
	// Handle the common double-pointer issue with Go plugins
	v := reflect.ValueOf(sym)

	// If it's a pointer to a pointer, dereference it
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	if !v.IsValid() {
		return nil
	}

	// Get the underlying value
	actualPlugin := v.Interface()

	// Try direct type assertion first
	if plugin, ok := actualPlugin.(Plugin); ok {
		return plugin
	}

	// If that fails, create a reflection-based wrapper
	return pm.createReflectionWrapper(actualPlugin)
}

// createReflectionWrapper creates a wrapper using reflection for interface compatibility
func (pm *PluginManager) createReflectionWrapper(obj interface{}) Plugin {
	v := reflect.ValueOf(obj)
	t := reflect.TypeOf(obj)

	// For struct values, we need to get the pointer type to check methods
	if t.Kind() == reflect.Struct {
		ptrType := reflect.PointerTo(t)
		t = ptrType
	}

	// Check if the object has all required methods
	requiredMethods := []string{"Name", "Version", "Initialize", "PreRequest", "PostRequest", "PreResponse", "PostResponse", "Cleanup"}

	for _, methodName := range requiredMethods {
		if _, found := t.MethodByName(methodName); !found {
			pm.logger.WithFields(logrus.Fields{
				"type":           t.String(),
				"missing_method": methodName,
			}).Debug("Plugin missing required method")
			return nil
		}
	}

	// If we have a struct value but need pointer methods, get the address
	if v.Kind() == reflect.Struct {
		if v.CanAddr() {
			v = v.Addr()
		} else {
			// Create a new addressable copy
			newV := reflect.New(v.Type())
			newV.Elem().Set(v)
			v = newV
		}
	}

	// Create a wrapper that implements Plugin interface
	return &pluginWrapper{
		obj:    obj,
		value:  v,
		logger: pm.logger,
	}
}

// pluginWrapper wraps a plugin object and implements the Plugin interface using reflection
type pluginWrapper struct {
	obj    interface{}
	value  reflect.Value
	logger *logrus.Logger
}

func (w *pluginWrapper) Name() string {
	result := w.value.MethodByName("Name").Call(nil)
	return result[0].String()
}

func (w *pluginWrapper) Version() string {
	result := w.value.MethodByName("Version").Call(nil)
	return result[0].String()
}

func (w *pluginWrapper) Initialize(config map[string]interface{}) error {
	args := []reflect.Value{reflect.ValueOf(config)}
	result := w.value.MethodByName("Initialize").Call(args)
	if !result[0].IsNil() {
		return result[0].Interface().(error)
	}
	return nil
}

func (w *pluginWrapper) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(pluginCtx)}
	result := w.value.MethodByName("PreRequest").Call(args)
	if !result[0].IsNil() {
		return result[0].Interface().(error)
	}
	return nil
}

func (w *pluginWrapper) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(pluginCtx)}
	result := w.value.MethodByName("PostRequest").Call(args)
	if !result[0].IsNil() {
		return result[0].Interface().(error)
	}
	return nil
}

func (w *pluginWrapper) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(pluginCtx)}
	result := w.value.MethodByName("PreResponse").Call(args)
	if !result[0].IsNil() {
		return result[0].Interface().(error)
	}
	return nil
}

func (w *pluginWrapper) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(pluginCtx)}
	result := w.value.MethodByName("PostResponse").Call(args)
	if !result[0].IsNil() {
		return result[0].Interface().(error)
	}
	return nil
}

func (w *pluginWrapper) Cleanup() error {
	result := w.value.MethodByName("Cleanup").Call(nil)
	if !result[0].IsNil() {
		return result[0].Interface().(error)
	}
	return nil
}

// LoadPlugin loads a plugin from a file
func (pm *PluginManager) LoadPlugin(name, path string, config map[string]interface{}, hooks []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Load the plugin file
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %w", name, err)
	}

	// Look for the plugin symbol
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export Plugin symbol: %w", name, err)
	}

	// Type assert to Plugin interface with reflection fallback
	pluginInstance, ok := sym.(Plugin)
	if !ok {
		// Try to handle interface compatibility issues with Go plugins
		pm.logger.WithFields(logrus.Fields{
			"plugin":      name,
			"symbol_type": fmt.Sprintf("%T", sym),
		}).Debug("Direct type assertion failed, attempting interface compatibility check")

		// Create a compatibility wrapper if the plugin has the right methods
		if wrapper := pm.createPluginWrapper(sym); wrapper != nil {
			pm.logger.WithField("plugin", name).Info("Created compatibility wrapper for plugin")
			pluginInstance = wrapper
		} else {
			return fmt.Errorf("plugin %s does not implement Plugin interface (got type %T)", name, sym)
		}
	}

	// Initialize the plugin
	if err := pluginInstance.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	// Store the plugin
	pm.plugins[name] = pluginInstance

	// Register hooks
	for _, hookType := range hooks {
		switch HookType(hookType) {
		case PreRequestHook:
			pm.hooks[PreRequestHook] = append(pm.hooks[PreRequestHook], pluginInstance)
		case PostRequestHook:
			pm.hooks[PostRequestHook] = append(pm.hooks[PostRequestHook], pluginInstance)
		case PreResponseHook:
			pm.hooks[PreResponseHook] = append(pm.hooks[PreResponseHook], pluginInstance)
		case PostResponseHook:
			pm.hooks[PostResponseHook] = append(pm.hooks[PostResponseHook], pluginInstance)
		}
	}

	pm.logger.WithFields(logrus.Fields{
		"plugin": name,
		"hooks":  hooks,
	}).Info("Plugin loaded successfully")

	return nil
}

// UnloadPlugin unloads a plugin
func (pm *PluginManager) UnloadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Cleanup the plugin
	if err := plugin.Cleanup(); err != nil {
		pm.logger.WithError(err).Warnf("Plugin %s cleanup failed", name)
	}

	// Remove from hooks
	for hookType, pluginList := range pm.hooks {
		for i, p := range pluginList {
			if p == plugin {
				pm.hooks[hookType] = append(pluginList[:i], pluginList[i+1:]...)
				break
			}
		}
	}

	// Remove from plugins map
	delete(pm.plugins, name)

	pm.logger.WithField("plugin", name).Info("Plugin unloaded")
	return nil
}

// ExecuteHook executes all plugins registered for a specific hook
func (pm *PluginManager) ExecuteHook(hookType HookType, ctx context.Context, pluginCtx *PluginContext) error {
	pm.mu.RLock()
	plugins := pm.hooks[hookType]
	pm.mu.RUnlock()

	for _, plugin := range plugins {
		var err error
		switch hookType {
		case PreRequestHook:
			err = plugin.PreRequest(ctx, pluginCtx)
		case PostRequestHook:
			err = plugin.PostRequest(ctx, pluginCtx)
		case PreResponseHook:
			err = plugin.PreResponse(ctx, pluginCtx)
		case PostResponseHook:
			err = plugin.PostResponse(ctx, pluginCtx)
		}

		if err != nil {
			pm.logger.WithError(err).WithFields(logrus.Fields{
				"plugin": plugin.Name(),
				"hook":   hookType,
			}).Error("Plugin hook execution failed")
			return fmt.Errorf("plugin %s hook %s failed: %w", plugin.Name(), hookType, err)
		}
	}

	return nil
}

// ListPlugins returns a list of loaded plugin names
func (pm *PluginManager) ListPlugins() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}
	return names
}

// GetPlugin returns a plugin by name
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// PluginMiddleware creates an Echo middleware that executes plugin hooks
func (pm *PluginManager) PluginMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Create plugin context
			pluginCtx := &PluginContext{
				RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				Path:      c.Request().URL.Path,
				Method:    c.Request().Method,
				Headers:   c.Request().Header,
				Metadata:  make(map[string]interface{}),
			}

			// Execute pre-request hooks
			if err := pm.ExecuteHook(PreRequestHook, c.Request().Context(), pluginCtx); err != nil {
				return err
			}

			// Execute the handler
			err := next(c)

			// Execute post-response hooks
			if hookErr := pm.ExecuteHook(PostResponseHook, c.Request().Context(), pluginCtx); hookErr != nil {
				pm.logger.WithError(hookErr).Error("Post-response hook failed")
			}

			return err
		}
	}
}

// LoadMiddleware loads a middleware plugin from a binary file
func (pm *PluginManager) LoadMiddleware(name, binaryPath string, config map[string]interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if already loaded
	if _, exists := pm.middlewares[name]; exists {
		return fmt.Errorf("middleware %s is already loaded", name)
	}

	// Load the plugin binary
	p, err := plugin.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to open middleware plugin: %w", err)
	}

	// Look up the Middleware symbol
	symMiddleware, err := p.Lookup("Middleware")
	if err != nil {
		return fmt.Errorf("middleware plugin does not export 'Middleware' symbol: %w", err)
	}

	// Try to cast to Middleware interface
	middleware, ok := symMiddleware.(Middleware)
	if !ok {
		// Try with pointer
		if mwPtr, ok := symMiddleware.(*Middleware); ok && mwPtr != nil {
			middleware = *mwPtr
		} else {
			return fmt.Errorf("middleware symbol is not of type Middleware")
		}
	}

	// Initialize middleware
	if err := middleware.Initialize(config); err != nil {
		return fmt.Errorf("failed to initialize middleware: %w", err)
	}

	// Store middleware
	pm.middlewares[name] = middleware

	pm.logger.Infof("Middleware %s (v%s) loaded successfully", middleware.Name(), middleware.Version())
	return nil
}

// UnloadMiddleware unloads a middleware plugin
func (pm *PluginManager) UnloadMiddleware(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	middleware, exists := pm.middlewares[name]
	if !exists {
		return fmt.Errorf("middleware %s is not loaded", name)
	}

	// Cleanup
	if err := middleware.Cleanup(); err != nil {
		pm.logger.WithError(err).Warnf("Error during middleware %s cleanup", name)
	}

	delete(pm.middlewares, name)
	pm.logger.Infof("Middleware %s unloaded", name)
	return nil
}

// GetMiddleware returns a middleware by name
func (pm *PluginManager) GetMiddleware(name string) (Middleware, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	middleware, exists := pm.middlewares[name]
	return middleware, exists
}

// ListMiddlewares returns a list of loaded middleware names
func (pm *PluginManager) ListMiddlewares() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.middlewares))
	for name := range pm.middlewares {
		names = append(names, name)
	}
	return names
}

// GetMiddlewareHandler returns an Echo middleware handler for a loaded middleware plugin
func (pm *PluginManager) GetMiddlewareHandler(name string) (echo.MiddlewareFunc, error) {
	middleware, exists := pm.GetMiddleware(name)
	if !exists {
		return nil, fmt.Errorf("middleware %s not found", name)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return middleware.Handle(next)
	}, nil
}

// ApplyMiddlewares applies all loaded middlewares to an Echo group or router
func (pm *PluginManager) ApplyMiddlewares(e *echo.Echo, routes []string) error {
	pm.mu.RLock()
	middlewareNames := make([]string, 0, len(pm.middlewares))
	for name := range pm.middlewares {
		middlewareNames = append(middlewareNames, name)
	}
	pm.mu.RUnlock()

	for _, name := range middlewareNames {
		middleware, exists := pm.GetMiddleware(name)
		if !exists {
			continue
		}

		// If specific routes are provided, apply only to those routes
		if len(routes) > 0 {
			for _, route := range routes {
				group := e.Group(route)
				group.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
					return middleware.Handle(next)
				})
			}
		} else {
			// Apply globally
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return middleware.Handle(next)
			})
		}

		pm.logger.Infof("Applied middleware %s to routes: %v", name, routes)
	}

	return nil
}

// GetTester returns the middleware tester
func (pm *PluginManager) GetTester() *MiddlewareTester {
	return pm.tester
}

// SetRollback sets the rollback manager (called after repository is available)
func (pm *PluginManager) SetRollback(rollback *MiddlewareRollback) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.rollback = rollback
}

// GetRollback returns the rollback manager
func (pm *PluginManager) GetRollback() *MiddlewareRollback {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.rollback
}
