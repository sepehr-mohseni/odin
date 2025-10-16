package postman

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBRepository implements persistence for Postman integration data
type MongoDBRepository struct {
	db                    *mongo.Database
	configCollection      *mongo.Collection
	syncCollection        *mongo.Collection
	testResultsCollection *mongo.Collection
}

// NewMongoDBRepository creates a new MongoDB repository
func NewMongoDBRepository(db *mongo.Database) (*MongoDBRepository, error) {
	repo := &MongoDBRepository{
		db:                    db,
		configCollection:      db.Collection("integration_configs"),
		syncCollection:        db.Collection("integration_syncs"),
		testResultsCollection: db.Collection("test_results"),
	}

	// Create indexes
	if err := repo.createIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return repo, nil
}

// createIndexes creates necessary MongoDB indexes
func (r *MongoDBRepository) createIndexes(ctx context.Context) error {
	// Config indexes
	configIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "provider", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys:    bson.D{{Key: "enabled", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
	}
	if _, err := r.configCollection.Indexes().CreateMany(ctx, configIndexes); err != nil {
		return fmt.Errorf("failed to create config indexes: %w", err)
	}

	// Sync indexes
	syncIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "collection_id", Value: 1}, {Key: "started_at", Value: -1}},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys:    bson.D{{Key: "service_name", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys:    bson.D{{Key: "direction", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
	}
	if _, err := r.syncCollection.Indexes().CreateMany(ctx, syncIndexes); err != nil {
		return fmt.Errorf("failed to create sync indexes: %w", err)
	}

	// Test results indexes
	testIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "collectionId", Value: 1}, {Key: "runAt", Value: -1}},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys:    bson.D{{Key: "serviceName", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetUnique(false),
		},
	}
	if _, err := r.testResultsCollection.Indexes().CreateMany(ctx, testIndexes); err != nil {
		return fmt.Errorf("failed to create test result indexes: %w", err)
	}

	return nil
}

// Config operations

// SaveConfig saves or updates an integration configuration
func (r *MongoDBRepository) SaveConfig(ctx context.Context, config *IntegrationConfig) error {
	config.UpdatedAt = time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}

	filter := bson.M{"provider": "postman"}
	update := bson.M{"$set": config}
	opts := options.Update().SetUpsert(true)

	_, err := r.configCollection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetConfig retrieves the integration configuration
func (r *MongoDBRepository) GetConfig(ctx context.Context) (*IntegrationConfig, error) {
	var config IntegrationConfig
	err := r.configCollection.FindOne(ctx, bson.M{"provider": "postman"}).Decode(&config)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	return &config, nil
}

// DeleteConfig removes the integration configuration
func (r *MongoDBRepository) DeleteConfig(ctx context.Context) error {
	_, err := r.configCollection.DeleteOne(ctx, bson.M{"provider": "postman"})
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	return nil
}

// Sync operations - Implementation of SyncRepository interface

// SaveSyncRecord saves a sync record
func (r *MongoDBRepository) SaveSyncRecord(ctx context.Context, record *SyncRecord) error {
	if record.ID == "" {
		// Generate new ID
		record.ID = primitive.NewObjectID().Hex()
		_, err := r.syncCollection.InsertOne(ctx, record)
		if err != nil {
			return fmt.Errorf("failed to insert sync record: %w", err)
		}
	} else {
		// Update existing
		filter := bson.M{"_id": record.ID}
		update := bson.M{"$set": record}
		_, err := r.syncCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update sync record: %w", err)
		}
	}

	return nil
}

// GetSyncRecord retrieves a sync record by ID
func (r *MongoDBRepository) GetSyncRecord(ctx context.Context, id string) (*SyncRecord, error) {
	var record SyncRecord
	err := r.syncCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&record)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync record: %w", err)
	}

	return &record, nil
}

// GetSyncHistory retrieves sync history for a collection
func (r *MongoDBRepository) GetSyncHistory(ctx context.Context, collectionID string, limit int) ([]*SyncRecord, error) {
	if limit <= 0 {
		limit = 50
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "started_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.syncCollection.Find(ctx, bson.M{"collection_id": collectionID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query sync history: %w", err)
	}
	defer cursor.Close(ctx)

	var records []*SyncRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("failed to decode sync history: %w", err)
	}

	return records, nil
}

// GetLastSync retrieves the last sync record for a collection and direction
func (r *MongoDBRepository) GetLastSync(ctx context.Context, collectionID string, direction SyncDirection) (*SyncRecord, error) {
	filter := bson.M{
		"collection_id": collectionID,
		"direction":     direction,
		"status":        SyncStatusCompleted,
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "started_at", Value: -1}})

	var record SyncRecord
	err := r.syncCollection.FindOne(ctx, filter, opts).Decode(&record)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last sync: %w", err)
	}

	return &record, nil
}

// UpdateSyncStatus updates the status of a sync record
func (r *MongoDBRepository) UpdateSyncStatus(ctx context.Context, id string, status SyncStatus, syncErr error) error {
	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	if syncErr != nil {
		update["$set"].(bson.M)["error"] = syncErr.Error()
	}

	if status == SyncStatusCompleted || status == SyncStatusFailed {
		now := time.Now()
		update["$set"].(bson.M)["completed_at"] = now
	}

	_, err := r.syncCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	return nil
}

