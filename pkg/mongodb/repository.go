package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// repository implements the Repository interface
type repository struct {
	client   *mongo.Client
	database *mongo.Database
	config   *Config
	logger   *logrus.Logger
}

// NewRepository creates a new MongoDB repository
func NewRepository(config *Config, logger *logrus.Logger) (Repository, error) {
	if !config.Enabled {
		logger.Info("MongoDB is disabled")
		return &noopRepository{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	// Create client options
	clientOpts := options.Client().ApplyURI(config.URI)

	// Set connection pool options
	if config.MaxPoolSize > 0 {
		clientOpts.SetMaxPoolSize(uint64(config.MaxPoolSize))
	}
	if config.MinPoolSize > 0 {
		clientOpts.SetMinPoolSize(uint64(config.MinPoolSize))
	}

	// Set timeouts
	if config.ConnectTimeout > 0 {
		clientOpts.SetConnectTimeout(config.ConnectTimeout)
	}

	// Set authentication
	if config.Auth.Username != "" {
		authDB := config.Auth.AuthDB
		if authDB == "" {
			authDB = "admin"
		}
		credential := options.Credential{
			Username:   config.Auth.Username,
			Password:   config.Auth.Password,
			AuthSource: authDB,
		}
		clientOpts.SetAuth(credential)
	}

	// Set TLS configuration
	if config.TLS.Enabled {
		tlsConfig, err := createTLSConfig(&config.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}
		clientOpts.SetTLSConfig(tlsConfig)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(config.Database)

	repo := &repository{
		client:   client,
		database: database,
		config:   config,
		logger:   logger,
	}

	// Create indexes
	if err := repo.createIndexes(ctx); err != nil {
		logger.WithError(err).Warn("Failed to create indexes")
	}

	logger.WithFields(logrus.Fields{
		"database": config.Database,
		"uri":      maskURI(config.URI),
	}).Info("Connected to MongoDB")

	return repo, nil
}

// createTLSConfig creates TLS configuration for MongoDB
func createTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{}

	// Load CA certificate
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// createIndexes creates necessary indexes
func (r *repository) createIndexes(ctx context.Context) error {
	// Services indexes
	servicesCol := r.database.Collection(ServicesCollection)
	_, err := servicesCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "enabled", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create services indexes: %w", err)
	}

	// Metrics indexes with TTL
	metricsCol := r.database.Collection(MetricsCollection)
	_, err = metricsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "name", Value: 1}, {Key: "timestamp", Value: -1}}},
		{Keys: bson.D{{Key: "labels", Value: 1}}},
		{Keys: bson.D{{Key: "ttl", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return fmt.Errorf("failed to create metrics indexes: %w", err)
	}

	// Traces indexes with TTL
	tracesCol := r.database.Collection(TracesCollection)
	_, err = tracesCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "traceId", Value: 1}}},
		{Keys: bson.D{{Key: "serviceName", Value: 1}, {Key: "startTime", Value: -1}}},
		{Keys: bson.D{{Key: "ttl", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return fmt.Errorf("failed to create traces indexes: %w", err)
	}

	// Alerts indexes
	alertsCol := r.database.Collection(AlertsCollection)
	_, err = alertsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "serviceName", Value: 1}}},
		{Keys: bson.D{{Key: "triggered", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create alerts indexes: %w", err)
	}

	// Health checks indexes with TTL
	healthCol := r.database.Collection(HealthChecksCollection)
	_, err = healthCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "serviceName", Value: 1}, {Key: "checkedAt", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "ttl", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return fmt.Errorf("failed to create health checks indexes: %w", err)
	}

	// Users indexes
	usersCol := r.database.Collection(UsersCollection)
	_, err = usersCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		return fmt.Errorf("failed to create users indexes: %w", err)
	}

	// API keys indexes
	keysCol := r.database.Collection(APIKeysCollection)
	_, err = keysCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		{Keys: bson.D{{Key: "enabled", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create API keys indexes: %w", err)
	}

	// Rate limits indexes with TTL
	rateLimitCol := r.database.Collection(RateLimitsCollection)
	_, err = rateLimitCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "key", Value: 1}}},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return fmt.Errorf("failed to create rate limits indexes: %w", err)
	}

	// Cache indexes with TTL
	cacheCol := r.database.Collection(CacheCollection)
	_, err = cacheCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "key", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return fmt.Errorf("failed to create cache indexes: %w", err)
	}

	// Audit logs indexes with TTL
	auditCol := r.database.Collection(AuditLogsCollection)
	_, err = auditCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "timestamp", Value: -1}}},
		{Keys: bson.D{{Key: "action", Value: 1}}},
		{Keys: bson.D{{Key: "ttl", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
	if err != nil {
		return fmt.Errorf("failed to create audit logs indexes: %w", err)
	}

	return nil
}

// Service operations

func (r *repository) CreateService(ctx context.Context, service *ServiceDocument) error {
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()

	col := r.database.Collection(ServicesCollection)
	_, err := col.InsertOne(ctx, service)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	r.logger.WithField("service", service.Name).Info("Service created in MongoDB")
	return nil
}

func (r *repository) GetService(ctx context.Context, id string) (*ServiceDocument, error) {
	col := r.database.Collection(ServicesCollection)

	var service ServiceDocument
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&service)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("service not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return &service, nil
}

func (r *repository) GetServiceByName(ctx context.Context, name string) (*ServiceDocument, error) {
	col := r.database.Collection(ServicesCollection)

	var service ServiceDocument
	err := col.FindOne(ctx, bson.M{"name": name}).Decode(&service)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("service not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return &service, nil
}

func (r *repository) ListServices(ctx context.Context, enabled *bool) ([]*ServiceDocument, error) {
	col := r.database.Collection(ServicesCollection)

	filter := bson.M{}
	if enabled != nil {
		filter["enabled"] = *enabled
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	defer cursor.Close(ctx)

	var services []*ServiceDocument
	if err := cursor.All(ctx, &services); err != nil {
		return nil, fmt.Errorf("failed to decode services: %w", err)
	}

	return services, nil
}

func (r *repository) UpdateService(ctx context.Context, id string, service *ServiceDocument) error {
	service.UpdatedAt = time.Now()

	col := r.database.Collection(ServicesCollection)
	result, err := col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": service},
	)
	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("service not found: %s", id)
	}

	r.logger.WithField("service", service.Name).Info("Service updated in MongoDB")
	return nil
}

func (r *repository) DeleteService(ctx context.Context, id string) error {
	col := r.database.Collection(ServicesCollection)
	result, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("service not found: %s", id)
	}

	r.logger.WithField("id", id).Info("Service deleted from MongoDB")
	return nil
}

// Config operations

func (r *repository) SaveConfig(ctx context.Context, config *ConfigDocument) error {
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	col := r.database.Collection(ConfigCollection)

	// Deactivate all other configs if this one is active
	if config.Active {
		_, err := col.UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"active": false}})
		if err != nil {
			return fmt.Errorf("failed to deactivate old configs: %w", err)
		}
	}

	_, err := col.InsertOne(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	r.logger.WithField("version", config.Version).Info("Config saved to MongoDB")
	return nil
}

