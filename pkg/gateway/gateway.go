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
	"odin/pkg/graphql"
	"odin/pkg/grpc"
	"odin/pkg/health"
	"odin/pkg/logging"
	"odin/pkg/middleware"
	"odin/pkg/monitoring"
	"odin/pkg/plugins"
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
	pluginManager   *plugins.PluginManager
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
			Protocol:       svcConfig.Protocol,
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

	// Initialize plugin manager
	pluginManager := plugins.NewPluginManager(logger)

	// Load plugins if enabled
	if cfg.Plugins.Enabled {
		for _, pluginCfg := range cfg.Plugins.Plugins {
			if pluginCfg.Enabled {
				if err := pluginManager.LoadPlugin(pluginCfg.Name, pluginCfg.Path, pluginCfg.Config, pluginCfg.Hooks); err != nil {
					logger.WithError(err).Warnf("Failed to load plugin %s", pluginCfg.Name)
				}
			}
		}
	}

	gateway := &Gateway{
		server:          e,
		config:          cfg,
		logger:          logger,
		adminHandler:    adminHandler,
		serviceRegistry: registry,
		router:          router,
		pluginManager:   pluginManager,
	}

	// Setup protocol-specific proxies
	for _, svcConfig := range cfg.Services {
		switch svcConfig.Protocol {
		case "graphql":
			if svcConfig.GraphQL != nil && len(svcConfig.Targets) > 0 {
				graphqlConfig := &graphql.ProxyConfig{
					Endpoint:            svcConfig.Targets[0],
					MaxQueryDepth:       svcConfig.GraphQL.MaxQueryDepth,
					MaxQueryComplexity:  svcConfig.GraphQL.MaxQueryComplexity,
					EnableIntrospection: svcConfig.GraphQL.EnableIntrospection,
					Timeout:             svcConfig.Timeout,
					EnableQueryCaching:  svcConfig.GraphQL.EnableQueryCaching,
					CacheTTL:            svcConfig.GraphQL.CacheTTL,
				}
				graphqlProxy := graphql.NewProxy(graphqlConfig, logger)
				graphqlProxy.RegisterRoutes(e, svcConfig.BasePath)
				logger.WithField("service", svcConfig.Name).Info("GraphQL proxy registered")
			}
		case "grpc":
			if svcConfig.GRPC != nil && len(svcConfig.Targets) > 0 {
				grpcConfig := &grpc.ProxyConfig{
					Target:           svcConfig.Targets[0],
					MaxMessageSize:   svcConfig.GRPC.MaxMessageSize,
					Timeout:          svcConfig.Timeout,
					EnableTLS:        svcConfig.GRPC.EnableTLS,
					TLSCertFile:      svcConfig.GRPC.TLSCertFile,
					TLSKeyFile:       svcConfig.GRPC.TLSKeyFile,
					EnableReflection: svcConfig.GRPC.EnableReflection,
				}
				grpcProxy, err := grpc.NewProxy(grpcConfig, logger)
				if err != nil {
					logger.WithError(err).Warnf("Failed to create gRPC proxy for service %s", svcConfig.Name)
				} else {
					grpcProxy.RegisterRoutes(e, svcConfig.BasePath)
					logger.WithField("service", svcConfig.Name).Info("gRPC proxy registered")
				}
			}
		}
	}

	logging.ConfigureLoggerLegacy(logger, cfg.Logging.Level, cfg.Logging.JSON)

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

	// Add plugin middleware if plugins are enabled
	if cfg.Plugins.Enabled {
		e.Use(pluginManager.PluginMiddleware())
		logger.Info("Plugin middleware enabled")
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
