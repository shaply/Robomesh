package comms

import (
	"context"
	"fmt"
	"roboserver/database"
	"roboserver/shared/event_bus"
	"sync"
	"sync/atomic"
)

// LocalBus implements Bus using the in-process event bus and Redis pub/sub.
// This is the default for a monolith deployment. Replace with KafkaBus,
// GRPCBus, etc. when splitting into microservices.
type LocalBus struct {
	eb  event_bus.EventBus
	rds *database.RedisHandler

	// Consumer groups for point-to-point delivery (round-robin in single-instance)
	groupsMu sync.RWMutex
	groups   map[string]*consumerGroup
}

// consumerGroupEntry wraps a handler with a stable ID for safe removal.
type consumerGroupEntry struct {
	id      uint64
	handler EventHandler
}

// consumerGroup tracks handlers for competing-consumer delivery.
type consumerGroup struct {
	mu       sync.Mutex
	handlers []*consumerGroupEntry
	counter  atomic.Uint64
	nextID   uint64
}

// NewLocalBus creates a Bus backed by the in-process event bus and Redis.
func NewLocalBus(eb event_bus.EventBus, rds *database.RedisHandler) *LocalBus {
	return &LocalBus{
		eb:     eb,
		rds:    rds,
		groups: make(map[string]*consumerGroup),
	}
}

func (b *LocalBus) PublishEvent(eventType string, data any) error {
	b.eb.PublishData(eventType, data)
	return nil
}

func (b *LocalBus) SubscribeEvent(eventType string, handler EventHandler) (func(), error) {
	sub := event_bus.NewSubscriber()
	b.eb.Subscribe(eventType, sub, func(event event_bus.Event) {
		handler(event.GetType(), event.GetData())
	})
	cancel := func() {
		b.eb.Unsubscribe(eventType, sub)
	}
	return cancel, nil
}

func groupKey(group, eventType string) string {
	return fmt.Sprintf("%s::%s", group, eventType)
}

func (b *LocalBus) PublishToGroup(group string, eventType string, data any) error {
	key := groupKey(group, eventType)

	b.groupsMu.RLock()
	cg, ok := b.groups[key]
	b.groupsMu.RUnlock()

	if !ok || len(cg.handlers) == 0 {
		return nil // no subscribers
	}

	cg.mu.Lock()
	n := len(cg.handlers)
	if n == 0 {
		cg.mu.Unlock()
		return nil
	}
	idx := cg.counter.Add(1) - 1
	entry := cg.handlers[idx%uint64(n)]
	cg.mu.Unlock()

	entry.handler(eventType, data)
	return nil
}

func (b *LocalBus) SubscribeAsGroup(group string, eventType string, handler EventHandler) (func(), error) {
	key := groupKey(group, eventType)

	b.groupsMu.Lock()
	cg, ok := b.groups[key]
	if !ok {
		cg = &consumerGroup{}
		b.groups[key] = cg
	}
	b.groupsMu.Unlock()

	cg.mu.Lock()
	cg.nextID++
	entryID := cg.nextID
	cg.handlers = append(cg.handlers, &consumerGroupEntry{id: entryID, handler: handler})
	cg.mu.Unlock()

	cancel := func() {
		cg.mu.Lock()
		defer cg.mu.Unlock()
		for i, entry := range cg.handlers {
			if entry.id == entryID {
				cg.handlers = append(cg.handlers[:i], cg.handlers[i+1:]...)
				return
			}
		}
	}
	return cancel, nil
}

func (b *LocalBus) PublishRegistrationResponse(ctx context.Context, uuid string, accepted bool) error {
	return b.rds.PublishRegistrationResponse(ctx, uuid, accepted)
}

func (b *LocalBus) WaitForRegistrationResponse(ctx context.Context, uuid string) (bool, error) {
	return b.rds.WaitForRegistrationResponse(ctx, uuid)
}
