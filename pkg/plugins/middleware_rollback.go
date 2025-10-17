package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MiddlewareSnapshot represents a snapshot of middleware state for rollback
type MiddlewareSnapshot struct {
	Name      string
	Priority  int
	Routes    []string
	Phase     string
	Config    map[string]interface{}
	Enabled   bool
	Timestamp time.Time
}

// MiddlewareRollback provides rollback capabilities for middleware changes
type MiddlewareRollback struct {
	manager      *PluginManager
	repo         *PluginRepository
	snapshots    map[string][]*MiddlewareSnapshot
	logger       *logrus.Logger
	mu           sync.RWMutex
	maxSnapshots int
}

// NewMiddlewareRollback creates a new middleware rollback manager
func NewMiddlewareRollback(manager *PluginManager, repo *PluginRepository, logger *logrus.Logger) *MiddlewareRollback {
	if logger == nil {
		logger = logrus.New()
	}

	return &MiddlewareRollback{
		manager:      manager,
		repo:         repo,
		snapshots:    make(map[string][]*MiddlewareSnapshot),
		logger:       logger,
		maxSnapshots: 10,
	}
}

// CreateSnapshot creates a snapshot of current middleware state
func (mr *MiddlewareRollback) CreateSnapshot(ctx context.Context, name string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Get current middleware state from repository
	plugin, err := mr.repo.GetPlugin(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get plugin state: %w", err)
	}

	snapshot := &MiddlewareSnapshot{
		Name:      plugin.Name,
		Priority:  plugin.Priority,
		Routes:    append([]string{}, plugin.AppliedTo...), // Deep copy
		Phase:     plugin.Phase,
		Config:    make(map[string]interface{}),
		Enabled:   plugin.Enabled,
		Timestamp: time.Now(),
	}

	// Deep copy config
	for k, v := range plugin.Config {
		snapshot.Config[k] = v
	}

	// Add to snapshots
	if _, exists := mr.snapshots[name]; !exists {
		mr.snapshots[name] = make([]*MiddlewareSnapshot, 0, mr.maxSnapshots)
	}

	mr.snapshots[name] = append(mr.snapshots[name], snapshot)

	// Limit snapshot history
	if len(mr.snapshots[name]) > mr.maxSnapshots {
		mr.snapshots[name] = mr.snapshots[name][1:]
	}

	mr.logger.WithFields(logrus.Fields{
		"middleware": name,
		"snapshots":  len(mr.snapshots[name]),
	}).Info("Created middleware snapshot")

	return nil
}

// Rollback rolls back a middleware to its previous snapshot
func (mr *MiddlewareRollback) Rollback(ctx context.Context, name string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	snapshots, exists := mr.snapshots[name]
	if !exists || len(snapshots) == 0 {
		return fmt.Errorf("no snapshots found for middleware %s", name)
	}

	// Get the last snapshot (skip current, go to previous)
	if len(snapshots) < 2 {
		return fmt.Errorf("no previous snapshot available for rollback")
	}

	snapshot := snapshots[len(snapshots)-2]

	mr.logger.WithFields(logrus.Fields{
		"middleware": name,
		"timestamp":  snapshot.Timestamp,
	}).Info("Rolling back middleware")

	// Update repository
	if err := mr.repo.UpdatePluginPriority(ctx, name, snapshot.Priority); err != nil {
		return fmt.Errorf("failed to restore priority: %w", err)
	}

	if err := mr.repo.UpdatePluginRoutes(ctx, name, snapshot.Routes); err != nil {
		return fmt.Errorf("failed to restore routes: %w", err)
	}

	// Update phase via UpdatePlugin
	plugin, err := mr.repo.GetPlugin(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get plugin: %w", err)
	}

	plugin.Phase = snapshot.Phase
	plugin.Config = snapshot.Config
	plugin.Enabled = snapshot.Enabled

	if err := mr.repo.UpdatePlugin(ctx, plugin); err != nil {
		return fmt.Errorf("failed to update plugin: %w", err)
	}

	// Update in-memory chain
	if err := mr.manager.UpdateMiddlewarePriority(name, snapshot.Priority); err != nil {
		mr.logger.WithError(err).Warn("Failed to update in-memory priority")
	}

	if err := mr.manager.UpdateMiddlewareRoutes(name, snapshot.Routes); err != nil {
		mr.logger.WithError(err).Warn("Failed to update in-memory routes")
	}

	// Remove the current snapshot since we rolled back
	mr.snapshots[name] = snapshots[:len(snapshots)-1]

	return nil
}

