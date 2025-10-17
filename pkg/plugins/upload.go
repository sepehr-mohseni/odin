package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PluginInfo represents metadata for an uploaded plugin
type PluginInfo struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	Name          string                 `bson:"name" json:"name"`
	Version       string                 `bson:"version" json:"version"`
	Description   string                 `bson:"description" json:"description"`
	FileID        primitive.ObjectID     `bson:"file_id" json:"file_id"`
	Filename      string                 `bson:"filename" json:"filename"`
	FileSize      int64                  `bson:"file_size" json:"file_size"`
	FileHash      string                 `bson:"file_hash" json:"file_hash"`
	Enabled       bool                   `bson:"enabled" json:"enabled"`
	Config        map[string]interface{} `bson:"config" json:"config"`
	GoVersion     string                 `bson:"go_version" json:"go_version"`
	GoOS          string                 `bson:"go_os" json:"go_os"`
	GoArch        string                 `bson:"go_arch" json:"go_arch"`
	Author        string                 `bson:"author" json:"author"`
	UploadedBy    string                 `bson:"uploaded_by" json:"uploaded_by"`
	UploadedAt    time.Time              `bson:"uploaded_at" json:"uploaded_at"`
	UpdatedAt     time.Time              `bson:"updated_at" json:"updated_at"`
	LastEnabledAt *time.Time             `bson:"last_enabled_at,omitempty" json:"last_enabled_at,omitempty"`
	UsageCount    int                    `bson:"usage_count" json:"usage_count"`
	Status        string                 `bson:"status" json:"status"` // active, disabled, error
	ErrorMessage  string                 `bson:"error_message,omitempty" json:"error_message,omitempty"`
	Metadata      PluginMetadata         `bson:"metadata" json:"metadata"`
}

// PluginMetadata contains additional plugin information
type PluginMetadata struct {
	Tags     []string `bson:"tags" json:"tags"`
	Routes   []string `bson:"routes" json:"routes"`
	Priority int      `bson:"priority" json:"priority"`
	Phase    string   `bson:"phase" json:"phase"` // pre-routing, post-routing, pre-response
}

// PluginUploader handles plugin file uploads and storage
type PluginUploader struct {
	db         *mongo.Database
	bucket     *gridfs.Bucket
	collection *mongo.Collection
	logger     *logrus.Logger
	uploadDir  string // Temporary upload directory
	maxSize    int64  // Max file size in bytes (default: 50MB)
}

// NewPluginUploader creates a new plugin uploader
func NewPluginUploader(db *mongo.Database, logger *logrus.Logger) (*PluginUploader, error) {
	// Create GridFS bucket
	bucket, err := gridfs.NewBucket(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create GridFS bucket: %w", err)
	}

	// Get plugins collection
	collection := db.Collection("plugins")

	// Create indexes
	ctx := context.Background()
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "name", Value: 1}, {Key: "version", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "enabled", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "uploaded_at", Value: -1}},
		},
	}

	_, err = collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		logger.WithError(err).Warn("Failed to create indexes for plugins collection")
	}

	// Create temporary upload directory
	uploadDir := filepath.Join(os.TempDir(), "odin-plugin-uploads")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	return &PluginUploader{
		db:         db,
		bucket:     bucket,
		collection: collection,
		logger:     logger,
		uploadDir:  uploadDir,
		maxSize:    50 * 1024 * 1024, // 50MB default
	}, nil
}

