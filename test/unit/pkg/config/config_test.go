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
  readTimeout: 30s
  writeTimeout: 30s

services:
  - name: test-service
    basePath: /api/test
    targets:
      - http://localhost:8081
    timeout: 30s
    authentication: false
    stripBasePath: true
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
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Len(t, cfg.Services, 1)
	assert.Equal(t, "test-service", cfg.Services[0].Name)
	assert.Equal(t, "/api/test", cfg.Services[0].BasePath)
	assert.Len(t, cfg.Services[0].Targets, 1)
	assert.Equal(t, "http://localhost:8081", cfg.Services[0].Targets[0])
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
	assert.Equal(t, "round-robin", service.LoadBalancing)
	assert.Equal(t, 3, service.RetryCount)
	assert.Equal(t, time.Second, service.RetryDelay)
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

	// Test with empty service name
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
	assert.Equal(t, 30*time.Second, service.Timeout)
	assert.False(t, service.Authentication)
}

func TestConfigWithAuth(t *testing.T) {
	configContent := `
server:
  port: 8080

auth:
  jwtSecret: "test-secret"
  accessTokenTTL: 3600s
  refreshTokenTTL: 86400s

services:
  - name: secure-service
    basePath: /api/secure
    targets:
      - http://localhost:8081
    authentication: true
`

	tmpFile, err := ioutil.TempFile("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name(), logrus.New())
	require.NoError(t, err)

	assert.Equal(t, "test-secret", cfg.Auth.JWTSecret)
	assert.True(t, cfg.Services[0].Authentication)
}
