package routing

import (
	"fmt"
	"odin/pkg/cache"
	"odin/pkg/service"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Router struct {
	echo           *echo.Echo
	registry       *service.Registry
	logger         *logrus.Logger
	cacheStore     cache.Store
	authMiddleware echo.MiddlewareFunc
}

func NewRouter(e *echo.Echo, registry *service.Registry, logger *logrus.Logger) *Router {
	return &Router{
		echo:     e,
		registry: registry,
		logger:   logger,
	}
}

func (r *Router) SetCacheStore(store cache.Store) {
	r.cacheStore = store
}

func (r *Router) SetAuthMiddleware(middleware echo.MiddlewareFunc) {
	r.authMiddleware = middleware
}

func (r *Router) RegisterRoutes() error {
	services := r.registry.GetAllServices()

	for _, svc := range services {
		r.logger.WithFields(logrus.Fields{
			"name":     svc.Name,
			"basePath": svc.BasePath,
		}).Info("Registering service route")

		routeGroup := r.echo.Group(svc.BasePath)

		if svc.Authentication && r.authMiddleware != nil {
			routeGroup.Use(r.authMiddleware)
		}

		handler, err := NewServiceHandler(svc, r.logger, r.cacheStore)
		if err != nil {
			return fmt.Errorf("failed to create proxy for service %s: %w", svc.Name, err)
		}

		routeGroup.Any("", handler.Handle)
		routeGroup.Any("/*", handler.Handle)
	}

	return nil
}

func (r *Router) RegisterHealthRoutes() {
	r.echo.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status": "UP",
		})
	})

	r.echo.GET("/routes", func(c echo.Context) error {
		var routes []map[string]string
		for _, route := range r.echo.Routes() {
			routes = append(routes, map[string]string{
				"method": route.Method,
				"path":   route.Path,
			})
		}
		return c.JSON(200, routes)
	})
}
