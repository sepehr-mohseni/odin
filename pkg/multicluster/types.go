package multicluster

import (
	"time"
)

// ClusterConfig defines configuration for a cluster
type ClusterConfig struct {
	Name        string            `yaml:"name" json:"name"`
	Endpoint    string            `yaml:"endpoint" json:"endpoint"` // API endpoint of remote cluster gateway
	Region      string            `yaml:"region" json:"region"`
	Zone        string            `yaml:"zone" json:"zone"`
	Priority    int               `yaml:"priority" json:"priority"` // Higher priority = preferred cluster
	Weight      int               `yaml:"weight" json:"weight"`     // For weighted load balancing
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	HealthCheck HealthCheckConfig `yaml:"healthCheck" json:"healthCheck"`
	Auth        AuthConfig        `yaml:"auth" json:"auth"`
	TLS         TLSConfig         `yaml:"tls" json:"tls"`
	Metadata    map[string]string `yaml:"metadata" json:"metadata"`
}

// HealthCheckConfig defines health check configuration for a cluster
type HealthCheckConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	Interval           time.Duration `yaml:"interval" json:"interval"`
	Timeout            time.Duration `yaml:"timeout" json:"timeout"`
	HealthyThreshold   int           `yaml:"healthyThreshold" json:"healthyThreshold"`
	UnhealthyThreshold int           `yaml:"unhealthyThreshold" json:"unhealthyThreshold"`
	Path               string        `yaml:"path" json:"path"` // Health check endpoint path
}

// AuthConfig defines authentication configuration for cluster communication
type AuthConfig struct {
	Type     string `yaml:"type" json:"type"`         // none, token, mtls, oauth2
	Token    string `yaml:"token" json:"token"`       // For token auth
	Username string `yaml:"username" json:"username"` // For basic auth
	Password string `yaml:"password" json:"password"` // For basic auth
}

// TLSConfig defines TLS configuration for cluster communication
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"certFile" json:"certFile"`
	KeyFile  string `yaml:"keyFile" json:"keyFile"`
	CAFile   string `yaml:"caFile" json:"caFile"`
	Insecure bool   `yaml:"insecure" json:"insecure"` // Skip TLS verification (not recommended)
}

// Config defines the multi-cluster configuration
type Config struct {
	Enabled          bool            `yaml:"enabled" json:"enabled"`
	LocalCluster     string          `yaml:"localCluster" json:"localCluster"` // Name of local cluster
	Clusters         []ClusterConfig `yaml:"clusters" json:"clusters"`
	FailoverStrategy string          `yaml:"failoverStrategy" json:"failoverStrategy"` // priority, round-robin, least-load
	SyncInterval     time.Duration   `yaml:"syncInterval" json:"syncInterval"`         // Service sync interval
	LoadBalancing    string          `yaml:"loadBalancing" json:"loadBalancing"`       // round-robin, weighted, latency
	AffinityEnabled  bool            `yaml:"affinityEnabled" json:"affinityEnabled"`   // Enable session affinity
	AffinityTTL      time.Duration   `yaml:"affinityTTL" json:"affinityTTL"`
}

// ClusterInfo represents runtime information about a cluster
type ClusterInfo struct {
	Name         string            `json:"name"`
	Endpoint     string            `json:"endpoint"`
	Region       string            `json:"region"`
	Zone         string            `json:"zone"`
	Status       ClusterStatus     `json:"status"`
	Healthy      bool              `json:"healthy"`
	Services     []string          `json:"services"` // Services available in this cluster
	Latency      time.Duration     `json:"latency"`  // Average latency to this cluster
	LastChecked  time.Time         `json:"lastChecked"`
	FailureCount int               `json:"failureCount"`
	Metadata     map[string]string `json:"metadata"`
}

// ClusterStatus represents the health status of a cluster
type ClusterStatus string

const (
	ClusterStatusHealthy   ClusterStatus = "healthy"
	ClusterStatusDegraded  ClusterStatus = "degraded"
	ClusterStatusUnhealthy ClusterStatus = "unhealthy"
	ClusterStatusUnknown   ClusterStatus = "unknown"
)

// ServiceLocation represents where a service is deployed
type ServiceLocation struct {
	ServiceName string   `json:"serviceName"`
	Clusters    []string `json:"clusters"` // List of cluster names where service is deployed
}

// RouteRequest represents a request to route to a cluster
type RouteRequest struct {
	ServiceName     string            `json:"serviceName"`
	Method          string            `json:"method"`
	Path            string            `json:"path"`
	Headers         map[string]string `json:"headers"`
	SessionID       string            `json:"sessionId"`       // For session affinity
	PreferredRegion string            `json:"preferredRegion"` // Prefer clusters in this region
}

// RouteDecision represents the routing decision
type RouteDecision struct {
	ClusterName string `json:"clusterName"`
	Endpoint    string `json:"endpoint"`
	Reason      string `json:"reason"`   // Why this cluster was chosen
	Fallback    bool   `json:"fallback"` // Whether this is a fallback choice
}

// Manager manages multi-cluster operations
type Manager interface {
	// GetClusterInfo returns information about a specific cluster
	GetClusterInfo(name string) (*ClusterInfo, error)

	// ListClusters returns all configured clusters
	ListClusters() []*ClusterInfo

	// RouteRequest determines which cluster to route a request to
	RouteRequest(req *RouteRequest) (*RouteDecision, error)

	// GetServiceLocations returns which clusters have the specified service
	GetServiceLocations(serviceName string) (*ServiceLocation, error)

	// SyncServices synchronizes service information across clusters
	SyncServices() error

	// Start begins cluster monitoring and synchronization
	Start() error

	// Stop gracefully shuts down the manager
	Stop() error
}
