package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"odin/pkg/plugins"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// uploadPlugin handles plugin binary file uploads
func (h *PluginHandler) uploadPlugin(c echo.Context) error {
	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Failed to get file: %v", err),
		})
	}

	// Get metadata
	metadataStr := c.FormValue("metadata")
	if metadataStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing plugin metadata",
		})
	}

	var plugin plugins.PluginRecord
	if err := json.Unmarshal([]byte(metadataStr), &plugin); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid metadata JSON: %v", err),
		})
	}

	// Validate file extension
	if !strings.HasSuffix(file.Filename, ".so") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Only .so files are allowed",
		})
	}

	// Create plugins directory if it doesn't exist
	pluginsDir := "/var/odin/plugins"
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create plugins directory: %v", err),
		})
	}

	// Save file to plugins directory
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to open uploaded file: %v", err),
		})
	}
	defer src.Close()

	binaryPath := filepath.Join(pluginsDir, fmt.Sprintf("%s.so", plugin.Name))
	dst, err := os.Create(binaryPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create destination file: %v", err),
		})
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to save file: %v", err),
		})
	}

	// Update plugin record with binary path
	plugin.BinaryPath = binaryPath
	plugin.CreatedAt = time.Now()
	plugin.UpdatedAt = time.Now()

	// Save to database
	if err := h.repo.SavePlugin(context.Background(), &plugin); err != nil {
		// Clean up uploaded file
		os.Remove(binaryPath)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to save plugin record: %v", err),
		})
	}

	// Auto-load if enabled
	if plugin.Enabled {
		if err := h.manager.LoadPlugin(plugin.Name, plugin.BinaryPath, plugin.Config, plugin.Hooks); err != nil {
			c.Logger().Errorf("Failed to load plugin %s: %v", plugin.Name, err)
			// Don't fail the request, just log
		}
	}

	return c.JSON(http.StatusCreated, plugin)
}

// buildPlugin builds a plugin from a template
func (h *PluginHandler) buildPlugin(c echo.Context) error {
	type BuildRequest struct {
		Name        string                 `json:"name"`
		Version     string                 `json:"version"`
		Author      string                 `json:"author"`
		Description string                 `json:"description"`
		Template    string                 `json:"template"`
		PluginType  string                 `json:"pluginType"`
		Hooks       []string               `json:"hooks"`
		AppliedTo   []string               `json:"appliedTo"`
		Enabled     bool                   `json:"enabled"`
		Config      map[string]interface{} `json:"config"`
	}

	var req BuildRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
	}

	// Validate required fields
	if req.Name == "" || req.Template == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Name and template are required",
		})
	}

	// Create build directory
	buildDir := filepath.Join("/tmp", "odin-plugin-build", req.Name)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create build directory: %v", err),
		})
	}
	defer os.RemoveAll(buildDir)

	// Generate plugin source code based on template
	sourceCode, err := generatePluginCode(req.Name, req.Template, req.PluginType, req.Hooks)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to generate plugin code: %v", err),
		})
	}

	// Write source file
	sourceFile := filepath.Join(buildDir, "plugin.go")
	if err := os.WriteFile(sourceFile, []byte(sourceCode), 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to write source file: %v", err),
		})
	}

	// Create go.mod file
	goModContent := fmt.Sprintf(`module %s

go 1.21

require (
	github.com/labstack/echo/v4 v4.11.4
	github.com/sirupsen/logrus v1.9.3
)
`, req.Name)

	if err := os.WriteFile(filepath.Join(buildDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to write go.mod: %v", err),
		})
	}

	// Build plugin
	pluginsDir := "/var/odin/plugins"
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create plugins directory: %v", err),
		})
	}

	outputPath := filepath.Join(pluginsDir, fmt.Sprintf("%s.so", req.Name))

	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", outputPath, sourceFile)
	cmd.Dir = buildDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":  fmt.Sprintf("Failed to build plugin: %v", err),
			"output": string(output),
		})
	}

	// Create plugin record
	plugin := plugins.PluginRecord{
		Name:        req.Name,
		Version:     req.Version,
		Author:      req.Author,
		Description: req.Description,
		BinaryPath:  outputPath,
		Hooks:       req.Hooks,
		AppliedTo:   req.AppliedTo,
		Enabled:     req.Enabled,
		Config:      req.Config,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save to database
	if err := h.repo.SavePlugin(context.Background(), &plugin); err != nil {
		os.Remove(outputPath)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to save plugin record: %v", err),
		})
	}

	// Auto-load if enabled
	if plugin.Enabled {
		if err := h.manager.LoadPlugin(plugin.Name, plugin.BinaryPath, plugin.Config, plugin.Hooks); err != nil {
			c.Logger().Errorf("Failed to load plugin %s: %v", plugin.Name, err)
		}
	}

	return c.JSON(http.StatusCreated, plugin)
}

