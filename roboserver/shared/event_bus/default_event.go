package event_bus

func NewDefaultPtrEvent(eventType string, data *interface{}) *DefaultPtrEvent {
	return &DefaultPtrEvent{
		Type: eventType,
		Data: data,
	}
}

func (e *DefaultPtrEvent) GetType() string {
	return e.Type
}

func (e *DefaultPtrEvent) GetData() interface{} {
	return *e.Data
}

func (e *DefaultPtrEvent) GetDataPtr() *interface{} {
	return e.Data
}

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

func (e *DefaultEvent) GetDataPtr() *interface{} {
	return &e.Data
}
