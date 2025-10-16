package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	"odin/pkg/mongodb"
	"odin/pkg/monitoring"
	"odin/pkg/plugins"
	"odin/pkg/routing"
	"odin/pkg/service"
	"odin/pkg/servicemesh"
	"odin/pkg/tracing"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

type Gateway struct {
	server          *echo.Echo
	config          *config.Config
	logger          *logrus.Logger
	adminHandler    *admin.AdminHandler
	serviceRegistry *service.Registry
	router          *routing.Router
	pluginManager   *plugins.PluginManager
	tracingManager  *tracing.Manager
	healthChecker   *health.TargetChecker
	alertManager    *health.AlertManager
	meshManager     *servicemesh.Manager
	mongoRepo       mongodb.Repository
}

func New(cfg *config.Config, configPath string, logger *logrus.Logger) (*Gateway, error) {
	e := echo.New()

	// Initialize distributed tracing
	tracingConfig := tracing.Config{
		Enabled:        cfg.Tracing.Enabled,
		ServiceName:    cfg.Tracing.ServiceName,
		ServiceVersion: cfg.Tracing.ServiceVersion,
		Environment:    cfg.Tracing.Environment,
		Endpoint:       cfg.Tracing.Endpoint,
		SampleRate:     cfg.Tracing.SampleRate,
		Insecure:       cfg.Tracing.Insecure,
	}

	tracingManager, err := tracing.NewManager(tracingConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Add OpenTelemetry middleware for Echo
	if cfg.Tracing.Enabled {
		e.Use(otelecho.Middleware(cfg.Tracing.ServiceName))
	}

	// Add monitoring middleware
	e.Use(middleware.MonitoringMiddleware())

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

	// Initialize MongoDB repository
	mongoConfig := &mongodb.Config{
		Enabled:        cfg.MongoDB.Enabled,
		URI:            cfg.MongoDB.URI,
		Database:       cfg.MongoDB.Database,
		ConnectTimeout: cfg.MongoDB.ConnectTimeout,
		MaxPoolSize:    cfg.MongoDB.MaxPoolSize,
		MinPoolSize:    cfg.MongoDB.MinPoolSize,
		Auth: mongodb.AuthConfig{
			Username: cfg.MongoDB.Auth.Username,
			Password: cfg.MongoDB.Auth.Password,
			AuthDB:   cfg.MongoDB.Auth.AuthDB,
		},
		TLS: mongodb.TLSConfig{
			Enabled:  cfg.MongoDB.TLS.Enabled,
			CertFile: cfg.MongoDB.TLS.CertFile,
			KeyFile:  cfg.MongoDB.TLS.KeyFile,
			CAFile:   cfg.MongoDB.TLS.CAFile,
		},
	}

	mongoRepo, err := mongodb.NewRepository(mongoConfig, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to initialize MongoDB repository, plugin persistence will be disabled")
		mongoRepo = nil
	}

	// Initialize plugin manager
	pluginManager := plugins.NewPluginManager(logger)

	// Initialize plugin repository if MongoDB is available
	var pluginRepo *plugins.PluginRepository
	if mongoRepo != nil {
		// Get MongoDB database from repository to create plugin repository
		mongoDB := mongoRepo.GetDatabase()
		if mongoDB != nil {
			pluginRepo = plugins.NewPluginRepository(mongoDB)
			logger.Info("Plugin repository initialized with MongoDB")

			// Load enabled plugins from MongoDB
			ctx := context.Background()
			enabledPlugins, err := pluginRepo.GetEnabledPlugins(ctx)
			if err != nil {
				logger.WithError(err).Warn("Failed to load enabled plugins from MongoDB")
			} else {
				logger.Infof("Loading %d enabled plugins from database", len(enabledPlugins))
				for _, plugin := range enabledPlugins {
					if err := pluginManager.LoadPlugin(plugin.Name, plugin.BinaryPath, plugin.Config, plugin.Hooks); err != nil {
						logger.WithError(err).Warnf("Failed to load plugin %s from database", plugin.Name)
					} else {
						logger.Infof("Loaded plugin %s from database", plugin.Name)
					}
				}
			}
		}
	}

	// Set plugin handler in admin if both manager and repository are available
	if pluginManager != nil && pluginRepo != nil {
		adminHandler.SetPluginHandler(pluginManager, pluginRepo)
		logger.Info("Plugin admin panel initialized")
	}

	// Load plugins from config if enabled (for backward compatibility)
	if cfg.Plugins.Enabled {
		for _, pluginCfg := range cfg.Plugins.Plugins {
			if pluginCfg.Enabled {
				if err := pluginManager.LoadPlugin(pluginCfg.Name, pluginCfg.Path, pluginCfg.Config, pluginCfg.Hooks); err != nil {
					logger.WithError(err).Warnf("Failed to load plugin %s", pluginCfg.Name)
				}
			}
		}
	}

	// Initialize alert manager for health checks
	alertManager := health.NewAlertManager(logger)

	// Add logging alert channel
	alertManager.AddChannel(health.NewLogChannel(logger))

	// Add webhook alert channel if configured
	if cfg.Monitoring.WebhookURL != "" {
		webhookChannel := health.NewWebhookChannel(cfg.Monitoring.WebhookURL, logger)
		alertManager.AddChannel(webhookChannel)
		logger.WithField("webhook", cfg.Monitoring.WebhookURL).Info("Health alert webhook configured")
	}

	alertManager.Start()
	logger.Info("Alert manager started")

	// Initialize health checker
	healthCheckerConfig := health.Config{
		Interval:           30 * time.Second,
		Timeout:            5 * time.Second,
		UnhealthyThreshold: 3,
		HealthyThreshold:   2,
		ExpectedStatus:     []int{200, 204},
		InsecureSkipVerify: false,
	}
	healthChecker := health.NewTargetChecker(healthCheckerConfig, logger, alertManager)

	// Initialize service mesh integration
	meshConfig := servicemesh.Config{
		Enabled:         cfg.ServiceMesh.Enabled,
		Type:            servicemesh.MeshType(cfg.ServiceMesh.Type),
		Namespace:       cfg.ServiceMesh.Namespace,
		TrustDomain:     cfg.ServiceMesh.TrustDomain,
		DiscoveryAddr:   cfg.ServiceMesh.DiscoveryAddr,
		RefreshInterval: cfg.ServiceMesh.RefreshInterval,
		MTLSEnabled:     cfg.ServiceMesh.MTLSEnabled,
		CertFile:        cfg.ServiceMesh.CertFile,
		KeyFile:         cfg.ServiceMesh.KeyFile,
		CAFile:          cfg.ServiceMesh.CAFile,
	}

	// Convert mesh-specific configs
	if cfg.ServiceMesh.Istio != nil {
		meshConfig.Istio = &servicemesh.IstioConfig{
			PilotAddr:         cfg.ServiceMesh.Istio.PilotAddr,
			MixerAddr:         cfg.ServiceMesh.Istio.MixerAddr,
			EnableTelemetry:   cfg.ServiceMesh.Istio.EnableTelemetry,
			EnablePolicyCheck: cfg.ServiceMesh.Istio.EnablePolicyCheck,
			CustomHeaders:     cfg.ServiceMesh.Istio.CustomHeaders,
			InjectSidecar:     cfg.ServiceMesh.Istio.InjectSidecar,
		}
	}

	if cfg.ServiceMesh.Linkerd != nil {
		meshConfig.Linkerd = &servicemesh.LinkerdConfig{
			ControlPlaneAddr: cfg.ServiceMesh.Linkerd.ControlPlaneAddr,
			TapAddr:          cfg.ServiceMesh.Linkerd.TapAddr,
			EnableTap:        cfg.ServiceMesh.Linkerd.EnableTap,
			ProfileNamespace: cfg.ServiceMesh.Linkerd.ProfileNamespace,
		}
	}

	if cfg.ServiceMesh.Consul != nil {
		meshConfig.Consul = &servicemesh.ConsulConfig{
			HTTPAddr:      cfg.ServiceMesh.Consul.HTTPAddr,
			Datacenter:    cfg.ServiceMesh.Consul.Datacenter,
			Token:         cfg.ServiceMesh.Consul.Token,
			EnableConnect: cfg.ServiceMesh.Consul.EnableConnect,
		}
	}

	meshManager, err := servicemesh.NewManager(meshConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize service mesh: %w", err)
	}

	gateway := &Gateway{
		server:          e,
		config:          cfg,
		logger:          logger,
		adminHandler:    adminHandler,
		serviceRegistry: registry,
		router:          router,
		pluginManager:   pluginManager,
		tracingManager:  tracingManager,
		healthChecker:   healthChecker,
		alertManager:    alertManager,
		meshManager:     meshManager,
		mongoRepo:       mongoRepo,
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

	// Start service mesh integration
	if cfg.ServiceMesh.Enabled {
		ctx := context.Background()
		if err := meshManager.Start(ctx); err != nil {
			logger.WithError(err).Warn("Failed to start service mesh integration, continuing without it")
		} else {
			// Add service mesh middleware
			e.Use(servicemesh.Middleware(meshManager, logger))
			e.Use(servicemesh.ProxyMiddleware(meshManager, logger))
			logger.WithField("type", cfg.ServiceMesh.Type).Info("Service mesh integration started")
		}
	}

	// Add all service targets to health checker
	for _, svcConfig := range cfg.Services {
		if svcConfig.HealthCheck != nil && svcConfig.HealthCheck.Enabled {
			// Override defaults with service-specific config
			svcHealthConfig := health.Config{
				Interval:           svcConfig.HealthCheck.Interval,
				Timeout:            svcConfig.HealthCheck.Timeout,
				UnhealthyThreshold: svcConfig.HealthCheck.UnhealthyThreshold,
				HealthyThreshold:   svcConfig.HealthCheck.HealthyThreshold,
				ExpectedStatus:     svcConfig.HealthCheck.ExpectedStatus,
				InsecureSkipVerify: svcConfig.HealthCheck.InsecureSkipVerify,
			}

			// Use service-specific checker if it has custom config, otherwise use global
			var checker *health.TargetChecker
			if svcHealthConfig.Interval != 0 || svcHealthConfig.Timeout != 0 {
				checker = health.NewTargetChecker(svcHealthConfig, logger, alertManager)
				checker.Start() // Start service-specific checker immediately
			} else {
				checker = healthChecker
			}

			for _, target := range svcConfig.Targets {
				checker.AddTarget(target)
				logger.WithFields(logrus.Fields{
					"service": svcConfig.Name,
					"target":  target,
				}).Info("Added target to health monitoring")
			}
		}
	}

	// Start the global health checker
	healthChecker.Start()
	logger.Info("Health monitoring started")

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

	// Start monitoring metrics broadcaster
	collector := admin.GetCollector()
	collector.StartMetricsBroadcaster()
	logger.Info("Monitoring metrics broadcaster started")

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
	g.logger.Info("Stopping health monitoring...")
	if g.healthChecker != nil {
		g.healthChecker.Stop()
	}
	if g.alertManager != nil {
		g.alertManager.Stop()
	}
	g.logger.Info("Health monitoring stopped")

	// Stop service mesh integration
	if g.meshManager != nil {
		g.logger.Info("Stopping service mesh integration...")
		if err := g.meshManager.Stop(); err != nil {
			g.logger.WithError(err).Warn("Error stopping service mesh")
		}
	}

	return g.server.Shutdown(ctx)
}