// testPlugin tests a plugin without loading it permanently
func (h *PluginHandler) testPlugin(c echo.Context) error {
	name := c.Param("name")

	plugin, err := h.repo.GetPlugin(context.Background(), name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Plugin not found",
		})
	}

	// Create test context
	type TestRequest struct {
		Method  string              `json:"method"`
		Path    string              `json:"path"`
		Headers map[string][]string `json:"headers"`
		Body    string              `json:"body"`
	}

	var testReq TestRequest
	if err := c.Bind(&testReq); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid test request: %v", err),
		})
	}

	// Try to load plugin temporarily
	logger := logrus.New()
	tempManager := plugins.NewPluginManager(logger)
	if err := tempManager.LoadPlugin(plugin.Name, plugin.BinaryPath, plugin.Config, plugin.Hooks); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "Failed to load plugin for testing",
			"details": err.Error(),
		})
	}
	defer tempManager.UnloadPlugin(plugin.Name)

	// Create test plugin context
	pluginCtx := &plugins.PluginContext{
		RequestID:   "test-request",
		ServiceName: "test-service",
		Path:        testReq.Path,
		Method:      testReq.Method,
		Headers:     testReq.Headers,
		Body:        []byte(testReq.Body),
		Metadata:    make(map[string]interface{}),
	}

	// Test hooks
	results := make(map[string]interface{})

	loadedPlugin, _ := tempManager.GetPlugin(plugin.Name)

	// Test pre-request hook
	if contains(plugin.Hooks, "pre-request") {
		err := loadedPlugin.PreRequest(context.Background(), pluginCtx)
		results["pre-request"] = map[string]interface{}{
			"success": err == nil,
			"error":   formatError(err),
		}
	}

	// Test post-request hook
	if contains(plugin.Hooks, "post-request") {
		err := loadedPlugin.PostRequest(context.Background(), pluginCtx)
		results["post-request"] = map[string]interface{}{
			"success": err == nil,
			"error":   formatError(err),
		}
	}

	// Test pre-response hook
	if contains(plugin.Hooks, "pre-response") {
		err := loadedPlugin.PreResponse(context.Background(), pluginCtx)
		results["pre-response"] = map[string]interface{}{
			"success": err == nil,
			"error":   formatError(err),
		}
	}

	// Test post-response hook
	if contains(plugin.Hooks, "post-response") {
		err := loadedPlugin.PostResponse(context.Background(), pluginCtx)
		results["post-response"] = map[string]interface{}{
			"success": err == nil,
			"error":   formatError(err),
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"plugin":  plugin.Name,
		"results": results,
		"context": pluginCtx,
	})
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func formatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func generatePluginCode(name, template, pluginType string, hooks []string) (string, error) {
	// Plugin templates
	templates := map[string]string{
		"auth":      authTemplate,
		"ratelimit": rateLimitTemplate,
		"logging":   loggingTemplate,
		"transform": transformTemplate,
		"cache":     cacheTemplate,
		"custom":    customTemplate,
	}

	templateCode, ok := templates[template]
	if !ok {
		return "", fmt.Errorf("unknown template: %s", template)
	}

	// Replace placeholders
	code := strings.ReplaceAll(templateCode, "{{.Name}}", name)
	code = strings.ReplaceAll(code, "{{.PluginType}}", pluginType)

	return code, nil
}

