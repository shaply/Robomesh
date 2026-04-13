# UDP Protocol

JSON packet-based protocol for lightweight IoT devices. All communication uses self-contained JSON packets with a `type` field for routing. Default port: **5001**.

## Packet Format

Every UDP packet is a single JSON object:

```json
{
  "type": "auth|heartbeat|message",
  "uuid": "robot-001",
  ...
}
```

Responses follow the same pattern:

```json
{
  "type": "auth_response|heartbeat_response|message_response",
  "status": "ok|nonce|error",
  ...
}
```

## Authentication

Two-step challenge-response, identical to the MQTT pattern:

### Step 1: Request Nonce

```json
// Robot sends:
{"type":"auth","uuid":"robot-001"}

// Server responds:
{"type":"auth_response","status":"nonce","nonce":"<hex>"}
```

### Step 2: Verify Signature

```json
// Robot sends:
{"type":"auth","uuid":"robot-001","nonce":"<hex>","signature":"<hex>"}

// Server responds (success):
{"type":"auth_response","status":"ok","jwt":"<jwt_token>"}

// Server responds (failure):
{"type":"auth_response","status":"error","error":"signature verification failed"}
```

**Signing:** The robot signs the raw nonce bytes (decoded from hex) with its Ed25519 private key. The signature is hex-encoded.

On success, the server:
1. Issues a session JWT
2. Stores the active session in Redis
3. Spawns or reattaches a handler process

## Heartbeat

Signed heartbeats keep the robot's session alive without re-authenticating.

```json
// Robot sends:
{
  "type": "heartbeat",
  "uuid": "robot-001",
  "payload": {"seq": 1, "ttl": 120, "extra_data": {"battery": 85}},
  "signature": "<hex>"
}

// Server responds:
{"type":"heartbeat_response","status":"ok"}
```

**Key differences from MQTT heartbeat:**
- `payload` is a **JSON object** (not a string) — the server reads it as `json.RawMessage`
- The signature is computed over the **compact JSON serialization** of the payload (e.g., `{"seq":1,"ttl":120}`)
- No JWT is required — the signature authenticates the robot

See [HEARTBEAT.md](HEARTBEAT.md) for payload fields and verification flow.

## Messaging

JWT-authenticated messages forwarded to the robot's handler:

```json
// Robot sends:
{
  "type": "message",
  "uuid": "robot-001",
  "jwt": "<session_jwt>",
  "payload": "hello from robot"
}

// Server responds:
{"type":"message_response","status":"ok"}
```

The server validates the JWT and checks that `claims.sub == uuid`. If no handler is running, an error is returned.

## Error Responses

All errors follow the same format:

```json
{"type":"<type>_response","status":"error","error":"<description>"}
```

Common errors:
- `unknown robot` — UUID not found in PostgreSQL
- `blacklisted` — Robot has been blacklisted
- `no pending nonce` — Step 2 sent without Step 1, or nonce expired (30s TTL)
- `signature verification failed` — Nonce signature doesn't match public key
- `invalid or mismatched JWT` — JWT expired, invalid, or doesn't match UUID
- `no handler running` — No handler process for this robot

## SDK Support

| SDK | Module | Status |
| --- | --- | --- |
| Python | `robomesh_sdk.udp_client.RobotUDPClient` | Full support |
| C | `robomesh_udp.h` / `librobomesh_udp.a` | Full support |

## Configuration

```yaml
server:
  udp_port: 5001   # Default UDP port
```

Env var override: `UDP_PORT`
