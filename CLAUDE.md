# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Robomesh is a centralized ecosystem for managing autonomous robots (home-assistants, rovers, trading bots, etc.). It consists of a Go backend (`roboserver/`), a Svelte 5 frontend (`frontend_app/`), and pluggable robot type implementations in `handlers/`.

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
go test ./shared/               # Config loading tests
go test ./mqtt_server/          # MQTT protocol tests
go test ./http_server/http_events/ # SSE event tests
```

Configuration loads from `config.yaml` (structural) + `.env` (secrets). Env vars always override. Startup sequence: config → event bus → database (PostgreSQL + Redis, seeds admin user) → comm bus, then 5 concurrent servers (Terminal, HTTP, TCP, UDP, MQTT).

### Frontend (frontend_app/)
```bash
cd frontend_app
npm install
npm run dev                     # Hot-reload dev server
npm run build                   # Production build
npm run check                   # svelte-check with TypeScript
```

### Handler Plugins (handlers/{type}/frontend/)
```bash
cd handlers/{type}/frontend
npm install
npm run build                   # Compiles Svelte components to dist/
npm run dev                     # Watch mode for development
```

### Robot SDKs

#### C SDK (robot_sdk/c/)
```bash
cd robot_sdk/c
mkdir -p build && cd build
cmake ..                        # Requires OpenSSL (libcrypto)
make                            # Builds librobomesh.a, test_integration, test_robot
./test_integration              # Integration tests (requires running roboserver)
./test_robot                    # Example robot (uses pre-seeded example-001)
```

#### Python SDK (robot_sdk/python/)
```bash
cd robot_sdk/python
pip install -e .                # Install in editable mode
python examples/test_robot.py   # Example robot
```

### Docker Compose
```bash
cp .env.example .env            # Fill in POSTGRES_PASSWORD, REDIS_PASSWORD, JWT_SECRET
docker compose up -d            # Start all services
docker compose logs -f backend  # View logs
```

## Architecture

### Backend Core Components

**main.go** initializes all servers and coordinates graceful shutdown (SIGINT/SIGTERM, 60s timeout) via context cancellation. On shutdown, all handler processes are stopped via `HandlerManager.StopAll()`.

**Configuration** (`shared/config.go`) — YAML + env var layered config system. `config.yaml` defines structure/defaults, env vars override. Access via `shared.AppConfig`.

**Auth** (`auth/`) — Cryptographic challenge-response handshake and user authentication:
- `nonce.go`: Generates random hex nonces
- `verify.go`: Ed25519/ECDSA signature verification (PEM and raw hex)
- `jwt.go`: HS256 JWT issuance and validation for robot sessions
- `user_jwt.go`: HS256 JWT issuance and validation for user sessions
- `handshake.go`: Full TCP handshake flow (UUID → Nonce → Sign → Verify → JWT)
- `heartbeat_handler.go`: Decoupled heartbeat processing — verifies signed payloads, tracks sequence numbers, updates Redis state independently of handler lifecycle

**Handler Engine** (`handler_engine/`) — Zero-idle OS process spawning with lifecycle management:
- `registry.go`: Maps DeviceType → `handlers/{type}/start_handler.sh` script path. Also `ListHandlerTypes()` and `ResolveHandlerDir()`.
- `process.go`: Spawns bash scripts, multiplexes stdin/stdout JSON-RPC. Supports event bus subscriptions, heartbeat forwarding, and config requests.
- `manager.go`: Global `HandlerManager` — thread-safe map of `UUID → *HandlerProcess`. Handlers survive TCP disconnects and can be started/killed via HTTP API.
- `reverse_connect.go`: Roboserver-initiated TCP/UDP connections to robots. Handler requests a port, roboserver resolves IP from Redis, dials out, verifies robot identity, then bridges I/O.
- `types.go`: JSON-RPC envelope format. Targets: `database`, `robot`, `event_bus`, `config`, `connect_robot`, `response`.

**Comm Bus** (`comms/`) — `Bus` interface abstracts inter-service communication. `LocalBus` wraps in-process event bus + Redis pub/sub. Swappable for Kafka/gRPC.

**Event Bus** (`shared/event_bus/`) — Typed pub/sub system with SafeMap-based subscriptions. Buffer size: 1000 events.

**Thread-Safe Data Structures** (`shared/data_structures/`) — Generic `SafeMap`, `SafeQueue`, `SafeSet` with RWMutex protection.

### Database

**PostgreSQL** — Permanent robot registry (UUID, PublicKey, DeviceType, IsBlacklisted). Migrations in `db/migrations/` use dbmate format.

**Redis** — Ephemeral state with TTL:
- `robot:{uuid}:active` — Active robot session (UUID, IP, DeviceType, JWT, PID)
- `robot:{uuid}:heartbeat` — Heartbeat state (UUID, IP, LastSeq, LastSeen) — independent of handler
- `robot:{uuid}:pending` — Pending registration (5 min TTL)
- `robot:{uuid}:pubkey` — Public key storage during REGISTER flow
- `mqtt:nonce:{uuid}` — MQTT auth nonce + cached robot info (30s TTL, pipe-delimited: `nonce|publicKey|deviceType`)
- `udp:nonce:{uuid}` — UDP auth nonce + cached robot info (30s TTL, same pipe-delimited format as MQTT)
- `handler:{uuid}:data:{key}` — Handler-scoped custom data storage (no TTL)
- `user:{username}` — User credentials (bcrypt hashed). Admin seeded on startup.
- `session:{token}` — User session tokens for server-side invalidation
- `ticket:{id}` — Single-use SSE auth tickets (30s TTL)

### Servers

- **HTTP** (`http_server/`): Chi router.
  - Public: `/auth` (login/logout/check/password), `/heartbeat` (robot heartbeat), `/plugins/{type}/*` (handler frontend assets)
  - Protected: `/robot` (list/detail/message), `/events` (SSE), `/provision`, `/register`, `/handler` (list/types/status/start/kill), `/ephemeral`, `/ws` (WebSocket)
- **TCP** (`tcp_server/`): Line-based protocol with 64KB max message size.
  - Commands: `AUTH` (crypto handshake), `REGISTER`, `HEARTBEAT <UUID> <payload> <signature>`
  - Handlers survive TCP disconnect (only notified, not killed)
- **MQTT** (`mqtt_server/`): Mochi-mqtt embedded broker with topic-based robot protocol.
  - `robomesh/auth/{uuid}` — Two-step challenge-response auth (nonce then signature). Robot record cached in Redis alongside nonce to avoid double PG lookup.
  - `robomesh/auth/{uuid}/response` — Server auth responses (nonce/JWT/error)
  - `robomesh/heartbeat/{uuid}` — Signed heartbeat payloads
  - `robomesh/heartbeat/{uuid}/response` — Heartbeat acknowledgements
  - `robomesh/message/{uuid}` — Robot→handler messages
  - `robomesh/to_robot/{uuid}` — Handler→robot messages
  - `acl_hook.go`: Custom ACL restricts topic subscriptions — response and `to_robot` topics only readable by the robot whose UUID matches
  - `bridge_hook.go`: Event bus bridge forwards only `robomesh/message/*` → internal event bus (auth/heartbeat protocol messages are excluded)
- **UDP** (`udp_server/`): JSON packet-based protocol for IoT devices (default port 5001).
  - All communication uses self-contained JSON packets with a `type` field
  - Auth: Two-step challenge-response (same as MQTT pattern). Step 1: `{"type":"auth","uuid":"..."}` → nonce. Step 2: `{"type":"auth","uuid":"...","nonce":"...","signature":"..."}` → JWT.
  - Heartbeat: `{"type":"heartbeat","uuid":"...","payload":"...","signature":"..."}` — signed heartbeat (same verification as TCP/HTTP)
  - Message: `{"type":"message","uuid":"...","jwt":"...","payload":"..."}` — JWT-authenticated messages forwarded to handler
  - Responses are JSON with `type`, `status`, and optional `nonce`/`jwt`/`error` fields
- **Terminal** (`terminal/`): Interactive CLI for debugging.

### Heartbeat Protocol

Robots send heartbeats independently of handler sessions. The heartbeat is a signed JSON payload verified against the robot's stored public key.

**TCP format:** `HEARTBEAT <UUID> <payloadJSON> <signatureHex>`
**HTTP format:** `POST /heartbeat` with `{"uuid": "...", "payload": "...", "signature": "..."}`
**UDP format:** `{"type":"heartbeat","uuid":"...","payload":"...","signature":"..."}`

**Payload fields:**
- `seq` (required): Monotonically increasing sequence number (replay protection)
- `ttl` (optional): Custom TTL in seconds (for battery saving)
- `extra_data` (optional): Arbitrary JSON data

The server verifies the signature, checks sequence > last seen, and stores state in Redis with the specified TTL. Heartbeat events are published on `robot.{uuid}.heartbeat` for handlers with `forward_heartbeats` enabled.

### Handler Communication Protocol

Handlers are bash scripts at `handlers/{type}/start_handler.sh` that communicate via JSON on stdin/stdout.

**Incoming (stdin):**
- `{"type":"connect","uuid":"...","device_type":"...","ip":"...","session_id":"..."}`
- `{"type":"incoming","uuid":"...","payload":"..."}`
- `{"type":"disconnect","uuid":"...","reason":"..."}`
- `{"type":"event","event_type":"...","data":{...}}`
- `{"type":"heartbeat","event_type":"...","data":{...}}`

**Outgoing (stdout) — JSON-RPC:**
- `{"target":"robot","id":"1","data":{...}}` — Send to robot
- `{"target":"database","id":"2","method":"get_robot","data":"uuid"}` — Query robot by UUID
- `{"target":"database","id":"3","method":"list_robots"}` — List all robots
- `{"target":"database","id":"4","method":"get_robots_by_type","data":"device_type"}` — Filter by type
- `{"target":"database","id":"5","method":"store_data","data":{"key":"k","value":"v"}}` — Store custom data
- `{"target":"database","id":"6","method":"get_data","data":"key"}` — Retrieve custom data
- `{"target":"database","id":"7","method":"delete_data","data":"key"}` — Delete custom data
- `{"target":"event_bus","method":"event.name","data":{...}}` — Publish event
- `{"target":"config","method":"forward_heartbeats","data":true}` — Enable heartbeat forwarding
- `{"target":"config","method":"subscribe","data":"event.type"}` — Subscribe to bus events
- `{"target":"connect_robot","data":{"port":8888,"protocol":"tcp"}}` — Initiate reverse connection

### Frontend

**SvelteKit 2.16** with Svelte 5 (runes mode), TypeScript strict mode, Tailwind CSS 4.1, Vite 6.2.

**Route groups**: `(auth)/` for unauthenticated pages, `(app)/` for protected pages (`/robots`, `/provision`, `/settings`).

**Auth flow**: POST `/auth/login` → JWT stored in `localStorage['auth-token']` → `fetchBackend()` wrapper auto-injects `Authorization: Bearer` header. JWT validated against Redis on each request. Password change via POST `/auth/password`. Passwords are validated 8–72 characters (bcrypt limit) on both login and change.

**Backend URL** (`lib/backend/fetch.ts`): `backendBaseUrl()` centralizes backend URL construction, deriving the protocol from `window.location.protocol` so both HTTP and HTTPS work without extra config. Used by `fetchBackend()`, `EventSourceManager`, plugin loader, and layout auth.

**Robot registry** (`lib/robots/registry.ts`): Maps robot types to Svelte components. Built-in types registered statically. Plugin types loaded dynamically via `getRobotComponentAsync()`.

**Plugin loader** (`lib/robots/plugin-loader.ts`): Dynamically imports `robot_card.js` and `robot_handler.js` from `/plugins/{type}/` endpoint. Cached after first load.

**Handler controls**: Robot cards show handler status (on/off) and provide Start/Kill buttons. Detail pages load plugin handler pages with tabbed navigation.

**SSE**: `EventSourceManager` (singleton) handles real-time robot status updates. Events are sent as single JSON envelopes (`{id, type, data}`) on SSE data lines.

**WebSocket**: Bidirectional communication via `/ws`. Actions: `subscribe`, `unsubscribe`, `send_to_robot` (forwards data to robot's TCP/MQTT connection), `send_to_handler` (forwards data to handler stdin).

**Robot search/filter**: Robots page supports filtering by name, UUID, IP, type via search bar and device type dropdown filter.

## Adding a New Robot Type

1. Copy `handlers/_template/` to `handlers/{device_type}/`
2. Edit `start_handler.sh` and implement your handler logic
3. (Optional) Build frontend components:
   ```bash
   cd handlers/{device_type}/frontend
   npm install && npm run build
   ```
4. Provision the robot's public key via `POST /provision`
5. Robot authenticates via TCP `AUTH` command (challenge-response)
6. Handler is auto-spawned. Can also be started manually via `POST /handler/{uuid}/start`

### Handler Directory Structure
```
handlers/
    _template/              # Boilerplate for new robot types
    {robot_type}/
        start_handler.sh    # Entry point (bash wrapper)
        handler.py          # Handler logic (any language)
        frontend/           # Optional micro-frontend
            src/
                RobotCard.svelte
                RobotHandler.svelte
            vite.config.js
            package.json
        dist/               # Compiled frontend assets (served via /plugins/)
            robot_card.js
            robot_handler.js
```

### Robot SDKs

Both SDKs live in `robot_sdk/` and implement TCP, UDP, and MQTT protocols (AUTH, REGISTER, PERSIST, HEARTBEAT, messaging).

**C SDK** (`robot_sdk/c/`) — Static libraries using OpenSSL for Ed25519 and POSIX sockets. Designed for embedded devices and low-level integration. Synchronous/blocking API. CMake build system.

- `librobomesh.a` — TCP client: `include/robomesh.h`, `src/robomesh.c`
- `librobomesh_udp.a` — UDP client: `include/robomesh_udp.h`, `src/robomesh_udp.c`
- `librobomesh_mqtt.a` — MQTT client (requires libmosquitto): `include/robomesh_mqtt.h`, `src/robomesh_mqtt.c`
- `tests/test_integration.c`: 8 integration tests (key gen/load, auth success/failure, provisioned auth, heartbeat, messaging)
- `examples/test_robot.c`: TCP demo, `examples/test_robot_udp.c`: UDP demo, `examples/test_robot_mqtt.c`: MQTT demo

**Python SDK** (`robot_sdk/python/`) — Higher-level clients with background threads for heartbeat and message receiving. Uses `cryptography` library for Ed25519.

- `robomesh_sdk/client.py`: `RobotClient` — TCP client with `on_message()` callback and `start_heartbeat()` background loop
- `robomesh_sdk/udp_client.py`: `RobotUDPClient` — UDP client with JSON packet-based protocol
- `robomesh_sdk/mqtt_client.py`: `RobotMQTTClient` — MQTT client (requires `paho-mqtt>=2.0`, install with `pip install robomesh-sdk[mqtt]`)
- `robomesh_sdk/keys.py`: Ed25519 key generation, loading, signing
- `robomesh_sdk/admin.py`: Admin API helpers (provision, registration approval)
- `tests/test_udp_unit.py`, `tests/test_mqtt_unit.py`: Unit tests with mocked transport layers (29 tests total)

**Key differences:** Python SDK has background heartbeat thread (`start_heartbeat()`), message callback (`on_message()`), and admin API wrapper. C SDK is synchronous-only with manual heartbeat calls. MQTT is optional in both SDKs (requires external library).

## Naming Conventions

- **Go**: Interfaces over concrete types; embed `BaseRobot` in specific robot structs
- **Database**: snake_case for tables/columns
- **Frontend components**: PascalCase; utility files: kebab-case
- **Handler directories**: `{device_type}/` in `handlers/`; entry point: `start_handler.sh`
