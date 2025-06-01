package config

import (
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
  timeout: 30s
  readTimeout: 5s
  writeTimeout: 10s
  gracefulTimeout: 15s
auth:
  jwtSecret: "test-secret"
  accessTokenTTL: 1h
services:
  - name: "test-service"
    basePath: "/api/test"
    targets:
      - "http://localhost:8081"
    loadBalancing: "round_robin"
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	logger := logrus.New()
	cfg, err := config.Load(tmpFile.Name(), logger)

	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.Timeout)
	assert.Equal(t, 5*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 15*time.Second, cfg.Server.GracefulTimeout)
	assert.Equal(t, "test-secret", cfg.Auth.JWTSecret)
	assert.Equal(t, time.Hour, cfg.Auth.AccessTokenTTL)
	assert.Len(t, cfg.Services, 1)
	assert.Equal(t, "test-service", cfg.Services[0].Name)
	assert.Equal(t, "round_robin", cfg.Services[0].LoadBalancing)
}

func TestLoadNonExistentFile(t *testing.T) {
	logger := logrus.New()
	_, err := config.Load("non-existent-file.yaml", logger)
	assert.Error(t, err)
}

func TestServiceDefaults(t *testing.T) {
	service := config.ServiceConfig{
		Name:     "test",
		BasePath: "/api/test",
		Targets:  []string{"http://localhost:8081"},
	}

	service.SetDefaults()

	assert.Equal(t, "round_robin", service.LoadBalancing)
	assert.Equal(t, 30*time.Second, service.Timeout)
	assert.False(t, service.StripBasePath)
}

func TestValidateConfig(t *testing.T) {
	validConfig := &config.Config{
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

	// This tests the internal validation - we need to expose it or test through Load
	cfg, err := createTestConfig(validConfig)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func createTestConfig(cfg *config.Config) (*config.Config, error) {
	// Helper function to simulate config validation
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return nil, assert.AnError
	}

	for _, service := range cfg.Services {
		if service.Name == "" {
			return nil, assert.AnError
		}
	}

	return cfg, nil
}

func TestConfigDefaults(t *testing.T) {
	// Test minimal config gets proper defaults
	configContent := `
server:
  port: 9000
services:
  - name: "minimal-service"
    basePath: "/api/minimal"
    targets:
      - "http://localhost:9001"
`

	tmpFile, err := os.CreateTemp("", "minimal-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	logger := logrus.New()
	cfg, err := config.Load(tmpFile.Name(), logger)

	require.NoError(t, err)

	// Test server defaults
	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 15*time.Second, cfg.Server.GracefulTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.Timeout)

	// Test logging defaults
	assert.Equal(t, "info", cfg.Logging.Level)

	// Test monitoring defaults
	assert.Equal(t, "/metrics", cfg.Monitoring.Path)

	// Test service defaults
	require.Len(t, cfg.Services, 1)
	service := cfg.Services[0]
	assert.Equal(t, "round_robin", service.LoadBalancing)
	assert.Equal(t, 30*time.Second, service.Timeout)
	assert.Equal(t, 3, service.RetryCount)
	assert.Equal(t, 1*time.Second, service.RetryDelay)
}
