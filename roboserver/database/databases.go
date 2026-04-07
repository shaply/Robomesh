package database

import (
	"context"
	"os"
	"roboserver/shared"

	"golang.org/x/crypto/bcrypt"
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

	// Seed default admin user if not already present
	seedDefaultUsers(dbCtx, rds)

	shared.DebugPrint("All databases initialized successfully")

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

// seedDefaultUsers ensures the admin user exists in Redis.
func seedDefaultUsers(ctx context.Context, rds *RedisHandler) {
	if rds == nil {
		return
	}

	// Check if admin already exists
	if _, err := rds.GetUser(ctx, "admin"); err == nil {
		shared.DebugPrint("Admin user already seeded")
		return
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "password1"
		shared.DebugPrint("WARNING: Using default admin credentials (admin/password1). Set ADMIN_PASSWORD env var for production.")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		shared.DebugPrint("Failed to hash default admin password: %v", err)
		return
	}

	user := &User{
		Username:     "admin",
		PasswordHash: string(hash),
	}
	if err := rds.SetUser(ctx, user); err != nil {
		shared.DebugPrint("Failed to seed admin user: %v", err)
		return
	}

	if os.Getenv("ADMIN_PASSWORD") != "" {
		shared.DebugPrint("Admin user seeded with password from ADMIN_PASSWORD env var")
	}
}