func (r *repository) GetActiveConfig(ctx context.Context) (*ConfigDocument, error) {
	col := r.database.Collection(ConfigCollection)

	var config ConfigDocument
	err := col.FindOne(ctx, bson.M{"active": true}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no active config found")
		}
		return nil, fmt.Errorf("failed to get active config: %w", err)
	}

	return &config, nil
}

func (r *repository) GetConfigByVersion(ctx context.Context, version string) (*ConfigDocument, error) {
	col := r.database.Collection(ConfigCollection)

	var config ConfigDocument
	err := col.FindOne(ctx, bson.M{"version": version}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("config not found: %s", version)
		}
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return &config, nil
}

func (r *repository) ListConfigs(ctx context.Context, limit int) ([]*ConfigDocument, error) {
	col := r.database.Collection(ConfigCollection)

	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}
	defer cursor.Close(ctx)

	var configs []*ConfigDocument
	if err := cursor.All(ctx, &configs); err != nil {
		return nil, fmt.Errorf("failed to decode configs: %w", err)
	}

	return configs, nil
}

// Metrics operations

func (r *repository) SaveMetric(ctx context.Context, metric *MetricDocument) error {
	metric.Timestamp = time.Now()
	// Set TTL to 30 days from now
	metric.TTL = time.Now().Add(30 * 24 * time.Hour)

	col := r.database.Collection(MetricsCollection)
	_, err := col.InsertOne(ctx, metric)
	if err != nil {
		return fmt.Errorf("failed to save metric: %w", err)
	}

	return nil
}