// RollbackToTimestamp rolls back to a specific snapshot by timestamp
func (mr *MiddlewareRollback) RollbackToTimestamp(ctx context.Context, name string, timestamp time.Time) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	snapshots, exists := mr.snapshots[name]
	if !exists || len(snapshots) == 0 {
		return fmt.Errorf("no snapshots found for middleware %s", name)
	}

	// Find snapshot by timestamp
	var snapshot *MiddlewareSnapshot
	var snapshotIndex int
	for i, s := range snapshots {
		if s.Timestamp.Equal(timestamp) {
			snapshot = s
			snapshotIndex = i
			break
		}
	}

	if snapshot == nil {
		return fmt.Errorf("snapshot not found for timestamp %v", timestamp)
	}

	mr.logger.WithFields(logrus.Fields{
		"middleware": name,
		"timestamp":  timestamp,
	}).Info("Rolling back middleware to specific timestamp")

	// Update repository
	if err := mr.repo.UpdatePluginPriority(ctx, name, snapshot.Priority); err != nil {
		return fmt.Errorf("failed to restore priority: %w", err)
	}

	if err := mr.repo.UpdatePluginRoutes(ctx, name, snapshot.Routes); err != nil {
		return fmt.Errorf("failed to restore routes: %w", err)
	}

	plugin, err := mr.repo.GetPlugin(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get plugin: %w", err)
	}

	plugin.Phase = snapshot.Phase
	plugin.Config = snapshot.Config
	plugin.Enabled = snapshot.Enabled

	if err := mr.repo.UpdatePlugin(ctx, plugin); err != nil {
		return fmt.Errorf("failed to update plugin: %w", err)
	}

	// Update in-memory chain
	if err := mr.manager.UpdateMiddlewarePriority(name, snapshot.Priority); err != nil {
		mr.logger.WithError(err).Warn("Failed to update in-memory priority")
	}

	if err := mr.manager.UpdateMiddlewareRoutes(name, snapshot.Routes); err != nil {
		mr.logger.WithError(err).Warn("Failed to update in-memory routes")
	}

	// Remove snapshots after the rollback point
	mr.snapshots[name] = snapshots[:snapshotIndex+1]

	return nil
}

// GetSnapshots returns all snapshots for a middleware
func (mr *MiddlewareRollback) GetSnapshots(name string) []*MiddlewareSnapshot {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	if snapshots, exists := mr.snapshots[name]; exists {
		// Return a copy
		result := make([]*MiddlewareSnapshot, len(snapshots))
		copy(result, snapshots)
		return result
	}

	return []*MiddlewareSnapshot{}
}

// ClearSnapshots clears all snapshots for a middleware
func (mr *MiddlewareRollback) ClearSnapshots(name string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	delete(mr.snapshots, name)
	mr.logger.WithField("middleware", name).Info("Cleared snapshots")
}

// ClearAllSnapshots clears all snapshots for all middleware
func (mr *MiddlewareRollback) ClearAllSnapshots() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.snapshots = make(map[string][]*MiddlewareSnapshot)
	mr.logger.Info("Cleared all snapshots")
}

// AutoRollbackOnError automatically rolls back if an error is detected
func (mr *MiddlewareRollback) AutoRollbackOnError(ctx context.Context, name string, tester *MiddlewareTester, threshold int) error {
	// Check health status
	health := tester.GetHealthStatus(name)

	if health.Status == "unhealthy" && health.ConsecutiveErrors >= threshold {
		mr.logger.WithFields(logrus.Fields{
			"middleware":        name,
			"consecutiveErrors": health.ConsecutiveErrors,
			"threshold":         threshold,
		}).Warn("Auto-rollback triggered due to consecutive errors")

		if err := mr.Rollback(ctx, name); err != nil {
			return fmt.Errorf("auto-rollback failed: %w", err)
		}

		mr.logger.WithField("middleware", name).Info("Auto-rollback completed successfully")
		return nil
	}

	return nil
}

// SnapshotAllMiddleware creates snapshots for all middleware in the chain
func (mr *MiddlewareRollback) SnapshotAllMiddleware(ctx context.Context) error {
	chain := mr.manager.GetMiddlewareChain()

	for _, entry := range chain {
		if err := mr.CreateSnapshot(ctx, entry.Name); err != nil {
			mr.logger.WithError(err).WithField("middleware", entry.Name).Error("Failed to create snapshot")
			continue
		}
	}

	mr.logger.WithField("count", len(chain)).Info("Created snapshots for all middleware")
	return nil
}

// GetSnapshotHistory returns snapshot history for all middleware
func (mr *MiddlewareRollback) GetSnapshotHistory() map[string][]*MiddlewareSnapshot {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	history := make(map[string][]*MiddlewareSnapshot)
	for name, snapshots := range mr.snapshots {
		history[name] = make([]*MiddlewareSnapshot, len(snapshots))
		copy(history[name], snapshots)
	}

	return history
}
