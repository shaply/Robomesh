package mqtt_server

import (
	"bytes"
	"roboserver/comms"
	"roboserver/shared"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

// eventBusBridgeHook bridges MQTT publish messages to the internal event bus.
// MQTT topic "robomesh/{event_type}" maps to event bus topic "{event_type}".
type eventBusBridgeHook struct {
	mqtt.HookBase
	bus comms.Bus
}

func (h *eventBusBridgeHook) ID() string {
	return "event-bus-bridge"
}

func (h *eventBusBridgeHook) Provides(b byte) bool {
	return bytes.Contains(
		[]byte{mqtt.OnPublished},
		[]byte{b},
	)
}

func (h *eventBusBridgeHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {
	topic := pk.TopicName
	payload := string(pk.Payload)

	// Forward MQTT messages with "robomesh/" prefix to the internal event bus
	const prefix = "robomesh/"
	if len(topic) > len(prefix) && topic[:len(prefix)] == prefix {
		eventType := topic[len(prefix):]
		if h.bus != nil {
			h.bus.PublishEvent(eventType, payload)
			shared.DebugPrint("MQTT→EventBus: %s → %s", topic, eventType)
		}
	}
}
