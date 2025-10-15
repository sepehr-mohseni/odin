package servicemesh

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// NoneMesh implements ServiceMesh interface for no mesh integration
type NoneMesh struct {
	logger *logrus.Logger
	client *http.Client
}

// NewNoneMesh creates a new no-mesh implementation
func NewNoneMesh(logger *logrus.Logger) *NoneMesh {
	return &NoneMesh{
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start is a no-op for NoneMesh
func (n *NoneMesh) Start(ctx context.Context) error {
	return nil
}

// Stop is a no-op for NoneMesh
func (n *NoneMesh) Stop() error {
	return nil
}

// DiscoverServices returns empty result for NoneMesh
func (n *NoneMesh) DiscoverServices() (*ServiceDiscoveryResult, error) {
	return &ServiceDiscoveryResult{
		Services:  make(map[string][]ServiceEndpoint),
		Timestamp: time.Now(),
	}, nil
}

// GetServiceEndpoints returns empty endpoints for NoneMesh
func (n *NoneMesh) GetServiceEndpoints(serviceName string) ([]ServiceEndpoint, error) {
	return []ServiceEndpoint{}, nil
}

// GetHTTPClient returns a basic HTTP client
func (n *NoneMesh) GetHTTPClient() (*http.Client, error) {
	return n.client, nil
}

// InjectHeaders is a no-op for NoneMesh
func (n *NoneMesh) InjectHeaders(headers http.Header) error {
	return nil
}

// GetMeshType returns MeshTypeNone
func (n *NoneMesh) GetMeshType() MeshType {
	return MeshTypeNone
}
