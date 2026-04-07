#!/usr/bin/env python3
"""
Test robot handler — network diagnostics (wifi speed, ping, signal strength).

Reads JSON messages from stdin, writes JSON-RPC responses to stdout.
Simulates network diagnostic operations for testing purposes.
"""
import sys
import json
import time
import random
import threading

# Simulated robot state
state = {
    "signal_strength": -45,   # dBm
    "ssid": "RoboNet-5G",
    "frequency": "5 GHz",
    "link_speed": 866,        # Mbps
    "last_speed_test": None,
    "speed_history": [],
    "connected": False,
    "uuid": None,
}

msg_counter = 0


def next_id():
    global msg_counter
    msg_counter += 1
    return str(msg_counter)


def send(target, method="", data=None, msg_id=""):
    """Send a JSON-RPC message to the roboserver."""
    msg = {"target": target, "id": msg_id or next_id()}
    if method:
        msg["method"] = method
    if data is not None:
        msg["data"] = data
    print(json.dumps(msg), flush=True)


def publish_event(event_type, data):
    """Publish an event on the event bus."""
    send("event_bus", method=event_type, data=data)


def send_to_robot(data):
    """Send data back to the robot."""
    send("robot", data=data)


def simulate_speed_test():
    """Simulate a wifi speed test with realistic-looking results."""
    # Simulate some processing time variation
    base_down = random.gauss(150, 30)
    base_up = random.gauss(50, 15)
    jitter = random.uniform(1.0, 15.0)
    latency = random.uniform(5.0, 35.0)

    result = {
        "download_mbps": round(max(10, base_down), 2),
        "upload_mbps": round(max(5, base_up), 2),
        "latency_ms": round(latency, 1),
        "jitter_ms": round(jitter, 1),
        "server": "speedtest-nyc.robomesh.local",
        "timestamp": int(time.time()),
    }

    state["last_speed_test"] = result
    state["speed_history"].append(result)
    # Keep last 20 results
    if len(state["speed_history"]) > 20:
        state["speed_history"] = state["speed_history"][-20:]

    return result


def simulate_ping(target="8.8.8.8", count=4):
    """Simulate a ping command."""
    results = []
    for i in range(count):
        rtt = random.gauss(15, 5)
        results.append(round(max(1, rtt), 2))

    avg = round(sum(results) / len(results), 2)
    return {
        "target": target,
        "count": count,
        "results_ms": results,
        "avg_ms": avg,
        "min_ms": min(results),
        "max_ms": max(results),
        "packet_loss": 0,
        "timestamp": int(time.time()),
    }


def get_status():
    """Return current network status."""
    # Drift signal slightly each check
    state["signal_strength"] = max(-90, min(-20,
        state["signal_strength"] + random.randint(-3, 3)))

    return {
        "signal_strength_dbm": state["signal_strength"],
        "signal_quality": signal_quality(state["signal_strength"]),
        "ssid": state["ssid"],
        "frequency": state["frequency"],
        "link_speed_mbps": state["link_speed"],
        "last_speed_test": state["last_speed_test"],
        "tests_run": len(state["speed_history"]),
        "uuid": state["uuid"],
        "timestamp": int(time.time()),
    }


def signal_quality(dbm):
    """Convert dBm to a quality label."""
    if dbm >= -30:
        return "excellent"
    elif dbm >= -50:
        return "good"
    elif dbm >= -60:
        return "fair"
    elif dbm >= -70:
        return "weak"
    else:
        return "poor"


def handle_connect(msg):
    """Called when the robot first authenticates."""
    state["uuid"] = msg.get("uuid")
    state["connected"] = True
    print(f"[test_robot] Robot {state['uuid']} connected from {msg.get('ip')}", file=sys.stderr)

    # Enable heartbeat forwarding
    send("config", method="forward_heartbeats", data=True)

    # Publish a connect event
    publish_event(f"test_robot.{state['uuid']}.connected", {
        "uuid": state["uuid"],
        "status": get_status(),
    })


def handle_incoming(msg):
    """Called when the robot sends a message or the frontend sends a command."""
    payload_raw = msg.get("payload", "")

    # Try to parse as JSON command
    try:
        if isinstance(payload_raw, str):
            payload = json.loads(payload_raw)
        else:
            payload = payload_raw
    except (json.JSONDecodeError, TypeError):
        payload = {"command": payload_raw}

    command = payload.get("command", "")
    print(f"[test_robot] Command: {command}", file=sys.stderr)

    if command == "speed_test":
        # Publish "running" status first
        publish_event(f"test_robot.{state['uuid']}.speed_test", {
            "status": "running",
            "uuid": state["uuid"],
        })
        result = simulate_speed_test()
        response = {"type": "speed_test_result", **result}
        send_to_robot(response)
        publish_event(f"test_robot.{state['uuid']}.speed_test", {
            "status": "complete",
            "uuid": state["uuid"],
            "result": result,
        })

    elif command == "ping":
        target = payload.get("target", "8.8.8.8")
        count = payload.get("count", 4)
        result = simulate_ping(target, count)
        response = {"type": "ping_result", **result}
        send_to_robot(response)

    elif command == "status":
        status = get_status()
        response = {"type": "status", **status}
        send_to_robot(response)

    elif command == "history":
        response = {
            "type": "speed_history",
            "history": state["speed_history"],
            "count": len(state["speed_history"]),
        }
        send_to_robot(response)

    else:
        # Echo unknown commands back
        send_to_robot({
            "type": "echo",
            "original": payload_raw,
            "message": f"Unknown command: {command}. Available: speed_test, ping, status, history",
        })


def handle_disconnect(msg):
    """Called when the robot's TCP connection closes."""
    reason = msg.get("reason", "unknown")
    state["connected"] = False
    print(f"[test_robot] Robot disconnected: {reason}", file=sys.stderr)
    publish_event(f"test_robot.{state['uuid']}.disconnected", {
        "uuid": state["uuid"],
        "reason": reason,
    })


def handle_heartbeat(msg):
    """Called when a heartbeat is forwarded to this handler."""
    data = msg.get("data", {})
    extra = data.get("extra_data", {})
    if extra:
        # Update signal strength if the robot reports it
        if "signal_dbm" in extra:
            state["signal_strength"] = extra["signal_dbm"]
    print(f"[test_robot] Heartbeat received", file=sys.stderr)


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
        elif msg_type == "heartbeat":
            handle_heartbeat(msg)
        elif msg_type == "event":
            print(f"[test_robot] Event: {msg.get('event_type')}", file=sys.stderr)


if __name__ == "__main__":
    main()
