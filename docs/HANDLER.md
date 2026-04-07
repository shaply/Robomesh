# Handler Script Protocol

Handler scripts (`handlers/{device_type}/start_handler.sh`) communicate with the Go backend via JSON on stdin/stdout. They are spawned per-robot and managed by the global `HandlerManager`.

## Lifecycle

- Handlers are spawned when a robot authenticates (AUTH or REGISTER) or manually via `POST /handler/{uuid}/start`
- Handlers **survive TCP disconnect** — they receive a `disconnect` message but keep running
- Handlers are killed via `POST /handler/{uuid}/kill`, server shutdown, or process exit
- Each robot has at most one handler running at a time

## Directory Structure

```text
handlers/
    {device_type}/
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

Device types are validated against `[a-zA-Z0-9_-]{1,64}`.

## Environment Variables

Handler scripts receive:

| Variable | Description |
| --- | --- |
| `ROBOT_UUID` | Robot identifier |
| `ROBOT_DEVICE_TYPE` | Device type |
| `ROBOT_IP` | Remote IP address |
| `ROBOT_SESSION_ID` | Session identifier |

## Messages Received on stdin

Each line is a JSON object with a `type` field:

```json
{"type": "connect", "uuid": "robot-001", "device_type": "example_robot", "ip": "192.168.1.50", "session_id": "abc123"}
{"type": "incoming", "uuid": "robot-001", "payload": "hello world"}
{"type": "disconnect", "uuid": "robot-001", "reason": "tcp_closed"}
{"type": "event", "event_type": "some.event", "data": {...}}
{"type": "heartbeat", "event_type": "robot.robot-001.heartbeat", "data": {...}}
```

| Type | Description |
| --- | --- |
| `connect` | Sent once when handler spawns |
| `incoming` | Forwarded from the robot's TCP connection |
| `disconnect` | TCP connection closed (handler keeps running) or handler being killed |
| `event` | Events from subscribed event bus topics |
| `heartbeat` | Heartbeat events (only if `forward_heartbeats` is enabled via config) |

## Requests Written to stdout (JSON-RPC)

Handlers write JSON-RPC envelopes to stdout. The Go backend routes them by `target`:

### Send message to robot

```json
{"target": "robot", "id": "1", "data": "message to send"}
```

### Query database

```json
{"target": "database", "id": "2", "method": "get_robot", "data": "robot-001"}
```

### Publish event

```json
{"target": "event_bus", "method": "sensor_update", "data": {"temp": 22.5}}
```

### Configure handler

```json
{"target": "config", "id": "3", "method": "forward_heartbeats", "data": true}
{"target": "config", "id": "4", "method": "subscribe", "data": "sensor.updates"}
```

| Config Method | Data Type | Description |
| --- | --- | --- |
| `forward_heartbeats` | `bool` | Enable/disable heartbeat event forwarding |
| `subscribe` | `string` | Subscribe to an arbitrary event bus topic |

### Request reverse connection to robot

```json
{"target": "connect_robot", "id": "5", "data": {"port": 8888, "protocol": "tcp"}}
```

See [Reverse Connection Flow](#reverse-connection-flow) below.

### Responses

Database, config, and reverse connection requests receive responses on stdin:

```json
{"target": "response", "id": "req1", "data": {...}, "error": ""}
```

The `error` field is empty on success. On failure, `data` may be `null` and `error` contains the reason.

## Reverse Connection Flow

When a handler requests a reverse connection, the server dials the robot and bridges I/O. Only one reverse connection per handler is allowed at a time.

### TCP Reverse Connect

```text
    Server                         Robot
      |                               |
      |---- TCP Connect (port) ------>|
      |---- ROBOSERVER_CONNECT {uuid}>|
      |<--- CONNECT_OK ---------------|
      |---- NONCE {random_hex} ------>|
      |<--- {signature_hex} ----------|
      |  Verify(sig, pubkey, nonce)   |
      |---- AUTH_OK ----------------->|
      |                               |
      |  (Bidirectional I/O bridge)   |
```

### UDP Reverse Connect

Direct UDP connection — no handshake. The server sends/receives datagrams on the specified port.

### IP Resolution

The server resolves the robot's IP in order:

1. Heartbeat state (`robot:{uuid}:heartbeat`)
2. Active session (`robot:{uuid}:active`)
3. Spawn-time IP (from the `connect` message)

### Response

On success: `{"target":"response","id":"5","data":"connected"}`

If a connection already exists: `{"target":"response","id":"5","data":null,"error":"robot already connected"}`
