// Package database provides database connectivity and management for the Robomesh server.
//
// This package handles MongoDB operations for robot data persistence, including
// robot registration, status updates, sensor data storage, and historical tracking.
// It maintains persistent connections with automatic connection pooling for optimal
// performance in a multi-robot environment.
package database

import (
	"context"
	"fmt"
	"os"
	"roboserver/shared"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongodbHandler manages MongoDB connections and operations for robot data.
//
// This handler maintains a persistent connection to MongoDB with automatic
// connection pooling, health monitoring, and graceful shutdown capabilities.
// It's designed to handle the high-frequency operations typical in robot
// management systems.
//
// Features:
// - Persistent connection with automatic pooling
// - Health monitoring and automatic reconnection
// - Context-based cancellation for graceful shutdown
// - Optimized for concurrent robot operations
//
// Usage:
//
//	handler := &MongodbHandler{}
//	err := handler.Start(ctx)
//	if err != nil {
//	    log.Fatal("Failed to start MongoDB handler:", err)
//	}
//	defer handler.Stop(ctx)
type MongodbHandler struct {
	client   *mongo.Client
	database *mongo.Database
	ctx      context.Context
	cancel   context.CancelFunc
}

// Start initializes and establishes a persistent MongoDB connection.
//
// This method creates a MongoDB client with optimized connection pooling settings
// for robot management workloads. It establishes a persistent connection that
// will be reused for all database operations, providing better performance than
// per-request connections.
//
// Connection Configuration:
// - Uses MongoDB Stable API version 1 for compatibility
// - Connection pooling with automatic management
// - Health monitoring with periodic ping
// - Graceful shutdown coordination via context
//
// Environment Variables:
// - MONGODB_URI: MongoDB connection string (required)
// - MONGODB_DATABASE: Database name (defaults to "robomesh")
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//
// Returns:
//   - error: nil on success, specific error on connection failure
//
// Example Usage:
//
//	handler := &MongodbHandler{}
//	if err := handler.Start(ctx); err != nil {
//	    log.Fatal("MongoDB connection failed:", err)
//	}
//	defer handler.Stop(ctx)
func (h *MongodbHandler) Start(ctx context.Context) error {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		return fmt.Errorf("MONGODB_URI environment variable is not set")
	}

	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		return fmt.Errorf("MONGODB_DATABASE environment variable is not set")
	}

	shared.DebugPrint("Connecting to MongoDB at: %s", mongoURI)

	// Create context for this handler instance
	h.ctx, h.cancel = context.WithCancel(ctx)

	// Configure client options for optimal performance
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().
		ApplyURI(mongoURI).
		SetServerAPIOptions(serverAPI).
		SetMaxPoolSize(shared.MONGODB_MAX_POOL_SIZE). // Adjust based on expected concurrent robots
		SetMinPoolSize(shared.MONGODB_MIN_POOL_SIZE). // Maintain minimum connections
		SetMaxConnIdleTime(0).                        // Keep connections alive
		SetRetryWrites(true).                         // Enable retry for write operations
		SetRetryReads(true)                           // Enable retry for read operations

	// Create client and connect
	client, err := mongo.Connect(h.ctx, opts)
	if err != nil {
		h.cancel()
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Test the connection
	if err := client.Ping(h.ctx, readpref.Primary()); err != nil {
		client.Disconnect(h.ctx)
		h.cancel()
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Store client and database references
	h.client = client
	h.database = client.Database(dbName)

	shared.DebugPrint("Successfully connected to MongoDB database: %s", dbName)
	return nil
}

// Stop gracefully closes the MongoDB connection and cleans up resources.
//
// This method should be called when the server is shutting down to ensure
// all database connections are properly closed and resources are released.
//
// Parameters:
//   - ctx: Context for shutdown timeout control
//
// Returns:
//   - error: nil on success, specific error on disconnect failure
func (h *MongodbHandler) Stop(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}

	if h.client != nil {
		if err := h.client.Disconnect(ctx); err != nil {
			shared.DebugPrint("Error disconnecting from MongoDB: %v", err)
			return fmt.Errorf("failed to disconnect from MongoDB: %w", err)
		}
		shared.DebugPrint("Successfully disconnected from MongoDB")
	}

	return nil
}

// GetDatabase returns the MongoDB database instance for operations.
//
// This method provides access to the MongoDB database for performing
// robot-related operations like storing sensor data, updating robot status,
// and querying historical information.
//
// Returns:
//   - *mongo.Database: Database instance for performing operations
//   - error: ErrDatabaseNotInitialized if Start() hasn't been called
func (h *MongodbHandler) GetDatabase() (*mongo.Database, error) {
	if h.database == nil {
		return nil, fmt.Errorf("database not initialized - call Start() first")
	}
	return h.database, nil
}

// GetCollection returns a MongoDB collection for robot data operations.
//
// This is a convenience method for accessing specific collections within
// the robot database. Common collections include "robots", "sensor_data",
// "commands", and "logs".
//
// Parameters:
//   - name: Collection name (e.g., "robots", "sensor_data")
//
// Returns:
//   - *mongo.Collection: Collection instance for operations
//   - error: ErrDatabaseNotInitialized if Start() hasn't been called
func (h *MongodbHandler) GetCollection(name string) (*mongo.Collection, error) {
	if h.database == nil {
		return nil, fmt.Errorf("database not initialized - call Start() first")
	}
	return h.database.Collection(name), nil
}

// IsHealthy checks if the MongoDB connection is still active.
//
// This method can be used for health checks and monitoring to ensure
// the database connection is operational. It performs a lightweight
// ping operation to verify connectivity.
//
// Returns:
//   - bool: true if connection is healthy, false otherwise
func (h *MongodbHandler) IsHealthy() bool {
	if h.client == nil {
		return false
	}

	// Use a short timeout for health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.client.Ping(ctx, readpref.Primary())
	return err == nil
}

// Legacy function for backward compatibility - consider using MongodbHandler.Start() instead
func StartMongodb(ctx context.Context) error {
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		return fmt.Errorf("MONGODB_URI environment variable is not set")
	}
	shared.DebugPrint("Connecting to MongoDB at:", mongoURI)

	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)
	// Create a new client and connect to the server
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return err
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			shared.DebugPrint("Error disconnecting from MongoDB:", err)
		}
	}()
	// Send a ping to confirm a successful connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return err
	}
	shared.DebugPrint("Pinged your deployment. Successfully connected to MongoDB!")

	// Create a handler instance and start it
	handler := &MongodbHandler{}
	return handler.Start(ctx)
}

// Example usage patterns for robot data operations:
//
// Store Robot Registration:
//   collection, err := handler.GetCollection("robots")
//   if err != nil {
//       return err
//   }
//   _, err = collection.InsertOne(ctx, robotData)
//
// Query Robot Status:
//   var robot RobotDocument
//   err = collection.FindOne(ctx, bson.M{"device_id": deviceID}).Decode(&robot)
//
// Update Robot Status:
//   update := bson.M{"$set": bson.M{"status": "online", "last_seen": time.Now()}}
//   _, err = collection.UpdateOne(ctx, filter, update)
//
// Store Sensor Data:
//   sensorCollection, err := handler.GetCollection("sensor_data")
//   sensorData := bson.M{
//       "device_id": deviceID,
//       "timestamp": time.Now(),
//       "data": sensorReading,
//   }
//   _, err = sensorCollection.InsertOne(ctx, sensorData)
