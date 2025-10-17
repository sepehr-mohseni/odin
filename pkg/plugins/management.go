package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ListPlugins returns all plugins with optional filters
func (pu *PluginUploader) ListPlugins(c echo.Context) error {
	ctx := context.Background()

	// Build filter
	filter := bson.M{}

	// Filter by enabled status
	if enabled := c.QueryParam("enabled"); enabled != "" {
		if enabled == "true" {
			filter["enabled"] = true
		} else if enabled == "false" {
			filter["enabled"] = false
		}
	}

	// Filter by name
	if name := c.QueryParam("name"); name != "" {
		filter["name"] = bson.M{"$regex": name, "$options": "i"}
	}

	// Filter by status
	if status := c.QueryParam("status"); status != "" {
		filter["status"] = status
	}

	// Query options
	opts := options.Find().SetSort(bson.D{{Key: "uploaded_at", Value: -1}})

	cursor, err := pu.collection.Find(ctx, filter, opts)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to list plugins")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to retrieve plugins",
		})
	}
	defer cursor.Close(ctx)

	var plugins []PluginInfo
	if err := cursor.All(ctx, &plugins); err != nil {
		pu.logger.WithError(err).Error("Failed to decode plugins")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to retrieve plugins",
		})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"plugins": plugins,
		"count":   len(plugins),
	})
}

// GetPlugin returns a single plugin by ID
func (pu *PluginUploader) GetPlugin(c echo.Context) error {
	ctx := context.Background()

	// Get plugin ID from path
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"error": "Invalid plugin ID",
		})
	}

	var plugin PluginInfo
	err = pu.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&plugin)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(404, map[string]interface{}{
				"error": "Plugin not found",
			})
		}
		pu.logger.WithError(err).Error("Failed to get plugin")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to retrieve plugin",
		})
	}

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"plugin":  plugin,
	})
}

// EnablePlugin enables a plugin and loads it into the middleware chain
func (pu *PluginUploader) EnablePlugin(c echo.Context, pm *PluginManager) error {
	ctx := context.Background()

	// Get plugin ID
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"error": "Invalid plugin ID",
		})
	}

	// Get plugin info
	var pluginInfo PluginInfo
	err = pu.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&pluginInfo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(404, map[string]interface{}{
				"error": "Plugin not found",
			})
		}
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to retrieve plugin",
		})
	}

	// Check if already enabled
	if pluginInfo.Enabled {
		return c.JSON(200, map[string]interface{}{
			"success": true,
			"message": "Plugin is already enabled",
			"plugin":  pluginInfo,
		})
	}

	// Download plugin from GridFS to temporary location
	tempPath := filepath.Join(pu.uploadDir, fmt.Sprintf("%s-%s.so", pluginInfo.Name, pluginInfo.Version))
	if err := pu.downloadPluginFromGridFS(ctx, pluginInfo.FileID, tempPath); err != nil {
		pu.logger.WithError(err).Error("Failed to download plugin from GridFS")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to load plugin file",
		})
	}
	defer os.Remove(tempPath) // Clean up after loading

	// Load the plugin
	p, err := plugin.Open(tempPath)
	if err != nil {
		pu.logger.WithError(err).WithField("path", tempPath).Error("Failed to open plugin")

		// Update plugin status to error
		pu.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{
			"$set": bson.M{
				"status":        "error",
				"error_message": err.Error(),
				"updated_at":    time.Now(),
			},
		})

		return c.JSON(500, map[string]interface{}{
			"error": fmt.Sprintf("Failed to load plugin: %s", err.Error()),
		})
	}

	// Look up the New function
	newSymbol, err := p.Lookup("New")
	if err != nil {
		pu.logger.WithError(err).Error("Plugin missing 'New' function")
		return c.JSON(400, map[string]interface{}{
			"error": "Plugin must export a 'New' function",
		})
	}

	// Call the New function to create middleware instance
	newFunc, ok := newSymbol.(func(map[string]interface{}) (Middleware, error))
	if !ok {
		return c.JSON(400, map[string]interface{}{
			"error": "Plugin 'New' function has incorrect signature",
		})
	}

	middleware, err := newFunc(pluginInfo.Config)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to initialize plugin")
		return c.JSON(500, map[string]interface{}{
			"error": fmt.Sprintf("Failed to initialize plugin: %s", err.Error()),
		})
	}

	// Register with middleware chain
	err = pm.RegisterMiddleware(
		pluginInfo.Name,
		middleware,
		pluginInfo.Metadata.Priority,
		pluginInfo.Metadata.Routes,
		pluginInfo.Metadata.Phase,
	)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to register middleware")
		return c.JSON(500, map[string]interface{}{
			"error": fmt.Sprintf("Failed to register middleware: %s", err.Error()),
		})
	}

	// Update plugin status
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"enabled":         true,
			"status":          "active",
			"last_enabled_at": now,
			"updated_at":      now,
			"error_message":   "",
		},
		"$inc": bson.M{
			"usage_count": 1,
		},
	}

	_, err = pu.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		pu.logger.WithError(err).Warn("Failed to update plugin status")
	}

	pu.logger.WithField("plugin", pluginInfo.Name).Info("Plugin enabled successfully")

	// Get updated plugin info
	pu.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&pluginInfo)

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Plugin enabled successfully",
		"plugin":  pluginInfo,
	})
}

