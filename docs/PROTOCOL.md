# Robomesh Protocol Reference

This document covers all communication protocols in Robomesh: TCP robot flows, handler script JSON-RPC, HTTP API, and terminal commands.

## TCP Protocol

Robots connect via TCP and send either `AUTH` (pre-registered) or `REGISTER` (new robot). Both flows end in an authenticated session where the handler script is spawned.

### AUTH Flow (Pre-Registered Robots)

For robots already stored in PostgreSQL via `POST /provision` or the PERSIST flow.

```text
    Robot                          Server (Go)
      │                               │
      │──── TCP Connect ─────────────>│
      │──── AUTH ────────────────────>│
      │<─── AUTH_CHALLENGE ───────────│
      │                               │
      │──── UUID ────────────────────>│
      │                               │ Look up PublicKey in PostgreSQL
      │                               │ Check IsBlacklisted = false
      │<─── NONCE {random_hex} ───────│
      │                               │
      │  Sign(nonce, PrivateKey)      │
      │──── {signature_hex} ─────────>│
      │                               │ Verify(signature, PublicKey, nonce)
      │                               │ Issue JWT + store in Redis
      │<─── AUTH_OK {jwt} ────────────│
      │                               │
      │  (Authenticated session)      │ Start heartbeat loop
      │                               │ Spawn handler process
```

**Error responses:** `ERROR NO_DATABASE`, `ERROR UNKNOWN_ROBOT`, `ERROR BLACKLISTED`, `ERROR INVALID_SIGNATURE`

### REGISTER Flow (New Robots)

For robots not yet in PostgreSQL. Stored ephemerally in Redis pending user approval. Duplicate UUIDs rejected (checked against both Redis active, Redis pending, and PostgreSQL).

```text
    Robot                          Server (Go)            User (Frontend/Terminal)
      │                               │                          │
      │──── TCP Connect ─────────────>│                          │
      │──── REGISTER ────────────────>│                          │
      │<─── REGISTER_CHALLENGE ───────│                          │
      │                               │                          │
      │──── UUID ────────────────────>│                          │
      │<─── SEND_DEVICE_TYPE ─────────│                          │
      │──── {device_type} ───────────>│                          │
      │<─── SEND_PUBLIC_KEY ──────────│                          │
      │──── {public_key_hex} ────────>│                          │
      │                               │ Store in Redis (pending) │
      │<─── REGISTER_PENDING ─────────│                          │
      │                               │                          │
      │  (Robot blocks, waiting...)   │   GET /register/pending  │
      │                               │<─────────────────────────│
      │                               │  POST /register          │
      │                               │<── {uuid, accept: true} ─│
      │                               │                          │
      │                               │ comms.Bus pub/sub notify │
      │<─── REGISTER_OK {jwt} ────────│                          │
      │                               │ Spawn handler process    │
```

**On rejection:** `REGISTER_REJECTED` is sent and the connection closes.

**Timeout:** Pending registrations expire after 5 minutes. Robot receives `ERROR REGISTRATION_TIMEOUT`.

### PERSIST Flow (Ephemeral to Permanent)

A registered (ephemeral) robot can promote itself to the PostgreSQL registry during an active session.

```text
    Robot                          Server (Go)
      │                               │
      │  (In active session)          │
      │──── PERSIST ─────────────────>│
      │                               │ Copy public key from Redis → PostgreSQL
      │<─── PERSIST_OK ───────────────│
```

If already persisted: `PERSIST_OK ALREADY_PERSISTED`. If no public key found in Redis: `ERROR NO_PUBLIC_KEY`.

### Session Mode

After AUTH or REGISTER succeeds, the connection enters session mode:

- All subsequent lines are forwarded to the handler script as `incoming` messages
- The `PERSIST` command is intercepted before reaching the handler (for REGISTER-originated sessions)
- A Redis heartbeat loop keeps the session alive
- On disconnect, the handler process is torn down and the Redis session removed

### Error Format

All errors follow: `ERROR <CODE> [detail]`

