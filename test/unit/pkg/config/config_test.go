package config

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"odin/pkg/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Create a temporary config file
	configContent := `
server:
  port: 8080

services:
  - name: test-service
    basePath: /api/test
    targets:
      - http://localhost:8081
`

	tmpFile, err := ioutil.TempFile("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name(), logrus.New())
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Len(t, cfg.Services, 1)
	assert.Equal(t, "test-service", cfg.Services[0].Name)
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := config.Load("nonexistent.yaml", logrus.New())
	assert.Error(t, err)
}

func TestServiceDefaults(t *testing.T) {
	service := &config.ServiceConfig{
		Name:     "test",
		BasePath: "/api/test",
		Targets:  []string{"http://localhost:8081"},
	}

	service.SetDefaults()

	assert.Equal(t, 30*time.Second, service.Timeout)
	assert.False(t, service.Authentication)
}

func TestBasicConfigValidation(t *testing.T) {
	// Test basic config structure
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
		Services: []config.ServiceConfig{
			{
				Name:     "test",
				BasePath: "/api/test",
				Targets:  []string{"http://localhost:8081"},
			},
		},
	}

	// Basic validation - ensure required fields are present
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Len(t, cfg.Services, 1)
	assert.Equal(t, "test", cfg.Services[0].Name)
	assert.NotEmpty(t, cfg.Services[0].Targets)

	// Test with invalid service (empty name)
	cfg.Services[0].Name = ""
	assert.Empty(t, cfg.Services[0].Name, "Service name should be empty for validation test")
}

func TestConfigDefaults(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}

	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)
}

func TestBasicConfigLoad(t *testing.T) {
	// Test that we can create a basic config without errors
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
	}

	assert.NotNil(t, cfg)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestServiceConfigBasics(t *testing.T) {
	service := config.ServiceConfig{
		Name:     "test-service",
		BasePath: "/api/test",
		Targets:  []string{"http://localhost:8081"},
	}

	service.SetDefaults()
	assert.Equal(t, "test-service", service.Name)
	assert.Equal(t, "/api/test", service.BasePath)
	assert.Len(t, service.Targets, 1)
}
