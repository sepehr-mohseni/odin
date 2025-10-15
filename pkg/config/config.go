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
	Server       ServerConfig       `yaml:"server"`
	Logging      LoggingConfig      `yaml:"logging"`
	Auth         AuthConfig         `yaml:"auth"`
	RateLimit    RateLimitConfig    `yaml:"rateLimit"`
	Cache        CacheConfig        `yaml:"cache"`
	Monitoring   MonitoringConfig   `yaml:"monitoring"`
	Admin        AdminConfig        `yaml:"admin"`
	Plugins      PluginsConfig      `yaml:"plugins"`
	Tracing      TracingConfig      `yaml:"tracing"`
	Services     []ServiceConfig    `yaml:"services"`
	ServiceMesh  ServiceMeshConfig  `yaml:"serviceMesh"`
	WASM         WASMConfig         `yaml:"wasm"`
	MultiCluster MultiClusterConfig `yaml:"multiCluster"`
	OpenAPI      OpenAPIConfig      `yaml:"openapi"`
	MongoDB      MongoDBConfig      `yaml:"mongodb"`
}

type ServerConfig struct {
	Port            int           `yaml:"port"`
	Timeout         time.Duration `yaml:"timeout"`
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

type PluginsConfig struct {
	Enabled   bool           `yaml:"enabled"`
	Directory string         `yaml:"directory"`
	Plugins   []PluginConfig `yaml:"plugins"`
}

type PluginConfig struct {
	Name    string                 `yaml:"name"`
	Path    string                 `yaml:"path"`
	Enabled bool                   `yaml:"enabled"`
	Config  map[string]interface{} `yaml:"config"`
	Hooks   []string               `yaml:"hooks"` // pre-request, post-request, pre-response, post-response
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
	Enabled    bool   `yaml:"enabled"`
	Path       string `yaml:"path"`
	WebhookURL string `yaml:"webhookUrl,omitempty"` // Optional webhook for health alerts
}

type TracingConfig struct {
	Enabled        bool    `yaml:"enabled"`
	ServiceName    string  `yaml:"serviceName"`
	ServiceVersion string  `yaml:"serviceVersion"`
	Environment    string  `yaml:"environment"`
	Endpoint       string  `yaml:"endpoint"`
	SampleRate     float64 `yaml:"sampleRate"`
	Insecure       bool    `yaml:"insecure"`
}

type ServiceMeshConfig struct {
	Enabled         bool               `yaml:"enabled"`
	Type            string             `yaml:"type"` // istio, linkerd, consul, none
	Namespace       string             `yaml:"namespace"`
	TrustDomain     string             `yaml:"trustDomain"`
	DiscoveryAddr   string             `yaml:"discoveryAddr"`
	RefreshInterval time.Duration      `yaml:"refreshInterval"`
	MTLSEnabled     bool               `yaml:"mtlsEnabled"`
	CertFile        string             `yaml:"certFile"`
	KeyFile         string             `yaml:"keyFile"`
	CAFile          string             `yaml:"caFile"`
	Istio           *IstioMeshConfig   `yaml:"istio,omitempty"`
	Linkerd         *LinkerdMeshConfig `yaml:"linkerd,omitempty"`
	Consul          *ConsulMeshConfig  `yaml:"consul,omitempty"`
}

type IstioMeshConfig struct {
	PilotAddr         string   `yaml:"pilotAddr"`
	MixerAddr         string   `yaml:"mixerAddr"`
	EnableTelemetry   bool     `yaml:"enableTelemetry"`
	EnablePolicyCheck bool     `yaml:"enablePolicyCheck"`
	CustomHeaders     []string `yaml:"customHeaders"`
	InjectSidecar     bool     `yaml:"injectSidecar"`
}

type LinkerdMeshConfig struct {
	ControlPlaneAddr string `yaml:"controlPlaneAddr"`
	TapAddr          string `yaml:"tapAddr"`
	EnableTap        bool   `yaml:"enableTap"`
	ProfileNamespace string `yaml:"profileNamespace"`
}

type ConsulMeshConfig struct {
	HTTPAddr      string `yaml:"httpAddr"`
	Datacenter    string `yaml:"datacenter"`
	Token         string `yaml:"token"`
	EnableConnect bool   `yaml:"enableConnect"`
}

type WASMConfig struct {
	Enabled        bool               `yaml:"enabled"`
	PluginDir      string             `yaml:"pluginDir"`
	Plugins        []WASMPluginConfig `yaml:"plugins"`
	MaxMemoryPages int                `yaml:"maxMemoryPages"`
	MaxInstances   int                `yaml:"maxInstances"`
	CacheEnabled   bool               `yaml:"cacheEnabled"`
}

type WASMPluginConfig struct {
	Name        string                 `yaml:"name"`
	Path        string                 `yaml:"path"`
	Type        string                 `yaml:"type"` // request, response, auth, ratelimit, middleware, aggregation
	Enabled     bool                   `yaml:"enabled"`
	Priority    int                    `yaml:"priority"`
	Config      map[string]interface{} `yaml:"config"`
	Timeout     time.Duration          `yaml:"timeout"`
	AllowedURLs []string               `yaml:"allowedUrls"`
	Services    []string               `yaml:"services"`
}

type MultiClusterConfig struct {
	Enabled          bool            `yaml:"enabled"`
	LocalCluster     string          `yaml:"localCluster"`
	Clusters         []ClusterConfig `yaml:"clusters"`
	FailoverStrategy string          `yaml:"failoverStrategy"`
	SyncInterval     time.Duration   `yaml:"syncInterval"`
	LoadBalancing    string          `yaml:"loadBalancing"`
	AffinityEnabled  bool            `yaml:"affinityEnabled"`
	AffinityTTL      time.Duration   `yaml:"affinityTTL"`
}

type ClusterConfig struct {
	Name        string             `yaml:"name"`
	Endpoint    string             `yaml:"endpoint"`
	Region      string             `yaml:"region"`
	Zone        string             `yaml:"zone"`
	Priority    int                `yaml:"priority"`
	Weight      int                `yaml:"weight"`
	Enabled     bool               `yaml:"enabled"`
	HealthCheck ClusterHealthCheck `yaml:"healthCheck"`
	Auth        ClusterAuth        `yaml:"auth"`
	TLS         ClusterTLS         `yaml:"tls"`
	Metadata    map[string]string  `yaml:"metadata"`
}

type ClusterHealthCheck struct {
	Enabled            bool          `yaml:"enabled"`
	Interval           time.Duration `yaml:"interval"`
	Timeout            time.Duration `yaml:"timeout"`
	HealthyThreshold   int           `yaml:"healthyThreshold"`
	UnhealthyThreshold int           `yaml:"unhealthyThreshold"`
	Path               string        `yaml:"path"`
}

type ClusterAuth struct {
	Type     string `yaml:"type"`
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ClusterTLS struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
	CAFile   string `yaml:"caFile"`
	Insecure bool   `yaml:"insecure"`
}

type OpenAPIConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Title        string `yaml:"title"`
	Version      string `yaml:"version"`
	Description  string `yaml:"description"`
	AutoGenerate bool   `yaml:"autoGenerate"`
	OutputPath   string `yaml:"outputPath"`
	UIEnabled    bool   `yaml:"uiEnabled"`
	UIPath       string `yaml:"uiPath"`
}