## Handler Script Protocol

Handler scripts (`handlers/{device_type}.sh`) communicate with Go via JSON on stdin/stdout. They are spawned per-robot and torn down on disconnect.

### Messages Received on stdin

```json
{"type": "connect", "uuid": "robot-001", "device_type": "example_robot", "ip": "192.168.1.50", "session_id": "abc123"}
{"type": "incoming", "uuid": "robot-001", "payload": "hello world"}
{"type": "disconnect", "uuid": "robot-001", "reason": "client_disconnect"}
```

### Requests Written to stdout

Handlers write JSON-RPC envelopes to stdout. Go routes them by `target`:

**Send message to robot:**

```json
{"target": "robot", "data": "message to send to robot"}
```

**Query database:**

```json
{"target": "database", "id": "req1", "method": "get_robot", "data": "robot-001"}
```

**Publish event:**

```json
{"target": "event_bus", "method": "sensor_update", "data": {"temp": 22.5}}
```

### Responses

Database requests receive responses on stdin:

```json
{"target": "response", "id": "req1", "data": {...}, "error": ""}
```

### Environment Variables

Handler scripts receive:

- `ROBOT_UUID` — robot identifier
- `ROBOT_DEVICE_TYPE` — device type
- `ROBOT_IP` — remote IP address
- `ROBOT_SESSION_ID` — session identifier

## HTTP API

All routes except `/auth/*` require `Authorization: Bearer <token>` header.

### Authentication

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/auth/login` | Login with username/password, returns JWT |

### Active Robots (Redis)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/robot` | List all active robots |
| `GET` | `/robot/{uuid}` | Get active robot detail (IP, type, PID, connected_at) |

### Robot Registry (PostgreSQL)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/provision` | List all registered robots |
| `GET` | `/provision/{uuid}` | Get registered robot detail |
| `POST` | `/provision` | Provision a robot: `{uuid, public_key, device_type}` |
| `PUT` | `/provision/{uuid}/blacklist` | Set blacklist status: `{blacklisted: true/false}` |

### Registration Approval

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/register/pending` | List all pending registrations |
| `POST` | `/register` | Accept/reject: `{uuid, accept: true/false}` |

### Ephemeral Sessions

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/ephemeral` | Create an ephemeral session directly |
| `DELETE` | `/ephemeral/{uuid}` | Remove an ephemeral session |

### Events (SSE)

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/events?events=type1,type2` | SSE stream, subscribe to event types |
| `POST` | `/events/subscribe` | Subscribe an existing SSE client to events |
| `POST` | `/events/unsubscribe` | Unsubscribe an SSE client from events |

## Terminal Commands

The debug terminal server (default port 6000) accepts TCP connections with a line-based CLI.

| Command | Description |
| --- | --- |
| `list` | List active robots (from Redis) |
| `robots` | List registered robots (from PostgreSQL) |
| `pending` | List pending robot registrations |
| `accept <uuid>` | Accept a pending registration |
| `reject <uuid>` | Reject a pending registration |
| `status <uuid>` | Get robot online status |
| `stop program` | Shut down the server |
| `subscribe <event>` | Subscribe to event type |
| `unsubscribe <event>` | Unsubscribe from event type |
| `publish <event> <data>` | Publish an event |
| `help [command]` | Show available commands |
| `exit` / `quit` | Close terminal session |

## Communication Bus (`comms.Bus`)

All inter-service communication flows through the `comms.Bus` interface. Services never import each other directly.

```go
type Bus interface {
    PublishEvent(eventType string, data any) error
    SubscribeEvent(eventType string, handler EventHandler) (cancel func(), err error)
    PublishRegistrationResponse(ctx context.Context, uuid string, accepted bool) error
    WaitForRegistrationResponse(ctx context.Context, uuid string) (bool, error)
}
```

**Current implementation:** `LocalBus` — in-process event bus + Redis pub/sub.

**Migration path:** Implement `Bus` with Kafka/gRPC/NATS. No service code changes required.
