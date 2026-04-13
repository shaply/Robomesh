# Robomesh

A centralized ecosystem for managing autonomous robots — from physical home-assistants and rovers to software-based trading bots. Designed for low-power Mini PC deployment with a stateless, decoupled, cryptographically secure, zero-idle architecture.

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                         Frontend                             │
│                  SvelteKit 5 + Tailwind CSS                  │
│                  (SSE for real-time updates)                 │
└────────────────────────┬─────────────────────────────────────┘
                         │ HTTP (REST + SSE)
┌────────────────────────▼──────────────────────────────────────┐
│                      roboserver (Go)                          │
│                                                               │
│  ┌───────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │HTTP Server│  │TCP Server│  │MQTT Stub │  │  Terminal    │  │
│  │(Chi)      │  │          │  │          │  │  (Debug CLI) │  │
│  └────┬──────┘  └───┬──────┘  └──────────┘  └──────────────┘  │
│       │             │                                         │
│       │       ┌─────▼──────────────────────┐                  │
│       │       │  Cryptographic Handshake   │                  │
│       │       │  (Nonce → Sign → Verify)   │                  │
│       │       └─────┬──────────────────────┘                  │
│       │             │                                         │
│  ┌────▼─────────────▼──────────────────────┐                  │
│  │           Handler Engine                │                  │
│  │  OS Process Spawner + JSON-RPC Router   │                  │
│  │  (stdin/stdout multiplexing)            │                  │
│  └────┬──────────────┬─────────────────────┘                  │
│       │              │                                        │
│  ┌────▼─────┐   ┌────▼────┐                                   │
│  │PostgreSQL│   │  Redis  │                                   │
│  │(registry)│   │ (state) │                                   │
│  └──────────┘   └─────────┘                                   │
└────────────────────────┬──────────────────────────────────────┘
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
├── docker-compose.yml          # Production: distributed deployment
├── docker-compose.dev.yml      # Local dev: all-in-one (Postgres + Redis + Backend)
├── .env.example                # Template for secrets
│
├── handlers/                   # Handler scripts (mounted into container)
│   └── example_robot.sh        # Example: echo handler
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
├── roboserver/                 # Go backend
│   ├── main.go                 # Entry point
│   ├── config.yaml             # Structural configuration
│   ├── Dockerfile
│   ├── auth/                   # Crypto handshake, JWT, heartbeat
│   ├── comms/                  # Communication abstraction (Bus interface)
│   ├── handler_engine/         # Zero-idle process spawner + JSON-RPC router
│   ├── database/               # PostgreSQL + Redis integration
│   ├── http_server/            # REST API (Chi router)
│   ├── tcp_server/             # Robot TCP protocol (AUTH, REGISTER, PERSIST)
│   ├── mqtt_server/            # MQTT (stub)
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
└── robots/                     # Arduino/embedded robot firmware
```

## Local Development

```bash
# Start PostgreSQL, Redis, and the Go backend
docker compose -f docker-compose.dev.yml up --build

# Services exposed:
#   HTTP API:   http://localhost:8080
#   TCP Server: localhost:5001
#   Terminal:   localhost:6001
#   PostgreSQL: localhost:5433
#   Redis:      localhost:6380
```

The dev compose file auto-initializes the database schema (`db/init.sql`) and seeds test data (`db/seed.sql`).

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
# Unit tests
cd roboserver && go test ./...

# E2E tests (requires running server)
pip install cryptography
python3 scripts/test_e2e.py
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
# Set POSTGRES_HOST, POSTGRES_PASSWORD, REDIS_PASSWORD, JWT_SECRET
docker compose up -d
docker compose exec backend dbmate up  # Run migrations
```

## Configuration

Configuration priority: **defaults < config.yaml < environment variables**.

| Variable | Description | Default |
| --- | --- | --- |
| `HTTP_PORT` | HTTP API port | 8080 |
| `TCP_PORT` | Robot TCP port | 5002 |
| `POSTGRES_HOST` | PostgreSQL host | localhost |
| `POSTGRES_PASSWORD` | PostgreSQL password | *(required)* |
| `REDIS_HOST` | Redis host | localhost |
| `REDIS_PASSWORD` | Redis password | *(required)* |
| `JWT_SECRET` | JWT signing secret | *(required)* |
| `DEBUG` | Enable debug logging | false |

See `roboserver/config.yaml` for the full configuration reference.
