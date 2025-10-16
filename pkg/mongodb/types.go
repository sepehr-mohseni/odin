package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Config defines MongoDB configuration
type Config struct {
	Enabled        bool          `yaml:"enabled" json:"enabled"`
	URI            string        `yaml:"uri" json:"uri"`
	Database       string        `yaml:"database" json:"database"`
	ConnectTimeout time.Duration `yaml:"connectTimeout" json:"connectTimeout"`
	QueryTimeout   time.Duration `yaml:"queryTimeout" json:"queryTimeout"`
	MaxPoolSize    int           `yaml:"maxPoolSize" json:"maxPoolSize"`
	MinPoolSize    int           `yaml:"minPoolSize" json:"minPoolSize"`
	TLS            TLSConfig     `yaml:"tls" json:"tls"`
	Auth           AuthConfig    `yaml:"auth" json:"auth"`
}

// TLSConfig defines TLS configuration for MongoDB
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CAFile   string `yaml:"caFile" json:"caFile"`
	CertFile string `yaml:"certFile" json:"certFile"`
	KeyFile  string `yaml:"keyFile" json:"keyFile"`
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	AuthDB   string `yaml:"authDB" json:"authDB"`
}

// Collections defines MongoDB collection names
const (
	ServicesCollection     = "services"
	ConfigCollection       = "config"
	MetricsCollection      = "metrics"
	TracesCollection       = "traces"
	AlertsCollection       = "alerts"
	HealthChecksCollection = "health_checks"
	ClustersCollection     = "clusters"
	PluginsCollection      = "plugins"
	UsersCollection        = "users"
	APIKeysCollection      = "api_keys"
	RateLimitsCollection   = "rate_limits"
	CacheCollection        = "cache"
	AuditLogsCollection    = "audit_logs"
)

// ServiceDocument represents a service in MongoDB
type ServiceDocument struct {
	ID             string                 `bson:"_id,omitempty" json:"id"`
	Name           string                 `bson:"name" json:"name"`
	BasePath       string                 `bson:"basePath" json:"basePath"`
	Targets        []string               `bson:"targets" json:"targets"`
	StripBasePath  bool                   `bson:"stripBasePath" json:"stripBasePath"`
	Timeout        int64                  `bson:"timeout" json:"timeout"` // milliseconds
	RetryCount     int                    `bson:"retryCount" json:"retryCount"`
	RetryDelay     int64                  `bson:"retryDelay" json:"retryDelay"` // milliseconds
	Authentication bool                   `bson:"authentication" json:"authentication"`
	LoadBalancing  string                 `bson:"loadBalancing" json:"loadBalancing"`
	Headers        map[string]string      `bson:"headers" json:"headers"`
	Protocol       string                 `bson:"protocol" json:"protocol"`
	Enabled        bool                   `bson:"enabled" json:"enabled"`
	Transform      map[string]interface{} `bson:"transform,omitempty" json:"transform,omitempty"`
	Aggregation    map[string]interface{} `bson:"aggregation,omitempty" json:"aggregation,omitempty"`
	HealthCheck    map[string]interface{} `bson:"healthCheck,omitempty" json:"healthCheck,omitempty"`
	CreatedAt      time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time              `bson:"updatedAt" json:"updatedAt"`
	Metadata       map[string]string      `bson:"metadata" json:"metadata"`
}

// ConfigDocument represents gateway configuration in MongoDB
type ConfigDocument struct {
	ID        string                 `bson:"_id,omitempty" json:"id"`
	Version   string                 `bson:"version" json:"version"`
	Config    map[string]interface{} `bson:"config" json:"config"`
	Active    bool                   `bson:"active" json:"active"`
	CreatedAt time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time              `bson:"updatedAt" json:"updatedAt"`
	CreatedBy string                 `bson:"createdBy" json:"createdBy"`
}

