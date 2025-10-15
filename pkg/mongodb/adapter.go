package mongodb

import (
	"context"
	"fmt"
	"time"

	"odin/pkg/config"

	"github.com/sirupsen/logrus"
)

// ServiceAdapter adapts between config.ServiceConfig and mongodb.ServiceDocument
type ServiceAdapter struct {
	repo   Repository
	logger *logrus.Logger
}

// NewServiceAdapter creates a new service adapter
func NewServiceAdapter(repo Repository, logger *logrus.Logger) *ServiceAdapter {
	return &ServiceAdapter{
		repo:   repo,
		logger: logger,
	}
}

// LoadServices loads services from MongoDB
func (a *ServiceAdapter) LoadServices(ctx context.Context) ([]config.ServiceConfig, error) {
	enabled := true
	docs, err := a.repo.ListServices(ctx, &enabled)
	if err != nil {
		return nil, fmt.Errorf("failed to load services from MongoDB: %w", err)
	}

	services := make([]config.ServiceConfig, 0, len(docs))
	for _, doc := range docs {
		svc := a.documentToConfig(doc)
		services = append(services, svc)
	}

	a.logger.WithField("count", len(services)).Info("Loaded services from MongoDB")
	return services, nil
}

// SaveService saves a service to MongoDB
func (a *ServiceAdapter) SaveService(ctx context.Context, svc *config.ServiceConfig) error {
	doc := a.configToDocument(svc)

	// Check if service exists
	existing, err := a.repo.GetServiceByName(ctx, svc.Name)
	if err == nil && existing != nil {
		// Update existing service
		return a.repo.UpdateService(ctx, existing.ID, doc)
	}

	// Create new service
	return a.repo.CreateService(ctx, doc)
}

// GetService retrieves a service by name
func (a *ServiceAdapter) GetService(ctx context.Context, name string) (*config.ServiceConfig, error) {
	doc, err := a.repo.GetServiceByName(ctx, name)
	if err != nil {
		return nil, err
	}

	svc := a.documentToConfig(doc)
	return &svc, nil
}

// DeleteService deletes a service by name
func (a *ServiceAdapter) DeleteService(ctx context.Context, name string) error {
	doc, err := a.repo.GetServiceByName(ctx, name)
	if err != nil {
		return err
	}

	return a.repo.DeleteService(ctx, doc.ID)
}

// UpdateService updates an existing service
func (a *ServiceAdapter) UpdateService(ctx context.Context, name string, svc *config.ServiceConfig) error {
	doc, err := a.repo.GetServiceByName(ctx, name)
	if err != nil {
		return err
	}

	updated := a.configToDocument(svc)
	updated.ID = doc.ID
	updated.CreatedAt = doc.CreatedAt

	return a.repo.UpdateService(ctx, doc.ID, updated)
}

// documentToConfig converts MongoDB document to service config
func (a *ServiceAdapter) documentToConfig(doc *ServiceDocument) config.ServiceConfig {
	svc := config.ServiceConfig{
		Name:           doc.Name,
		BasePath:       doc.BasePath,
		Targets:        doc.Targets,
		StripBasePath:  doc.StripBasePath,
		Timeout:        time.Duration(doc.Timeout) * time.Millisecond,
		RetryCount:     doc.RetryCount,
		RetryDelay:     time.Duration(doc.RetryDelay) * time.Millisecond,
		Authentication: doc.Authentication,
		LoadBalancing:  doc.LoadBalancing,
		Headers:        doc.Headers,
		Protocol:       doc.Protocol,
	}

	// Convert transform config
	if doc.Transform != nil {
		svc.Transform = config.TransformConfig{
			Request:  convertTransformRules(doc.Transform["request"]),
			Response: convertTransformRules(doc.Transform["response"]),
		}
	}

	// Convert aggregation config
	if doc.Aggregation != nil {
		svc.Aggregation = &config.AggregationConfig{
			Dependencies: convertDependencies(doc.Aggregation),
		}
	}

	// Convert health check config
	if doc.HealthCheck != nil {
		hc := doc.HealthCheck
		svc.HealthCheck = &config.HealthCheckConfig{
			Enabled:            hc["enabled"].(bool),
			Interval:           time.Duration(hc["interval"].(int64)) * time.Millisecond,
			Timeout:            time.Duration(hc["timeout"].(int64)) * time.Millisecond,
			UnhealthyThreshold: int(hc["unhealthyThreshold"].(int64)),
			HealthyThreshold:   int(hc["healthyThreshold"].(int64)),
			InsecureSkipVerify: hc["insecureSkipVerify"].(bool),
		}
		if expectedStatus, ok := hc["expectedStatus"].([]interface{}); ok {
			statuses := make([]int, 0, len(expectedStatus))
			for _, s := range expectedStatus {
				statuses = append(statuses, int(s.(int64)))
			}
			svc.HealthCheck.ExpectedStatus = statuses
		}
	}

	return svc
}

