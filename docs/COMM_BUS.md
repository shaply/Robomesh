# Communication Bus

All inter-service communication flows through the `comms.Bus` interface. Services never import each other directly — they publish and subscribe to events through the bus.

## Interface

```go
type Bus interface {
    PublishEvent(eventType string, data any) error
    SubscribeEvent(eventType string, handler EventHandler) (cancel func(), err error)
    PublishToGroup(group string, eventType string, data any) error
    SubscribeAsGroup(group string, eventType string, handler EventHandler) (cancel func(), err error)
    PublishRegistrationResponse(ctx context.Context, uuid string, accepted bool) error
    WaitForRegistrationResponse(ctx context.Context, uuid string) (bool, error)
}
```

- `PublishToGroup` sends an event that only one subscriber in the named consumer group receives (round-robin).
- `SubscribeAsGroup` joins a consumer group for load-balanced event processing.

## Current Implementation

`LocalBus` — wraps the in-process event bus (`shared/event_bus/`) + Redis pub/sub for cross-process communication.

The event bus uses SafeMap-based subscriptions with a buffer size of 1000 events per subscriber.

## Migration Path

To scale beyond a single process, implement the `Bus` interface with Kafka, gRPC, NATS, or any other messaging system. No service code changes required — only the bus implementation needs to change.

## Standard Event Topics

| Topic Pattern | Publisher | Subscriber | Description |
| --- | --- | --- | --- |
| `robot.registering` | TCP server | Frontend (SSE), Terminal | New robot requesting registration |
| `robot.{uuid}.heartbeat` | Heartbeat handler | Handlers (opt-in) | Robot heartbeat received |
| `handler.{uuid}.message` | HTTP API, other handlers | Target handler | Directed message to a specific handler |

## Usage in Handlers

Handlers can interact with the bus via JSON-RPC on stdout:

**Publish an event:**

```json
{"target": "event_bus", "method": "sensor_update", "data": {"temp": 22.5}}
```

**Subscribe to events (via config):**

```json
{"target": "config", "method": "subscribe", "data": "sensor.updates"}
```

Subscribed events arrive on stdin as:

```json
{"type": "event", "event_type": "sensor.updates", "data": {...}}
```
