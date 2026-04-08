package shared

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.Server.HTTPPort != 8080 {
		t.Errorf("Expected default HTTP port 8080, got %d", cfg.Server.HTTPPort)
	}
	if cfg.Server.TCPPort != 5000 {
		t.Errorf("Expected default TCP port 5000, got %d", cfg.Server.TCPPort)
	}
	if cfg.Server.MQTTPort != 1883 {
		t.Errorf("Expected default MQTT port 1883, got %d", cfg.Server.MQTTPort)
	}
	if cfg.Database.Postgres.Host != "localhost" {
		t.Errorf("Expected default postgres host 'localhost', got %s", cfg.Database.Postgres.Host)
	}
	if cfg.Database.Redis.Host != "localhost" {
		t.Errorf("Expected default redis host 'localhost', got %s", cfg.Database.Redis.Host)
	}
	if cfg.Handlers.BasePath != "../handlers" {
		t.Errorf("Expected default handler base path '../handlers', got %s", cfg.Handlers.BasePath)
	}
}

func TestLoadConfig_NonexistentFile(t *testing.T) {
	err := LoadConfig("/nonexistent/config.yaml")
	if err != nil {
		t.Errorf("Expected nil error for missing config (should use defaults), got %v", err)
	}
	if AppConfig.Server.HTTPPort != 8080 {
		t.Errorf("Expected default HTTP port after missing config, got %d", AppConfig.Server.HTTPPort)
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("TCP_PORT", "6000")
	os.Setenv("DEBUG", "true")
	os.Setenv("POSTGRES_HOST", "dbhost")
	os.Setenv("JWT_SECRET", "supersecret")
	defer func() {
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("TCP_PORT")
		os.Unsetenv("DEBUG")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("JWT_SECRET")
	}()

	cfg := defaultConfig()
	applyEnvOverrides(&cfg)

	if cfg.Server.HTTPPort != 9090 {
		t.Errorf("Expected HTTP port 9090 from env, got %d", cfg.Server.HTTPPort)
	}
	if cfg.Server.TCPPort != 6000 {
		t.Errorf("Expected TCP port 6000 from env, got %d", cfg.Server.TCPPort)
	}
	if !cfg.Server.Debug {
		t.Error("Expected debug=true from env")
	}
	if cfg.Database.Postgres.Host != "dbhost" {
		t.Errorf("Expected postgres host 'dbhost' from env, got %s", cfg.Database.Postgres.Host)
	}
	if cfg.Auth.JWTSecret != "supersecret" {
		t.Errorf("Expected JWT secret from env, got %s", cfg.Auth.JWTSecret)
	}
}

func TestEnvCSV(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://a.com, http://b.com, http://c.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	cfg := defaultConfig()
	applyEnvOverrides(&cfg)

	if len(cfg.Server.AllowedOrigins) != 3 {
		t.Errorf("Expected 3 origins, got %d", len(cfg.Server.AllowedOrigins))
	}
	if cfg.Server.AllowedOrigins[0] != "http://a.com" {
		t.Errorf("Expected first origin 'http://a.com', got %s", cfg.Server.AllowedOrigins[0])
	}
}

func TestPostgresConfig_DSN(t *testing.T) {
	cfg := PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Database: "testdb",
		SSLMode:  "disable",
	}
	dsn := cfg.DSN()
	expected := "postgres://user:pass@localhost:5432/testdb?sslmode=disable"
	if dsn != expected {
		t.Errorf("Expected DSN %q, got %q", expected, dsn)
	}
}

func TestRedisConfig_Addr(t *testing.T) {
	cfg := RedisConfig{Host: "redis-host", Port: 6380}
	if cfg.Addr() != "redis-host:6380" {
		t.Errorf("Expected 'redis-host:6380', got %s", cfg.Addr())
	}
}

func TestRedisConfig_TTL(t *testing.T) {
	cfg := RedisConfig{SessionTTL: "120s"}
	if cfg.TTL() != 120*time.Second {
		t.Errorf("Expected 120s TTL, got %v", cfg.TTL())
	}

	// Invalid TTL should return default
	cfg.SessionTTL = "invalid"
	if cfg.TTL() != 60*time.Second {
		t.Errorf("Expected default 60s TTL for invalid input, got %v", cfg.TTL())
	}
}

func TestRedisConfig_UserTTL(t *testing.T) {
	cfg := RedisConfig{UserSessionTTL: "48h"}
	if cfg.UserTTL() != 48*time.Hour {
		t.Errorf("Expected 48h TTL, got %v", cfg.UserTTL())
	}

	cfg.UserSessionTTL = "invalid"
	if cfg.UserTTL() != 24*time.Hour {
		t.Errorf("Expected default 24h TTL for invalid input, got %v", cfg.UserTTL())
	}
}

func TestTimeoutsConfig(t *testing.T) {
	cfg := TimeoutsConfig{
		Handshake:      "15s",
		ProcessKill:    "5s",
		ReverseConnect: "20s",
	}

	if cfg.HandshakeTimeout() != 15*time.Second {
		t.Errorf("Expected 15s handshake timeout, got %v", cfg.HandshakeTimeout())
	}
	if cfg.ProcessKillTimeout() != 5*time.Second {
		t.Errorf("Expected 5s process kill timeout, got %v", cfg.ProcessKillTimeout())
	}
	if cfg.ReverseConnectTimeout() != 20*time.Second {
		t.Errorf("Expected 20s reverse connect timeout, got %v", cfg.ReverseConnectTimeout())
	}

	// Invalid durations should return defaults
	cfg = TimeoutsConfig{Handshake: "bad", ProcessKill: "bad", ReverseConnect: "bad"}
	if cfg.HandshakeTimeout() != 30*time.Second {
		t.Errorf("Expected default 30s handshake timeout, got %v", cfg.HandshakeTimeout())
	}
	if cfg.ProcessKillTimeout() != 10*time.Second {
		t.Errorf("Expected default 10s process kill timeout, got %v", cfg.ProcessKillTimeout())
	}
	if cfg.ReverseConnectTimeout() != 10*time.Second {
		t.Errorf("Expected default 10s reverse connect timeout, got %v", cfg.ReverseConnectTimeout())
	}
}

func TestConnLifetime(t *testing.T) {
	cfg := PostgresConfig{ConnMaxLifetime: "2h"}
	if cfg.ConnLifetime() != 2*time.Hour {
		t.Errorf("Expected 2h conn lifetime, got %v", cfg.ConnLifetime())
	}

	cfg.ConnMaxLifetime = "invalid"
	if cfg.ConnLifetime() != time.Hour {
		t.Errorf("Expected default 1h conn lifetime, got %v", cfg.ConnLifetime())
	}
}
