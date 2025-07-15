// Package database provides database initialization and coordination for the Robomesh server.
//
// This package manages the lifecycle of all database connections and provides
// a centralized way to start and stop database services. It coordinates between
// different database types and ensures proper initialization order.
package database

import (
	"context"
	"roboserver/shared"
)

// DBManager coordinates all database connections and provides access to database services.
//
// This manager maintains references to all database handlers and provides a
// unified interface for database operations across the application.
type DBManager_t struct {
	MongoDB *MongodbHandler
	ctx     context.Context
	cancel  context.CancelFunc
}

// Start initializes all database connections and returns a DBManager.
//
// This function creates and initializes all database handlers, ensuring they're
// properly connected and ready for use. The returned DBManager can be
// passed to components that need database access.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - rm: Robot manager instance (for potential future database integration)
//
// Returns:
//   - *DBManager: Initialized database manager with all handlers
//   - error: nil on success, specific error on initialization failure
//
// Example Usage:
//
//	dbManager, err := database.Start(ctx, robotManager)
//	if err != nil {
//	    log.Fatal("Database initialization failed:", err)
//	}
//	defer dbManager.Stop()
func Start(ctx context.Context) (DBManager, error) {
	// Create database manager
	dbCtx, cancel := context.WithCancel(ctx)
	manager := &DBManager_t{
		ctx:    dbCtx,
		cancel: cancel,
	}

	// Initialize MongoDB handler
	manager.MongoDB = &MongodbHandler{}
	if err := manager.MongoDB.Start(dbCtx); err != nil {
		cancel()
		return nil, err
	}

	shared.DebugPrint("All databases initialized successfully")

	// Start monitoring goroutine
	go func() {
		<-dbCtx.Done()
		shared.DebugPrint("Database context cancelled, shutting down databases...")
		manager.Stop()
	}()

	return manager, nil
}

// Stop gracefully shuts down all database connections.
//
// This method should be called during server shutdown to ensure all database
// connections are properly closed and resources are released.
func (dm *DBManager_t) Stop() {
	if dm.cancel != nil {
		dm.cancel()
	}

	if dm.MongoDB != nil {
		if err := dm.MongoDB.Stop(dm.ctx); err != nil {
			shared.DebugPrint("Error stopping MongoDB: %v", err)
		}
	}

	shared.DebugPrint("All databases stopped successfully")
}

// GetMongoDB returns the MongoDB handler for database operations.
//
// This method provides access to the MongoDB handler for components that need
// to perform database operations.
//
// Returns:
//   - *MongodbHandler: MongoDB handler instance (nil if not initialized)
func (dm *DBManager_t) GetMongoDB() *MongodbHandler {
	return dm.MongoDB
}

// IsHealthy checks if all database connections are healthy.
//
// This method can be used for health checks and monitoring to ensure
// all database services are operational.
//
// Returns:
//   - bool: true if all databases are healthy, false otherwise
func (dm *DBManager_t) IsHealthy() bool {
	if dm.MongoDB == nil || !dm.MongoDB.IsHealthy() {
		return false
	}

	// Add checks for other databases here as they're added
	return true
}
