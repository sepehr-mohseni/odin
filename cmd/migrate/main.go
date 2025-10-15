package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"odin/pkg/config"
	"odin/pkg/mongodb"

	"github.com/sirupsen/logrus"
)

func main() {
	var (
		configFile    = flag.String("config", "config/config.yaml", "Path to configuration file")
		servicesFile  = flag.String("services", "", "Path to services YAML file (optional)")
		routesFile    = flag.String("routes", "", "Path to routes YAML file (optional)")
		mongoURI      = flag.String("mongodb-uri", "", "MongoDB connection URI (overrides config)")
		mongoDatabase = flag.String("mongodb-database", "odin_gateway", "MongoDB database name")
		dryRun        = flag.Bool("dry-run", false, "Perform dry run without actual migration")
		force         = flag.Bool("force", false, "Force migration even if services exist")
		verbose       = flag.Bool("verbose", false, "Enable verbose logging")
	)

	flag.Parse()

	// Setup logger
	logger := logrus.New()
	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logger.Info("Starting MongoDB migration tool")

	// Load main configuration
	cfg, err := config.Load(*configFile, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Override MongoDB settings if provided
	if *mongoURI != "" {
		cfg.MongoDB.URI = *mongoURI
		cfg.MongoDB.Enabled = true
	}
	if *mongoDatabase != "" {
		cfg.MongoDB.Database = *mongoDatabase
	}

	if !cfg.MongoDB.Enabled {
		logger.Fatal("MongoDB is not enabled in configuration")
	}

	// Validate MongoDB configuration
	if cfg.MongoDB.URI == "" {
		logger.Fatal("MongoDB URI is required")
	}
	if cfg.MongoDB.Database == "" {
		logger.Fatal("MongoDB database name is required")
	}

	logger.WithFields(logrus.Fields{
		"uri":      maskURI(cfg.MongoDB.URI),
		"database": cfg.MongoDB.Database,
		"dry_run":  *dryRun,
	}).Info("MongoDB configuration")

	// Create MongoDB repository
	mongoConfig := &mongodb.Config{
		Enabled:        cfg.MongoDB.Enabled,
		URI:            cfg.MongoDB.URI,
		Database:       cfg.MongoDB.Database,
		MaxPoolSize:    cfg.MongoDB.MaxPoolSize,
		MinPoolSize:    cfg.MongoDB.MinPoolSize,
		ConnectTimeout: cfg.MongoDB.ConnectTimeout,
		TLS: mongodb.TLSConfig{
			Enabled:  cfg.MongoDB.TLS.Enabled,
			CAFile:   cfg.MongoDB.TLS.CAFile,
			CertFile: cfg.MongoDB.TLS.CertFile,
			KeyFile:  cfg.MongoDB.TLS.KeyFile,
		},
		Auth: mongodb.AuthConfig{
			Username: cfg.MongoDB.Auth.Username,
			Password: cfg.MongoDB.Auth.Password,
			AuthDB:   cfg.MongoDB.Auth.AuthDB,
		},
	}

	repo, err := mongodb.NewRepository(mongoConfig, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to MongoDB")
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := repo.Close(ctx); err != nil {
			logger.WithError(err).Error("Failed to close MongoDB connection")
		}
	}()

	logger.Info("Connected to MongoDB successfully")

	// Check if services already exist
	ctx := context.Background()
	existingServices, err := repo.ListServices(ctx, nil)
	if err != nil {
		logger.WithError(err).Fatal("Failed to list existing services")
	}

	if len(existingServices) > 0 && !*force {
		logger.WithField("count", len(existingServices)).Fatal(
			"Services already exist in MongoDB. Use --force to overwrite")
	}

	if len(existingServices) > 0 {
		logger.WithField("count", len(existingServices)).Warn("Existing services will be replaced")
	}

	// Collect services from various sources
	var allServices []config.ServiceConfig

	// 1. Services from main config
	if len(cfg.Services) > 0 {
		logger.WithField("count", len(cfg.Services)).Info("Found services in main config")
		allServices = append(allServices, cfg.Services...)
	}

	// 2. Services from separate services file
	if *servicesFile != "" {
		services, err := loadServicesFromYAML(*servicesFile, logger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to load services file")
		}
		logger.WithField("count", len(services)).Info("Loaded services from services file")
		allServices = append(allServices, services...)
	}

	// 3. Services from routes file (if different format)
	if *routesFile != "" {
		routes, err := loadServicesFromYAML(*routesFile, logger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to load routes file")
		}
		logger.WithField("count", len(routes)).Info("Loaded services from routes file")
		allServices = append(allServices, routes...)
	}

	if len(allServices) == 0 {
		logger.Fatal("No services found to migrate")
	}

	// Remove duplicates by name
	uniqueServices := make(map[string]config.ServiceConfig)
	for _, svc := range allServices {
		if _, exists := uniqueServices[svc.Name]; exists {
			logger.WithField("service", svc.Name).Warn("Duplicate service found, using latest definition")
		}
		uniqueServices[svc.Name] = svc
	}

	// Convert to slice
	servicesToMigrate := make([]config.ServiceConfig, 0, len(uniqueServices))
	for _, svc := range uniqueServices {
		servicesToMigrate = append(servicesToMigrate, svc)
	}

	logger.WithField("count", len(servicesToMigrate)).Info("Total unique services to migrate")

	// Dry run - just print what would be migrated
	if *dryRun {
		logger.Info("DRY RUN MODE - No changes will be made")
		logger.Info("Services that would be migrated:")
		for i, svc := range servicesToMigrate {
			logger.WithFields(logrus.Fields{
				"index":    i + 1,
				"name":     svc.Name,
				"basePath": svc.BasePath,
				"targets":  len(svc.Targets),
				"protocol": svc.Protocol,
				"enabled":  true,
			}).Info("Service")
		}
		logger.Info("Migration would complete successfully (dry run)")
		return
	}

	// Perform actual migration
	adapter := mongodb.NewServiceAdapter(repo, logger)

	migratedCount := 0
	failedCount := 0
	var errors []error

	for i, svc := range servicesToMigrate {
		logger.WithFields(logrus.Fields{
			"index":    i + 1,
			"total":    len(servicesToMigrate),
			"service":  svc.Name,
			"basePath": svc.BasePath,
		}).Info("Migrating service")

		// Set defaults
		svc.SetDefaults()

		if err := adapter.SaveService(ctx, &svc); err != nil {
			logger.WithError(err).WithField("service", svc.Name).Error("Failed to migrate service")
			failedCount++
			errors = append(errors, fmt.Errorf("service %s: %w", svc.Name, err))
			continue
		}

		migratedCount++
		logger.WithField("service", svc.Name).Info("Service migrated successfully")
	}

	// Summary
	logger.Info("=====================================")
	logger.WithFields(logrus.Fields{
		"total":    len(servicesToMigrate),
		"migrated": migratedCount,
		"failed":   failedCount,
	}).Info("Migration completed")

	if failedCount > 0 {
		logger.Error("Some services failed to migrate:")
		for _, err := range errors {
			logger.Error(err.Error())
		}
		os.Exit(1)
	}

	// Verify migration
	logger.Info("Verifying migration...")
	verifiedServices, err := repo.ListServices(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("Failed to verify migration")
		os.Exit(1)
	}

	logger.WithField("count", len(verifiedServices)).Info("Services in MongoDB after migration")

	// Create audit log entry
	auditLog := &mongodb.AuditLogDocument{
		UserID:    "migration-tool",
		Username:  "migration-tool",
		Action:    "migrate_services",
		Resource:  "services",
		Timestamp: time.Now(),
		Changes: map[string]interface{}{
			"migrated_count": migratedCount,
			"failed_count":   failedCount,
			"total_count":    len(servicesToMigrate),
		},
		IPAddress: "localhost",
		Status:    "success",
		Message:   fmt.Sprintf("Migrated %d services successfully", migratedCount),
	}

	if err := repo.CreateAuditLog(ctx, auditLog); err != nil {
		logger.WithError(err).Warn("Failed to create audit log")
	}

	logger.Info("✅ Migration completed successfully!")
	logger.Info("You can now enable MongoDB in your configuration and restart Odin")
}

func loadServicesFromYAML(path string, logger *logrus.Logger) ([]config.ServiceConfig, error) {
	cfg, err := config.Load(path, logger)
	if err != nil {
		return nil, err
	}

	if len(cfg.Services) == 0 {
		logger.Warn("No services found in file")
	}

	return cfg.Services, nil
}

func maskURI(uri string) string {
	// Simple masking for security
	return "mongodb://***:***@***"
}
