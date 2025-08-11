package http_events

import (
	"roboserver/shared"
	"roboserver/shared/utils"
	"time"
)

func NewEventSession(session *shared.Session) *EventSession {
	return &EventSession{
		Session:   *session,
		Timestamp: time.Now().UnixMilli(),
		RandomID:  utils.GenerateRandomString(16), // Generate a random ID for the session
	}
}
