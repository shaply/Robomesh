package http_events

import "roboserver/shared"

type Message struct {
	Type string
	Data *interface{}
}

type EventSession struct {
	Session   shared.Session `json:"session"`
	Timestamp int64          `json:"timestamp"`
	RandomID  string         `json:"random_id"`
}

type EventStruct struct {
	ESess      EventSession `json:"event_session"`
	EventTypes []string     `json:"event_types"`
}

type SentEvent struct {
	Id          string `json:"id"`
	Type        string `json:"type"`
	EncodedData string `json:"encoded_data"`
}
