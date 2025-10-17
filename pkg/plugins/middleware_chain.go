package plugins

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// RegisterMiddleware adds a middleware to the chain with priority and route targeting
func (pm *PluginManager) RegisterMiddleware(name string, middleware Middleware, priority int, routes []string, phase string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Store the middleware
	pm.middlewares[name] = middleware

	// Add to middleware chain
	entry := MiddlewareEntry{
		Name:       name,
		Middleware: middleware,
		Priority:   priority,
		Routes:     routes,
		Phase:      phase,
	}

	pm.middlewareChain.mu.Lock()
	pm.middlewareChain.Middlewares = append(pm.middlewareChain.Middlewares, entry)

	// Sort by priority
	sort.Slice(pm.middlewareChain.Middlewares, func(i, j int) bool {
		return pm.middlewareChain.Middlewares[i].Priority < pm.middlewareChain.Middlewares[j].Priority
	})
	pm.middlewareChain.mu.Unlock()

	pm.logger.WithFields(logrus.Fields{
		"middleware": name,
		"priority":   priority,
		"routes":     routes,
		"phase":      phase,
	}).Info("Middleware registered successfully")

	return nil
}

// UnregisterMiddleware removes a middleware from the chain
func (pm *PluginManager) UnregisterMiddleware(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Remove from middlewares map
	middleware, exists := pm.middlewares[name]
	if !exists {
		return nil // Already unregistered
	}

	// Cleanup the middleware
	if err := middleware.Cleanup(); err != nil {
		pm.logger.WithError(err).WithField("middleware", name).Warn("Middleware cleanup failed")
	}

	delete(pm.middlewares, name)

	// Remove from middleware chain
	pm.middlewareChain.mu.Lock()
	filtered := make([]MiddlewareEntry, 0, len(pm.middlewareChain.Middlewares))
	for _, entry := range pm.middlewareChain.Middlewares {
		if entry.Name != name {
			filtered = append(filtered, entry)
		}
	}
	pm.middlewareChain.Middlewares = filtered
	pm.middlewareChain.mu.Unlock()

	pm.logger.WithField("middleware", name).Info("Middleware unregistered successfully")

	return nil
}

// LoadMiddlewareWithChain loads a middleware plugin and adds it to the middleware chain
func (pm *PluginManager) LoadMiddlewareWithChain(name, path string, config map[string]interface{}, priority int, routes []string, phase string) error {
	// First load the middleware using existing LoadMiddleware method
	if err := pm.LoadMiddleware(name, path, config); err != nil {
		return err
	}

	// Then register it in the middleware chain
	middleware, exists := pm.GetMiddleware(name)
	if !exists {
		return fmt.Errorf("middleware %s was loaded but not found", name)
	}

	return pm.RegisterMiddleware(name, middleware, priority, routes, phase)
}

// GetMiddlewareChain returns the current middleware chain
func (pm *PluginManager) GetMiddlewareChain() []MiddlewareEntry {
	pm.middlewareChain.mu.RLock()
	defer pm.middlewareChain.mu.RUnlock()

	// Return a copy to prevent external modification
	chain := make([]MiddlewareEntry, len(pm.middlewareChain.Middlewares))
	copy(chain, pm.middlewareChain.Middlewares)
	return chain
}

// UpdateMiddlewarePriority changes the priority of a middleware and resorts the chain
func (pm *PluginManager) UpdateMiddlewarePriority(name string, newPriority int) error {
	pm.middlewareChain.mu.Lock()
	defer pm.middlewareChain.mu.Unlock()

	for i := range pm.middlewareChain.Middlewares {
		if pm.middlewareChain.Middlewares[i].Name == name {
			pm.middlewareChain.Middlewares[i].Priority = newPriority

			// Resort the chain
			sort.Slice(pm.middlewareChain.Middlewares, func(i, j int) bool {
				return pm.middlewareChain.Middlewares[i].Priority < pm.middlewareChain.Middlewares[j].Priority
			})

			pm.logger.WithFields(logrus.Fields{
				"middleware": name,
				"priority":   newPriority,
			}).Info("Middleware priority updated")

			return nil
		}
	}

	return fmt.Errorf("middleware %s not found in chain", name)
}

// UpdateMiddlewareRoutes changes the routes a middleware applies to
func (pm *PluginManager) UpdateMiddlewareRoutes(name string, routes []string) error {
	pm.middlewareChain.mu.Lock()
	defer pm.middlewareChain.mu.Unlock()

	for i := range pm.middlewareChain.Middlewares {
		if pm.middlewareChain.Middlewares[i].Name == name {
			pm.middlewareChain.Middlewares[i].Routes = routes

			pm.logger.WithFields(logrus.Fields{
				"middleware": name,
				"routes":     routes,
			}).Info("Middleware routes updated")

			return nil
		}
	}

	return fmt.Errorf("middleware %s not found in chain", name)
}

// DynamicMiddleware creates an Echo middleware that applies plugins based on route matching
func (pm *PluginManager) DynamicMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestPath := c.Request().URL.Path

			// Get all middlewares that apply to this route
			pm.middlewareChain.mu.RLock()
			applicableMiddlewares := make([]MiddlewareEntry, 0)
			for _, entry := range pm.middlewareChain.Middlewares {
				if pm.matchesRoute(requestPath, entry.Routes) {
					applicableMiddlewares = append(applicableMiddlewares, entry)
				}
			}
			pm.middlewareChain.mu.RUnlock()

			// Build the middleware chain
			handler := next
			for i := len(applicableMiddlewares) - 1; i >= 0; i-- {
				entry := applicableMiddlewares[i]
				handler = entry.Middleware.Handle(handler)
			}

			return handler(c)
		}
	}
}

// matchesRoute checks if a request path matches any of the route patterns
func (pm *PluginManager) matchesRoute(requestPath string, routePatterns []string) bool {
	if len(routePatterns) == 0 {
		return false
	}

	for _, pattern := range routePatterns {
		// Handle wildcard "*" - matches all routes
		if pattern == "*" {
			return true
		}

		// Handle exact match
		if pattern == requestPath {
			return true
		}

		// Handle prefix match with wildcard (e.g., "/api/*")
		if strings.HasSuffix(pattern, "/*") {
			prefix := strings.TrimSuffix(pattern, "/*")
			if strings.HasPrefix(requestPath, prefix) {
				return true
			}
		}

		// Handle glob-style match using filepath.Match
		matched, err := path.Match(pattern, requestPath)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// ReloadMiddlewareChain reloads all middleware from repository
func (pm *PluginManager) ReloadMiddlewareChain(repo *PluginRepository) error {
	// This would be called to reload middleware from MongoDB
	// Implementation depends on repository integration
	pm.logger.Info("Reloading middleware chain from repository")

	// Clear existing middleware chain
	pm.middlewareChain.mu.Lock()
	pm.middlewareChain.Middlewares = []MiddlewareEntry{}
	pm.middlewareChain.mu.Unlock()

	// Reload from repository would happen here
	// For now, this is a placeholder for future integration

	return nil
}