// UploadPlugin handles plugin file upload
func (pu *PluginUploader) UploadPlugin(c echo.Context) error {
	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(400, map[string]interface{}{
			"error": "No file provided",
		})
	}

	// Get form fields
	name := c.FormValue("name")
	version := c.FormValue("version")
	description := c.FormValue("description")
	author := c.FormValue("author")
	priority := c.FormValue("priority")
	phase := c.FormValue("phase")

	// Get config (JSON string)
	configStr := c.FormValue("config")
	config := make(map[string]interface{})
	if configStr != "" {
		// Parse JSON config (simplified for now)
		config["raw"] = configStr
	}

	// Get routes (comma-separated)
	routesStr := c.FormValue("routes")
	routes := []string{"/*"}
	if routesStr != "" {
		routes = strings.Split(routesStr, ",")
		for i := range routes {
			routes[i] = strings.TrimSpace(routes[i])
		}
	}

	// Validate input
	if name == "" || version == "" {
		return c.JSON(400, map[string]interface{}{
			"error": "Name and version are required",
		})
	}

	// Validate file
	if err := pu.validateFile(file); err != nil {
		pu.logger.WithError(err).WithFields(logrus.Fields{
			"filename": file.Filename,
			"size":     file.Size,
		}).Warn("Plugin file validation failed")
		return c.JSON(400, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Check if plugin already exists
	exists, err := pu.pluginExists(name, version)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to check plugin existence")
		return c.JSON(500, map[string]interface{}{
			"error": "Database error",
		})
	}
	if exists {
		return c.JSON(409, map[string]interface{}{
			"error": fmt.Sprintf("Plugin %s version %s already exists", name, version),
		})
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		pu.logger.WithError(err).Error("Failed to open uploaded file")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to read file",
		})
	}
	defer src.Close()

	// Save to temporary location for validation
	tempPath := filepath.Join(pu.uploadDir, fmt.Sprintf("%s-%s-%d.so", name, version, time.Now().Unix()))
	dst, err := os.Create(tempPath)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to create temporary file")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to save file",
		})
	}
	defer dst.Close()
	defer os.Remove(tempPath) // Clean up temp file

	// Copy file and calculate hash
	hash := sha256.New()
	multiWriter := io.MultiWriter(dst, hash)
	written, err := io.Copy(multiWriter, src)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to copy file")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to save file",
		})
	}

	fileHash := hex.EncodeToString(hash.Sum(nil))

	// Validate plugin binary
	if err := pu.validatePluginBinary(tempPath, name); err != nil {
		pu.logger.WithError(err).WithField("file", tempPath).Warn("Plugin binary validation failed")
		return c.JSON(400, map[string]interface{}{
			"error": fmt.Sprintf("Plugin validation failed: %s", err.Error()),
		})
	}

	// Reset file pointer for GridFS upload
	if _, err := dst.Seek(0, 0); err != nil {
		pu.logger.WithError(err).Error("Failed to reset file pointer")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to process file",
		})
	}

	// Upload to GridFS
	ctx := context.Background()
	uploadStream, err := pu.bucket.OpenUploadStream(file.Filename)
	if err != nil {
		pu.logger.WithError(err).Error("Failed to open GridFS upload stream")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to store file",
		})
	}
	defer uploadStream.Close()

	if _, err := io.Copy(uploadStream, dst); err != nil {
		pu.logger.WithError(err).Error("Failed to upload to GridFS")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to store file",
		})
	}

	fileID := uploadStream.FileID.(primitive.ObjectID)

	// Create plugin info
	priorityInt := 100
	if priority != "" {
		fmt.Sscanf(priority, "%d", &priorityInt)
	}

	if phase == "" {
		phase = "pre-routing"
	}

	// Extract plugin metadata from binary
	goVersion, goOS, goArch, err := ExtractPluginMetadata(tempPath)
	if err != nil {
		pu.logger.WithError(err).Warn("Failed to extract plugin metadata, using defaults")
	}

	pluginInfo := PluginInfo{
		Name:        name,
		Version:     version,
		Description: description,
		FileID:      fileID,
		Filename:    file.Filename,
		FileSize:    written,
		FileHash:    fileHash,
		Enabled:     false, // Disabled by default
		Config:      config,
		GoVersion:   goVersion,
		GoOS:        goOS,
		GoArch:      goArch,
		Author:      author,
		UploadedBy:  "admin", // TODO: Get from JWT context
		UploadedAt:  time.Now(),
		UpdatedAt:   time.Now(),
		Status:      "uploaded",
		Metadata: PluginMetadata{
			Tags:     []string{},
			Routes:   routes,
			Priority: priorityInt,
			Phase:    phase,
		},
	}

	// Save metadata to MongoDB
	result, err := pu.collection.InsertOne(ctx, pluginInfo)
	if err != nil {
		// Clean up GridFS file
		pu.bucket.Delete(fileID)
		pu.logger.WithError(err).Error("Failed to save plugin metadata")
		return c.JSON(500, map[string]interface{}{
			"error": "Failed to save plugin",
		})
	}

	pluginInfo.ID = result.InsertedID.(primitive.ObjectID)

	pu.logger.WithFields(logrus.Fields{
		"plugin_id": pluginInfo.ID.Hex(),
		"name":      name,
		"version":   version,
		"size":      written,
	}).Info("Plugin uploaded successfully")

	return c.JSON(201, map[string]interface{}{
		"success": true,
		"plugin":  pluginInfo,
	})
}

// validateFile performs basic file validation
func (pu *PluginUploader) validateFile(file *multipart.FileHeader) error {
	// Check file extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".so") {
		return fmt.Errorf("only .so files are allowed")
	}

	// Check file size
	if file.Size > pu.maxSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d MB", pu.maxSize/(1024*1024))
	}

	if file.Size == 0 {
		return fmt.Errorf("file is empty")
	}

	return nil
}

// pluginExists checks if a plugin with the same name and version already exists
func (pu *PluginUploader) pluginExists(name, version string) (bool, error) {
	ctx := context.Background()
	filter := bson.M{"name": name, "version": version}

	count, err := pu.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// validatePluginBinary validates the plugin binary file
func (pu *PluginUploader) validatePluginBinary(filePath, expectedName string) error {
	// Use the comprehensive validator
	validator := NewPluginValidator()
	return validator.ValidatePlugin(filePath, expectedName)
}
