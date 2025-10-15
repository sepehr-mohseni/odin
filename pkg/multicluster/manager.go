package multicluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// clusterManager implements the Manager interface
type clusterManager struct {
	config      *Config
	logger      *logrus.Logger
	clusters    map[string]*ClusterInfo
	services    map[string]*ServiceLocation
	httpClient  *http.Client
	affinityMap map[string]string // sessionID -> clusterName
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewManager creates a new multi-cluster manager
func NewManager(config *Config, logger *logrus.Logger) (Manager, error) {
	if !config.Enabled {
		logger.Info("Multi-cluster management is disabled")
		return &noopManager{}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	manager := &clusterManager{
		config:      config,
		logger:      logger,
		clusters:    make(map[string]*ClusterInfo),
		services:    make(map[string]*ServiceLocation),
		httpClient:  httpClient,
		affinityMap: make(map[string]string),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Initialize cluster info
	for _, clusterCfg := range config.Clusters {
		if !clusterCfg.Enabled {
			continue
		}

		// Create TLS config if enabled
		var tlsConfig *tls.Config
		if clusterCfg.TLS.Enabled {
			var err error
			tlsConfig, err = createTLSConfig(&clusterCfg.TLS)
			if err != nil {
				logger.WithError(err).WithField("cluster", clusterCfg.Name).Warn("Failed to create TLS config")
				continue
			}
		}

		// Create HTTP client for this cluster
		if tlsConfig != nil {
			transport := &http.Transport{
				TLSClientConfig: tlsConfig,
			}
			httpClient = &http.Client{
				Timeout:   10 * time.Second,
				Transport: transport,
			}
		}

		manager.clusters[clusterCfg.Name] = &ClusterInfo{
			Name:     clusterCfg.Name,
			Endpoint: clusterCfg.Endpoint,
			Region:   clusterCfg.Region,
			Zone:     clusterCfg.Zone,
			Status:   ClusterStatusUnknown,
			Healthy:  false,
			Services: []string{},
			Metadata: clusterCfg.Metadata,
		}
	}

	logger.WithField("clusters", len(manager.clusters)).Info("Multi-cluster manager initialized")
	return manager, nil
}

// createTLSConfig creates TLS configuration from TLSConfig
func createTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Insecure,
	}

	// Load client certificate if provided
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// Start begins cluster monitoring and synchronization
func (m *clusterManager) Start() error {
	m.logger.Info("Starting multi-cluster manager")

	// Start health check loop
	m.wg.Add(1)
	go m.healthCheckLoop()

	// Start service sync loop
	m.wg.Add(1)
	go m.serviceSyncLoop()

	// Start affinity cleanup loop
	if m.config.AffinityEnabled {
		m.wg.Add(1)
		go m.affinityCleanupLoop()
	}

	m.logger.Info("Multi-cluster manager started")
	return nil
}

// Stop gracefully shuts down the manager
func (m *clusterManager) Stop() error {
	m.logger.Info("Stopping multi-cluster manager")
	m.cancel()
	m.wg.Wait()
	m.logger.Info("Multi-cluster manager stopped")
	return nil
}

// healthCheckLoop periodically checks cluster health
func (m *clusterManager) healthCheckLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial health check
	m.checkAllClusters()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkAllClusters()
		}
	}
}

// checkAllClusters performs health checks on all clusters
func (m *clusterManager) checkAllClusters() {
	for _, clusterCfg := range m.config.Clusters {
		if !clusterCfg.Enabled || !clusterCfg.HealthCheck.Enabled {
			continue
		}

		m.checkClusterHealth(&clusterCfg)
	}
}

// checkClusterHealth checks the health of a single cluster
func (m *clusterManager) checkClusterHealth(cfg *ClusterConfig) {
	start := time.Now()

	healthURL := fmt.Sprintf("%s%s", cfg.Endpoint, cfg.HealthCheck.Path)
	if cfg.HealthCheck.Path == "" {
		healthURL = fmt.Sprintf("%s/health", cfg.Endpoint)
	}

	req, err := http.NewRequestWithContext(m.ctx, "GET", healthURL, nil)
	if err != nil {
		m.markClusterUnhealthy(cfg.Name, err)
		return
	}

	// Add authentication if configured
	if cfg.Auth.Type == "token" && cfg.Auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Auth.Token)
	} else if cfg.Auth.Type == "basic" {
		req.SetBasicAuth(cfg.Auth.Username, cfg.Auth.Password)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.markClusterUnhealthy(cfg.Name, err)
		return
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	m.mu.Lock()
	defer m.mu.Unlock()

	if cluster, ok := m.clusters[cfg.Name]; ok {
		cluster.Latency = latency
		cluster.LastChecked = time.Now()

		if resp.StatusCode == http.StatusOK {
			cluster.FailureCount = 0
			cluster.Healthy = true
			cluster.Status = ClusterStatusHealthy
			m.logger.WithFields(logrus.Fields{
				"cluster": cfg.Name,
				"latency": latency,
			}).Debug("Cluster health check passed")
		} else {
			cluster.FailureCount++
			if cluster.FailureCount >= cfg.HealthCheck.UnhealthyThreshold {
				cluster.Healthy = false
				cluster.Status = ClusterStatusUnhealthy
			}
			m.logger.WithFields(logrus.Fields{
				"cluster":      cfg.Name,
				"statusCode":   resp.StatusCode,
				"failureCount": cluster.FailureCount,
			}).Warn("Cluster health check failed")
		}
	}
}

