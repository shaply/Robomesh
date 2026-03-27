package database

import (
	"context"
	"roboserver/shared"
)

type DBManager_t struct {
	postgres *PostgresHandler
	redis    *RedisHandler
	ctx      context.Context
	cancel   context.CancelFunc
}

// Start initializes PostgreSQL and Redis connections and returns a DBManager.
func Start(ctx context.Context) (DBManager, error) {
	dbCtx, cancel := context.WithCancel(ctx)
	manager := &DBManager_t{
		ctx:    dbCtx,
		cancel: cancel,
	}

	// Initialize PostgreSQL
	pg, err := NewPostgresHandler(dbCtx)
	if err != nil {
		cancel()
		return nil, err
	}
	manager.postgres = pg

	// Initialize Redis
	rds, err := NewRedisHandler(dbCtx)
	if err != nil {
		cancel()
		return nil, err
	}
	manager.redis = rds

	shared.DebugPrint("All databases initialized successfully")

	go func() {
		<-dbCtx.Done()
		shared.DebugPrint("Database context cancelled, shutting down databases...")
		manager.Stop()
	}()

	return manager, nil
}

func (dm *DBManager_t) Postgres() *PostgresHandler { return dm.postgres }
func (dm *DBManager_t) Redis() *RedisHandler       { return dm.redis }

func (dm *DBManager_t) Stop() {
	if dm.cancel != nil {
		dm.cancel()
	}

	if dm.postgres != nil {
		dm.postgres.Close()
	}
	if dm.redis != nil {
		dm.redis.Close()
	}
	shared.DebugPrint("All databases stopped successfully")
}

func (dm *DBManager_t) IsHealthy(ctx context.Context) bool {
	if dm.postgres == nil || !dm.postgres.IsHealthy(ctx) {
		return false
	}
	if dm.redis == nil || !dm.redis.IsHealthy(ctx) {
		return false
	}
	return true
}
