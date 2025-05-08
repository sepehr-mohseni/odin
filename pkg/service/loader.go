package service

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type ServiceConfigFile struct {
	Services []*Config `yaml:"services"`
}

func LoadFromFile(filename string, logger *logrus.Logger) (*Registry, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read service configuration: %w", err)
	}

	var configFile ServiceConfigFile
	if err := yaml.Unmarshal(data, &configFile); err != nil {
		return nil, fmt.Errorf("failed to parse service configuration: %w", err)
	}

	registry := NewRegistry(logger)

	for _, svc := range configFile.Services {
		if err := registry.Register(svc); err != nil {
			logger.WithError(err).Warnf("Failed to register service %s", svc.Name)
		}
	}

	return registry, nil
}

func (r *Registry) SaveToFile(filename string) error {
	services := r.GetAllServices()

	configFile := ServiceConfigFile{
		Services: services,
	}

	data, err := yaml.Marshal(configFile)
	if err != nil {
		return fmt.Errorf("failed to marshal service configuration: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write service configuration file: %w", err)
	}

	return nil
}