// configToDocument converts service config to MongoDB document
func (a *ServiceAdapter) configToDocument(svc *config.ServiceConfig) *ServiceDocument {
	doc := &ServiceDocument{
		Name:           svc.Name,
		BasePath:       svc.BasePath,
		Targets:        svc.Targets,
		StripBasePath:  svc.StripBasePath,
		Timeout:        int64(svc.Timeout / time.Millisecond),
		RetryCount:     svc.RetryCount,
		RetryDelay:     int64(svc.RetryDelay / time.Millisecond),
		Authentication: svc.Authentication,
		LoadBalancing:  svc.LoadBalancing,
		Headers:        svc.Headers,
		Protocol:       svc.Protocol,
		Enabled:        true,
		Metadata:       make(map[string]string),
	}

	// Convert transform config
	if len(svc.Transform.Request) > 0 || len(svc.Transform.Response) > 0 {
		doc.Transform = map[string]interface{}{
			"request":  transformRulesToMap(svc.Transform.Request),
			"response": transformRulesToMap(svc.Transform.Response),
		}
	}

	// Convert aggregation config
	if svc.Aggregation != nil {
		doc.Aggregation = dependenciesToMap(svc.Aggregation.Dependencies)
	}

	// Convert health check config
	if svc.HealthCheck != nil {
		doc.HealthCheck = map[string]interface{}{
			"enabled":            svc.HealthCheck.Enabled,
			"interval":           int64(svc.HealthCheck.Interval / time.Millisecond),
			"timeout":            int64(svc.HealthCheck.Timeout / time.Millisecond),
			"unhealthyThreshold": svc.HealthCheck.UnhealthyThreshold,
			"healthyThreshold":   svc.HealthCheck.HealthyThreshold,
			"insecureSkipVerify": svc.HealthCheck.InsecureSkipVerify,
		}
		if len(svc.HealthCheck.ExpectedStatus) > 0 {
			statuses := make([]int, len(svc.HealthCheck.ExpectedStatus))
			copy(statuses, svc.HealthCheck.ExpectedStatus)
			doc.HealthCheck["expectedStatus"] = statuses
		}
	}

	return doc
}

// Helper functions for type conversion

func convertTransformRules(data interface{}) []config.TransformRule {
	if data == nil {
		return nil
	}

	rules, ok := data.([]interface{})
	if !ok {
		return nil
	}

	result := make([]config.TransformRule, 0, len(rules))
	for _, r := range rules {
		ruleMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		rule := config.TransformRule{
			From:    getString(ruleMap, "from"),
			To:      getString(ruleMap, "to"),
			Default: getString(ruleMap, "default"),
		}
		result = append(result, rule)
	}

	return result
}

