package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Alert operations

func (r *repository) CreateAlert(ctx context.Context, alert *AlertDocument) error {
	alert.Triggered = time.Now()

	col := r.database.Collection(AlertsCollection)
	_, err := col.InsertOne(ctx, alert)
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	r.logger.WithField("service", alert.ServiceName).Warn("Alert triggered")
	return nil
}

func (r *repository) GetAlert(ctx context.Context, id string) (*AlertDocument, error) {
	col := r.database.Collection(AlertsCollection)

	var alert AlertDocument
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&alert)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("alert not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	return &alert, nil
}

func (r *repository) ListAlerts(ctx context.Context, status string) ([]*AlertDocument, error) {
	col := r.database.Collection(AlertsCollection)

	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "triggered", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}
	defer cursor.Close(ctx)

	var alerts []*AlertDocument
	if err := cursor.All(ctx, &alerts); err != nil {
		return nil, fmt.Errorf("failed to decode alerts: %w", err)
	}

	return alerts, nil
}

func (r *repository) UpdateAlert(ctx context.Context, id string, alert *AlertDocument) error {
	col := r.database.Collection(AlertsCollection)
	result, err := col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": alert},
	)
	if err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("alert not found: %s", id)
	}

	return nil
}

func (r *repository) ResolveAlert(ctx context.Context, id string) error {
	col := r.database.Collection(AlertsCollection)
	result, err := col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"status":   "resolved",
				"resolved": time.Now(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("alert not found: %s", id)
	}

	r.logger.WithField("id", id).Info("Alert resolved")
	return nil
}

// Health check operations

func (r *repository) SaveHealthCheck(ctx context.Context, check *HealthCheckDocument) error {
	check.CheckedAt = time.Now()
	// Set TTL to 24 hours from now
	check.TTL = time.Now().Add(24 * time.Hour)

	col := r.database.Collection(HealthChecksCollection)
	_, err := col.InsertOne(ctx, check)
	if err != nil {
		return fmt.Errorf("failed to save health check: %w", err)
	}

	return nil
}

func (r *repository) GetLatestHealthCheck(ctx context.Context, serviceName string) (*HealthCheckDocument, error) {
	col := r.database.Collection(HealthChecksCollection)

	var check HealthCheckDocument
	err := col.FindOne(
		ctx,
		bson.M{"serviceName": serviceName},
		options.FindOne().SetSort(bson.D{{Key: "checkedAt", Value: -1}}),
	).Decode(&check)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no health check found for service: %s", serviceName)
		}
		return nil, fmt.Errorf("failed to get health check: %w", err)
	}

	return &check, nil
}

func (r *repository) QueryHealthChecks(ctx context.Context, serviceName string, start, end time.Time) ([]*HealthCheckDocument, error) {
	col := r.database.Collection(HealthChecksCollection)

	filter := bson.M{
		"serviceName": serviceName,
		"checkedAt": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "checkedAt", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to query health checks: %w", err)
	}
	defer cursor.Close(ctx)

	var checks []*HealthCheckDocument
	if err := cursor.All(ctx, &checks); err != nil {
		return nil, fmt.Errorf("failed to decode health checks: %w", err)
	}

	return checks, nil
}

// Cluster operations

func (r *repository) CreateCluster(ctx context.Context, cluster *ClusterDocument) error {
	cluster.CreatedAt = time.Now()
	cluster.UpdatedAt = time.Now()

	col := r.database.Collection(ClustersCollection)
	_, err := col.InsertOne(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	r.logger.WithField("cluster", cluster.Name).Info("Cluster created in MongoDB")
	return nil
}

func (r *repository) GetCluster(ctx context.Context, id string) (*ClusterDocument, error) {
	col := r.database.Collection(ClustersCollection)

	var cluster ClusterDocument
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&cluster)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("cluster not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	return &cluster, nil
}

func (r *repository) ListClusters(ctx context.Context, enabled *bool) ([]*ClusterDocument, error) {
	col := r.database.Collection(ClustersCollection)

	filter := bson.M{}
	if enabled != nil {
		filter["enabled"] = *enabled
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer cursor.Close(ctx)

	var clusters []*ClusterDocument
	if err := cursor.All(ctx, &clusters); err != nil {
		return nil, fmt.Errorf("failed to decode clusters: %w", err)
	}

	return clusters, nil
}

func (r *repository) UpdateCluster(ctx context.Context, id string, cluster *ClusterDocument) error {
	cluster.UpdatedAt = time.Now()

	col := r.database.Collection(ClustersCollection)
	result, err := col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": cluster},
	)
	if err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("cluster not found: %s", id)
	}

	r.logger.WithField("cluster", cluster.Name).Info("Cluster updated in MongoDB")
	return nil
}

