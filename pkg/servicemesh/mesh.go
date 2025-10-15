package servicemesh

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ServiceMesh interface defines the contract for service mesh integrations
type ServiceMesh interface {
	// Start begins the service mesh integration
	Start(ctx context.Context) error

	// Stop gracefully shuts down the service mesh integration
	Stop() error

	// DiscoverServices returns all available services
	DiscoverServices() (*ServiceDiscoveryResult, error)

	// GetServiceEndpoints returns endpoints for a specific service
	GetServiceEndpoints(serviceName string) ([]ServiceEndpoint, error)

	// GetHTTPClient returns an HTTP client configured for mesh communication
	GetHTTPClient() (*http.Client, error)

	// InjectHeaders adds required mesh headers to a request
	InjectHeaders(headers http.Header) error

	// GetMeshType returns the type of service mesh
	GetMeshType() MeshType
}

// Manager manages service mesh integration
type Manager struct {
	config   Config
	logger   *logrus.Logger
	mesh     ServiceMesh
	mu       sync.RWMutex
	services *ServiceDiscoveryResult
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewManager creates a new service mesh manager
func NewManager(config Config, logger *logrus.Logger) (*Manager, error) {
	if !config.Enabled {
		logger.Info("Service mesh integration is disabled")
		return &Manager{
			config: config,
			logger: logger,
			mesh:   NewNoneMesh(logger),
		}, nil
	}

	// Set defaults
	if config.RefreshInterval == 0 {
		config.RefreshInterval = 30 * time.Second
	}
	if config.Namespace == "" {
		config.Namespace = "default"
	}

	var mesh ServiceMesh
	var err error

	switch config.Type {
	case MeshTypeIstio:
		if config.Istio == nil {
			return nil, fmt.Errorf("istio configuration is required when type is istio")
		}
		mesh, err = NewIstioMesh(config, logger)
	case MeshTypeLinkerd:
		if config.Linkerd == nil {
			return nil, fmt.Errorf("linkerd configuration is required when type is linkerd")
		}
		mesh, err = NewLinkerdMesh(config, logger)
	case MeshTypeConsul:
		if config.Consul == nil {
			return nil, fmt.Errorf("consul configuration is required when type is consul")
		}
		mesh, err = NewConsulMesh(config, logger)
	case MeshTypeNone:
		mesh = NewNoneMesh(logger)
	default:
		return nil, fmt.Errorf("unsupported service mesh type: %s", config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s mesh: %w", config.Type, err)
	}

	manager := &Manager{
		config:   config,
		logger:   logger,
		mesh:     mesh,
		stopChan: make(chan struct{}),
		services: &ServiceDiscoveryResult{
			Services:  make(map[string][]ServiceEndpoint),
			Timestamp: time.Now(),
		},
	}

	logger.WithFields(logrus.Fields{
		"type":      config.Type,
		"namespace": config.Namespace,
		"mtls":      config.MTLSEnabled,
	}).Info("Service mesh manager initialized")

	return manager, nil
}

// Start begins the service mesh integration
func (m *Manager) Start(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}

	m.logger.Info("Starting service mesh integration")

	// Start the mesh-specific implementation
	if err := m.mesh.Start(ctx); err != nil {
		return fmt.Errorf("failed to start mesh: %w", err)
	}

	// Start periodic service discovery
	m.wg.Add(1)
	go m.discoveryLoop()

	// Initial service discovery
	if _, err := m.refreshServices(); err != nil {
		m.logger.WithError(err).Warn("Initial service discovery failed")
	}

	return nil
}

// Stop gracefully shuts down the service mesh integration
func (m *Manager) Stop() error {
	if !m.config.Enabled {
		return nil
	}

	m.logger.Info("Stopping service mesh integration")
	close(m.stopChan)
	m.wg.Wait()

	return m.mesh.Stop()
}

// discoveryLoop periodically refreshes service discovery
func (m *Manager) discoveryLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := m.refreshServices(); err != nil {
				m.logger.WithError(err).Warn("Service discovery refresh failed")
			}
		case <-m.stopChan:
			return
		}
	}
}

// refreshServices updates the cached service discovery data
func (m *Manager) refreshServices() (*ServiceDiscoveryResult, error) {
	result, err := m.mesh.DiscoverServices()
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.services = result
	m.mu.Unlock()

	serviceCount := 0
	for _, endpoints := range result.Services {
		serviceCount += len(endpoints)
	}

	m.logger.WithFields(logrus.Fields{
		"services":  len(result.Services),
		"endpoints": serviceCount,
	}).Debug("Service discovery refreshed")

	return result, nil
}

// GetServices returns all discovered services
func (m *Manager) GetServices() *ServiceDiscoveryResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.services
}

// GetServiceEndpoints returns endpoints for a specific service
func (m *Manager) GetServiceEndpoints(serviceName string) ([]ServiceEndpoint, error) {
	return m.mesh.GetServiceEndpoints(serviceName)
}

// GetHTTPClient returns an HTTP client configured for mesh communication
func (m *Manager) GetHTTPClient() (*http.Client, error) {
	return m.mesh.GetHTTPClient()
}

// InjectHeaders adds required mesh headers to a request
func (m *Manager) InjectHeaders(headers http.Header) error {
	return m.mesh.InjectHeaders(headers)
}

// GetMeshType returns the type of service mesh
func (m *Manager) GetMeshType() MeshType {
	return m.mesh.GetMeshType()
}

// createTLSConfig creates TLS configuration for mTLS
func createTLSConfig(config Config) (*tls.Config, error) {
	if !config.MTLSEnabled {
		return nil, nil
	}

	// Load client cert
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client cert: %w", err)
	}

	// Load CA cert
	caCert, err := os.ReadFile(config.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}
