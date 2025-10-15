package service

import (
	"context"
	"fmt"

	"odin/pkg/config"
	"odin/pkg/mongodb"

	"github.com/sirupsen/logrus"
)

// LoaderInterface defines interface for loading services
type LoaderInterface interface {
	LoadServices(ctx context.Context) ([]Config, error)
	SaveService(ctx context.Context, svc *Config) error
	DeleteService(ctx context.Context, name string) error
	UpdateService(ctx context.Context, name string, svc *Config) error
}

// MongoDBLoader loads services from MongoDB
type MongoDBLoader struct {
	adapter *mongodb.ServiceAdapter
	logger  *logrus.Logger
}

// FileLoader loads services from YAML files
type FileLoader struct {
	configPath string
	logger     *logrus.Logger
}

// NewLoader creates appropriate loader based on configuration
func NewLoader(cfg *config.Config, logger *logrus.Logger) (LoaderInterface, error) {
	if cfg.MongoDB.Enabled {
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
			return nil, fmt.Errorf("failed to create MongoDB repository: %w", err)
		}

		adapter := mongodb.NewServiceAdapter(repo, logger)
		logger.Info("Using MongoDB for service storage")

		return &MongoDBLoader{
			adapter: adapter,
			logger:  logger,
		}, nil
	}

	// Fall back to file-based loader
	logger.Info("Using file-based service storage")
	return &FileLoader{
		logger: logger,
	}, nil
}

// MongoDB Loader implementation

func (l *MongoDBLoader) LoadServices(ctx context.Context) ([]Config, error) {
	cfgs, err := l.adapter.LoadServices(ctx)
	if err != nil {
		return nil, err
	}

	// Convert config.ServiceConfig to service.Config
	services := make([]Config, 0, len(cfgs))
	for _, cfg := range cfgs {
		svc := Config{
			Name:           cfg.Name,
			BasePath:       cfg.BasePath,
			Targets:        cfg.Targets,
			StripBasePath:  cfg.StripBasePath,
			Timeout:        cfg.Timeout,
			RetryCount:     cfg.RetryCount,
			RetryDelay:     cfg.RetryDelay,
			Authentication: cfg.Authentication,
			LoadBalancing:  cfg.LoadBalancing,
			Headers:        cfg.Headers,
			Protocol:       cfg.Protocol,
		}
		services = append(services, svc)
	}

	return services, nil
}

func (l *MongoDBLoader) SaveService(ctx context.Context, svc *Config) error {
	cfg := &config.ServiceConfig{
		Name:           svc.Name,
		BasePath:       svc.BasePath,
		Targets:        svc.Targets,
		StripBasePath:  svc.StripBasePath,
		Timeout:        svc.Timeout,
		RetryCount:     svc.RetryCount,
		RetryDelay:     svc.RetryDelay,
		Authentication: svc.Authentication,
		LoadBalancing:  svc.LoadBalancing,
		Headers:        svc.Headers,
		Protocol:       svc.Protocol,
	}

	return l.adapter.SaveService(ctx, cfg)
}

func (l *MongoDBLoader) DeleteService(ctx context.Context, name string) error {
	return l.adapter.DeleteService(ctx, name)
}

func (l *MongoDBLoader) UpdateService(ctx context.Context, name string, svc *Config) error {
	cfg := &config.ServiceConfig{
		Name:           svc.Name,
		BasePath:       svc.BasePath,
		Targets:        svc.Targets,
		StripBasePath:  svc.StripBasePath,
		Timeout:        svc.Timeout,
		RetryCount:     svc.RetryCount,
		RetryDelay:     svc.RetryDelay,
		Authentication: svc.Authentication,
		LoadBalancing:  svc.LoadBalancing,
		Headers:        svc.Headers,
		Protocol:       svc.Protocol,
	}

	return l.adapter.UpdateService(ctx, name, cfg)
}

// File Loader implementation

func (l *FileLoader) LoadServices(ctx context.Context) ([]Config, error) {
	// This will be loaded from the main config, not from a separate file
	l.logger.Info("Services loaded from main configuration file")
	return []Config{}, nil
}

func (l *FileLoader) SaveService(ctx context.Context, svc *Config) error {
	return fmt.Errorf("file-based loader does not support dynamic service updates")
}

func (l *FileLoader) DeleteService(ctx context.Context, name string) error {
	return fmt.Errorf("file-based loader does not support service deletion")
}

func (l *FileLoader) UpdateService(ctx context.Context, name string, svc *Config) error {
	return fmt.Errorf("file-based loader does not support service updates")
}
