#!/usr/bin/env python3
"""Example test robot using the Robomesh SDK over MQTT.

Demonstrates the full MQTT lifecycle:
1. Generate keys (or use existing ones)
2. Provision via admin API
3. Authenticate via MQTT topic-based challenge-response
4. Send heartbeats over MQTT
5. Exchange messages over MQTT

Requires: pip install paho-mqtt>=2.0

Usage:
    python test_robot_mqtt.py

    # Custom server:
    ROBOMESH_HOST=192.168.1.50 ROBOMESH_MQTT_PORT=1883 python test_robot_mqtt.py
"""

import logging
import os
import signal
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from robomesh_sdk import generate_ed25519_keypair
from robomesh_sdk.mqtt_client import RobotMQTTClient
from robomesh_sdk.admin import AdminClient

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(name)s] %(message)s")
logger = logging.getLogger("test_robot_mqtt")

HOST = os.environ.get("ROBOMESH_HOST", "localhost")
MQTT_PORT = int(os.environ.get("ROBOMESH_MQTT_PORT", "1883"))
HTTP_PORT = int(os.environ.get("ROBOMESH_HTTP_PORT", "8080"))


def main():
    # 1. Generate a fresh keypair
    _, priv_hex, pub_hex = generate_ed25519_keypair()
    robot_uuid = "test-robot-mqtt-demo"
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

    # 3. Connect and authenticate over MQTT
    client = RobotMQTTClient(
        uuid=robot_uuid,
        private_key_hex=priv_hex,
        host=HOST,
        mqtt_port=MQTT_PORT,
    )

    client.connect()
    jwt = client.authenticate()
    logger.info("Authenticated via MQTT! JWT: %s...%s", jwt[:20], jwt[-10:])

    # 4. Register message callback
    def on_message(msg):
        logger.info("Received from handler: %s", msg)

    client.on_message(on_message)

    # 5. Send heartbeats in background
    client.start_heartbeat(interval=25, ttl=60)

    # 6. Send some messages
    client.send("hello from MQTT test robot")
    time.sleep(1)
    client.send('{"type": "status", "battery": 88, "protocol": "mqtt"}')

    # 7. Wait for Ctrl+C
    logger.info("MQTT robot running. Press Ctrl+C to stop.")

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
