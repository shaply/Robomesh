# HTTP API

All routes except public ones require `Authorization: Bearer <token>` header or `session-token` cookie.

Base URL: `http://{host}:{http_port}` (default port 8080).

## Authentication

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/auth/login` | Public | Login with username/password, returns JWT. Rate limited: 5 attempts / 5 minutes per IP. |
| `POST` | `/auth/logout` | JWT | Invalidate session (removes from Redis) |
| `GET` | `/auth` | JWT | Check if current token is valid |
| `POST` | `/auth/ticket` | JWT | Get a short-lived single-use ticket for SSE (30s TTL) |
| `POST` | `/auth/password` | JWT | Change password: `{current_password, new_password}` (8-72 chars) |

**Token extraction:** Authorization header (`Bearer <token>`) or cookie (`session-token`). Query parameters are **not** accepted for JWTs.

**Session validation:** JWT is validated and session existence is verified against Redis. Tokens are invalid immediately after logout.

### Login

```text
POST /auth/login
Content-Type: application/json
```

```json
{"username": "admin", "password": "..."}
```

**Response (200):**

```json
{"status": "success", "message": "Logged in successfully", "token": "<jwt>"}
```

**Rate limiting:** 5 failed attempts per IP within a 5-minute window results in `429 Too Many Requests`.

### Ticket Exchange (for SSE)

Prevents JWT exposure in URLs. Used by the frontend before connecting EventSource.

```text
POST /auth/ticket
Authorization: Bearer <jwt>
```

**Response (200):**

```json
{"ticket": "<random_hex>"}
```

Tickets are valid for 30 seconds and consumed on first use.

## Heartbeat

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/heartbeat` | Signature | Submit a signed heartbeat (public — robots authenticate via signature) |

See [HEARTBEAT.md](HEARTBEAT.md) for payload format and verification flow.

## Active Robots (Redis)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/robot` | JWT | List all active robots |
| `GET` | `/robot/{uuid}` | JWT | Get active robot detail (IP, type, PID, connected_at) |

## Robot Registry (PostgreSQL)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/provision` | JWT | List all registered robots |
| `GET` | `/provision/{uuid}` | JWT | Get registered robot detail |
| `POST` | `/provision` | JWT | Provision a robot: `{uuid, public_key, device_type}` |
| `POST` | `/provision/{uuid}/blacklist` | JWT | Set blacklist status: `{blacklisted: true/false}` |
| `GET` | `/provision/{uuid}/status` | JWT | Check robot's active session status in Redis |

### Provision a Robot

```text
POST /provision
Content-Type: application/json
```

```json
{"uuid": "robot-001", "public_key": "<hex>", "device_type": "sensor"}
```

## Registration Approval

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/register/pending` | JWT | List all pending registrations |
| `POST` | `/register` | JWT | Accept/reject: `{uuid, accept: true/false}` |

## Handler Lifecycle

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/handler/` | JWT | List all running handlers (UUID -> PID map) |
| `GET` | `/handler/{uuid}` | JWT | Get handler status: `{uuid, active, pid, device_type}` |
| `POST` | `/handler/{uuid}/start` | JWT | Manually spawn a handler (even without TCP connection) |
| `POST` | `/handler/{uuid}/kill` | JWT | Kill a running handler process |
| `GET` | `/handler/{uuid}/logs` | JWT | SSE stream of handler stdout/stderr log lines |

Handlers survive TCP disconnects. They can be started/killed independently via these endpoints.

## WebSocket

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/ws` | JWT | WebSocket connection for bidirectional event streaming |

## Ephemeral Sessions

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `POST` | `/ephemeral` | JWT | Create an ephemeral session directly |
| `DELETE` | `/ephemeral/{uuid}` | JWT | Remove an ephemeral session |

## Events (SSE)

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/events?events=type1,type2&ticket=...` | Ticket | SSE stream. Uses single-use ticket from `/auth/ticket`. |
| `POST` | `/events/subscribe` | JWT | Subscribe an existing SSE client to additional events |
| `POST` | `/events/unsubscribe` | JWT | Unsubscribe an SSE client from events |

### SSE Connection Flow

1. Frontend sends `POST /auth/ticket` with JWT in Authorization header
2. Server returns `{"ticket": "<random_hex>"}` (valid for 30 seconds, single-use)
3. Frontend connects `EventSource` with `?events=type1,type2&ticket=<ticket>`
4. Server consumes and deletes ticket on first use
5. Server sends initial `sessID` event with client session ID
6. Events stream as JSON envelopes on SSE data lines:

```json
{"id": "evt-1", "type": "robot.registering", "data": "<json_string>"}
```

## Plugin System

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| `GET` | `/plugins/` | Public | List available handler plugin types |
| `GET` | `/plugins/{type}/*` | Public | Serve compiled handler frontend assets from `handlers/{type}/dist/` |

Plugin assets are compiled Svelte components (`robot_card.js`, `robot_handler.js`) loaded dynamically by the frontend via ES module imports. Plugin cache in the frontend has a 5-minute TTL.