func (r *repository) QueryMetrics(ctx context.Context, name string, start, end time.Time, labels map[string]string) ([]*MetricDocument, error) {
	col := r.database.Collection(MetricsCollection)

	filter := bson.M{
		"name": name,
		"timestamp": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	if len(labels) > 0 {
		for k, v := range labels {
			filter["labels."+k] = v
		}
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var metrics []*MetricDocument
	if err := cursor.All(ctx, &metrics); err != nil {
		return nil, fmt.Errorf("failed to decode metrics: %w", err)
	}

	return metrics, nil
}

// Trace operations

func (r *repository) SaveTrace(ctx context.Context, trace *TraceDocument) error {
	// Set TTL to 7 days from now
	trace.TTL = time.Now().Add(7 * 24 * time.Hour)

	col := r.database.Collection(TracesCollection)
	_, err := col.InsertOne(ctx, trace)
	if err != nil {
		return fmt.Errorf("failed to save trace: %w", err)
	}

	return nil
}

func (r *repository) GetTrace(ctx context.Context, traceID string) ([]*TraceDocument, error) {
	col := r.database.Collection(TracesCollection)

	cursor, err := col.Find(ctx, bson.M{"traceId": traceID}, options.Find().SetSort(bson.D{{Key: "startTime", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to get trace: %w", err)
	}
	defer cursor.Close(ctx)

	var traces []*TraceDocument
	if err := cursor.All(ctx, &traces); err != nil {
		return nil, fmt.Errorf("failed to decode traces: %w", err)
	}

	return traces, nil
}

func (r *repository) QueryTraces(ctx context.Context, serviceName string, start, end time.Time) ([]*TraceDocument, error) {
	col := r.database.Collection(TracesCollection)

	filter := bson.M{
		"serviceName": serviceName,
		"startTime": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "startTime", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to query traces: %w", err)
	}
	defer cursor.Close(ctx)

	var traces []*TraceDocument
	if err := cursor.All(ctx, &traces); err != nil {
		return nil, fmt.Errorf("failed to decode traces: %w", err)
	}

	return traces, nil
}

// Continue with remaining operations in next part...
// (Alert, Health Check, Cluster, Plugin, User, API Key, Rate Limit, Cache, Audit Log operations)

// Ping checks the MongoDB connection
func (r *repository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx, nil)
}

// Close closes the MongoDB connection
func (r *repository) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}

// GetDatabase returns the MongoDB database instance
func (r *repository) GetDatabase() *mongo.Database {
	return r.database
}

// maskURI masks sensitive information in MongoDB URI
func maskURI(uri string) string {
	// Simple masking - in production, use proper URL parsing
	return "mongodb://***:***@***"
}

// noopRepository is a no-op implementation when MongoDB is disabled
type noopRepository struct{}

func (n *noopRepository) GetDatabase() *mongo.Database {
	return nil
}
func (n *noopRepository) CreateService(ctx context.Context, service *ServiceDocument) error {
	return nil
}
func (n *noopRepository) GetService(ctx context.Context, id string) (*ServiceDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) GetServiceByName(ctx context.Context, name string) (*ServiceDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) ListServices(ctx context.Context, enabled *bool) ([]*ServiceDocument, error) {
	return nil, nil
}
func (n *noopRepository) UpdateService(ctx context.Context, id string, service *ServiceDocument) error {
	return nil
}
func (n *noopRepository) DeleteService(ctx context.Context, id string) error {
	return nil
}
func (n *noopRepository) SaveConfig(ctx context.Context, config *ConfigDocument) error {
	return nil
}
func (n *noopRepository) GetActiveConfig(ctx context.Context) (*ConfigDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) GetConfigByVersion(ctx context.Context, version string) (*ConfigDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) ListConfigs(ctx context.Context, limit int) ([]*ConfigDocument, error) {
	return nil, nil
}
func (n *noopRepository) SaveMetric(ctx context.Context, metric *MetricDocument) error {
	return nil
}
func (n *noopRepository) QueryMetrics(ctx context.Context, name string, start, end time.Time, labels map[string]string) ([]*MetricDocument, error) {
	return nil, nil
}
func (n *noopRepository) SaveTrace(ctx context.Context, trace *TraceDocument) error {
	return nil
}
func (n *noopRepository) GetTrace(ctx context.Context, traceID string) ([]*TraceDocument, error) {
	return nil, nil
}
func (n *noopRepository) QueryTraces(ctx context.Context, serviceName string, start, end time.Time) ([]*TraceDocument, error) {
	return nil, nil
}
func (n *noopRepository) CreateAlert(ctx context.Context, alert *AlertDocument) error {
	return nil
}
func (n *noopRepository) GetAlert(ctx context.Context, id string) (*AlertDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) ListAlerts(ctx context.Context, status string) ([]*AlertDocument, error) {
	return nil, nil
}
func (n *noopRepository) UpdateAlert(ctx context.Context, id string, alert *AlertDocument) error {
	return nil
}
func (n *noopRepository) ResolveAlert(ctx context.Context, id string) error {
	return nil
}
func (n *noopRepository) SaveHealthCheck(ctx context.Context, check *HealthCheckDocument) error {
	return nil
}
func (n *noopRepository) GetLatestHealthCheck(ctx context.Context, serviceName string) (*HealthCheckDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) QueryHealthChecks(ctx context.Context, serviceName string, start, end time.Time) ([]*HealthCheckDocument, error) {
	return nil, nil
}
func (n *noopRepository) CreateCluster(ctx context.Context, cluster *ClusterDocument) error {
	return nil
}
func (n *noopRepository) GetCluster(ctx context.Context, id string) (*ClusterDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) ListClusters(ctx context.Context, enabled *bool) ([]*ClusterDocument, error) {
	return nil, nil
}
func (n *noopRepository) UpdateCluster(ctx context.Context, id string, cluster *ClusterDocument) error {
	return nil
}
func (n *noopRepository) DeleteCluster(ctx context.Context, id string) error {
	return nil
}
func (n *noopRepository) CreatePlugin(ctx context.Context, plugin *PluginDocument) error {
	return nil
}
func (n *noopRepository) GetPlugin(ctx context.Context, id string) (*PluginDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) ListPlugins(ctx context.Context, enabled *bool) ([]*PluginDocument, error) {
	return nil, nil
}
func (n *noopRepository) UpdatePlugin(ctx context.Context, id string, plugin *PluginDocument) error {
	return nil
}
func (n *noopRepository) DeletePlugin(ctx context.Context, id string) error {
	return nil
}
func (n *noopRepository) CreateUser(ctx context.Context, user *UserDocument) error {
	return nil
}
func (n *noopRepository) GetUser(ctx context.Context, id string) (*UserDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) GetUserByUsername(ctx context.Context, username string) (*UserDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) ListUsers(ctx context.Context) ([]*UserDocument, error) {
	return nil, nil
}
func (n *noopRepository) UpdateUser(ctx context.Context, id string, user *UserDocument) error {
	return nil
}
func (n *noopRepository) DeleteUser(ctx context.Context, id string) error {
	return nil
}
func (n *noopRepository) CreateAPIKey(ctx context.Context, key *APIKeyDocument) error {
	return nil
}
func (n *noopRepository) GetAPIKey(ctx context.Context, key string) (*APIKeyDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) ListAPIKeys(ctx context.Context, userID string) ([]*APIKeyDocument, error) {
	return nil, nil
}
func (n *noopRepository) UpdateAPIKey(ctx context.Context, id string, key *APIKeyDocument) error {
	return nil
}
func (n *noopRepository) DeleteAPIKey(ctx context.Context, id string) error {
	return nil
}
func (n *noopRepository) GetRateLimit(ctx context.Context, key string) (*RateLimitDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) UpdateRateLimit(ctx context.Context, limit *RateLimitDocument) error {
	return nil
}
func (n *noopRepository) GetCache(ctx context.Context, key string) (*CacheDocument, error) {
	return nil, fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) SetCache(ctx context.Context, cache *CacheDocument) error {
	return nil
}
func (n *noopRepository) DeleteCache(ctx context.Context, key string) error {
	return nil
}
func (n *noopRepository) CreateAuditLog(ctx context.Context, log *AuditLogDocument) error {
	return nil
}
func (n *noopRepository) QueryAuditLogs(ctx context.Context, userID string, start, end time.Time) ([]*AuditLogDocument, error) {
	return nil, nil
}
func (n *noopRepository) Ping(ctx context.Context) error {
	return fmt.Errorf("MongoDB is disabled")
}
func (n *noopRepository) Close(ctx context.Context) error {
	return nil
}
