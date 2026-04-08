package database

import (
	"context"
	"database/sql"
	"fmt"
	"roboserver/shared"
	"time"

	_ "github.com/lib/pq"
)

type PostgresHandler struct {
	DB *sql.DB
}

func NewPostgresHandler(ctx context.Context) (*PostgresHandler, error) {
	cfg := shared.AppConfig.Database.Postgres
	dsn := cfg.DSN()

	shared.DebugPrint("Connecting to PostgreSQL at %s:%d", cfg.Host, cfg.Port)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnLifetime())

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	shared.DebugPrint("Successfully connected to PostgreSQL database: %s", cfg.Database)
	return &PostgresHandler{DB: db}, nil
}

func (h *PostgresHandler) Close() {
	if h.DB != nil {
		h.DB.Close()
	}
}

func (h *PostgresHandler) IsHealthy(ctx context.Context) bool {
	if h.DB == nil {
		return false
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return h.DB.PingContext(pingCtx) == nil
}

// --- Robot Registry Queries ---

type RobotRecord struct {
	UUID          string
	PublicKey     string
	DeviceType    string
	IsBlacklisted bool
	CreatedAt     time.Time
}

func (h *PostgresHandler) GetRobotByUUID(ctx context.Context, uuid string) (*RobotRecord, error) {
	row := h.DB.QueryRowContext(ctx,
		`SELECT uuid, public_key, device_type, is_blacklisted, created_at
		 FROM robots WHERE uuid = $1`, uuid)

	r := &RobotRecord{}
	if err := row.Scan(&r.UUID, &r.PublicKey, &r.DeviceType, &r.IsBlacklisted, &r.CreatedAt); err != nil {
		return nil, err
	}
	return r, nil
}

func (h *PostgresHandler) RegisterRobot(ctx context.Context, uuid, publicKey, deviceType string) error {
	_, err := h.DB.ExecContext(ctx,
		`INSERT INTO robots (uuid, public_key, device_type) VALUES ($1, $2, $3)`,
		uuid, publicKey, deviceType)
	return err
}

func (h *PostgresHandler) BlacklistRobot(ctx context.Context, uuid string, blacklisted bool) error {
	_, err := h.DB.ExecContext(ctx,
		`UPDATE robots SET is_blacklisted = $1 WHERE uuid = $2`,
		blacklisted, uuid)
	return err
}

func (h *PostgresHandler) GetRobotsByType(ctx context.Context, deviceType string) ([]*RobotRecord, error) {
	rows, err := h.DB.QueryContext(ctx,
		`SELECT uuid, public_key, device_type, is_blacklisted, created_at FROM robots WHERE device_type = $1 ORDER BY created_at`, deviceType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var robots []*RobotRecord
	for rows.Next() {
		r := &RobotRecord{}
		if err := rows.Scan(&r.UUID, &r.PublicKey, &r.DeviceType, &r.IsBlacklisted, &r.CreatedAt); err != nil {
			return nil, err
		}
		robots = append(robots, r)
	}
	return robots, rows.Err()
}

func (h *PostgresHandler) GetAllRobots(ctx context.Context) ([]*RobotRecord, error) {
	rows, err := h.DB.QueryContext(ctx,
		`SELECT uuid, public_key, device_type, is_blacklisted, created_at FROM robots ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var robots []*RobotRecord
	for rows.Next() {
		r := &RobotRecord{}
		if err := rows.Scan(&r.UUID, &r.PublicKey, &r.DeviceType, &r.IsBlacklisted, &r.CreatedAt); err != nil {
			return nil, err
		}
		robots = append(robots, r)
	}
	return robots, rows.Err()
}
