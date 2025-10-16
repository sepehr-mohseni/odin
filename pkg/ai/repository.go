package ai

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Repository defines the interface for AI data persistence
type Repository interface {
	// Traffic patterns
	SaveTrafficPattern(ctx context.Context, pattern *TrafficPattern) error
	GetTrafficPatterns(ctx context.Context, serviceName, endpoint string, startTime, endTime time.Time) ([]*TrafficPattern, error)

	// Baselines
	SaveBaseline(ctx context.Context, baseline *Baseline) error
	GetBaseline(ctx context.Context, serviceName, endpoint, timeWindow string) (*Baseline, error)
	ListBaselines(ctx context.Context, serviceName string) ([]*Baseline, error)

	// Anomalies
	SaveAnomaly(ctx context.Context, anomaly *Anomaly) error
	GetAnomaly(ctx context.Context, id string) (*Anomaly, error)
	ListAnomalies(ctx context.Context, filter AnomalyFilter) ([]*Anomaly, error)
	UpdateAnomaly(ctx context.Context, anomaly *Anomaly) error
	MarkAnomalyResolved(ctx context.Context, id string) error
	MarkAnomalyFalsePositive(ctx context.Context, id string) error

	// Alerts
	SaveAlert(ctx context.Context, alert *Alert) error
	GetPendingAlerts(ctx context.Context) ([]*Alert, error)
	MarkAlertSent(ctx context.Context, id string) error

	// Cleanup
	CleanupOldData(ctx context.Context, retentionDays int) error
}

// AnomalyFilter represents filter criteria for anomaly queries
type AnomalyFilter struct {
	ServiceName string
	Endpoint    string
	AnomalyType string
	Severity    string
	Resolved    *bool
	StartTime   *time.Time
	EndTime     *time.Time
	Limit       int
}

// MongoRepository implements Repository using MongoDB
type MongoRepository struct {
	db *mongo.Database
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository(db *mongo.Database) (*MongoRepository, error) {
	repo := &MongoRepository{db: db}

	// Create indexes
	if err := repo.createIndexes(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

// createIndexes creates necessary indexes for AI collections
func (r *MongoRepository) createIndexes(ctx context.Context) error {
	// Traffic patterns indexes
	patternsCol := r.db.Collection("traffic_patterns")
	_, err := patternsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "service_name", Value: 1},
				{Key: "endpoint", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys:    bson.D{{Key: "timestamp", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(7 * 24 * 60 * 60), // 7 days TTL
		},
	})
	if err != nil {
		return err
	}

	// Baselines indexes
	baselinesCol := r.db.Collection("ai_baselines")
	_, err = baselinesCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "service_name", Value: 1},
				{Key: "endpoint", Value: 1},
				{Key: "time_window", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "last_updated", Value: -1}},
		},
	})
	if err != nil {
		return err
	}

	// Anomalies indexes
	anomaliesCol := r.db.Collection("ai_anomalies")
	_, err = anomaliesCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "service_name", Value: 1},
				{Key: "timestamp", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "anomaly_type", Value: 1},
				{Key: "severity", Value: 1},
			},
		},
		{
			Keys: bson.D{{Key: "resolved", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "timestamp", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(90 * 24 * 60 * 60), // 90 days TTL
		},
	})
	if err != nil {
		return err
	}

	// Alerts indexes
	alertsCol := r.db.Collection("ai_alerts")
	_, err = alertsCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "sent", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "timestamp", Value: -1}},
		},
		{
			Keys:    bson.D{{Key: "timestamp", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(30 * 24 * 60 * 60), // 30 days TTL
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// SaveTrafficPattern saves a traffic pattern to the database
func (r *MongoRepository) SaveTrafficPattern(ctx context.Context, pattern *TrafficPattern) error {
	col := r.db.Collection("traffic_patterns")
	_, err := col.InsertOne(ctx, pattern)
	return err
}

// GetTrafficPatterns retrieves traffic patterns for analysis
func (r *MongoRepository) GetTrafficPatterns(ctx context.Context, serviceName, endpoint string, startTime, endTime time.Time) ([]*TrafficPattern, error) {
	col := r.db.Collection("traffic_patterns")

	filter := bson.M{
		"service_name": serviceName,
		"timestamp": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
	}

	if endpoint != "" {
		filter["endpoint"] = endpoint
	}

	cursor, err := col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var patterns []*TrafficPattern
	if err := cursor.All(ctx, &patterns); err != nil {
		return nil, err
	}

	return patterns, nil
}

// SaveBaseline saves a baseline to the database
func (r *MongoRepository) SaveBaseline(ctx context.Context, baseline *Baseline) error {
	col := r.db.Collection("ai_baselines")

	filter := bson.M{
		"service_name": baseline.ServiceName,
		"endpoint":     baseline.Endpoint,
		"time_window":  baseline.TimeWindow,
	}

	update := bson.M{"$set": baseline}
	opts := options.Update().SetUpsert(true)

	_, err := col.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetBaseline retrieves a baseline from the database
func (r *MongoRepository) GetBaseline(ctx context.Context, serviceName, endpoint, timeWindow string) (*Baseline, error) {
	col := r.db.Collection("ai_baselines")

	filter := bson.M{
		"service_name": serviceName,
		"endpoint":     endpoint,
		"time_window":  timeWindow,
	}

	var baseline Baseline
	if err := col.FindOne(ctx, filter).Decode(&baseline); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &baseline, nil
}

// ListBaselines lists all baselines for a service
func (r *MongoRepository) ListBaselines(ctx context.Context, serviceName string) ([]*Baseline, error) {
	col := r.db.Collection("ai_baselines")

	filter := bson.M{"service_name": serviceName}
	cursor, err := col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var baselines []*Baseline
	if err := cursor.All(ctx, &baselines); err != nil {
		return nil, err
	}

	return baselines, nil
}

// SaveAnomaly saves an anomaly to the database
func (r *MongoRepository) SaveAnomaly(ctx context.Context, anomaly *Anomaly) error {
	col := r.db.Collection("ai_anomalies")
	_, err := col.InsertOne(ctx, anomaly)
	return err
}

// GetAnomaly retrieves an anomaly by ID
func (r *MongoRepository) GetAnomaly(ctx context.Context, id string) (*Anomaly, error) {
	col := r.db.Collection("ai_anomalies")

	var anomaly Anomaly
	if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&anomaly); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	return &anomaly, nil
}

// ListAnomalies lists anomalies based on filter criteria
func (r *MongoRepository) ListAnomalies(ctx context.Context, filter AnomalyFilter) ([]*Anomaly, error) {
	col := r.db.Collection("ai_anomalies")

	query := bson.M{}

	if filter.ServiceName != "" {
		query["service_name"] = filter.ServiceName
	}
	if filter.Endpoint != "" {
		query["endpoint"] = filter.Endpoint
	}
	if filter.AnomalyType != "" {
		query["anomaly_type"] = filter.AnomalyType
	}
	if filter.Severity != "" {
		query["severity"] = filter.Severity
	}
	if filter.Resolved != nil {
		query["resolved"] = *filter.Resolved
	}
	if filter.StartTime != nil || filter.EndTime != nil {
		timeQuery := bson.M{}
		if filter.StartTime != nil {
			timeQuery["$gte"] = *filter.StartTime
		}
		if filter.EndTime != nil {
			timeQuery["$lte"] = *filter.EndTime
		}
		query["timestamp"] = timeQuery
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}})
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}

	cursor, err := col.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var anomalies []*Anomaly
	if err := cursor.All(ctx, &anomalies); err != nil {
		return nil, err
	}

	return anomalies, nil
}

