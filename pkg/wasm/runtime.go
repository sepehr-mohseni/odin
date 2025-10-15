package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// wasmRuntime implements the Runtime interface using wazero
type wasmRuntime struct {
	config  *Config
	logger  *logrus.Logger
	runtime wazero.Runtime
	plugins map[string]*wasmPlugin
	modules map[string]wazero.CompiledModule // Cached compiled modules
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// wasmPlugin implements the Plugin interface
type wasmPlugin struct {
	config    *PluginConfig
	runtime   *wasmRuntime
	module    api.Module
	exports   map[string]api.Function
	instances int
	mu        sync.Mutex
}

// NewRuntime creates a new WASM runtime
func NewRuntime(config *Config, logger *logrus.Logger) (Runtime, error) {
	if !config.Enabled {
		logger.Info("WASM runtime is disabled")
		return &noopRuntime{}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create wazero runtime with configuration
	runtimeConfig := wazero.NewRuntimeConfig().
		WithMemoryLimitPages(uint32(config.MaxMemoryPages))

	if config.CacheEnabled {
		// Enable compilation cache
		cache := wazero.NewCompilationCache()
		runtimeConfig = runtimeConfig.WithCompilationCache(cache)
	}

	r := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)

	// Instantiate WASI for filesystem and environment access
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, r); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}

	runtime := &wasmRuntime{
		config:  config,
		logger:  logger,
		runtime: r,
		plugins: make(map[string]*wasmPlugin),
		modules: make(map[string]wazero.CompiledModule),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Register host functions
	if err := runtime.registerHostFunctions(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register host functions: %w", err)
	}

	// Load configured plugins
	if err := runtime.loadConfiguredPlugins(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	logger.WithField("plugins", len(runtime.plugins)).Info("WASM runtime initialized")
	return runtime, nil
}

// registerHostFunctions registers functions that WASM modules can call
func (r *wasmRuntime) registerHostFunctions() error {
	// Create host module with functions
	_, err := r.runtime.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, offset, size uint32) {
			// log function: reads string from memory and logs it
			data, ok := m.Memory().Read(offset, size)
			if !ok {
				r.logger.Error("Failed to read log message from WASM memory")
				return
			}
			r.logger.Info(string(data))
		}).
		Export("log").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module) int64 {
			// get_time function: returns current unix timestamp
			return time.Now().Unix()
		}).
		Export("get_time").
		Instantiate(r.ctx)

	return err
}

// loadConfiguredPlugins loads all plugins from configuration
func (r *wasmRuntime) loadConfiguredPlugins() error {
	for _, pluginConfig := range r.config.Plugins {
		if !pluginConfig.Enabled {
			r.logger.WithField("plugin", pluginConfig.Name).Debug("Plugin is disabled, skipping")
			continue
		}

		// Resolve plugin path
		pluginPath := pluginConfig.Path
		if !filepath.IsAbs(pluginPath) {
			pluginPath = filepath.Join(r.config.PluginDir, pluginPath)
		}

		if _, err := r.LoadPlugin(&pluginConfig); err != nil {
			r.logger.WithError(err).WithField("plugin", pluginConfig.Name).Error("Failed to load plugin")
			// Continue loading other plugins
			continue
		}

		r.logger.WithFields(logrus.Fields{
			"plugin": pluginConfig.Name,
			"type":   pluginConfig.Type,
			"path":   pluginPath,
		}).Info("Plugin loaded successfully")
	}

	return nil
}

// LoadPlugin loads a WASM plugin from file
func (r *wasmRuntime) LoadPlugin(config *PluginConfig) (Plugin, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if plugin already loaded
	if existing, ok := r.plugins[config.Name]; ok {
		return existing, nil
	}

	// Resolve plugin path
	pluginPath := config.Path
	if !filepath.IsAbs(pluginPath) {
		pluginPath = filepath.Join(r.config.PluginDir, pluginPath)
	}

	// Read WASM binary
	wasmBytes, err := os.ReadFile(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin file: %w", err)
	}

	// Compile or get from cache
	var compiled wazero.CompiledModule
	if r.config.CacheEnabled {
		if cached, ok := r.modules[config.Name]; ok {
			compiled = cached
		}
	}

	if compiled == nil {
		compiled, err = r.runtime.CompileModule(r.ctx, wasmBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to compile plugin: %w", err)
		}

		if r.config.CacheEnabled {
			r.modules[config.Name] = compiled
		}
	}

	// Instantiate module
	moduleConfig := wazero.NewModuleConfig().WithName(config.Name)
	module, err := r.runtime.InstantiateModule(r.ctx, compiled, moduleConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate plugin: %w", err)
	}

	// Get exported functions
	exports := make(map[string]api.Function)
	for _, name := range []string{"init", "execute", "cleanup"} {
		if fn := module.ExportedFunction(name); fn != nil {
			exports[name] = fn
		}
	}

	// Call init function if exists
	if initFn, ok := exports["init"]; ok {
		if _, err := initFn.Call(r.ctx); err != nil {
			return nil, fmt.Errorf("plugin init failed: %w", err)
		}
	}

	plugin := &wasmPlugin{
		config:  config,
		runtime: r,
		module:  module,
		exports: exports,
	}

	r.plugins[config.Name] = plugin
	return plugin, nil
}

