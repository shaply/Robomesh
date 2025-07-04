// Package shared provides implementation methods for the DefaultMsg struct.
//
// This file contains the method implementations that make DefaultMsg conform
// to the Msg interface. These methods provide access to the message components
// in a standardized way across all message types in the system.
package shared

// GetMsg returns the primary message content or command.
//
// This method provides access to the main message field, which typically
// contains the command type or primary content of the message.
//
// Returns:
//   - string: The main message content/command
//
// Example:
//
//	msg := &DefaultMsg{Msg: "STATUS_CHECK"}
//	command := msg.GetMsg()  // Returns "STATUS_CHECK"
func (msg *DefaultMsg) GetMsg() string {
	return msg.Msg
}

// GetPayload returns the structured data payload of the message.
//
// This method provides access to additional data that accompanies the message.
// The payload can be any type and is typically used for command parameters,
// sensor data, or other structured information.
//
// Returns:
//   - any: The payload data (can be nil if no payload is attached)
//
// Example:
//
//	msg := &DefaultMsg{
//	    Msg: "MOVE",
//	    Payload: map[string]int{"x": 10, "y": 5},
//	}
//	data := msg.GetPayload()  // Returns the coordinate map
func (msg *DefaultMsg) GetPayload() any {
	return msg.Payload
}

// GetSource returns the identifier of the component that created this message.
//
// This method provides traceability by identifying the originating component,
// which is useful for debugging, routing, and logging purposes.
//
// Returns:
//   - string: The source component identifier (empty string if not set)
//
// Example:
//
//	msg := &DefaultMsg{
//	    Msg: "ALERT",
//	    Source: "health_monitor",
//	}
//	origin := msg.GetSource()  // Returns "health_monitor"
func (msg *DefaultMsg) GetSource() string {
	return msg.Source
}

// GetReplyChan returns the channel for sending responses to this message.
//
// This method provides access to the reply channel, which enables request-response
// patterns. The sender can listen on this channel for responses to their message.
//
// Returns:
//   - chan any: The reply channel (nil if no response is expected)
//
// Usage Pattern:
//
//	// Sender creates message with reply channel
//	replyChan := make(chan any, 1)
//	msg := &DefaultMsg{
//	    Msg: "GET_STATUS",
//	    ReplyChan: replyChan,
//	}
//
//	// Send message and wait for response
//	handler.SendMsg(msg)
//	response := <-replyChan
//
// Note: Reply channels should be buffered to prevent blocking the responder.
func (msg *DefaultMsg) GetReplyChan() chan any {
	return msg.ReplyChan
}
