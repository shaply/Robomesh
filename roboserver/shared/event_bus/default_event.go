package event_bus

func NewDefaultEvent(eventType string, data interface{}) *DefaultEvent {
	return &DefaultEvent{
		Type: eventType,
		Data: data,
	}
}

func (e *DefaultEvent) GetType() string {
	return e.Type
}

func (e *DefaultEvent) GetData() interface{} {
	return e.Data
}
