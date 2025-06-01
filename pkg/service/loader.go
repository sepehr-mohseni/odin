package service

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Services []Config `yaml:"services"`
}

func LoadServices(configPath string, logger *logrus.Logger) ([]Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read services config file: %w", err)
	}

	var serviceConfig ServiceConfig
	if err := yaml.Unmarshal(data, &serviceConfig); err != nil {
		return nil, fmt.Errorf("failed to parse services config file: %w", err)
	}

	// Set defaults for each service
	for i := range serviceConfig.Services {
		setServiceDefaults(&serviceConfig.Services[i])
	}

	logger.WithField("count", len(serviceConfig.Services)).Info("Loaded services configuration")

	return serviceConfig.Services, nil
}

func setServiceDefaults(config *Config) {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 1 * time.Second
	}
	if config.LoadBalancing == "" {
		config.LoadBalancing = "round_robin"
	}
}

func ValidateService(config *Config) error {
	if config.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if config.BasePath == "" {
		return fmt.Errorf("service basePath cannot be empty")
	}
	if len(config.Targets) == 0 {
		return fmt.Errorf("service must have at least one target")
	}

	return nil
}
