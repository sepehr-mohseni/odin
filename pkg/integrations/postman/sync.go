package postman

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"odin/pkg/config"

	"github.com/sirupsen/logrus"
)

// SyncDirection indicates sync direction
type SyncDirection string

const (
	SyncDirectionImport      SyncDirection = "import"        // Postman -> Odin
	SyncDirectionExport      SyncDirection = "export"        // Odin -> Postman
	SyncDirectionBidirection SyncDirection = "bidirectional" // Both ways
)

// SyncStatus represents the status of a sync operation
type SyncStatus string

const (
	SyncStatusPending    SyncStatus = "pending"
	SyncStatusInProgress SyncStatus = "in_progress"
	SyncStatusCompleted  SyncStatus = "completed"
	SyncStatusFailed     SyncStatus = "failed"
)

// SyncRecord tracks a sync operation
type SyncRecord struct {
	ID               string        `json:"id" bson:"_id,omitempty"`
	CollectionID     string        `json:"collection_id" bson:"collection_id"`
	CollectionName   string        `json:"collection_name" bson:"collection_name"`
	ServiceName      string        `json:"service_name" bson:"service_name"`
	Direction        SyncDirection `json:"direction" bson:"direction"`
	Status           SyncStatus    `json:"status" bson:"status"`
	StartedAt        time.Time     `json:"started_at" bson:"started_at"`
	CompletedAt      *time.Time    `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	Error            string        `json:"error,omitempty" bson:"error,omitempty"`
	ItemsImported    int           `json:"items_imported" bson:"items_imported"`
	ItemsExported    int           `json:"items_exported" bson:"items_exported"`
	SourceHash       string        `json:"source_hash" bson:"source_hash"`
	DestinationHash  string        `json:"destination_hash" bson:"destination_hash"`
	ChangesDetected  bool          `json:"changes_detected" bson:"changes_detected"`
	ConflictsFound   int           `json:"conflicts_found" bson:"conflicts_found"`
	ConflictStrategy string        `json:"conflict_strategy" bson:"conflict_strategy"`
}

// SyncEngine handles synchronization between Postman and Odin
type SyncEngine struct {
	client       *Client
	transformer  *Transformer
	config       *IntegrationConfig
	logger       *logrus.Logger
	repository   SyncRepository
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
	lastSyncTime map[string]time.Time // collectionID -> last sync time
}

// SyncRepository interface for persisting sync records
type SyncRepository interface {
	SaveSyncRecord(ctx context.Context, record *SyncRecord) error
	GetSyncRecord(ctx context.Context, id string) (*SyncRecord, error)
	GetSyncHistory(ctx context.Context, collectionID string, limit int) ([]*SyncRecord, error)
	GetLastSync(ctx context.Context, collectionID string, direction SyncDirection) (*SyncRecord, error)
	UpdateSyncStatus(ctx context.Context, id string, status SyncStatus, err error) error
}

// NewSyncEngine creates a new sync engine
func NewSyncEngine(client *Client, config *IntegrationConfig, repository SyncRepository, logger *logrus.Logger) *SyncEngine {
	return &SyncEngine{
		client:       client,
		transformer:  NewTransformer(),
		config:       config,
		logger:       logger,
		repository:   repository,
		stopChan:     make(chan struct{}),
		lastSyncTime: make(map[string]time.Time),
	}
}

// Start begins the background sync process
func (s *SyncEngine) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("sync engine already running")
	}
	s.running = true
	s.mu.Unlock()

	if !s.config.AutoSync {
		s.logger.Info("Auto-sync disabled, sync engine will not start background job")
		return nil
	}

	interval, err := time.ParseDuration(s.config.SyncInterval)
	if err != nil {
		return fmt.Errorf("invalid sync interval: %w", err)
	}

	s.logger.WithField("interval", interval).Info("Starting sync engine")

	s.wg.Add(1)
	go s.syncLoop(ctx, interval)

	return nil
}

// Stop stops the background sync process
func (s *SyncEngine) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("sync engine not running")
	}
	s.running = false
	s.mu.Unlock()

	s.logger.Info("Stopping sync engine")
	close(s.stopChan)
	s.wg.Wait()

	return nil
}

// syncLoop runs the periodic sync
func (s *SyncEngine) syncLoop(ctx context.Context, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial sync
	s.logger.Info("Running initial sync")
	if err := s.SyncAll(ctx); err != nil {
		s.logger.WithError(err).Error("Initial sync failed")
	}

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Context cancelled, stopping sync loop")
			return
		case <-s.stopChan:
			s.logger.Info("Stop signal received, stopping sync loop")
			return
		case <-ticker.C:
			s.logger.Info("Running scheduled sync")
			if err := s.SyncAll(ctx); err != nil {
				s.logger.WithError(err).Error("Scheduled sync failed")
			}
		}
	}
}

// SyncAll syncs all configured collections
func (s *SyncEngine) SyncAll(ctx context.Context) error {
	s.logger.Info("Starting sync for all configured collections")

	var errors []error
	for _, mapping := range s.config.Mappings {
		if mapping.AutoSync {
			if err := s.SyncCollection(ctx, mapping.PostmanCollectionID, mapping.OdinServiceName, SyncDirectionBidirection); err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"collection_id": mapping.PostmanCollectionID,
					"service_name":  mapping.OdinServiceName,
				}).Error("Failed to sync collection")
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("sync completed with %d errors", len(errors))
	}

	return nil
}

// SyncCollection syncs a specific collection
func (s *SyncEngine) SyncCollection(ctx context.Context, collectionID, serviceName string, direction SyncDirection) error {
	logger := s.logger.WithFields(logrus.Fields{
		"collection_id": collectionID,
		"service_name":  serviceName,
		"direction":     direction,
	})

	logger.Info("Starting collection sync")

	// Create sync record
	record := &SyncRecord{
		CollectionID:     collectionID,
		ServiceName:      serviceName,
		Direction:        direction,
		Status:           SyncStatusInProgress,
		StartedAt:        time.Now(),
		ConflictStrategy: "postman_wins", // Default strategy
	}

	// Save initial record
	if err := s.repository.SaveSyncRecord(ctx, record); err != nil {
		return fmt.Errorf("failed to save sync record: %w", err)
	}

	// Check if changes exist
	changesDetected, err := s.DetectChanges(ctx, collectionID)
	if err != nil {
		logger.WithError(err).Warn("Failed to detect changes, proceeding with sync anyway")
		changesDetected = true // Assume changes to be safe
	}

	record.ChangesDetected = changesDetected

	if !changesDetected {
		logger.Info("No changes detected, skipping sync")
		record.Status = SyncStatusCompleted
		now := time.Now()
		record.CompletedAt = &now
		_ = s.repository.SaveSyncRecord(ctx, record)
		return nil
	}

	// Perform sync based on direction
	var syncErr error
	switch direction {
	case SyncDirectionImport:
		syncErr = s.importFromPostman(ctx, collectionID, serviceName, record)
	case SyncDirectionExport:
		syncErr = s.exportToPostman(ctx, collectionID, serviceName, record)
	case SyncDirectionBidirection:
		// For bidirectional, import first then check conflicts
		syncErr = s.importFromPostman(ctx, collectionID, serviceName, record)
		// Could add conflict detection and resolution here
	default:
		syncErr = fmt.Errorf("unknown sync direction: %s", direction)
	}

	// Update record status
	now := time.Now()
	record.CompletedAt = &now
	if syncErr != nil {
		record.Status = SyncStatusFailed
		record.Error = syncErr.Error()
		logger.WithError(syncErr).Error("Sync failed")
	} else {
		record.Status = SyncStatusCompleted
		s.updateLastSyncTime(collectionID)
		logger.Info("Sync completed successfully")
	}

	// Save final record
	if err := s.repository.SaveSyncRecord(ctx, record); err != nil {
		logger.WithError(err).Error("Failed to save final sync record")
	}

	return syncErr
}

// importFromPostman imports collection from Postman to Odin
func (s *SyncEngine) importFromPostman(ctx context.Context, collectionID, serviceName string, record *SyncRecord) error {
	// Fetch collection from Postman
	collection, err := s.client.GetCollection(ctx, collectionID)
	if err != nil {
		return fmt.Errorf("failed to fetch collection: %w", err)
	}

	record.CollectionName = collection.Info.Name

	// Transform to Odin service
	serviceConfig, err := s.transformer.PostmanToOdinService(collection)
	if err != nil {
		return fmt.Errorf("failed to transform collection: %w", err)
	}

	// Override name if specified
	if serviceName != "" {
		serviceConfig.Name = serviceName
	}

	// Calculate hash
	hash, err := s.calculateHash(collection)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to calculate hash")
	} else {
		record.SourceHash = hash
	}

	// Here you would save the service config to Odin's configuration
	// For now, we just log it as this would require integration with config management
	s.logger.WithFields(logrus.Fields{
		"service_name": serviceConfig.Name,
		"base_path":    serviceConfig.BasePath,
		"targets":      serviceConfig.Targets,
	}).Info("Service configuration ready for import")

	record.ItemsImported = len(collection.Item)

	return nil
}

// exportToPostman exports Odin service to Postman
func (s *SyncEngine) exportToPostman(ctx context.Context, collectionID, serviceName string, record *SyncRecord) error {
	// This would fetch the service config from Odin
	// For now, we'll create a placeholder
	serviceConfig := &config.ServiceConfig{
		Name:     serviceName,
		BasePath: "/api/v1/" + serviceName,
	}

	// Transform to Postman collection
	collection, err := s.transformer.OdinServiceToPostman(serviceConfig)
	if err != nil {
		return fmt.Errorf("failed to transform service: %w", err)
	}

	record.CollectionName = collection.Info.Name

	// Update or create collection in Postman
	if collectionID != "" {
		// Update existing
		_, err = s.client.UpdateCollection(ctx, collectionID, collection)
		if err != nil {
			return fmt.Errorf("failed to update collection: %w", err)
		}
		record.ItemsExported = len(collection.Item)
	} else {
		// Create new
		created, err := s.client.CreateCollection(ctx, collection)
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
		record.CollectionID = created.Info.ID
		record.ItemsExported = len(created.Item)
	}

	return nil
}

// DetectChanges checks if a collection has changed since last sync
func (s *SyncEngine) DetectChanges(ctx context.Context, collectionID string) (bool, error) {
	// Get last sync record
	lastSync, err := s.repository.GetLastSync(ctx, collectionID, SyncDirectionImport)
	if err != nil || lastSync == nil {
		// No previous sync, assume changes
		return true, nil
	}

	// Fetch current collection
	collection, err := s.client.GetCollection(ctx, collectionID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch collection: %w", err)
	}

	// Calculate current hash
	currentHash, err := s.calculateHash(collection)
	if err != nil {
		return false, fmt.Errorf("failed to calculate hash: %w", err)
	}

	// Compare with last sync hash
	changed := currentHash != lastSync.SourceHash

	s.logger.WithFields(logrus.Fields{
		"collection_id": collectionID,
		"current_hash":  currentHash,
		"last_hash":     lastSync.SourceHash,
		"changed":       changed,
	}).Debug("Change detection result")

	return changed, nil
}

// calculateHash calculates MD5 hash of collection content
func (s *SyncEngine) calculateHash(collection *PostmanCollection) (string, error) {
	// Serialize collection to JSON
	data, err := json.Marshal(collection)
	if err != nil {
		return "", err
	}

	// Calculate MD5 hash
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash), nil
}

// updateLastSyncTime updates the last sync time for a collection
func (s *SyncEngine) updateLastSyncTime(collectionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSyncTime[collectionID] = time.Now()
}

// GetLastSyncTime returns the last sync time for a collection
func (s *SyncEngine) GetLastSyncTime(collectionID string) time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastSyncTime[collectionID]
}

// IsRunning returns whether the sync engine is running
func (s *SyncEngine) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetSyncHistory retrieves sync history for a collection
func (s *SyncEngine) GetSyncHistory(ctx context.Context, collectionID string, limit int) ([]*SyncRecord, error) {
	return s.repository.GetSyncHistory(ctx, collectionID, limit)
}

// ForceSync forces an immediate sync of a collection
func (s *SyncEngine) ForceSync(ctx context.Context, collectionID, serviceName string, direction SyncDirection) error {
	s.logger.WithFields(logrus.Fields{
		"collection_id": collectionID,
		"service_name":  serviceName,
		"direction":     direction,
	}).Info("Force sync requested")

	return s.SyncCollection(ctx, collectionID, serviceName, direction)
}