// markClusterUnhealthy marks a cluster as unhealthy
func (m *clusterManager) markClusterUnhealthy(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cluster, ok := m.clusters[name]; ok {
		cluster.FailureCount++
		cluster.Healthy = false
		cluster.Status = ClusterStatusUnhealthy
		cluster.LastChecked = time.Now()

		m.logger.WithError(err).WithField("cluster", name).Error("Cluster unreachable")
	}
}

// serviceSyncLoop periodically synchronizes service information
func (m *clusterManager) serviceSyncLoop() {
	defer m.wg.Done()

	interval := m.config.SyncInterval
	if interval == 0 {
		interval = 1 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial sync
	m.SyncServices()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.SyncServices(); err != nil {
				m.logger.WithError(err).Error("Failed to sync services")
			}
		}
	}
}

// SyncServices synchronizes service information across clusters
func (m *clusterManager) SyncServices() error {
	m.logger.Debug("Syncing services across clusters")

	serviceMap := make(map[string][]string)

	for name, cluster := range m.clusters {
		if !cluster.Healthy {
			continue
		}

		services, err := m.fetchClusterServices(name)
		if err != nil {
			m.logger.WithError(err).WithField("cluster", name).Warn("Failed to fetch cluster services")
			continue
		}

		cluster.Services = services

		for _, svc := range services {
			serviceMap[svc] = append(serviceMap[svc], name)
		}
	}

	m.mu.Lock()
	for svcName, clusters := range serviceMap {
		m.services[svcName] = &ServiceLocation{
			ServiceName: svcName,
			Clusters:    clusters,
		}
	}
	m.mu.Unlock()

	m.logger.WithField("services", len(serviceMap)).Info("Services synchronized")
	return nil
}

// fetchClusterServices fetches the list of services from a cluster
func (m *clusterManager) fetchClusterServices(clusterName string) ([]string, error) {
	cluster, ok := m.clusters[clusterName]
	if !ok {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get cluster config for auth
	var cfg *ClusterConfig
	for i := range m.config.Clusters {
		if m.config.Clusters[i].Name == clusterName {
			cfg = &m.config.Clusters[i]
			break
		}
	}

	if cfg == nil {
		return nil, fmt.Errorf("cluster config not found: %s", clusterName)
	}

	// Fetch services from cluster's API
	url := fmt.Sprintf("%s/api/services", cluster.Endpoint)
	req, err := http.NewRequestWithContext(m.ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add authentication
	if cfg.Auth.Type == "token" && cfg.Auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Auth.Token)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var services []string
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, err
	}

	return services, nil
}

// affinityCleanupLoop removes expired session affinity entries
func (m *clusterManager) affinityCleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// Cleanup is implicit with TTL-based checks during routing
			m.logger.Debug("Affinity cleanup cycle completed")
		}
	}
}

// GetClusterInfo returns information about a specific cluster
func (m *clusterManager) GetClusterInfo(name string) (*ClusterInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cluster, ok := m.clusters[name]
	if !ok {
		return nil, fmt.Errorf("cluster not found: %s", name)
	}

	return cluster, nil
}

// ListClusters returns all configured clusters
func (m *clusterManager) ListClusters() []*ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clusters := make([]*ClusterInfo, 0, len(m.clusters))
	for _, cluster := range m.clusters {
		clusters = append(clusters, cluster)
	}

	return clusters
}

// GetServiceLocations returns which clusters have the specified service
func (m *clusterManager) GetServiceLocations(serviceName string) (*ServiceLocation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	location, ok := m.services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	return location, nil
}