// GetAllSyncs retrieves all sync records
func (r *MongoDBRepository) GetAllSyncs(ctx context.Context, limit int) ([]*SyncRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "started_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.syncCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query syncs: %w", err)
	}
	defer cursor.Close(ctx)

	var records []*SyncRecord
	if err := cursor.All(ctx, &records); err != nil {
		return nil, fmt.Errorf("failed to decode syncs: %w", err)
	}

	return records, nil
}

// Test results operations - Implementation of NewmanRepository interface

// SaveTestResult saves a test result
func (r *MongoDBRepository) SaveTestResult(ctx context.Context, result *NewmanResult) error {
	if result.ID == "" {
		// Generate new ID
		result.ID = primitive.NewObjectID().Hex()
		_, err := r.testResultsCollection.InsertOne(ctx, result)
		if err != nil {
			return fmt.Errorf("failed to insert test result: %w", err)
		}
	} else {
		// Update existing
		filter := bson.M{"_id": result.ID}
		update := bson.M{"$set": result}
		_, err := r.testResultsCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update test result: %w", err)
		}
	}

	return nil
}

// GetTestResult retrieves a test result by ID
func (r *MongoDBRepository) GetTestResult(ctx context.Context, id string) (*NewmanResult, error) {
	var result NewmanResult
	err := r.testResultsCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get test result: %w", err)
	}

	return &result, nil
}

// GetTestHistory retrieves test history for a collection
func (r *MongoDBRepository) GetTestHistory(ctx context.Context, collectionID string, limit int) ([]*NewmanResult, error) {
	if limit <= 0 {
		limit = 50
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "runAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.testResultsCollection.Find(ctx, bson.M{"collectionId": collectionID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query test history: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*NewmanResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode test history: %w", err)
	}

	return results, nil
}

// GetLatestTestResult retrieves the latest test result for a collection
func (r *MongoDBRepository) GetLatestTestResult(ctx context.Context, collectionID string) (*NewmanResult, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "runAt", Value: -1}})

	var result NewmanResult
	err := r.testResultsCollection.FindOne(ctx, bson.M{"collectionId": collectionID}, opts).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest test result: %w", err)
	}

	return &result, nil
}

// GetTestResultsByService retrieves test results for a service
func (r *MongoDBRepository) GetTestResultsByService(ctx context.Context, serviceName string, limit int) ([]*NewmanResult, error) {
	if limit <= 0 {
		limit = 50
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "runAt", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.testResultsCollection.Find(ctx, bson.M{"serviceName": serviceName}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query test results by service: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*NewmanResult
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode test results: %w", err)
	}

	return results, nil
}

// GetTestStats retrieves test statistics
func (r *MongoDBRepository) GetTestStats(ctx context.Context, collectionID string) (*TestStats, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "collectionId", Value: collectionID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "totalRuns", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "passedRuns", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{bson.D{{Key: "$eq", Value: bson.A{"$status", "passed"}}}, 1, 0}}}}}},
			{Key: "failedRuns", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{bson.D{{Key: "$eq", Value: bson.A{"$status", "failed"}}}, 1, 0}}}}}},
			{Key: "avgDuration", Value: bson.D{{Key: "$avg", Value: "$duration"}}},
			{Key: "lastRun", Value: bson.D{{Key: "$max", Value: "$runAt"}}},
		}}},
	}

	cursor, err := r.testResultsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate test stats: %w", err)
	}
	defer cursor.Close(ctx)

	var results []struct {
		TotalRuns   int       `bson:"totalRuns"`
		PassedRuns  int       `bson:"passedRuns"`
		FailedRuns  int       `bson:"failedRuns"`
		AvgDuration float64   `bson:"avgDuration"`
		LastRun     time.Time `bson:"lastRun"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode test stats: %w", err)
	}

	if len(results) == 0 {
		return &TestStats{}, nil
	}

	stats := &TestStats{
		TotalRuns:    results[0].TotalRuns,
		PassedRuns:   results[0].PassedRuns,
		FailedRuns:   results[0].FailedRuns,
		AvgDuration:  int64(results[0].AvgDuration),
		LastRun:      results[0].LastRun,
		CollectionID: collectionID,
		SuccessRate:  float64(results[0].PassedRuns) / float64(results[0].TotalRuns) * 100,
	}

	return stats, nil
}

// TestStats represents test execution statistics
type TestStats struct {
	CollectionID string    `json:"collectionId"`
	TotalRuns    int       `json:"totalRuns"`
	PassedRuns   int       `json:"passedRuns"`
	FailedRuns   int       `json:"failedRuns"`
	AvgDuration  int64     `json:"avgDuration"`
	LastRun      time.Time `json:"lastRun"`
	SuccessRate  float64   `json:"successRate"`
}

// DeleteOldTestResults deletes test results older than the specified duration
func (r *MongoDBRepository) DeleteOldTestResults(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	filter := bson.M{"runAt": bson.M{"$lt": cutoff}}

	result, err := r.testResultsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old test results: %w", err)
	}

	return result.DeletedCount, nil
}

// DeleteOldSyncRecords deletes sync records older than the specified duration
func (r *MongoDBRepository) DeleteOldSyncRecords(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	filter := bson.M{"started_at": bson.M{"$lt": cutoff}}

	result, err := r.syncCollection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old sync records: %w", err)
	}

	return result.DeletedCount, nil
}
