package event_bus

import "github.com/google/uuid"

func NewSubscriber() *Subscriber {
	return &Subscriber{
		ID: uuid.New().String(), // Generate a new unique ID for the subscriber
	}
}