type MongoDBConfig struct {
	Enabled        bool          `yaml:"enabled"`
	URI            string        `yaml:"uri"`
	Database       string        `yaml:"database"`
	MaxPoolSize    int           `yaml:"maxPoolSize"`
	MinPoolSize    int           `yaml:"minPoolSize"`
	ConnectTimeout time.Duration `yaml:"connectTimeout"`
	Auth           MongoDBAuth   `yaml:"auth"`
	TLS            MongoDBTLS    `yaml:"tls"`
}

type MongoDBAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	AuthDB   string `yaml:"authDB"`
}

type MongoDBTLS struct {
	Enabled  bool   `yaml:"enabled"`
	CAFile   string `yaml:"caFile"`
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

type ServiceConfig struct {
	Name           string             `yaml:"name"`
	BasePath       string             `yaml:"basePath"`
	Targets        []string           `yaml:"targets"`
	StripBasePath  bool               `yaml:"stripBasePath"`
	Timeout        time.Duration      `yaml:"timeout"`
	RetryCount     int                `yaml:"retryCount"`
	RetryDelay     time.Duration      `yaml:"retryDelay"`
	Authentication bool               `yaml:"authentication"`
	LoadBalancing  string             `yaml:"loadBalancing"`
	Headers        map[string]string  `yaml:"headers"`
	Protocol       string             `yaml:"protocol"` // http, graphql, grpc
	Transform      TransformConfig    `yaml:"transform"`
	Aggregation    *AggregationConfig `yaml:"aggregation,omitempty"`
	GraphQL        *GraphQLConfig     `yaml:"graphql,omitempty"`
	GRPC           *GRPCConfig        `yaml:"grpc,omitempty"`
	HealthCheck    *HealthCheckConfig `yaml:"healthCheck,omitempty"`
}

type TransformConfig struct {
	Request  []TransformRule `yaml:"request"`
	Response []TransformRule `yaml:"response"`
}

type TransformRule struct {
	From    string `yaml:"from"`
	To      string `yaml:"to"`
	Default string `yaml:"default"`
}

type AggregationConfig struct {
	Dependencies []DependencyConfig `yaml:"dependencies"`
}

type GraphQLConfig struct {
	MaxQueryDepth       int           `yaml:"maxQueryDepth"`
	MaxQueryComplexity  int           `yaml:"maxQueryComplexity"`
	EnableIntrospection bool          `yaml:"enableIntrospection"`
	EnableQueryCaching  bool          `yaml:"enableQueryCaching"`
	CacheTTL            time.Duration `yaml:"cacheTTL"`
}

type GRPCConfig struct {
	ProtoFiles       []string `yaml:"protoFiles"`
	ImportPaths      []string `yaml:"importPaths"`
	EnableReflection bool     `yaml:"enableReflection"`
	MaxMessageSize   int      `yaml:"maxMessageSize"`
	EnableTLS        bool     `yaml:"enableTLS"`
	TLSCertFile      string   `yaml:"tlsCertFile"`
	TLSKeyFile       string   `yaml:"tlsKeyFile"`
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

// HealthCheckConfig holds health check configuration for backend targets
type HealthCheckConfig struct {
	Enabled            bool          `yaml:"enabled"`
	Interval           time.Duration `yaml:"interval"`           // How often to check (default: 30s)
	Timeout            time.Duration `yaml:"timeout"`            // Request timeout (default: 5s)
	UnhealthyThreshold int           `yaml:"unhealthyThreshold"` // Failures before unhealthy (default: 3)
	HealthyThreshold   int           `yaml:"healthyThreshold"`   // Successes before healthy (default: 2)
	ExpectedStatus     []int         `yaml:"expectedStatus"`     // Expected HTTP status codes (default: [200, 204])
	InsecureSkipVerify bool          `yaml:"insecureSkipVerify"` // Skip TLS verification
}

// SetDefaults sets default values for ServiceConfig
func (s *ServiceConfig) SetDefaults() {
	if s.LoadBalancing == "" {
		s.LoadBalancing = "round-robin"
	}
	if s.Timeout == 0 {
		s.Timeout = 30 * time.Second
	}
	if s.RetryCount == 0 {
		s.RetryCount = 3
	}
	if s.RetryDelay == 0 {
		s.RetryDelay = 1 * time.Second
	}
	if s.Protocol == "" {
		s.Protocol = "http"
	}
}

func Load(configPath string, logger *logrus.Logger) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}
	if config.Server.GracefulTimeout == 0 {
		config.Server.GracefulTimeout = 15 * time.Second
	}
	if config.Server.Timeout == 0 {
		config.Server.Timeout = 30 * time.Second
	}

	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}

	if config.Monitoring.Path == "" {
		config.Monitoring.Path = "/metrics"
	}

	// Set tracing defaults
	if config.Tracing.ServiceName == "" {
		config.Tracing.ServiceName = "odin-gateway"
	}
	if config.Tracing.ServiceVersion == "" {
		config.Tracing.ServiceVersion = "1.0.0"
	}
	if config.Tracing.Environment == "" {
		config.Tracing.Environment = "development"
	}
	if config.Tracing.Endpoint == "" {
		config.Tracing.Endpoint = "http://localhost:4318/v1/traces"
	}
	if config.Tracing.SampleRate == 0 {
		config.Tracing.SampleRate = 1.0
	}

	// Load services from external file if available
	servicesPath := filepath.Join(filepath.Dir(configPath), "services.yaml")
	if _, err := os.Stat(servicesPath); err == nil {
		servicesData, err := os.ReadFile(servicesPath)
		if err == nil {
			var servicesConfig struct {
				Services []ServiceConfig `yaml:"services"`
			}
			if err := yaml.Unmarshal(servicesData, &servicesConfig); err == nil {
				config.Services = append(config.Services, servicesConfig.Services...)
			}
		}
	}

	// Set defaults for services
	for i := range config.Services {
		config.Services[i].SetDefaults()
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	for _, service := range config.Services {
		if service.Name == "" {
			return fmt.Errorf("service name cannot be empty")
		}
		if service.BasePath == "" {
			return fmt.Errorf("service %s: basePath cannot be empty", service.Name)
		}
		if len(service.Targets) == 0 {
			return fmt.Errorf("service %s: at least one target must be specified", service.Name)
		}
	}

	return nil
}
