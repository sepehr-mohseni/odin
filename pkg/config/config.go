package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Logging    LoggingConfig    `yaml:"logging"`
	Auth       AuthConfig       `yaml:"auth"`
	RateLimit  RateLimitConfig  `yaml:"rateLimit"`
	Cache      CacheConfig      `yaml:"cache"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Admin      AdminConfig      `yaml:"admin"`
	Services   []ServiceConfig  `yaml:"-"`
}

type ServerConfig struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"readTimeout"`
	WriteTimeout    time.Duration `yaml:"writeTimeout"`
	GracefulTimeout time.Duration `yaml:"gracefulTimeout"`
	Compression     bool          `yaml:"compression"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	JSON  bool   `yaml:"json"`
}

type AuthConfig struct {
	JWTSecret         string        `yaml:"jwtSecret"`
	AccessTokenTTL    time.Duration `yaml:"accessTokenTTL"`
	RefreshTokenTTL   time.Duration `yaml:"refreshTokenTTL"`
	IgnorePathRegexes []string      `yaml:"ignorePathRegexes"`
}

type AdminConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type RateLimitConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Limit    int           `yaml:"limit"`
	Duration time.Duration `yaml:"duration"`
	Strategy string        `yaml:"strategy"`
	RedisURL string        `yaml:"redisUrl"`
}

type CacheConfig struct {
	Enabled     bool          `yaml:"enabled"`
	TTL         time.Duration `yaml:"ttl"`
	RedisURL    string        `yaml:"redisUrl"`
	Strategy    string        `yaml:"strategy"`
	MaxSizeInMB int           `yaml:"maxSizeInMB"`
}

type MonitoringConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type ServiceConfig struct {
	Name           string            `yaml:"name"`
	BasePath       string            `yaml:"basePath"`
	Targets        []string          `yaml:"targets"`
	StripBasePath  bool              `yaml:"stripBasePath"`
	Timeout        time.Duration     `yaml:"timeout"`
	RetryCount     int               `yaml:"retryCount"`
	RetryDelay     time.Duration     `yaml:"retryDelay"`
	Authentication bool              `yaml:"authentication"`
	LoadBalancing  string            `yaml:"loadBalancing"`
	Headers        map[string]string `yaml:"headers"`
	Transform      struct {
		Request  []TransformRule `yaml:"request"`
		Response []TransformRule `yaml:"response"`
	} `yaml:"transform"`
	Aggregation *AggregationConfig `yaml:"aggregation,omitempty"`
}

type AggregationConfig struct {
	Dependencies []DependencyConfig `yaml:"dependencies"`
}

type DependencyConfig struct {
	Service          string          `yaml:"service"`
	Path             string          `yaml:"path"`
	ParameterMapping []MappingConfig `yaml:"parameterMapping"`
	ResultMapping    []MappingConfig `yaml:"resultMapping"`
}

type MappingConfig struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

type TransformRule struct {
	From    string `yaml:"from"`
	To      string `yaml:"to"`
	Default string `yaml:"default"`
}

func Load(path string, logger *logrus.Logger) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if envPort := os.Getenv("GATEWAY_PORT"); envPort != "" {
		var port int
		if _, err := fmt.Sscanf(envPort, "%d", &port); err == nil {
			config.Server.Port = port
		}
	}

	servicesConfig, err := LoadServices(filepath.Join(filepath.Dir(path), "services.yaml"), logger)
	if err != nil {
		logger.Warnf("Failed to load services configuration: %v", err)
		logger.Info("Proceeding with empty services configuration")
	} else {
		config.Services = servicesConfig
	}

	return &config, nil
}

func LoadServices(path string, logger *logrus.Logger) ([]ServiceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read services config file: %w", err)
	}

	var servicesWrapper struct {
		Services []ServiceConfig `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &servicesWrapper); err != nil {
		return nil, fmt.Errorf("failed to parse services config file: %w", err)
	}

	logger.Infof("Loaded %d services from %s", len(servicesWrapper.Services), path)

	return servicesWrapper.Services, nil
}
