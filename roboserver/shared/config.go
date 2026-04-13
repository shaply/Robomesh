package shared

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AppConfig is the global application configuration singleton.
var AppConfig Config

// DEBUG_MODE controls debug logging throughout the server.
var DEBUG_MODE = false

const (
	EVENT_BUS_BUFFER_SIZE = 1000
)

// Config is the top-level application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Handlers HandlersConfig `yaml:"handlers"`
	Timeouts TimeoutsConfig `yaml:"timeouts"`
}

type TimeoutsConfig struct {
	Handshake      string `yaml:"handshake"`
	ProcessKill    string `yaml:"process_kill"`
	ReverseConnect string `yaml:"reverse_connect"`
}

func (t *TimeoutsConfig) HandshakeTimeout() time.Duration {
	d, err := time.ParseDuration(t.Handshake)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

func (t *TimeoutsConfig) ProcessKillTimeout() time.Duration {
	d, err := time.ParseDuration(t.ProcessKill)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

func (t *TimeoutsConfig) ReverseConnectTimeout() time.Duration {
	d, err := time.ParseDuration(t.ReverseConnect)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

type ServerConfig struct {
	HTTPPort       int       `yaml:"http_port"`
	TCPPort        int       `yaml:"tcp_port"`
	UDPPort        int       `yaml:"udp_port"`
	MQTTPort       int       `yaml:"mqtt_port"`
	TerminalPort   int       `yaml:"terminal_port"`
	Debug          bool      `yaml:"debug"`
	AllowedOrigins []string  `yaml:"allowed_origins"`
	TLS            TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type DatabaseConfig struct {
	Postgres PostgresConfig `yaml:"postgres"`
	Redis    RedisConfig    `yaml:"redis"`
}

type PostgresConfig struct {
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	User            string `yaml:"user"`
	Password        string `yaml:"-"`
	Database        string `yaml:"database"`
	SSLMode         string `yaml:"ssl_mode"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Password       string `yaml:"-"`
	DB             int    `yaml:"db"`
	SessionTTL     string `yaml:"session_ttl"`
	UserSessionTTL string `yaml:"user_session_ttl"`
}

type AuthConfig struct {
	JWTSecret   string `yaml:"-"`
	JWTExpiry   int    `yaml:"jwt_expiry"`
	NonceLength int    `yaml:"nonce_length"`
}

type HandlersConfig struct {
	BasePath string `yaml:"base_path"`
}

// DSN returns the PostgreSQL connection string.
func (p *PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode)
}

// Addr returns the Redis address as host:port.
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// TTL returns the robot session TTL as a time.Duration.
func (r *RedisConfig) TTL() time.Duration {
	d, err := time.ParseDuration(r.SessionTTL)
	if err != nil {
		return 60 * time.Second
	}
	return d
}

// UserTTL returns the user session TTL as a time.Duration.
func (r *RedisConfig) UserTTL() time.Duration {
	d, err := time.ParseDuration(r.UserSessionTTL)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

// ConnLifetime returns the connection max lifetime as a time.Duration.
func (p *PostgresConfig) ConnLifetime() time.Duration {
	d, err := time.ParseDuration(p.ConnMaxLifetime)
	if err != nil {
		return time.Hour
	}
	return d
}

func defaultConfig() Config {
	return Config{
		Server: ServerConfig{
			HTTPPort:       8080,
			TCPPort:        5002,
			UDPPort:        5001,
			MQTTPort:       1883,
			TerminalPort:   6000,
			Debug:          false,
			AllowedOrigins: []string{"http://localhost:5173", "http://localhost:4173"},
		},
		Database: DatabaseConfig{
			Postgres: PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				User:            "robomesh",
				Database:        "robomesh_db",
				SSLMode:         "disable",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: "1h",
			},
			Redis: RedisConfig{
				Host:           "localhost",
				Port:           6379,
				DB:             0,
				SessionTTL:     "60s",
				UserSessionTTL: "24h",
			},
		},
		Auth: AuthConfig{
			JWTExpiry:   3600,
			NonceLength: 32,
		},
		Handlers: HandlersConfig{
			BasePath: "../handlers",
		},
		Timeouts: TimeoutsConfig{
			Handshake:      "30s",
			ProcessKill:    "10s",
			ReverseConnect: "10s",
		},
	}
}

// LoadConfig loads configuration with priority: defaults < YAML file < environment variables.
func LoadConfig(path string) error {
	AppConfig = defaultConfig()

	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, &AppConfig); err != nil {
			return fmt.Errorf("failed to parse config file %s: %w", path, err)
		}
	}

	applyEnvOverrides(&AppConfig)
	DEBUG_MODE = AppConfig.Server.Debug
	return nil
}

func applyEnvOverrides(cfg *Config) {
	// Server
	envBool("DEBUG", &cfg.Server.Debug)
	envInt("HTTP_PORT", &cfg.Server.HTTPPort)
	envInt("TCP_PORT", &cfg.Server.TCPPort)
	envInt("UDP_PORT", &cfg.Server.UDPPort)
	envInt("MQTT_PORT", &cfg.Server.MQTTPort)
	envInt("TERMINAL_PORT", &cfg.Server.TerminalPort)

	// PostgreSQL
	envStr("POSTGRES_HOST", &cfg.Database.Postgres.Host)
	envInt("POSTGRES_PORT", &cfg.Database.Postgres.Port)
	envStr("POSTGRES_USER", &cfg.Database.Postgres.User)
	envStr("POSTGRES_PASSWORD", &cfg.Database.Postgres.Password)
	envStr("POSTGRES_DB", &cfg.Database.Postgres.Database)
	envStr("POSTGRES_SSL_MODE", &cfg.Database.Postgres.SSLMode)

	// Redis
	envStr("REDIS_HOST", &cfg.Database.Redis.Host)
	envInt("REDIS_PORT", &cfg.Database.Redis.Port)
	envStr("REDIS_PASSWORD", &cfg.Database.Redis.Password)
	envInt("REDIS_DB", &cfg.Database.Redis.DB)

	// Auth
	envStr("JWT_SECRET", &cfg.Auth.JWTSecret)
	envInt("JWT_EXPIRY", &cfg.Auth.JWTExpiry)

	// Handlers
	envStr("HANDLERS_BASE_PATH", &cfg.Handlers.BasePath)

	// TLS
	envBool("TLS_ENABLED", &cfg.Server.TLS.Enabled)
	envStr("TLS_CERT_FILE", &cfg.Server.TLS.CertFile)
	envStr("TLS_KEY_FILE", &cfg.Server.TLS.KeyFile)

	// CORS
	envCSV("ALLOWED_ORIGINS", &cfg.Server.AllowedOrigins)
}

func envStr(key string, dst *string) {
	if v := os.Getenv(key); v != "" {
		*dst = v
	}
}

func envInt(key string, dst *int) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*dst = n
		}
	}
}

func envBool(key string, dst *bool) {
	if v := os.Getenv(key); v != "" {
		*dst = v == "true"
	}
}

func envCSV(key string, dst *[]string) {
	if v := os.Getenv(key); v != "" {
		var parts []string
		for _, s := range strings.Split(v, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				parts = append(parts, s)
			}
		}
		if len(parts) > 0 {
			*dst = parts
		}
	}
}
