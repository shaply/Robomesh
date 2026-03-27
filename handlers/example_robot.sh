#!/bin/bash
# Handler script for example_robot device type.
#
# This script is spawned by the Go handler_engine when an example_robot
# authenticates via TCP. Communication with Go is via JSON on stdin/stdout.
#
# Environment variables set by Go:
#   ROBOT_UUID, ROBOT_DEVICE_TYPE, ROBOT_IP, ROBOT_SESSION_ID
#
# Protocol:
#   stdin  <- JSON messages from Go (connect, incoming, disconnect)
#   stdout -> JSON-RPC envelopes routed by Go (target: robot|database|event_bus)

log() {
    echo "[example_robot:$ROBOT_UUID] $1" >&2
}

# Send a message back to the robot via TCP
send_to_robot() {
    local msg="$1"
    echo "{\"target\":\"robot\",\"data\":\"$msg\"}"
}

# Publish an event to the event bus
publish_event() {
    local method="$1"
    local data="$2"
    echo "{\"target\":\"event_bus\",\"method\":\"$method\",\"data\":\"$data\"}"
}

log "Handler started"

# Read JSON messages from stdin, one per line
while IFS= read -r line; do
    # Parse message type using basic string matching
    msg_type=$(echo "$line" | grep -o '"type":"[^"]*"' | cut -d'"' -f4)

    case "$msg_type" in
        connect)
            log "Robot connected from $ROBOT_IP"
            send_to_robot "WELCOME Hello from example_robot handler!"
            publish_event "robot_connected" "$ROBOT_UUID"
            ;;
        incoming)
            payload=$(echo "$line" | grep -o '"payload":"[^"]*"' | cut -d'"' -f4)
            log "Received: $payload"

            # Echo the message back with a prefix
            send_to_robot "ECHO $payload"
            ;;
        disconnect)
            reason=$(echo "$line" | grep -o '"reason":"[^"]*"' | cut -d'"' -f4)
            log "Robot disconnected: $reason"
            publish_event "robot_disconnected" "$ROBOT_UUID"
            ;;
        *)
            log "Unknown message type: $msg_type"
            ;;
    esac
done

log "Handler exiting"