func convertDependencies(data map[string]interface{}) []config.DependencyConfig {
	if data == nil {
		return nil
	}

	deps, ok := data["dependencies"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]config.DependencyConfig, 0, len(deps))
	for _, d := range deps {
		depMap, ok := d.(map[string]interface{})
		if !ok {
			continue
		}

		dep := config.DependencyConfig{
			Service: getString(depMap, "service"),
			Path:    getString(depMap, "path"),
		}

		if paramMap, ok := depMap["parameterMapping"].([]interface{}); ok {
			dep.ParameterMapping = convertMappings(paramMap)
		}

		if resultMap, ok := depMap["resultMapping"].([]interface{}); ok {
			dep.ResultMapping = convertMappings(resultMap)
		}

		result = append(result, dep)
	}

	return result
}

func convertMappings(data []interface{}) []config.MappingConfig {
	if data == nil {
		return nil
	}

	result := make([]config.MappingConfig, 0, len(data))
	for _, m := range data {
		mMap, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		mapping := config.MappingConfig{
			From: getString(mMap, "from"),
			To:   getString(mMap, "to"),
		}
		result = append(result, mapping)
	}

	return result
}

func transformRulesToMap(rules []config.TransformRule) []interface{} {
	if rules == nil {
		return nil
	}

	result := make([]interface{}, 0, len(rules))
	for _, rule := range rules {
		result = append(result, map[string]interface{}{
			"from":    rule.From,
			"to":      rule.To,
			"default": rule.Default,
		})
	}

	return result
}

func dependenciesToMap(deps []config.DependencyConfig) map[string]interface{} {
	if deps == nil {
		return nil
	}

	depsArray := make([]interface{}, 0, len(deps))
	for _, dep := range deps {
		depMap := map[string]interface{}{
			"service": dep.Service,
			"path":    dep.Path,
		}

		if len(dep.ParameterMapping) > 0 {
			depMap["parameterMapping"] = mappingsToArray(dep.ParameterMapping)
		}

		if len(dep.ResultMapping) > 0 {
			depMap["resultMapping"] = mappingsToArray(dep.ResultMapping)
		}

		depsArray = append(depsArray, depMap)
	}

	return map[string]interface{}{
		"dependencies": depsArray,
	}
}

func mappingsToArray(mappings []config.MappingConfig) []interface{} {
	result := make([]interface{}, 0, len(mappings))
	for _, m := range mappings {
		result = append(result, map[string]interface{}{
			"from": m.From,
			"to":   m.To,
		})
	}
	return result
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ConfigManager manages gateway configuration in MongoDB
type ConfigManager struct {
	repo   Repository
	logger *logrus.Logger
}

// NewConfigManager creates a new config manager
func NewConfigManager(repo Repository, logger *logrus.Logger) *ConfigManager {
	return &ConfigManager{
		repo:   repo,
		logger: logger,
	}
}

// SaveConfig saves the current configuration to MongoDB
func (m *ConfigManager) SaveConfig(ctx context.Context, cfg *config.Config, version string) error {
	doc := &ConfigDocument{
		Version: version,
		Active:  true,
		Config:  configToMap(cfg),
	}

	if err := m.repo.SaveConfig(ctx, doc); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	m.logger.WithField("version", version).Info("Configuration saved to MongoDB")
	return nil
}

// GetActiveConfig retrieves the active configuration
func (m *ConfigManager) GetActiveConfig(ctx context.Context) (*config.Config, error) {
	doc, err := m.repo.GetActiveConfig(ctx)
	if err != nil {
		return nil, err
	}

	cfg := mapToConfig(doc.Config)
	m.logger.WithField("version", doc.Version).Info("Loaded active configuration from MongoDB")
	return cfg, nil
}

// Helper function to convert config to map (simplified - you may need to expand this)
func configToMap(cfg *config.Config) map[string]interface{} {
	// This is a simplified version - in production, you'd want full serialization
	return map[string]interface{}{
		"server":  cfg.Server,
		"logging": cfg.Logging,
		// Add other fields as needed
	}
}

// Helper function to convert map to config (simplified)
func mapToConfig(data map[string]interface{}) *config.Config {
	// This is a simplified version - in production, you'd want full deserialization
	cfg := &config.Config{}
	// Populate fields from data
	return cfg
}
