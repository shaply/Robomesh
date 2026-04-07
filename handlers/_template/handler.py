#!/usr/bin/env python3
"""
Template handler for Robomesh robot type.

Reads JSON messages from stdin and writes JSON responses to stdout.
"""
import sys
import json


def send(target, method="", data=None, msg_id=""):
    """Send a JSON-RPC message to the roboserver."""
    msg = {"target": target, "id": msg_id}
    if method:
        msg["method"] = method
    if data is not None:
        msg["data"] = data
    print(json.dumps(msg), flush=True)


def handle_connect(msg):
    """Called when the robot first authenticates."""
    uuid = msg.get("uuid")
    print(f"[handler] Robot {uuid} connected", file=sys.stderr)


def handle_incoming(msg):
    """Called when the robot sends a message."""
    payload = msg.get("payload", "")
    print(f"[handler] Received: {payload}", file=sys.stderr)
    # Echo back to robot as an example
    send("robot", data={"echo": payload}, msg_id="echo-1")


def handle_disconnect(msg):
    """Called when the robot's TCP connection closes."""
    reason = msg.get("reason", "unknown")
    print(f"[handler] Robot disconnected: {reason}", file=sys.stderr)


def main():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            msg = json.loads(line)
        except json.JSONDecodeError:
            continue

        msg_type = msg.get("type", "")
        if msg_type == "connect":
            handle_connect(msg)
        elif msg_type == "incoming":
            handle_incoming(msg)
        elif msg_type == "disconnect":
            handle_disconnect(msg)
        elif msg_type == "event":
            print(f"[handler] Event: {msg.get('event_type')}", file=sys.stderr)
        elif msg_type == "heartbeat":
            print(f"[handler] Heartbeat received", file=sys.stderr)


if __name__ == "__main__":
    main()