// GetPlugin retrieves a loaded plugin by name
func (r *wasmRuntime) GetPlugin(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return plugin, nil
}

// ListPlugins returns all loaded plugins
func (r *wasmRuntime) ListPlugins() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// UnloadPlugin unloads a plugin
func (r *wasmRuntime) UnloadPlugin(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, ok := r.plugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	if err := plugin.Close(); err != nil {
		return fmt.Errorf("failed to close plugin: %w", err)
	}

	delete(r.plugins, name)
	return nil
}

// Close shuts down the runtime
func (r *wasmRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Close all plugins
	for name, plugin := range r.plugins {
		if err := plugin.Close(); err != nil {
			r.logger.WithError(err).WithField("plugin", name).Error("Failed to close plugin")
		}
	}

	r.cancel()
	return r.runtime.Close(r.ctx)
}

// Plugin implementation

// Name returns the plugin name
func (p *wasmPlugin) Name() string {
	return p.config.Name
}

// Type returns the plugin type
func (p *wasmPlugin) Type() PluginType {
	return p.config.Type
}

// Execute runs the plugin with the given context and input
func (p *wasmPlugin) Execute(ctx context.Context, pluginCtx *PluginContext, input interface{}) (*PluginResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check instance limit
	if p.runtime.config.MaxInstances > 0 && p.instances >= p.runtime.config.MaxInstances {
		return nil, fmt.Errorf("max instances reached for plugin %s", p.config.Name)
	}

	p.instances++
	defer func() { p.instances-- }()

	// Get execute function
	executeFn, ok := p.exports["execute"]
	if !ok {
		return nil, fmt.Errorf("plugin does not export 'execute' function")
	}

	// Prepare input data
	inputData := map[string]interface{}{
		"context": pluginCtx,
		"input":   input,
		"config":  p.config.Config,
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	// Get memory and write input
	mem := p.module.Memory()
	inputSize := uint32(len(inputJSON))

	// Get current memory size and allocate at the end
	memSize := mem.Size()
	inputPtr := memSize

	// Grow memory if needed
	pagesNeeded := (inputSize + 65535) / 65536 // Calculate pages needed (64KB per page)
	if pagesNeeded > 0 {
		if _, ok := mem.Grow(pagesNeeded); !ok {
			return nil, fmt.Errorf("failed to grow memory for input")
		}
	}

	// Write input to memory
	if !mem.Write(inputPtr, inputJSON) {
		return nil, fmt.Errorf("failed to write input to memory")
	}

	// Execute with timeout
	execCtx := ctx
	if p.config.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, p.config.Timeout)
		defer cancel()
	}

	// Call execute function (passes input pointer and size, returns output pointer and size)
	results, err := executeFn.Call(execCtx, uint64(inputPtr), uint64(inputSize))
	if err != nil {
		return nil, fmt.Errorf("plugin execution failed: %w", err)
	}

	if len(results) < 2 {
		return nil, fmt.Errorf("plugin returned invalid results")
	}

	// Read output from memory
	outputPtr := uint32(results[0])
	outputSize := uint32(results[1])

	outputJSON, ok := mem.Read(outputPtr, outputSize)
	if !ok {
		return nil, fmt.Errorf("failed to read output from memory")
	}

	// Parse result
	var result PluginResult
	if err := json.Unmarshal(outputJSON, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return &result, nil
}

// Close cleans up plugin resources
func (p *wasmPlugin) Close() error {
	// Call cleanup function if exists
	if cleanupFn, ok := p.exports["cleanup"]; ok {
		if _, err := cleanupFn.Call(p.runtime.ctx); err != nil {
			return fmt.Errorf("plugin cleanup failed: %w", err)
		}
	}

	return p.module.Close(p.runtime.ctx)
}

// noopRuntime is a no-op implementation when WASM is disabled
type noopRuntime struct{}

func (n *noopRuntime) LoadPlugin(config *PluginConfig) (Plugin, error) {
	return nil, fmt.Errorf("WASM runtime is disabled")
}

func (n *noopRuntime) GetPlugin(name string) (Plugin, error) {
	return nil, fmt.Errorf("WASM runtime is disabled")
}

func (n *noopRuntime) ListPlugins() []Plugin {
	return nil
}

func (n *noopRuntime) UnloadPlugin(name string) error {
	return fmt.Errorf("WASM runtime is disabled")
}

func (n *noopRuntime) Close() error {
	return nil
}