// MetricDocument represents a metric entry in MongoDB
type MetricDocument struct {
	ID        string                 `bson:"_id,omitempty" json:"id"`
	Name      string                 `bson:"name" json:"name"`
	Type      string                 `bson:"type" json:"type"` // counter, gauge, histogram
	Value     float64                `bson:"value" json:"value"`
	Labels    map[string]string      `bson:"labels" json:"labels"`
	Timestamp time.Time              `bson:"timestamp" json:"timestamp"`
	TTL       time.Time              `bson:"ttl" json:"ttl"` // For automatic expiration
	Metadata  map[string]interface{} `bson:"metadata" json:"metadata"`
}

// TraceDocument represents a trace in MongoDB
type TraceDocument struct {
	ID          string            `bson:"_id,omitempty" json:"id"`
	TraceID     string            `bson:"traceId" json:"traceId"`
	SpanID      string            `bson:"spanId" json:"spanId"`
	ParentID    string            `bson:"parentId,omitempty" json:"parentId,omitempty"`
	ServiceName string            `bson:"serviceName" json:"serviceName"`
	Operation   string            `bson:"operation" json:"operation"`
	StartTime   time.Time         `bson:"startTime" json:"startTime"`
	Duration    int64             `bson:"duration" json:"duration"` // microseconds
	Tags        map[string]string `bson:"tags" json:"tags"`
	Logs        []TraceLog        `bson:"logs" json:"logs"`
	Status      string            `bson:"status" json:"status"`
	TTL         time.Time         `bson:"ttl" json:"ttl"`
}

// TraceLog represents a log entry in a trace
type TraceLog struct {
	Timestamp time.Time              `bson:"timestamp" json:"timestamp"`
	Fields    map[string]interface{} `bson:"fields" json:"fields"`
}

// AlertDocument represents an alert in MongoDB
type AlertDocument struct {
	ID          string                 `bson:"_id,omitempty" json:"id"`
	Name        string                 `bson:"name" json:"name"`
	Type        string                 `bson:"type" json:"type"` // email, webhook, slack
	Severity    string                 `bson:"severity" json:"severity"`
	Message     string                 `bson:"message" json:"message"`
	Source      string                 `bson:"source" json:"source"`
	ServiceName string                 `bson:"serviceName,omitempty" json:"serviceName,omitempty"`
	Metadata    map[string]interface{} `bson:"metadata" json:"metadata"`
	Triggered   time.Time              `bson:"triggered" json:"triggered"`
	Resolved    *time.Time             `bson:"resolved,omitempty" json:"resolved,omitempty"`
	Status      string                 `bson:"status" json:"status"` // active, resolved, acknowledged
}

// HealthCheckDocument represents a health check result in MongoDB
type HealthCheckDocument struct {
	ID          string                 `bson:"_id,omitempty" json:"id"`
	ServiceName string                 `bson:"serviceName" json:"serviceName"`
	Target      string                 `bson:"target" json:"target"`
	Status      string                 `bson:"status" json:"status"` // healthy, unhealthy, degraded
	StatusCode  int                    `bson:"statusCode,omitempty" json:"statusCode,omitempty"`
	Latency     int64                  `bson:"latency" json:"latency"` // milliseconds
	Message     string                 `bson:"message,omitempty" json:"message,omitempty"`
	CheckedAt   time.Time              `bson:"checkedAt" json:"checkedAt"`
	Metadata    map[string]interface{} `bson:"metadata" json:"metadata"`
	TTL         time.Time              `bson:"ttl" json:"ttl"`
}

// ClusterDocument represents a cluster in MongoDB
type ClusterDocument struct {
	ID          string                 `bson:"_id,omitempty" json:"id"`
	Name        string                 `bson:"name" json:"name"`
	Endpoint    string                 `bson:"endpoint" json:"endpoint"`
	Region      string                 `bson:"region" json:"region"`
	Zone        string                 `bson:"zone" json:"zone"`
	Priority    int                    `bson:"priority" json:"priority"`
	Weight      int                    `bson:"weight" json:"weight"`
	Enabled     bool                   `bson:"enabled" json:"enabled"`
	Status      string                 `bson:"status" json:"status"` // healthy, degraded, unhealthy
	Services    []string               `bson:"services" json:"services"`
	Metadata    map[string]interface{} `bson:"metadata" json:"metadata"`
	LastChecked time.Time              `bson:"lastChecked" json:"lastChecked"`
	CreatedAt   time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `bson:"updatedAt" json:"updatedAt"`
}

