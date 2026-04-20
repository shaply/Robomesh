# Configuration

Configuration is layered: `config.yaml` defines structure and defaults, environment variables override any value. Env vars always take precedence.

Default ports and settings are centralized in `defaults.env` at the project root. Docker Compose loads this automatically. When changing a port or default, update `defaults.env` — this propagates to all Docker services. For non-Docker usage, `config.yaml` provides the same defaults and can be overridden with env vars.

Access in code via `shared.AppConfig`.

## Server

```yaml
server:
  http_port: 8080
  tcp_port: 5002
  udp_port: 5001
  mqtt_port: 1883
  terminal_port: 6000
  debug: false
  allowed_origins:
    - "http://localhost:5173"
    - "http://localhost:4173"
```

| Env Var | Description |
| --- | --- |
| `HTTP_PORT` | HTTP server port |
| `TCP_PORT` | TCP server port |
| `UDP_PORT` | UDP server port |
| `MQTT_PORT` | MQTT server port |
| `TERMINAL_PORT` | Terminal server port |
| `DEBUG` | Enable debug logging (`true`/`false`) |
| `ALLOWED_ORIGINS` | Comma-separated list of CORS origins |

## Database

```yaml
database:
  postgres:
    host: "localhost"
    port: 5432
    user: "robomesh"
    database: "robomesh_db"
    ssl_mode: "disable"
    max_open_conns: 10
    max_idle_conns: 5
    conn_max_lifetime: "1h"
  redis:
    host: "localhost"
    port: 6379
    db: 0
    session_ttl: "60s"
    user_session_ttl: "24h"
```

| Env Var | Description |
| --- | --- |
| `POSTGRES_HOST` | PostgreSQL host |
| `POSTGRES_PORT` | PostgreSQL port |
| `POSTGRES_USER` | PostgreSQL user |
| `POSTGRES_PASSWORD` | PostgreSQL password |
| `POSTGRES_DB` | PostgreSQL database name |
| `POSTGRES_SSL_MODE` | PostgreSQL SSL mode (default: `disable`) |
| `REDIS_HOST` | Redis host |
| `REDIS_PORT` | Redis port |
| `REDIS_PASSWORD` | Redis password |
| `REDIS_DB` | Redis database number (default: `0`) |

**TTL configuration:**

- `session_ttl` — Robot session TTL in Redis (default: 60s). Controls how long active robot sessions persist without heartbeat renewal.
- `user_session_ttl` — User (web UI) session TTL in Redis (default: 24h). Controls how long user login sessions last.

## Authentication

```yaml
auth:
  jwt_expiry: 3600
  nonce_length: 32
```

`jwt_secret` is loaded exclusively from the `JWT_SECRET` environment variable (not from YAML).

| Env Var | Description |
| --- | --- |
| `JWT_SECRET` | Secret key for HS256 JWT signing (required in production) |

## Handlers

```yaml
handlers:
  base_path: "../handlers"
```

| Env Var | Description |
| --- | --- |
| `HANDLERS_BASE_PATH` | Path to handler scripts directory |

## Timeouts

```yaml
timeouts:
  handshake: "30s"
  process_kill: "10s"
  reverse_connect: "10s"
```

| Setting | Default | Description |
| --- | --- | --- |
| `handshake` | 30s | TCP read deadline during AUTH/REGISTER handshake |
| `process_kill` | 10s | Grace period before force-killing a handler process on `Stop()` |
| `reverse_connect` | 10s | Dial timeout and read deadline for reverse connections to robots |

## Redis Key Schema

| Key Pattern | Type | TTL | Description |
| --- | --- | --- | --- |
| `robot:{uuid}:active` | JSON | `session_ttl` | Active robot session (UUID, IP, DeviceType, JWT, PID) |
| `robot:{uuid}:heartbeat` | JSON | Per-heartbeat | Heartbeat state (UUID, IP, LastSeq, LastSeen) |
| `robot:{uuid}:pending` | JSON | 5 min | Pending registration |
| `robot:{uuid}:pubkey` | String | 5 min | Public key storage during REGISTER flow |
| `mqtt:nonce:{uuid}` | String | 30s | MQTT auth nonce + cached robot info (`nonce\|publicKey\|deviceType`) |
| `udp:nonce:{uuid}` | String | 30s | UDP auth nonce + cached robot info (`nonce\|publicKey\|deviceType`) |
| `handler:{uuid}:data:{key}` | String | None | Handler-scoped custom data storage |
| `user:{username}` | JSON | None | User credentials (bcrypt hashed) |
| `session:{token}` | String | `user_session_ttl` | User session for server-side invalidation |
| `ticket:{ticket}` | String | 30s | Single-use SSE ticket |

## Startup Sequence

1. Load config from `config.yaml` + env vars
2. Initialize event bus
3. Connect databases (PostgreSQL + Redis)
4. Seed default admin user (if not exists)
5. Initialize comm bus
6. Start 5 concurrent servers: Terminal, HTTP, TCP, UDP, MQTT

## Graceful Shutdown

On SIGINT/SIGTERM:

1. Cancel root context (60s timeout)
2. Stop all handler processes via `HandlerManager.StopAll()`
3. Shut down all servers
4. Close database connections
