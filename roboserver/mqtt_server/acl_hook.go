package mqtt_server

import (
	"bytes"
	"strings"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

// robotACLHook allows all MQTT CONNECT credentials (identity is verified at
// the application layer via challenge-response on auth topics) but restricts
// topic subscriptions so that clients can only subscribe to response topics
// matching their own client ID (which must be the robot UUID).
//
// Publish (write) is unrestricted — the protocol hook validates payloads.
// Subscribe (read) rules:
//   - robomesh/auth/{uuid}/response  → only if uuid == client ID
//   - robomesh/heartbeat/{uuid}/response → only if uuid == client ID
//   - robomesh/to_robot/{uuid}       → only if uuid == client ID
//   - all other topics               → allowed (e.g. publishing to auth/heartbeat/message)
type robotACLHook struct {
	mqtt.HookBase
}

func (h *robotACLHook) ID() string {
	return "robot-acl"
}

func (h *robotACLHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnectAuthenticate,
		mqtt.OnACLCheck,
	}, []byte{b})
}

// OnConnectAuthenticate allows all connections — robot identity is verified
// via the challenge-response protocol on robomesh/auth/ topics.
func (h *robotACLHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	return true
}

// OnACLCheck restricts subscribe access on response and to_robot topics to the
// client's own UUID. Publish access is unrestricted.
func (h *robotACLHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	// Publish (write) is always allowed — protocol hook validates content
	if write {
		return true
	}

	// For subscribes (reads), restrict sensitive per-robot topics
	clientID := cl.ID

	// robomesh/auth/{uuid}/response — restrict to own UUID
	if strings.HasPrefix(topic, "robomesh/auth/") && strings.HasSuffix(topic, "/response") {
		uuid := strings.TrimPrefix(topic, "robomesh/auth/")
		uuid = strings.TrimSuffix(uuid, "/response")
		return uuid == clientID
	}

	// robomesh/heartbeat/{uuid}/response — restrict to own UUID
	if strings.HasPrefix(topic, "robomesh/heartbeat/") && strings.HasSuffix(topic, "/response") {
		uuid := strings.TrimPrefix(topic, "robomesh/heartbeat/")
		uuid = strings.TrimSuffix(uuid, "/response")
		return uuid == clientID
	}

	// robomesh/to_robot/{uuid} — restrict to own UUID
	if strings.HasPrefix(topic, "robomesh/to_robot/") {
		uuid := strings.TrimPrefix(topic, "robomesh/to_robot/")
		return uuid == clientID
	}

	return true
}