// Plugin templates (simplified versions - users can customize)

const authTemplate = `package main

import (
	"context"
	"fmt"
	"strings"
)

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

type {{.Name}} struct {
	config map[string]interface{}
}

var Plugin {{.Name}}

func (p *{{.Name}}) Name() string {
	return "{{.Name}}"
}

func (p *{{.Name}}) Version() string {
	return "1.0.0"
}

func (p *{{.Name}}) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *{{.Name}}) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Check for authentication token
	authHeader := pluginCtx.Headers["Authorization"]
	if len(authHeader) == 0 {
		return fmt.Errorf("missing authorization header")
	}
	
	token := strings.TrimPrefix(authHeader[0], "Bearer ")
	if token == "" {
		return fmt.Errorf("invalid token format")
	}
	
	// Add user context (in real plugin, validate token here)
	pluginCtx.UserID = "user-from-token"
	pluginCtx.Metadata["authenticated"] = true
	
	return nil
}

func (p *{{.Name}}) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) Cleanup() error {
	return nil
}
`

const rateLimitTemplate = `package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

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

type {{.Name}} struct {
	config   map[string]interface{}
	counters map[string]*counter
	mu       sync.Mutex
}

type counter struct {
	count     int
	resetTime time.Time
}

var Plugin {{.Name}}

func (p *{{.Name}}) Name() string {
	return "{{.Name}}"
}

func (p *{{.Name}}) Version() string {
	return "1.0.0"
}

func (p *{{.Name}}) Initialize(config map[string]interface{}) error {
	p.config = config
	p.counters = make(map[string]*counter)
	return nil
}

func (p *{{.Name}}) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Get limit from config (default: 100 requests per minute)
	limit := 100
	if l, ok := p.config["limit"].(float64); ok {
		limit = int(l)
	}
	
	// Use IP address as key (in real plugin, get from headers)
	key := "default-key"
	
	now := time.Now()
	c, exists := p.counters[key]
	
	if !exists || now.After(c.resetTime) {
		p.counters[key] = &counter{
			count:     1,
			resetTime: now.Add(time.Minute),
		}
		return nil
	}
	
	if c.count >= limit {
		return fmt.Errorf("rate limit exceeded: %d requests per minute", limit)
	}
	
	c.count++
	return nil
}

func (p *{{.Name}}) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) Cleanup() error {
	return nil
}
`

const loggingTemplate = `package main

import (
	"context"
	"fmt"
	"time"
)

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

type {{.Name}} struct {
	config map[string]interface{}
}

var Plugin {{.Name}}

func (p *{{.Name}}) Name() string {
	return "{{.Name}}"
}

func (p *{{.Name}}) Version() string {
	return "1.0.0"
}

func (p *{{.Name}}) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *{{.Name}}) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	fmt.Printf("[%s] Request: %s %s\n", time.Now().Format(time.RFC3339), pluginCtx.Method, pluginCtx.Path)
	pluginCtx.Metadata["request_time"] = time.Now()
	return nil
}

func (p *{{.Name}}) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	if startTime, ok := pluginCtx.Metadata["request_time"].(time.Time); ok {
		duration := time.Since(startTime)
		fmt.Printf("[%s] Response: %s %s - Duration: %v\n", 
			time.Now().Format(time.RFC3339), pluginCtx.Method, pluginCtx.Path, duration)
	}
	return nil
}

func (p *{{.Name}}) Cleanup() error {
	return nil
}
`