// PluginDocument represents a WASM plugin in MongoDB
type PluginDocument struct {
	ID          string                 `bson:"_id,omitempty" json:"id"`
	Name        string                 `bson:"name" json:"name"`
	Path        string                 `bson:"path" json:"path"`
	Type        string                 `bson:"type" json:"type"`
	Enabled     bool                   `bson:"enabled" json:"enabled"`
	Priority    int                    `bson:"priority" json:"priority"`
	Config      map[string]interface{} `bson:"config" json:"config"`
	Timeout     int64                  `bson:"timeout" json:"timeout"` // milliseconds
	AllowedURLs []string               `bson:"allowedUrls" json:"allowedUrls"`
	Services    []string               `bson:"services" json:"services"`
	Version     string                 `bson:"version" json:"version"`
	CreatedAt   time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `bson:"updatedAt" json:"updatedAt"`
}

// UserDocument represents a user in MongoDB
type UserDocument struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	Username  string    `bson:"username" json:"username"`
	Email     string    `bson:"email" json:"email"`
	Password  string    `bson:"password" json:"password"` // Hashed
	Role      string    `bson:"role" json:"role"`         // admin, user, viewer
	Active    bool      `bson:"active" json:"active"`
	APIKeys   []string  `bson:"apiKeys" json:"apiKeys"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
	LastLogin time.Time `bson:"lastLogin" json:"lastLogin"`
}

// APIKeyDocument represents an API key in MongoDB
type APIKeyDocument struct {
	ID          string            `bson:"_id,omitempty" json:"id"`
	Key         string            `bson:"key" json:"key"`
	Name        string            `bson:"name" json:"name"`
	UserID      string            `bson:"userId" json:"userId"`
	Permissions []string          `bson:"permissions" json:"permissions"`
	RateLimit   int               `bson:"rateLimit" json:"rateLimit"`
	Enabled     bool              `bson:"enabled" json:"enabled"`
	ExpiresAt   *time.Time        `bson:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	CreatedAt   time.Time         `bson:"createdAt" json:"createdAt"`
	LastUsed    time.Time         `bson:"lastUsed" json:"lastUsed"`
	Metadata    map[string]string `bson:"metadata" json:"metadata"`
}

// RateLimitDocument represents rate limit state in MongoDB
type RateLimitDocument struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	Key       string    `bson:"key" json:"key"` // User ID, IP, or API key
	Count     int       `bson:"count" json:"count"`
	Window    time.Time `bson:"window" json:"window"`
	ExpiresAt time.Time `bson:"expiresAt" json:"expiresAt"`
}

