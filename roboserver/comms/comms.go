// Package comms abstracts inter-service communication.
//
// All methods block until the operation completes. For async, use `go`:
//
//	go bus.PublishEvent("robot.connected", data)
//
// Current implementation (LocalBus) uses the in-process event bus + Redis
// pub/sub. To migrate to Kafka/gRPC, implement the Bus interface and swap
// at initialization — no service code changes required.
package comms

import "context"

// Bus is the single abstraction all services use for communication.
// Services import comms.Bus instead of depending on each other.
type Bus interface {
	// PublishEvent broadcasts an event to all subscribers of the given type.
	PublishEvent(eventType string, data any) error

	// SubscribeEvent registers a handler called for each event of the given type.
	// Returns a cancel function that unsubscribes. The handler is called
	// asynchronously — long-running handlers should be aware of concurrency.
	SubscribeEvent(eventType string, handler EventHandler) (cancel func(), err error)

	// PublishRegistrationResponse sends an accept/reject decision for a
	// pending robot registration. Unblocks any corresponding
	// WaitForRegistrationResponse call.
	PublishRegistrationResponse(ctx context.Context, uuid string, accepted bool) error

	// WaitForRegistrationResponse blocks until a registration decision
	// arrives for the given UUID. Returns true if accepted.
	WaitForRegistrationResponse(ctx context.Context, uuid string) (bool, error)
}

// EventHandler is called when a subscribed event fires.
type EventHandler func(eventType string, data any)

// Event is a simple value type for events flowing through the system.
type Event struct {
	Type string
	Data any
}

// TODO: For microservice migration, add competing-consumer (point-to-point)
// delivery. When multiple instances of a service exist (e.g., N TCP servers),
// some events must reach exactly ONE instance — not broadcast to all.
//
// Proposed additions to Bus:
//
//   // PublishToGroup sends an event that only ONE subscriber in the named
//   // group will receive (competing consumers). Maps to Kafka consumer
//   // groups, NATS queue subscriptions, etc.
//   PublishToGroup(group string, eventType string, data any) error
//
//   // SubscribeAsGroup joins a consumer group. Only one member per group
//   // receives each published event.
//   SubscribeAsGroup(group string, eventType string, handler EventHandler) (cancel func(), err error)
//
// Current state: WaitForRegistrationResponse already achieves point-to-point
// via UUID-scoped Redis pub/sub channels, so this isn't needed yet. Add these
// methods when actually deploying multiple service instances behind Kafka/NATS.
