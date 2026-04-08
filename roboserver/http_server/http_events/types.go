package http_events

import "roboserver/shared"

type EventSession struct {
	Session   shared.Session `json:"session"`
	Timestamp int64          `json:"timestamp"`
	RandomID  string         `json:"random_id"`
}

type EventStruct struct {
	ESess      EventSession `json:"event_session"`
	EventTypes []string     `json:"event_types"`
}

// SentEvent is the JSON envelope sent over SSE.
// Data contains JSON-encoded event data as a string.
type SentEvent struct {
	Id   string `json:"id"`
	Type string `json:"type"`
	Data string `json:"data"`
}
