# Robomesh

A centralized ecosystem for managing autonomous robots — from physical home-assistants and rovers to software-based trading bots. Designed for low-power Mini PC deployment with a stateless, decoupled, cryptographically secure, zero-idle architecture.

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                         Frontend                             │
│                  SvelteKit 5 + Tailwind CSS                  │
│                  (SSE + WebSocket for real-time updates)      │
└────────────────────────┬─────────────────────────────────────┘
                         │ HTTP (REST + SSE + WS)
┌────────────────────────▼──────────────────────────────────────┐
│                      roboserver (Go)                          │
│                                                               │
│  ┌───────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐        │
│  │HTTP Server│ │TCP Server│ │UDP Server│ │MQTT Broker│        │
│  │(Chi)      │ │          │ │(JSON)    │ │(Mochi)    │        │
│  └────┬──────┘ └───┬──────┘ └───┬──────┘ └───┬──────┘        │
│       │            │            │             │         ┌────────────┐
│       │            │            │             │         │  Terminal   │
│       │            └──────┬─────┘             │         │ (Debug CLI)│
│       │                   │                   │         └────────────┘
│       │       ┌───────────▼───────────────────▼──┐                │
│       │       │  Cryptographic Handshake         │                │
│       │       │  (Nonce → Sign → Verify → JWT)   │                │
│       │       └───────────┬──────────────────────┘                │
│       │                   │                                       │
│  ┌────▼───────────────────▼──────────────────────┐                │
│  │           Handler Engine                      │                │
│  │  OS Process Spawner + JSON-RPC Router         │                │
│  │  (stdin/stdout multiplexing)                  │                │
│  └────┬──────────────┬───────────────────────────┘                │
│       │              │                                            │
│  ┌────▼─────┐   ┌────▼────┐                                      │
│  │PostgreSQL│   │  Redis  │                                      │
│  │(registry)│   │ (state) │                                      │
│  └──────────┘   └─────────┘                                      │
└────────────────────────┬──────────────────────────────────────────┘
                         │
              ┌──────────▼───────────┐
              │   Handler Scripts    │
              │  (Bash/Python/Node)  │
              │  ./handlers/*.sh     │
              │  Zero-idle: spawned  │
              │  on-demand per robot │
              └──────────────────────┘
```

**Key Design Pillars:**

1. **Cryptographic Device Authentication** — Ed25519 challenge-response handshake. On success, a session JWT is issued.
2. **Zero-Idle Execution** — Handler processes spawned on connect, torn down on disconnect. Zero compute when idle.
3. **Multiplexed JSON-RPC** — Handler scripts communicate via stdin/stdout JSON envelopes.
4. **Stateless Server** — Active state in Redis (TTL-based). Permanent registry in PostgreSQL.
5. **Communication Abstraction** — All inter-service communication goes through `comms.Bus`. Swap the implementation (Kafka, gRPC) without changing service code.

For protocol details and API reference, see [docs/PROTOCOL.md](docs/PROTOCOL.md).

## Project Structure

```text
Robomesh/
├── defaults.env                # Single source of truth for default ports/settings
├── docker-compose.yml          # Production: distributed deployment
├── docker-compose.dev.yml      # Local dev: all-in-one (Postgres + Redis + Backend)
├── .env.example                # Template for secrets and overrides
│
├── handlers/                   # Handler scripts (mounted into container)
│   ├── _template/              # Boilerplate for new robot types
│   └── test_robot/             # Example: echo handler
│
├── db/                         # Database management
│   ├── init.sql                # Schema (Docker entrypoint)
│   ├── seed.sql                # Dev seed data
│   ├── migrations/             # dbmate migrations
│   └── docker-compose.yml      # Standalone DB deployment
│
├── scripts/                    # Development utilities
│   └── test_e2e.py             # E2E test suite (Python)
│
├── tests/                      # Integration testing
│   └── integration/            # Deployment readiness test suite
│
├── roboserver/                 # Go backend
│   ├── main.go                 # Entry point
│   ├── config.yaml             # Structural configuration
│   ├── Dockerfile
│   ├── auth/                   # Crypto handshake, JWT, heartbeat
│   ├── comms/                  # Communication abstraction (Bus interface)
│   ├── handler_engine/         # Zero-idle process spawner + JSON-RPC router
│   ├── database/               # PostgreSQL + Redis integration
│   ├── http_server/            # REST API + SSE + WebSocket (Chi router)
│   ├── tcp_server/             # Robot TCP protocol (AUTH, REGISTER, PERSIST)
│   ├── udp_server/             # Robot UDP protocol (JSON packet-based)
│   ├── mqtt_server/            # MQTT broker (Mochi-MQTT embedded)
│   ├── terminal/               # Debug CLI
│   └── shared/                 # Core types, config, event bus
│
├── frontend_app/               # SvelteKit 5 frontend
│   ├── Dockerfile
│   ├── src/
│   │   ├── routes/             # (auth)/ and (app)/ route groups
│   │   └── lib/                # Components, stores, robot registry
│   └── package.json
│
├── robot_sdk/                  # Robot client SDKs
│   ├── python/                 # Python SDK (TCP, UDP, MQTT)
│   └── c/                      # C SDK (TCP, UDP, MQTT)
│
└── docs/                       # Protocol and API documentation
    ├── PROTOCOL.md             # Index of all protocol docs
    ├── TCP.md                  # TCP protocol
    ├── UDP.md                  # UDP protocol
    ├── MQTT.md                 # MQTT protocol
    ├── HEARTBEAT.md            # Heartbeat protocol (all transports)
    ├── HANDLER.md              # Handler script JSON-RPC protocol
    ├── HTTP_API.md             # REST API reference
    ├── COMM_BUS.md             # Communication bus abstraction
    ├── TERMINAL.md             # Debug terminal commands
    └── CONFIGURATION.md        # Configuration reference
```

## Local Development

```bash
# Start PostgreSQL, Redis, Backend, and Frontend
docker compose -f docker-compose.dev.yml up --build

# Services exposed:
#   HTTP API:   http://localhost:8080
#   TCP Server: localhost:5002
#   UDP Server: localhost:5001
#   MQTT Broker:localhost:1883
#   Terminal:   localhost:6000
#   Frontend:   http://localhost:3000
#   PostgreSQL: localhost:5433
#   Redis:      localhost:6380
```

The dev compose file auto-initializes the database schema (`db/init.sql`) and seeds test data (`db/seed.sql`).

All default ports are defined in `defaults.env`. To override any port, set the corresponding variable in your `.env` file.

### Running Without Docker

```bash
# Backend (requires local PostgreSQL + Redis)
cd roboserver
cp .env.example .env  # Fill in credentials
go run .

# Frontend
cd frontend_app
cp .env.example .env  # Set PUBLIC_BACKEND_IP and PUBLIC_BACKEND_PORT
npm install
npm run dev
```

## Tests

```bash
# Unit tests (Go backend)
cd roboserver && go test ./...

# Unit tests (Python SDK)
cd robot_sdk/python && pip install -e . && pytest tests/test_udp_unit.py tests/test_mqtt_unit.py -v

# E2E tests (requires running server)
pip install cryptography
python3 scripts/test_e2e.py

# Integration tests — deployment readiness (requires running server)
cd tests/integration
pip install -r requirements.txt
pytest -v

# SDK integration tests (requires running server)
cd robot_sdk/python && pip install -e . && pytest tests/test_integration.py -v
```

## Production Build

Robomesh is designed for distributed deployment: a Database Machine and a Mini PC.

### Machine A: Database Host

```bash
cd db
cp .env.example .env  # Set POSTGRES_PASSWORD
docker compose up -d
```

### Machine B: Mini PC (Server + Frontend + Redis)

```bash
cp .env.example .env
# Set POSTGRES_PASSWORD, REDIS_PASSWORD, JWT_SECRET
# Set POSTGRES_HOST to Machine A's LAN IP
docker compose up -d
docker compose exec backend dbmate up  # Run migrations
```

## Configuration

Configuration priority: **defaults < config.yaml < environment variables**.

Default ports and settings are centralized in `defaults.env`. Docker Compose loads this automatically. For non-Docker usage, `config.yaml` provides the same defaults.

| Variable | Description | Default |
| --- | --- | --- |
| `HTTP_PORT` | HTTP API port | 8080 |
| `TCP_PORT` | Robot TCP port | 5002 |
| `UDP_PORT` | Robot UDP port | 5001 |
| `MQTT_PORT` | MQTT broker port | 1883 |
| `TERMINAL_PORT` | Debug terminal port | 6000 |
| `FRONTEND_PORT` | Frontend port | 3000 |
| `POSTGRES_HOST` | PostgreSQL host | localhost |
| `POSTGRES_PASSWORD` | PostgreSQL password | *(required)* |
| `REDIS_HOST` | Redis host | localhost |
| `REDIS_PASSWORD` | Redis password | *(required)* |
| `JWT_SECRET` | JWT signing secret | *(required)* |
| `DEBUG` | Enable debug logging | false |

See `roboserver/config.yaml` for the full structural configuration reference, or [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for detailed documentation.
