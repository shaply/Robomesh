# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Robomesh is a centralized ecosystem for managing autonomous robots (home-assistants, rovers, trading bots, etc.). It consists of a Go backend (`roboserver/`), a Svelte 5 frontend (`frontend_app/`), and pluggable robot type implementations.

Designed for low-power Mini PC deployment with a stateless, decoupled, cryptographically secure, zero-idle architecture.

## Build & Dev Commands

### Backend (roboserver/)
```bash
cd roboserver
go build -o roboserver          # Build
go run .                        # Run (requires .env or config.yaml)
go test ./...                   # Run all tests
go test ./auth/                 # Auth/crypto tests
go test ./handler_engine/       # Handler engine tests
go test ./shared/data_structures/  # Data structure tests
```

Configuration loads from `config.yaml` (structural) + `.env` (secrets). Env vars always override. Startup sequence: config → event bus → database (PostgreSQL + Redis) → robot manager, then 4 concurrent servers (Terminal, HTTP, TCP, MQTT).

### Frontend (frontend_app/)
```bash
cd frontend_app
npm install
npm run dev                     # Hot-reload dev server
npm run build                   # Production build
npm run check                   # svelte-check with TypeScript
```

### Docker Compose
```bash
cp .env.example .env            # Fill in POSTGRES_PASSWORD, REDIS_PASSWORD, JWT_SECRET
docker compose up -d            # Start all services
docker compose logs -f backend  # View logs
```

## Architecture

### Backend Core Components

**main.go** initializes all servers and coordinates graceful shutdown (SIGINT/SIGTERM, 60s timeout) via context cancellation.

**Configuration** (`shared/config.go`) — YAML + env var layered config system. `config.yaml` defines structure/defaults, env vars override. Access via `shared.AppConfig`.

**Auth** (`auth/`) — Cryptographic challenge-response handshake:
- `nonce.go`: Generates random hex nonces
- `verify.go`: Ed25519/ECDSA signature verification (PEM and raw hex)
- `jwt.go`: HS256 JWT issuance and validation
- `handshake.go`: Full TCP handshake flow (UUID → Nonce → Sign → Verify → JWT)
- `heartbeat.go`: Redis TTL refresh loop for active sessions

**Handler Engine** (`handler_engine/`) — Zero-idle OS process spawning:
- `registry.go`: Maps DeviceType → `./handlers/{type}.sh` script path
- `process.go`: Spawns bash scripts via `exec.CommandContext`, multiplexes stdin/stdout JSON-RPC
- `types.go`: JSON-RPC envelope format (`target`: database/robot/event_bus)

**Robot Manager** (`shared/robot_manager/`) — Central coordinator for robot connections. Dual-indexed by ID and IP with RWMutex protection.

**Event Bus** (`shared/event_bus/`) — Typed pub/sub system with SafeMap-based subscriptions. Buffer size: 1000 events.

**Thread-Safe Data Structures** (`shared/data_structures/`) — Generic `SafeMap`, `SafeQueue`, `SafeSet` with RWMutex protection. Comprehensive test suites.

### Database

**PostgreSQL** — Permanent robot registry (UUID, PublicKey, DeviceType, IsBlacklisted). Migrations in `db/migrations/` use dbmate format.

**Redis** — Ephemeral active session state with TTL. Stores connected robot metadata, JWT, PID. Heartbeat loop refreshes TTL.

### Servers

- **HTTP** (`http_server/`): Chi router. Routes: `/auth`, `/robot`, `/events` (SSE), `/provision` (robot key registration), `/ephemeral` (Redis-only sessions).
- **TCP** (`tcp_server/`): Line-based protocol. Commands: `AUTH` (crypto handshake), `REGISTER`, `TRANSFER`, `UNREGISTER`.
- **MQTT** (`mqtt_server/`): Stub, not yet implemented.
- **Terminal** (`terminal/`): Interactive CLI for debugging.

### Frontend

**SvelteKit 2.16** with Svelte 5, TypeScript strict mode, Tailwind CSS 4.1, Vite 6.2.

**Route groups**: `(auth)/` for unauthenticated pages, `(app)/` for protected pages.

**Auth flow**: POST `/auth/login` → token stored in `localStorage['auth-token']` → `fetchBackend()` wrapper auto-injects `Authorization: Bearer` header.

**Robot registry** (`lib/robots/registry.ts`): Maps robot types to Svelte components, display metadata, and capabilities. Falls back to 'default' type.

**SSE**: `EventSourceManager` (singleton) handles real-time robot status updates from the backend.

## Adding a New Robot Type

1. Create a handler script: `roboserver/handlers/{device_type}.sh`
2. Provision the robot's public key via `POST /provision`
3. Robot authenticates via TCP `AUTH` command (challenge-response)
4. Go spawns the handler script and routes JSON-RPC messages

Legacy robot packages under `roboserver/robots/` still work via the factory pattern for in-process handlers.

## Naming Conventions

- **Go**: Interfaces over concrete types; embed `BaseRobot` in specific robot structs
- **Database**: snake_case for tables/columns
- **Frontend components**: PascalCase; utility files: kebab-case
- **Handler scripts**: `{device_type}.sh` in `roboserver/handlers/`