// DisablePlugin disables a plugin and removes it from the middleware chain
func (pu *PluginUploader) DisablePlugin(c echo.Context, pm *PluginManager) error {
	ctx := context.Background()

	// Get plugin ID
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"error": "Invalid plugin ID",
		})
	}

	// Get plugin info
	var pluginInfo PluginInfo
	err = pu.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&pluginInfo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(404, map[string]interface{}{
				"error": "Plugin not found",
			})
		}
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to retrieve plugin",
		})
	}

	// Check if already disabled
	if !pluginInfo.Enabled {
		return c.JSON(200, map[string]interface{}{
			"success": true,
			"message": "Plugin is already disabled",
			"plugin":  pluginInfo,
		})
	}

	// Unregister from middleware chain
	if err := pm.UnregisterMiddleware(pluginInfo.Name); err != nil {
		pu.logger.WithError(err).Warn("Failed to unregister middleware")
	}

	// Update plugin status
	update := bson.M{
		"$set": bson.M{
			"enabled":    false,
			"status":     "disabled",
			"updated_at": time.Now(),
		},
	}

	_, err = pu.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to update plugin status")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to disable plugin",
		})
	}

	pu.logger.WithField("plugin", pluginInfo.Name).Info("Plugin disabled successfully")

	// Get updated plugin info
	pu.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&pluginInfo)

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Plugin disabled successfully",
		"plugin":  pluginInfo,
	})
}

// DeletePlugin deletes a plugin and its associated file
func (pu *PluginUploader) DeletePlugin(c echo.Context, pm *PluginManager) error {
	ctx := context.Background()

	// Get plugin ID
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"error": "Invalid plugin ID",
		})
	}

	// Get plugin info
	var pluginInfo PluginInfo
	err = pu.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&pluginInfo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(404, map[string]interface{}{
				"error": "Plugin not found",
			})
		}
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to retrieve plugin",
		})
	}

	// If plugin is enabled, disable it first
	if pluginInfo.Enabled {
		if err := pm.UnregisterMiddleware(pluginInfo.Name); err != nil {
			pu.logger.WithError(err).Warn("Failed to unregister middleware before deletion")
		}
	}

	// Delete the file from GridFS
	if err := pu.bucket.Delete(pluginInfo.FileID); err != nil {
		pu.logger.WithError(err).Warn("Failed to delete file from GridFS")
	}

	// Delete plugin metadata
	_, err = pu.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		pu.logger.WithError(err).Error("Failed to delete plugin")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to delete plugin",
		})
	}

	pu.logger.WithField("plugin", pluginInfo.Name).Info("Plugin deleted successfully")

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Plugin deleted successfully",
	})
}

// UpdatePluginConfig updates the configuration of a plugin
func (pu *PluginUploader) UpdatePluginConfig(c echo.Context) error {
	ctx := context.Background()

	// Get plugin ID
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"error": "Invalid plugin ID",
		})
	}

	// Parse request body
	var req struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Update plugin config
	update := bson.M{
		"$set": bson.M{
			"config":     req.Config,
			"updated_at": time.Now(),
		},
	}

	result, err := pu.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to update plugin config")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to update plugin configuration",
		})
	}

	if result.MatchedCount == 0 {
		return c.JSON(404, map[string]interface{}{
			"error": "Plugin not found",
		})
	}

	// Get updated plugin
	var pluginInfo PluginInfo
	pu.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&pluginInfo)

	pu.logger.WithField("plugin", pluginInfo.Name).Info("Plugin configuration updated")

	return c.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
		"plugin":  pluginInfo,
	})
}

// downloadPluginFromGridFS downloads a plugin file from GridFS to a local path
func (pu *PluginUploader) downloadPluginFromGridFS(ctx context.Context, fileID primitive.ObjectID, destPath string) error {
	// Open download stream
	downloadStream, err := pu.bucket.OpenDownloadStream(fileID)
	if err != nil {
		return fmt.Errorf("failed to open download stream: %w", err)
	}
	defer downloadStream.Close()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy data
	_, err = io.Copy(destFile, downloadStream)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

// GetPluginStats returns statistics about plugins
func (pu *PluginUploader) GetPluginStats(c echo.Context) error {
	ctx := context.Background()

	// Count total plugins
	totalCount, _ := pu.collection.CountDocuments(ctx, bson.M{})

	// Count enabled plugins
	enabledCount, _ := pu.collection.CountDocuments(ctx, bson.M{"enabled": true})

	// Count disabled plugins
	disabledCount, _ := pu.collection.CountDocuments(ctx, bson.M{"enabled": false})

	// Count plugins by status
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, _ := pu.collection.Aggregate(ctx, pipeline)
	defer cursor.Close(ctx)

	statusCounts := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err == nil {
			statusCounts[result.ID] = result.Count
		}
	}

	return c.JSON(200, map[string]interface{}{
		"success":       true,
		"total_plugins": totalCount,
		"enabled":       enabledCount,
		"disabled":      disabledCount,
		"status_counts": statusCounts,
	})
}
