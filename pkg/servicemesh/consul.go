package servicemesh

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// ConsulMesh implements ServiceMesh interface for Consul Connect
type ConsulMesh struct {
	config     Config
	logger     *logrus.Logger
	client     *http.Client
	consulAddr string
}

// NewConsulMesh creates a new Consul service mesh integration
func NewConsulMesh(config Config, logger *logrus.Logger) (*ConsulMesh, error) {
	if config.Consul == nil {
		return nil, fmt.Errorf("consul configuration is required")
	}

	tlsConfig, err := createTLSConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	mesh := &ConsulMesh{
		config:     config,
		logger:     logger,
		client:     client,
		consulAddr: config.Consul.HTTPAddr,
	}

	logger.WithFields(logrus.Fields{
		"http_addr":       config.Consul.HTTPAddr,
		"datacenter":      config.Consul.Datacenter,
		"connect_enabled": config.Consul.EnableConnect,
		"mtls_enabled":    config.MTLSEnabled,
	}).Info("Consul mesh integration configured")

	return mesh, nil
}

// Start begins the Consul service mesh integration
func (c *ConsulMesh) Start(ctx context.Context) error {
	c.logger.Info("Starting Consul mesh integration")

	// Verify connection to Consul via HTTP
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/v1/status/leader", c.consulAddr), nil)
	if err != nil {
		return fmt.Errorf("failed to create consul request: %w", err)
	}

	if c.config.Consul.Token != "" {
		req.Header.Set("X-Consul-Token", c.config.Consul.Token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to consul: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("consul returned status %d", resp.StatusCode)
	}

	c.logger.Info("Successfully connected to Consul")
	return nil
}

// Stop gracefully shuts down the Consul integration
func (c *ConsulMesh) Stop() error {
	c.logger.Info("Stopping Consul mesh integration")
	return nil
}

// DiscoverServices returns all available services from Consul
func (c *ConsulMesh) DiscoverServices() (*ServiceDiscoveryResult, error) {
	c.logger.Debug("Discovering services from Consul")

	result := &ServiceDiscoveryResult{
		Services:  make(map[string][]ServiceEndpoint),
		Timestamp: time.Now(),
	}

	// In a real implementation, this would query Consul's HTTP API
	// GET /v1/catalog/services to get all services
	// Then for each service, GET /v1/health/service/{name}?passing=true

	c.logger.Debug("Service discovery completed")

	return result, nil
}

// GetServiceEndpoints returns endpoints for a specific service
func (c *ConsulMesh) GetServiceEndpoints(serviceName string) ([]ServiceEndpoint, error) {
	c.logger.WithField("service", serviceName).Debug("Getting service endpoints from Consul")

	// In a real implementation, this would query:
	// GET /v1/health/service/{serviceName}?passing=true
	// and parse the response to build ServiceEndpoint list

	return []ServiceEndpoint{}, nil
}

// GetHTTPClient returns an HTTP client configured for Consul Connect
func (c *ConsulMesh) GetHTTPClient() (*http.Client, error) {
	return c.client, nil
}

// InjectHeaders adds Consul-specific headers to the request
func (c *ConsulMesh) InjectHeaders(headers http.Header) error {
	// Add Consul trace headers
	if headers.Get("X-Consul-Token") == "" && c.config.Consul.Token != "" {
		headers.Set("X-Consul-Token", c.config.Consul.Token)
	}

	// Add request tracing
	if headers.Get("X-Request-Id") == "" {
		headers.Set("X-Request-Id", generateRequestID())
	}

	// Add datacenter info
	if c.config.Consul.Datacenter != "" {
		headers.Set("X-Consul-Datacenter", c.config.Consul.Datacenter)
	}

	return nil
}

// GetMeshType returns the mesh type
func (c *ConsulMesh) GetMeshType() MeshType {
	return MeshTypeConsul
}