func (r *repository) DeleteCluster(ctx context.Context, id string) error {
	col := r.database.Collection(ClustersCollection)
	result, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("cluster not found: %s", id)
	}

	r.logger.WithField("id", id).Info("Cluster deleted from MongoDB")
	return nil
}

// Plugin operations

func (r *repository) CreatePlugin(ctx context.Context, plugin *PluginDocument) error {
	plugin.CreatedAt = time.Now()
	plugin.UpdatedAt = time.Now()

	col := r.database.Collection(PluginsCollection)
	_, err := col.InsertOne(ctx, plugin)
	if err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}

	r.logger.WithField("plugin", plugin.Name).Info("Plugin created in MongoDB")
	return nil
}

func (r *repository) GetPlugin(ctx context.Context, id string) (*PluginDocument, error) {
	col := r.database.Collection(PluginsCollection)

	var plugin PluginDocument
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&plugin)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("plugin not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	return &plugin, nil
}

func (r *repository) ListPlugins(ctx context.Context, enabled *bool) ([]*PluginDocument, error) {
	col := r.database.Collection(PluginsCollection)

	filter := bson.M{}
	if enabled != nil {
		filter["enabled"] = *enabled
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	defer cursor.Close(ctx)

	var plugins []*PluginDocument
	if err := cursor.All(ctx, &plugins); err != nil {
		return nil, fmt.Errorf("failed to decode plugins: %w", err)
	}

	return plugins, nil
}

func (r *repository) UpdatePlugin(ctx context.Context, id string, plugin *PluginDocument) error {
	plugin.UpdatedAt = time.Now()

	col := r.database.Collection(PluginsCollection)
	result, err := col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": plugin},
	)
	if err != nil {
		return fmt.Errorf("failed to update plugin: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("plugin not found: %s", id)
	}

	r.logger.WithField("plugin", plugin.Name).Info("Plugin updated in MongoDB")
	return nil
}

func (r *repository) DeletePlugin(ctx context.Context, id string) error {
	col := r.database.Collection(PluginsCollection)
	result, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete plugin: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("plugin not found: %s", id)
	}

	r.logger.WithField("id", id).Info("Plugin deleted from MongoDB")
	return nil
}

// User operations

func (r *repository) CreateUser(ctx context.Context, user *UserDocument) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	col := r.database.Collection(UsersCollection)
	_, err := col.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.WithField("username", user.Username).Info("User created in MongoDB")
	return nil
}

func (r *repository) GetUser(ctx context.Context, id string) (*UserDocument, error) {
	col := r.database.Collection(UsersCollection)

	var user UserDocument
	err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *repository) GetUserByUsername(ctx context.Context, username string) (*UserDocument, error) {
	col := r.database.Collection(UsersCollection)

	var user UserDocument
	err := col.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *repository) ListUsers(ctx context.Context) ([]*UserDocument, error) {
	col := r.database.Collection(UsersCollection)

	cursor, err := col.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "username", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*UserDocument
	if err := cursor.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}

func (r *repository) UpdateUser(ctx context.Context, id string, user *UserDocument) error {
	user.UpdatedAt = time.Now()

	col := r.database.Collection(UsersCollection)
	result, err := col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": user},
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	r.logger.WithField("username", user.Username).Info("User updated in MongoDB")
	return nil
}

func (r *repository) DeleteUser(ctx context.Context, id string) error {
	col := r.database.Collection(UsersCollection)
	result, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	r.logger.WithField("id", id).Info("User deleted from MongoDB")
	return nil
}

// API Key operations

