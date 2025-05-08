package gateway

import (
	"context"
	"fmt"
	"net/http"

	"odin/pkg/admin"
	"odin/pkg/aggregator"
	"odin/pkg/auth"
	"odin/pkg/cache"
	"odin/pkg/config"
	"odin/pkg/health"
	"odin/pkg/logging"
	"odin/pkg/middleware"
	"odin/pkg/monitoring"
	"odin/pkg/proxy"
	"odin/pkg/routing"
	"odin/pkg/service"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

type Gateway struct {
	server          *echo.Echo
	config          *config.Config
	logger          *logrus.Logger
	adminHandler    *admin.AdminHandler
	serviceRegistry *service.Registry
	router          *routing.Router
}

func New(cfg *config.Config, configPath string, logger *logrus.Logger) (*Gateway, error) {
	e := echo.New()

	agg := aggregator.New(logger, cfg.Services)

	agg.RegisterRoutes(e)

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("aggregator", agg)
			return next(c)
		}
	})

	registry := service.NewRegistry(logger)

	for _, svcConfig := range cfg.Services {
		svc := &service.Config{
			Name:           svcConfig.Name,
			BasePath:       svcConfig.BasePath,
			Targets:        svcConfig.Targets,
			StripBasePath:  svcConfig.StripBasePath,
			Timeout:        svcConfig.Timeout,
			RetryCount:     svcConfig.RetryCount,
			RetryDelay:     svcConfig.RetryDelay,
			Authentication: svcConfig.Authentication,
			LoadBalancing:  svcConfig.LoadBalancing,
			Headers:        svcConfig.Headers,
		}

		svc.Transform.Request = make([]service.TransformRule, len(svcConfig.Transform.Request))
		for i, rule := range svcConfig.Transform.Request {
			svc.Transform.Request[i] = service.TransformRule{
				From:    rule.From,
				To:      rule.To,
				Default: rule.Default,
			}
		}

		svc.Transform.Response = make([]service.TransformRule, len(svcConfig.Transform.Response))
		for i, rule := range svcConfig.Transform.Response {
			svc.Transform.Response[i] = service.TransformRule{
				From:    rule.From,
				To:      rule.To,
				Default: rule.Default,
			}
		}

		if svcConfig.Aggregation != nil {
			aggregation := &service.AggregationConfig{
				Dependencies: make([]service.DependencyConfig, len(svcConfig.Aggregation.Dependencies)),
			}

			for i, dep := range svcConfig.Aggregation.Dependencies {
				dependency := service.DependencyConfig{
					Service:          dep.Service,
					Path:             dep.Path,
					ParameterMapping: make([]service.MappingConfig, len(dep.ParameterMapping)),
					ResultMapping:    make([]service.MappingConfig, len(dep.ResultMapping)),
				}

				for j, mapping := range dep.ParameterMapping {
					dependency.ParameterMapping[j] = service.MappingConfig{
						From: mapping.From,
						To:   mapping.To,
					}
				}

				for j, mapping := range dep.ResultMapping {
					dependency.ResultMapping[j] = service.MappingConfig{
						From: mapping.From,
						To:   mapping.To,
					}
				}

				aggregation.Dependencies[i] = dependency
			}

			svc.Aggregation = aggregation
		}

		if err := registry.Register(svc); err != nil {
			logger.WithError(err).Warnf("Failed to register service %s", svc.Name)
		}
	}

	router := routing.NewRouter(e, registry, logger)

	adminHandler := admin.New(cfg, configPath, logger)

	gateway := &Gateway{
		server:          e,
		config:          cfg,
		logger:          logger,
		adminHandler:    adminHandler,
		serviceRegistry: registry,
		router:          router,
	}

	logging.ConfigureLogger(logger, cfg.Logging.Level, cfg.Logging.JSON)

	health.Register(e, logger)

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("config", cfg)
			return next(c)
		}
	})

	e.Use(echomw.LoggerWithConfig(echomw.LoggerConfig{
		Format: "${time_rfc3339} | ${remote_ip} | ${method} ${uri} | ${status} | ${latency_human}\n",
	}))

	if cfg.Monitoring.Enabled {
		monitoring.Register(e, cfg.Monitoring.Path)
	}

	if cfg.RateLimit.Enabled {
		// Initialize rate limiter
		logger.Info("Rate limiting enabled")
	}

	if cfg.Server.Compression {
		e.Use(echomw.Gzip())
	}

	authMiddleware := auth.NewJWTMiddleware(cfg.Auth)
	router.SetAuthMiddleware(authMiddleware)

	var cacheStore cache.Store
	if cfg.Cache.Enabled {
		var err error
		cacheStore, err = cache.NewStore(cfg.Cache)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cache: %w", err)
		}
		e.Use(middleware.CacheMiddleware(cacheStore, logger))
		router.SetCacheStore(cacheStore)
	}

	if err := router.RegisterRoutes(); err != nil {
		return nil, fmt.Errorf("failed to register routes: %w", err)
	}

	adminHandler.Register(e)
	logger.Info("Admin interface registered at /admin")

	for _, svc := range cfg.Services {
		logger.WithFields(logrus.Fields{
			"name":     svc.Name,
			"basePath": svc.BasePath,
			"targets":  svc.Targets,
		}).Info("Registering service route")

		routeGroup := e.Group(svc.BasePath)

		if svc.Authentication {
			routeGroup.Use(authMiddleware)
		}

		proxyHandler, err := proxy.NewHandler(svc, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy for service %s: %w", svc.Name, err)
		}

		routeGroup.Any("", proxyHandler)
		routeGroup.Any("/*", proxyHandler)
	}

	return gateway, nil
}

func (g *Gateway) Start() error {
	addr := fmt.Sprintf(":%d", g.config.Server.Port)

	s := &http.Server{
		Addr:         addr,
		ReadTimeout:  g.config.Server.ReadTimeout,
		WriteTimeout: g.config.Server.WriteTimeout,
	}

	g.logger.Infof("Server starting on %s", addr)
	return g.server.StartServer(s)
}

func (g *Gateway) Shutdown(ctx context.Context) error {
	return g.server.Shutdown(ctx)
}
