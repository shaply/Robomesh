# MQTT Protocol

Topic-based protocol using an embedded Mochi-MQTT broker. Default port: **1883**.

## Topic Structure

| Topic | Direction | Description |
| --- | --- | --- |
| `robomesh/auth/{uuid}` | Robot → Server | Auth requests (step 1 and step 2) |
| `robomesh/auth/{uuid}/response` | Server → Robot | Auth responses (nonce, JWT, errors) |
| `robomesh/heartbeat/{uuid}` | Robot → Server | Signed heartbeat payloads |
| `robomesh/heartbeat/{uuid}/response` | Server → Robot | Heartbeat acknowledgements |
| `robomesh/message/{uuid}` | Robot → Server | Messages forwarded to handler |
| `robomesh/to_robot/{uuid}` | Server → Robot | Messages from handler to robot |

## ACL (Access Control)

The broker enforces topic restrictions via a custom ACL hook:

- **Subscribe:** Robots can only subscribe to their own response and `to_robot` topics
- **Publish:** No restrictions on publish (protocol validation happens at the application layer)
- **Connection:** All MQTT connections are accepted — identity is verified via the challenge-response auth, not at the transport layer

## Authentication

Two-step challenge-response over MQTT publish/subscribe:

### Step 1: Request Nonce

Robot publishes to `robomesh/auth/{uuid}`:

```json
{"uuid": "robot-001"}
```

Server publishes to `robomesh/auth/{uuid}/response`:

```json
{"status": "nonce", "nonce": "<hex>"}
```

### Step 2: Verify Signature

Robot publishes to `robomesh/auth/{uuid}`:

```json
{"uuid": "robot-001", "signature": "<hex>", "nonce": "<hex>"}
```

Server publishes to `robomesh/auth/{uuid}/response`:

```json
{"status": "ok", "jwt": "<jwt_token>"}
```

**Signing:** Same as UDP/TCP — sign the raw nonce bytes (decoded from hex) with Ed25519.

The server caches the nonce + robot info (public key, device type) in Redis for 30s to avoid a double PostgreSQL lookup. Redis key: `mqtt:nonce:{uuid}`.

## Heartbeat

Robot publishes to `robomesh/heartbeat/{uuid}`:

```json
{
  "payload": "{\"seq\":1,\"ttl\":120}",
  "signature": "<hex>"
}
```

**Key difference from UDP:** The `payload` field is a **JSON string** (not a nested object). The server passes this string directly to `ProcessHeartbeat`.

**Signing:** The signature is computed over the raw bytes of the payload string (e.g., the UTF-8 bytes of `{"seq":1,"ttl":120}`).

Server publishes to `robomesh/heartbeat/{uuid}/response`:

```json
{"status": "ok"}
```

See [HEARTBEAT.md](HEARTBEAT.md) for payload fields and verification flow.

## Messaging

### Robot → Handler

Robot publishes raw payload to `robomesh/message/{uuid}`. The broker's bridge hook forwards the message to the internal event bus, which delivers it to the handler's stdin.

Messages are authorized by verifying the robot has an active session in Redis (completed the auth flow). No separate JWT is required per message.

### Handler → Robot

Handler sends a message via the JSON-RPC protocol:

```json
{"target": "robot", "id": "1", "data": {"action": "move"}}
```

The server publishes this to `robomesh/to_robot/{uuid}` via the outbound bridge.

## Event Bus Bridge

The `eventBusBridgeHook` bridges MQTT messages to the internal event bus:

- Only `robomesh/message/*` topics are bridged (auth and heartbeat protocol messages are excluded)
- Messages are published as `mqtt.message.{uuid}` events

## Error Handling

Errors are published as JSON to the relevant response topic:

```json
{"status": "error", "error": "unknown robot"}
```

## SDK Support

| SDK | Module | Dependency |
| --- | --- | --- |
| Python | `robomesh_sdk.mqtt_client.RobotMQTTClient` | `paho-mqtt>=2.0` |
| C | `robomesh_mqtt.h` / `librobomesh_mqtt.a` | `libmosquitto` |

## Configuration

```yaml
server:
  mqtt_port: 1883   # Default MQTT port
```

Env var override: `MQTT_PORT`
