#!/usr/bin/env python3
"""Example test robot using the Robomesh SDK.

This script demonstrates the full lifecycle:
1. Generate keys (or use existing ones)
2. Provision via admin API
3. Authenticate via TCP
4. Send heartbeats
5. Exchange messages

Usage:
    # With default dev docker-compose settings:
    python test_robot.py

    # Custom server:
    ROBOMESH_HOST=192.168.1.50 ROBOMESH_TCP_PORT=5002 python test_robot.py
"""

import logging
import os
import signal
import sys
import time

# Add parent to path so we can import without installing
sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from robomesh_sdk import RobotClient, generate_ed25519_keypair
from robomesh_sdk.admin import AdminClient

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(name)s] %(message)s")
logger = logging.getLogger("test_robot")

HOST = os.environ.get("ROBOMESH_HOST", "localhost")
TCP_PORT = int(os.environ.get("ROBOMESH_TCP_PORT", "5002"))
HTTP_PORT = int(os.environ.get("ROBOMESH_HTTP_PORT", "8080"))


def main():
    # 1. Generate a fresh keypair
    _, priv_hex, pub_hex = generate_ed25519_keypair()
    robot_uuid = "test-robot-demo"
    device_type = "test_robot"

    logger.info("Robot UUID:    %s", robot_uuid)
    logger.info("Public key:    %s", pub_hex)

    # 2. Provision via admin API
    admin = AdminClient(host=HOST, http_port=HTTP_PORT)
    admin.login("admin", "password1")
    try:
        admin.provision_robot(robot_uuid, pub_hex, device_type)
        logger.info("Robot provisioned successfully")
    except Exception as e:
        logger.warning("Provision failed (may already exist): %s", e)

    # 3. Connect and authenticate
    client = RobotClient(
        uuid=robot_uuid,
        private_key_hex=priv_hex,
        host=HOST,
        tcp_port=TCP_PORT,
    )

    jwt = client.authenticate()
    logger.info("Authenticated! JWT: %s...%s", jwt[:20], jwt[-10:])

    # 4. Register message callback
    def on_message(msg):
        logger.info("Received from server: %s", msg)

    client.on_message(on_message)

    # 5. Send heartbeats in background
    client.start_heartbeat(interval=25, ttl=60)

    # 6. Send some messages
    client.send("hello from test robot")
    time.sleep(1)
    client.send('{"type": "status", "battery": 95, "temp": 37.2}')

    # 7. Wait for Ctrl+C
    logger.info("Robot running. Press Ctrl+C to stop.")

    def handle_sigint(sig, frame):
        logger.info("Shutting down...")
        client.stop_heartbeat()
        client.disconnect()
        sys.exit(0)

    signal.signal(signal.SIGINT, handle_sigint)

    while True:
        time.sleep(1)


if __name__ == "__main__":
    main()