// CacheDocument represents cached data in MongoDB
type CacheDocument struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	Key       string    `bson:"key" json:"key"`
	Value     []byte    `bson:"value" json:"value"`
	ExpiresAt time.Time `bson:"expiresAt" json:"expiresAt"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}

// AuditLogDocument represents an audit log entry
type AuditLogDocument struct {
	ID        string                 `bson:"_id,omitempty" json:"id"`
	Action    string                 `bson:"action" json:"action"`
	Resource  string                 `bson:"resource" json:"resource"`
	UserID    string                 `bson:"userId" json:"userId"`
	Username  string                 `bson:"username" json:"username"`
	IPAddress string                 `bson:"ipAddress" json:"ipAddress"`
	Changes   map[string]interface{} `bson:"changes" json:"changes"`
	Status    string                 `bson:"status" json:"status"` // success, failure
	Message   string                 `bson:"message,omitempty" json:"message,omitempty"`
	Timestamp time.Time              `bson:"timestamp" json:"timestamp"`
	TTL       time.Time              `bson:"ttl" json:"ttl"`
}

// Repository defines the interface for MongoDB operations
type Repository interface {
	// Database access
	GetDatabase() *mongo.Database

	// Service operations
	CreateService(ctx context.Context, service *ServiceDocument) error
	GetService(ctx context.Context, id string) (*ServiceDocument, error)
	GetServiceByName(ctx context.Context, name string) (*ServiceDocument, error)
	ListServices(ctx context.Context, enabled *bool) ([]*ServiceDocument, error)
	UpdateService(ctx context.Context, id string, service *ServiceDocument) error
	DeleteService(ctx context.Context, id string) error

	// Config operations
	SaveConfig(ctx context.Context, config *ConfigDocument) error
	GetActiveConfig(ctx context.Context) (*ConfigDocument, error)
	GetConfigByVersion(ctx context.Context, version string) (*ConfigDocument, error)
	ListConfigs(ctx context.Context, limit int) ([]*ConfigDocument, error)

	// Metrics operations
	SaveMetric(ctx context.Context, metric *MetricDocument) error
	QueryMetrics(ctx context.Context, name string, start, end time.Time, labels map[string]string) ([]*MetricDocument, error)

	// Trace operations
	SaveTrace(ctx context.Context, trace *TraceDocument) error
	GetTrace(ctx context.Context, traceID string) ([]*TraceDocument, error)
	QueryTraces(ctx context.Context, serviceName string, start, end time.Time) ([]*TraceDocument, error)

	// Alert operations
	CreateAlert(ctx context.Context, alert *AlertDocument) error
	GetAlert(ctx context.Context, id string) (*AlertDocument, error)
	ListAlerts(ctx context.Context, status string) ([]*AlertDocument, error)
	UpdateAlert(ctx context.Context, id string, alert *AlertDocument) error
	ResolveAlert(ctx context.Context, id string) error

	// Health check operations
	SaveHealthCheck(ctx context.Context, check *HealthCheckDocument) error
	GetLatestHealthCheck(ctx context.Context, serviceName string) (*HealthCheckDocument, error)
	QueryHealthChecks(ctx context.Context, serviceName string, start, end time.Time) ([]*HealthCheckDocument, error)

	// Cluster operations
	CreateCluster(ctx context.Context, cluster *ClusterDocument) error
	GetCluster(ctx context.Context, id string) (*ClusterDocument, error)
	ListClusters(ctx context.Context, enabled *bool) ([]*ClusterDocument, error)
	UpdateCluster(ctx context.Context, id string, cluster *ClusterDocument) error
	DeleteCluster(ctx context.Context, id string) error

	// Plugin operations
	CreatePlugin(ctx context.Context, plugin *PluginDocument) error
	GetPlugin(ctx context.Context, id string) (*PluginDocument, error)
	ListPlugins(ctx context.Context, enabled *bool) ([]*PluginDocument, error)
	UpdatePlugin(ctx context.Context, id string, plugin *PluginDocument) error
	DeletePlugin(ctx context.Context, id string) error

	// User operations
	CreateUser(ctx context.Context, user *UserDocument) error
	GetUser(ctx context.Context, id string) (*UserDocument, error)
	GetUserByUsername(ctx context.Context, username string) (*UserDocument, error)
	ListUsers(ctx context.Context) ([]*UserDocument, error)
	UpdateUser(ctx context.Context, id string, user *UserDocument) error
	DeleteUser(ctx context.Context, id string) error

	// API Key operations
	CreateAPIKey(ctx context.Context, key *APIKeyDocument) error
	GetAPIKey(ctx context.Context, key string) (*APIKeyDocument, error)
	ListAPIKeys(ctx context.Context, userID string) ([]*APIKeyDocument, error)
	UpdateAPIKey(ctx context.Context, id string, key *APIKeyDocument) error
	DeleteAPIKey(ctx context.Context, id string) error

	// Rate limit operations
	GetRateLimit(ctx context.Context, key string) (*RateLimitDocument, error)
	UpdateRateLimit(ctx context.Context, limit *RateLimitDocument) error

	// Cache operations
	GetCache(ctx context.Context, key string) (*CacheDocument, error)
	SetCache(ctx context.Context, cache *CacheDocument) error
	DeleteCache(ctx context.Context, key string) error

	// Audit log operations
	CreateAuditLog(ctx context.Context, log *AuditLogDocument) error
	QueryAuditLogs(ctx context.Context, userID string, start, end time.Time) ([]*AuditLogDocument, error)

	// Health and utility
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
}