// UpdateAnomaly updates an anomaly
func (r *MongoRepository) UpdateAnomaly(ctx context.Context, anomaly *Anomaly) error {
	col := r.db.Collection("ai_anomalies")

	filter := bson.M{"_id": anomaly.ID}
	update := bson.M{"$set": anomaly}

	_, err := col.UpdateOne(ctx, filter, update)
	return err
}

// MarkAnomalyResolved marks an anomaly as resolved
func (r *MongoRepository) MarkAnomalyResolved(ctx context.Context, id string) error {
	col := r.db.Collection("ai_anomalies")

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"resolved":    true,
			"resolved_at": now,
		},
	}

	_, err := col.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// MarkAnomalyFalsePositive marks an anomaly as a false positive
func (r *MongoRepository) MarkAnomalyFalsePositive(ctx context.Context, id string) error {
	col := r.db.Collection("ai_anomalies")

	update := bson.M{
		"$set": bson.M{
			"false_positive": true,
		},
	}

	_, err := col.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// SaveAlert saves an alert to the database
func (r *MongoRepository) SaveAlert(ctx context.Context, alert *Alert) error {
	col := r.db.Collection("ai_alerts")
	_, err := col.InsertOne(ctx, alert)
	return err
}

// GetPendingAlerts retrieves unsent alerts
func (r *MongoRepository) GetPendingAlerts(ctx context.Context) ([]*Alert, error) {
	col := r.db.Collection("ai_alerts")

	cursor, err := col.Find(ctx, bson.M{"sent": false}, options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var alerts []*Alert
	if err := cursor.All(ctx, &alerts); err != nil {
		return nil, err
	}

	return alerts, nil
}

// MarkAlertSent marks an alert as sent
func (r *MongoRepository) MarkAlertSent(ctx context.Context, id string) error {
	col := r.db.Collection("ai_alerts")

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"sent":    true,
			"sent_at": now,
		},
	}

	_, err := col.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// CleanupOldData removes old traffic patterns beyond retention period
func (r *MongoRepository) CleanupOldData(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Traffic patterns are handled by TTL index
	// This is for manual cleanup if needed
	patternsCol := r.db.Collection("traffic_patterns")
	_, err := patternsCol.DeleteMany(ctx, bson.M{
		"timestamp": bson.M{"$lt": cutoff},
	})

	return err
}
