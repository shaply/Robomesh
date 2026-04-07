# Heartbeat Protocol

Robots send signed heartbeats independently of handler sessions. This keeps the robot "online" in Redis without requiring a persistent TCP connection.

Heartbeats are decoupled from handler lifecycle — a robot can heartbeat without a handler running, and a handler can run without heartbeats.

## Transport

**TCP format:**

```text
HEARTBEAT <UUID> <payloadJSON> <signatureHex>
```

**HTTP format:**

```text
POST /heartbeat
Content-Type: application/json
```

```json
{"uuid": "...", "payload": "<payloadJSON>", "signature": "<signatureHex>"}
```

Both transports are public (no JWT required) — the signature itself authenticates the robot.

## Payload

```json
{
  "seq": 42,           // Required: monotonically increasing (replay protection)
  "ttl": 120,          // Optional: custom TTL in seconds (for battery saving)
  "extra_data": {}     // Optional: arbitrary JSON data
}
```

- `seq` must be strictly greater than the last seen sequence number. Out-of-order or replayed heartbeats are rejected.
- `ttl` controls how long the heartbeat state lives in Redis. Defaults to the configured `session_ttl` if omitted. Useful for battery-powered robots that heartbeat infrequently.
- `extra_data` is passed through to handlers via heartbeat events (e.g., battery level, sensor readings).

## Verification Flow

1. Robot signs the payload JSON string with its private key (Ed25519 or ECDSA)
2. Server looks up robot's public key in PostgreSQL
3. Server verifies signature against public key
4. Server checks `seq > last_seen_seq` (replay protection)
5. Server stores heartbeat state in Redis (`robot:{uuid}:heartbeat`) with specified TTL
6. Server refreshes active session TTL if one exists (`robot:{uuid}:active`)
7. Server publishes heartbeat event on `robot.{uuid}.heartbeat` for handlers

## Redis State

Heartbeat state stored at `robot:{uuid}:heartbeat`:

```json
{
  "uuid": "robot-001",
  "ip": "192.168.1.50",
  "last_seq": 42,
  "last_seen": 1711584000
}
```

## Handler Forwarding

Handlers can opt into receiving heartbeat events by sending a config request:

```json
{"target": "config", "method": "forward_heartbeats", "data": true}
```

When enabled, heartbeats arrive on stdin as:

```json
{"type": "heartbeat", "event_type": "robot.robot-001.heartbeat", "data": {...}}
```

## Responses

- **TCP:** `HEARTBEAT_OK` on success, `ERROR <reason>` on failure.
- **HTTP:** `200 OK` with `{"status": "ok"}` on success, `4xx`/`5xx` with error message on failure.