func (r *repository) CreateAPIKey(ctx context.Context, key *APIKeyDocument) error {
	key.CreatedAt = time.Now()
	key.LastUsed = time.Now()

	col := r.database.Collection(APIKeysCollection)
	_, err := col.InsertOne(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	r.logger.WithField("name", key.Name).Info("API key created in MongoDB")
	return nil
}

func (r *repository) GetAPIKey(ctx context.Context, key string) (*APIKeyDocument, error) {
	col := r.database.Collection(APIKeysCollection)

	var apiKey APIKeyDocument
	err := col.FindOne(ctx, bson.M{"key": key}).Decode(&apiKey)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Update last used timestamp
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		col.UpdateOne(ctx, bson.M{"key": key}, bson.M{"$set": bson.M{"lastUsed": time.Now()}})
	}()

	return &apiKey, nil
}

func (r *repository) ListAPIKeys(ctx context.Context, userID string) ([]*APIKeyDocument, error) {
	col := r.database.Collection(APIKeysCollection)

	filter := bson.M{}
	if userID != "" {
		filter["userId"] = userID
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer cursor.Close(ctx)

	var keys []*APIKeyDocument
	if err := cursor.All(ctx, &keys); err != nil {
		return nil, fmt.Errorf("failed to decode API keys: %w", err)
	}

	return keys, nil
}

func (r *repository) UpdateAPIKey(ctx context.Context, id string, key *APIKeyDocument) error {
	col := r.database.Collection(APIKeysCollection)
	result, err := col.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": key},
	)
	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("API key not found: %s", id)
	}

	return nil
}

func (r *repository) DeleteAPIKey(ctx context.Context, id string) error {
	col := r.database.Collection(APIKeysCollection)
	result, err := col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("API key not found: %s", id)
	}

	r.logger.WithField("id", id).Info("API key deleted from MongoDB")
	return nil
}

// Rate limit operations

func (r *repository) GetRateLimit(ctx context.Context, key string) (*RateLimitDocument, error) {
	col := r.database.Collection(RateLimitsCollection)

	var limit RateLimitDocument
	err := col.FindOne(ctx, bson.M{"key": key}).Decode(&limit)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("rate limit not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get rate limit: %w", err)
	}

	return &limit, nil
}

func (r *repository) UpdateRateLimit(ctx context.Context, limit *RateLimitDocument) error {
	col := r.database.Collection(RateLimitsCollection)

	_, err := col.UpdateOne(
		ctx,
		bson.M{"key": limit.Key},
		bson.M{
			"$set": limit,
			"$setOnInsert": bson.M{
				"createdAt": time.Now(),
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("failed to update rate limit: %w", err)
	}

	return nil
}

// Cache operations

func (r *repository) GetCache(ctx context.Context, key string) (*CacheDocument, error) {
	col := r.database.Collection(CacheCollection)

	var cache CacheDocument
	err := col.FindOne(ctx, bson.M{"key": key}).Decode(&cache)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("cache not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	// Check if expired
	if time.Now().After(cache.ExpiresAt) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			r.DeleteCache(ctx, key)
		}()
		return nil, fmt.Errorf("cache expired: %s", key)
	}

	return &cache, nil
}

func (r *repository) SetCache(ctx context.Context, cache *CacheDocument) error {
	col := r.database.Collection(CacheCollection)

	_, err := col.UpdateOne(
		ctx,
		bson.M{"key": cache.Key},
		bson.M{
			"$set": cache,
			"$setOnInsert": bson.M{
				"createdAt": time.Now(),
			},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

func (r *repository) DeleteCache(ctx context.Context, key string) error {
	col := r.database.Collection(CacheCollection)
	_, err := col.DeleteOne(ctx, bson.M{"key": key})
	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}

// Audit log operations

func (r *repository) CreateAuditLog(ctx context.Context, log *AuditLogDocument) error {
	log.Timestamp = time.Now()
	// Set TTL to 90 days from now
	log.TTL = time.Now().Add(90 * 24 * time.Hour)

	col := r.database.Collection(AuditLogsCollection)
	_, err := col.InsertOne(ctx, log)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (r *repository) QueryAuditLogs(ctx context.Context, userID string, start, end time.Time) ([]*AuditLogDocument, error) {
	col := r.database.Collection(AuditLogsCollection)

	filter := bson.M{
		"timestamp": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}
	if userID != "" {
		filter["userId"] = userID
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*AuditLogDocument
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, fmt.Errorf("failed to decode audit logs: %w", err)
	}

	return logs, nil
}
