package comms

import (
	"context"
	"roboserver/database"
	"roboserver/shared/event_bus"
)

// LocalBus implements Bus using the in-process event bus and Redis pub/sub.
// This is the default for a monolith deployment. Replace with KafkaBus,
// GRPCBus, etc. when splitting into microservices.
type LocalBus struct {
	eb  event_bus.EventBus
	rds *database.RedisHandler
}

// NewLocalBus creates a Bus backed by the in-process event bus and Redis.
func NewLocalBus(eb event_bus.EventBus, rds *database.RedisHandler) *LocalBus {
	return &LocalBus{eb: eb, rds: rds}
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

func (b *LocalBus) PublishRegistrationResponse(ctx context.Context, uuid string, accepted bool) error {
	return b.rds.PublishRegistrationResponse(ctx, uuid, accepted)
}

func (b *LocalBus) WaitForRegistrationResponse(ctx context.Context, uuid string) (bool, error) {
	return b.rds.WaitForRegistrationResponse(ctx, uuid)
}
