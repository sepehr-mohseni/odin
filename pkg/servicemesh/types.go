package servicemesh

import (
	"time"
)

// MeshType represents the type of service mesh
type MeshType string

const (
	MeshTypeIstio   MeshType = "istio"
	MeshTypeLinkerd MeshType = "linkerd"
	MeshTypeConsul  MeshType = "consul"
	MeshTypeNone    MeshType = "none"
)

// Config holds service mesh configuration
type Config struct {
	Enabled         bool          `yaml:"enabled"`
	Type            MeshType      `yaml:"type"`
	Namespace       string        `yaml:"namespace"`
	TrustDomain     string        `yaml:"trustDomain"`
	DiscoveryAddr   string        `yaml:"discoveryAddr"`
	RefreshInterval time.Duration `yaml:"refreshInterval"`

	// mTLS settings
	MTLSEnabled bool   `yaml:"mtlsEnabled"`
	CertFile    string `yaml:"certFile"`
	KeyFile     string `yaml:"keyFile"`
	CAFile      string `yaml:"caFile"`

	// Istio-specific
	Istio *IstioConfig `yaml:"istio,omitempty"`

	// Linkerd-specific
	Linkerd *LinkerdConfig `yaml:"linkerd,omitempty"`

	// Consul-specific
	Consul *ConsulConfig `yaml:"consul,omitempty"`
}

// IstioConfig holds Istio-specific configuration
type IstioConfig struct {
	PilotAddr         string   `yaml:"pilotAddr"`
	MixerAddr         string   `yaml:"mixerAddr"`
	EnableTelemetry   bool     `yaml:"enableTelemetry"`
	EnablePolicyCheck bool     `yaml:"enablePolicyCheck"`
	CustomHeaders     []string `yaml:"customHeaders"`
	InjectSidecar     bool     `yaml:"injectSidecar"`
}

// LinkerdConfig holds Linkerd-specific configuration
type LinkerdConfig struct {
	ControlPlaneAddr string `yaml:"controlPlaneAddr"`
	TapAddr          string `yaml:"tapAddr"`
	EnableTap        bool   `yaml:"enableTap"`
	ProfileNamespace string `yaml:"profileNamespace"`
}

// ConsulConfig holds Consul-specific configuration
type ConsulConfig struct {
	HTTPAddr      string `yaml:"httpAddr"`
	Datacenter    string `yaml:"datacenter"`
	Token         string `yaml:"token"`
	EnableConnect bool   `yaml:"enableConnect"`
}

// ServiceEndpoint represents a discovered service endpoint
type ServiceEndpoint struct {
	ServiceName string            `json:"serviceName"`
	Address     string            `json:"address"`
	Port        int               `json:"port"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	Healthy     bool              `json:"healthy"`
	Weight      int               `json:"weight"`
}

// ServiceDiscoveryResult holds discovered services
type ServiceDiscoveryResult struct {
	Services  map[string][]ServiceEndpoint `json:"services"`
	Timestamp time.Time                    `json:"timestamp"`
}