const transformTemplate = `package main

import (
	"context"
	"strings"
)

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

type {{.Name}} struct {
	config map[string]interface{}
}

var Plugin {{.Name}}

func (p *{{.Name}}) Name() string {
	return "{{.Name}}"
}

func (p *{{.Name}}) Version() string {
	return "1.0.0"
}

func (p *{{.Name}}) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *{{.Name}}) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// Add custom header
	if pluginCtx.Headers == nil {
		pluginCtx.Headers = make(map[string][]string)
	}
	pluginCtx.Headers["X-Plugin-Processed"] = []string{"true"}
	
	// Transform body (example: convert to uppercase)
	if len(pluginCtx.Body) > 0 {
		pluginCtx.Body = []byte(strings.ToUpper(string(pluginCtx.Body)))
	}
	
	return nil
}

func (p *{{.Name}}) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	// Transform response before sending to client
	if len(pluginCtx.Body) > 0 {
		// Example: Add wrapper
		wrapped := []byte("{\"data\":" + string(pluginCtx.Body) + "}")
		pluginCtx.Body = wrapped
	}
	return nil
}

func (p *{{.Name}}) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) Cleanup() error {
	return nil
}
`

const cacheTemplate = `package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

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

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type {{.Name}} struct {
	config map[string]interface{}
	cache  map[string]*cacheEntry
	mu     sync.RWMutex
}

var Plugin {{.Name}}

func (p *{{.Name}}) Name() string {
	return "{{.Name}}"
}

func (p *{{.Name}}) Version() string {
	return "1.0.0"
}

func (p *{{.Name}}) Initialize(config map[string]interface{}) error {
	p.config = config
	p.cache = make(map[string]*cacheEntry)
	return nil
}

func (p *{{.Name}}) getCacheKey(pluginCtx *PluginContext) string {
	key := fmt.Sprintf("%s:%s:%s", pluginCtx.Method, pluginCtx.Path, string(pluginCtx.Body))
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash)
}

func (p *{{.Name}}) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	if pluginCtx.Method != "GET" {
		return nil // Only cache GET requests
	}
	
	key := p.getCacheKey(pluginCtx)
	
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if entry, ok := p.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		// Cache hit - skip backend request
		pluginCtx.Body = entry.data
		pluginCtx.Metadata["cache_hit"] = true
		return nil
	}
	
	pluginCtx.Metadata["cache_hit"] = false
	return nil
}

func (p *{{.Name}}) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	if pluginCtx.Method != "GET" || pluginCtx.Metadata["cache_hit"] == true {
		return nil
	}
	
	// Get TTL from config (default: 5 minutes)
	ttl := 5 * time.Minute
	if t, ok := p.config["ttl"].(float64); ok {
		ttl = time.Duration(t) * time.Second
	}
	
	key := p.getCacheKey(pluginCtx)
	
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.cache[key] = &cacheEntry{
		data:      pluginCtx.Body,
		expiresAt: time.Now().Add(ttl),
	}
	
	return nil
}

func (p *{{.Name}}) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	return nil
}

func (p *{{.Name}}) Cleanup() error {
	return nil
}
`

const customTemplate = `package main

import (
	"context"
)

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

type {{.Name}} struct {
	config map[string]interface{}
}

var Plugin {{.Name}}

func (p *{{.Name}}) Name() string {
	return "{{.Name}}"
}

func (p *{{.Name}}) Version() string {
	return "1.0.0"
}

func (p *{{.Name}}) Initialize(config map[string]interface{}) error {
	p.config = config
	// TODO: Initialize your plugin here
	return nil
}

func (p *{{.Name}}) PreRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// TODO: Implement pre-request logic
	return nil
}

func (p *{{.Name}}) PostRequest(ctx context.Context, pluginCtx *PluginContext) error {
	// TODO: Implement post-request logic
	return nil
}

func (p *{{.Name}}) PreResponse(ctx context.Context, pluginCtx *PluginContext) error {
	// TODO: Implement pre-response logic
	return nil
}

func (p *{{.Name}}) PostResponse(ctx context.Context, pluginCtx *PluginContext) error {
	// TODO: Implement post-response logic
	return nil
}

func (p *{{.Name}}) Cleanup() error {
	// TODO: Cleanup resources
	return nil
}
`