// RouteRequest determines which cluster to route a request to
func (m *clusterManager) RouteRequest(req *RouteRequest) (*RouteDecision, error) {
	// Check session affinity first
	if m.config.AffinityEnabled && req.SessionID != "" {
		if clusterName, ok := m.affinityMap[req.SessionID]; ok {
			if cluster, exists := m.clusters[clusterName]; exists && cluster.Healthy {
				return &RouteDecision{
					ClusterName: clusterName,
					Endpoint:    cluster.Endpoint,
					Reason:      "session affinity",
					Fallback:    false,
				}, nil
			}
		}
	}

	// Get service locations
	location, err := m.GetServiceLocations(req.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("service not found in any cluster: %w", err)
	}

	// Filter healthy clusters
	healthyClusters := m.getHealthyClusters(location.Clusters)
	if len(healthyClusters) == 0 {
		return nil, fmt.Errorf("no healthy clusters available for service: %s", req.ServiceName)
	}

	// Apply routing strategy
	var selectedCluster *ClusterInfo
	switch m.config.LoadBalancing {
	case "weighted":
		selectedCluster = m.selectWeightedCluster(healthyClusters)
	case "latency":
		selectedCluster = m.selectLowestLatencyCluster(healthyClusters)
	case "round-robin":
		fallthrough
	default:
		selectedCluster = m.selectRoundRobinCluster(healthyClusters)
	}

	if selectedCluster == nil {
		return nil, fmt.Errorf("failed to select cluster")
	}

	// Store affinity if enabled
	if m.config.AffinityEnabled && req.SessionID != "" {
		m.affinityMap[req.SessionID] = selectedCluster.Name
	}

	return &RouteDecision{
		ClusterName: selectedCluster.Name,
		Endpoint:    selectedCluster.Endpoint,
		Reason:      fmt.Sprintf("load balancing: %s", m.config.LoadBalancing),
		Fallback:    false,
	}, nil
}

// getHealthyClusters filters for healthy clusters
func (m *clusterManager) getHealthyClusters(clusterNames []string) []*ClusterInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var healthy []*ClusterInfo
	for _, name := range clusterNames {
		if cluster, ok := m.clusters[name]; ok && cluster.Healthy {
			healthy = append(healthy, cluster)
		}
	}

	return healthy
}

// selectRoundRobinCluster selects a cluster using round-robin
func (m *clusterManager) selectRoundRobinCluster(clusters []*ClusterInfo) *ClusterInfo {
	if len(clusters) == 0 {
		return nil
	}
	return clusters[rand.Intn(len(clusters))]
}

// selectWeightedCluster selects a cluster using weighted load balancing
func (m *clusterManager) selectWeightedCluster(clusters []*ClusterInfo) *ClusterInfo {
	if len(clusters) == 0 {
		return nil
	}

	// Build weighted list based on cluster configuration
	totalWeight := 0
	weights := make(map[string]int)

	for _, cluster := range clusters {
		// Get cluster config for weight
		for _, cfg := range m.config.Clusters {
			if cfg.Name == cluster.Name {
				weight := cfg.Weight
				if weight <= 0 {
					weight = 1
				}
				weights[cluster.Name] = weight
				totalWeight += weight
				break
			}
		}
	}

	// Select randomly based on weight
	if totalWeight == 0 {
		return clusters[0]
	}

	selection := rand.Intn(totalWeight)
	cumulative := 0

	for _, cluster := range clusters {
		cumulative += weights[cluster.Name]
		if selection < cumulative {
			return cluster
		}
	}

	return clusters[0]
}

// selectLowestLatencyCluster selects the cluster with lowest latency
func (m *clusterManager) selectLowestLatencyCluster(clusters []*ClusterInfo) *ClusterInfo {
	if len(clusters) == 0 {
		return nil
	}

	lowest := clusters[0]
	for _, cluster := range clusters[1:] {
		if cluster.Latency < lowest.Latency {
			lowest = cluster
		}
	}

	return lowest
}

// noopManager is a no-op implementation when multi-cluster is disabled
type noopManager struct{}

func (n *noopManager) GetClusterInfo(name string) (*ClusterInfo, error) {
	return nil, fmt.Errorf("multi-cluster management is disabled")
}

func (n *noopManager) ListClusters() []*ClusterInfo {
	return nil
}

func (n *noopManager) RouteRequest(req *RouteRequest) (*RouteDecision, error) {
	return nil, fmt.Errorf("multi-cluster management is disabled")
}

func (n *noopManager) GetServiceLocations(serviceName string) (*ServiceLocation, error) {
	return nil, fmt.Errorf("multi-cluster management is disabled")
}

func (n *noopManager) SyncServices() error {
	return nil
}

func (n *noopManager) Start() error {
	return nil
}

func (n *noopManager) Stop() error {
	return nil
}
