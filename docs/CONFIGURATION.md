# Configuration

Configuration is layered: `config.yaml` defines structure and defaults, environment variables override any value. Env vars always take precedence.

Access in code via `shared.AppConfig`.

## Server

```yaml
server:
  http_port: 8080
  tcp_port: 5000
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
| `MQTT_PORT` | MQTT server port |
| `TERMINAL_PORT` | Terminal server port |
| `DEBUG_MODE` | Enable debug logging (`true`/`false`) |
| `ALLOWED_ORIGINS` | Comma-separated list of CORS origins |

## Database

```yaml
database:
  postgres:
    host: "localhost"
    port: 5432
    user: "robomesh"
    db_name: "robomesh"
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
| `REDIS_HOST` | Redis host |
| `REDIS_PORT` | Redis port |
| `REDIS_PASSWORD` | Redis password |

**TTL configuration:**

- `session_ttl` — Robot session TTL in Redis (default: 60s). Controls how long active robot sessions persist without heartbeat renewal.
- `user_session_ttl` — User (web UI) session TTL in Redis (default: 24h). Controls how long user login sessions last.

## Authentication

```yaml
auth:
  jwt_secret: ""
```

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
| `user:{username}` | JSON | None | User credentials (bcrypt hashed) |
| `session:{token}` | String | `user_session_ttl` | User session for server-side invalidation |
| `ticket:{ticket}` | String | 30s | Single-use SSE ticket |

## Startup Sequence

1. Load config from `config.yaml` + env vars
2. Initialize event bus
3. Connect databases (PostgreSQL + Redis)
4. Run PostgreSQL migrations
5. Seed default admin user (if not exists)
6. Initialize comm bus
7. Start 4 concurrent servers: Terminal, HTTP, TCP, MQTT

## Graceful Shutdown

On SIGINT/SIGTERM:

1. Cancel root context (60s timeout)
2. Stop all handler processes via `HandlerManager.StopAll()`
3. Shut down all servers
4. Close database connections
