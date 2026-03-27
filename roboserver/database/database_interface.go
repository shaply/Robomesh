package database

import "context"

// DBManager provides access to all database backends.
type DBManager interface {
	Postgres() *PostgresHandler
	Redis() *RedisHandler
	Stop()
	IsHealthy(ctx context.Context) bool
}
