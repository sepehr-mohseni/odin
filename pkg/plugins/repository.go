package plugins

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PluginRecord represents a plugin stored in MongoDB
type PluginRecord struct {
	Name        string                 `bson:"name" json:"name"`
	Version     string                 `bson:"version" json:"version"`
	Description string                 `bson:"description" json:"description"`
	Author      string                 `bson:"author" json:"author"`
	BinaryPath  string                 `bson:"binaryPath" json:"binaryPath"`
	Config      map[string]interface{} `bson:"config" json:"config"`
	Hooks       []string               `bson:"hooks" json:"hooks"`
	Enabled     bool                   `bson:"enabled" json:"enabled"`
	AppliedTo   []string               `bson:"appliedTo" json:"appliedTo"` // Routes where plugin is applied
	CreatedAt   time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `bson:"updatedAt" json:"updatedAt"`
}

// PluginRepository handles plugin database operations
type PluginRepository struct {
	collection *mongo.Collection
}

// NewPluginRepository creates a new plugin repository
func NewPluginRepository(db *mongo.Database) *PluginRepository {
	collection := db.Collection("plugins")

	// Create indexes
	ctx := context.Background()
	_, _ = collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
		},
	})

	return &PluginRepository{
		collection: collection,
	}
}

// SavePlugin stores a plugin in the database
func (r *PluginRepository) SavePlugin(ctx context.Context, plugin *PluginRecord) error {
	plugin.CreatedAt = time.Now()
	plugin.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, plugin)
	if err != nil {
		return fmt.Errorf("failed to save plugin: %w", err)
	}

	return nil
}

// GetPlugin retrieves a plugin by name
func (r *PluginRepository) GetPlugin(ctx context.Context, name string) (*PluginRecord, error) {
	var plugin PluginRecord

	err := r.collection.FindOne(ctx, bson.M{"name": name}).Decode(&plugin)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("plugin %s not found", name)
		}
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	return &plugin, nil
}

// ListPlugins returns all plugins matching the filter
func (r *PluginRepository) ListPlugins(ctx context.Context, filter bson.M) ([]*PluginRecord, error) {
	if filter == nil {
		filter = bson.M{}
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	defer cursor.Close(ctx)

	var plugins []*PluginRecord
	if err := cursor.All(ctx, &plugins); err != nil {
		return nil, fmt.Errorf("failed to decode plugins: %w", err)
	}

	return plugins, nil
}

// UpdatePlugin updates an existing plugin
func (r *PluginRepository) UpdatePlugin(ctx context.Context, plugin *PluginRecord) error {
	plugin.UpdatedAt = time.Now()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"name": plugin.Name},
		bson.M{"$set": plugin},
	)
	if err != nil {
		return fmt.Errorf("failed to update plugin: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("plugin %s not found", plugin.Name)
	}

	return nil
}

// DeletePlugin removes a plugin from the database
func (r *PluginRepository) DeletePlugin(ctx context.Context, name string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"name": name})
	if err != nil {
		return fmt.Errorf("failed to delete plugin: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("plugin %s not found", name)
	}

	return nil
}

// EnablePlugin enables a plugin
func (r *PluginRepository) EnablePlugin(ctx context.Context, name string) error {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"name": name},
		bson.M{
			"$set": bson.M{
				"enabled":   true,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to enable plugin: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("plugin %s not found", name)
	}

	return nil
}

// DisablePlugin disables a plugin
func (r *PluginRepository) DisablePlugin(ctx context.Context, name string) error {
	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"name": name},
		bson.M{
			"$set": bson.M{
				"enabled":   false,
				"updatedAt": time.Now(),
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to disable plugin: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("plugin %s not found", name)
	}

	return nil
}

// GetEnabledPlugins returns all enabled plugins
func (r *PluginRepository) GetEnabledPlugins(ctx context.Context) ([]*PluginRecord, error) {
	return r.ListPlugins(ctx, bson.M{"enabled": true})
}
