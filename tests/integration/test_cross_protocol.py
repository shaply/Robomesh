"""Cross-protocol integration tests.

Tests interactions that span multiple protocols — for example, authenticating
on one transport and heartbeating on another, or verifying that state changes
are visible across protocols.
"""

import json
import time

import pytest

from conftest import (
    HOST, HTTP_PORT, TCP_PORT, UDP_PORT, MQTT_PORT,
)
from robomesh_sdk import RobotClient, RobotUDPClient, generate_ed25519_keypair
from robomesh_sdk.admin import AdminClient

try:
    from robomesh_sdk.mqtt_client import RobotMQTTClient
    MQTT_AVAILABLE = True
except ImportError:
    MQTT_AVAILABLE = False


class TestCrossProtocolAuth:
    """Verify that authentication on one protocol creates visible state on another."""

    def test_tcp_auth_visible_via_http(self, admin, provisioned_robot):
        """TCP-authenticated robot is visible in HTTP robot list."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            time.sleep(0.3)
            status = admin.get_robot_status(robot_uuid)
            assert status["online"] is True
        finally:
            client.disconnect()

    def test_udp_auth_visible_via_http(self, admin, provisioned_robot):
        """UDP-authenticated robot is visible in HTTP robot list."""
        robot_uuid, priv_hex = provisioned_robot
        udp = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            udp.authenticate()
            time.sleep(0.3)
            status = admin.get_robot_status(robot_uuid)
            assert status["online"] is True
        finally:
            udp.disconnect()

    @pytest.mark.skipif(not MQTT_AVAILABLE, reason="paho-mqtt not installed")
    def test_mqtt_auth_visible_via_http(self, admin, provisioned_robot):
        """MQTT-authenticated robot is visible in HTTP robot list."""
        robot_uuid, priv_hex = provisioned_robot
        mqtt = RobotMQTTClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            mqtt.authenticate()
            time.sleep(0.3)
            status = admin.get_robot_status(robot_uuid)
            assert status["online"] is True
        finally:
            mqtt.disconnect()


class TestCrossProtocolHeartbeat:
    """Heartbeat via one protocol while authenticated on another."""

    def test_tcp_auth_http_heartbeat(self, admin, provisioned_robot):
        """Authenticate via TCP, heartbeat via HTTP POST."""
        robot_uuid, priv_hex = provisioned_robot
        from robomesh_sdk.keys import sign_message, load_private_key

        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()

            # Heartbeat via HTTP
            payload = json.dumps({"seq": 1}, separators=(",", ":"))
            key = load_private_key(priv_hex)
            sig = sign_message(key, payload.encode())

            import urllib.request
            req_data = json.dumps({
                "uuid": robot_uuid,
                "payload": payload,
                "signature": sig,
            }).encode()
            req = urllib.request.Request(
                f"http://{HOST}:{HTTP_PORT}/heartbeat",
                data=req_data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req, timeout=10) as resp:
                assert resp.status == 200
        finally:
            client.disconnect()

    def test_tcp_auth_udp_heartbeat(self, provisioned_robot):
        """Authenticate via TCP, heartbeat via UDP."""
        robot_uuid, priv_hex = provisioned_robot
        from robomesh_sdk.keys import sign_message, load_private_key

        tcp_client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        udp_client = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            tcp_client.authenticate()
            udp_client.connect()
            udp_client.send_heartbeat()
        finally:
            tcp_client.disconnect()
            udp_client.disconnect()


class TestCrossProtocolProvision:
    """Provisioned robot works across all protocols."""

    def test_provision_then_auth_tcp(self, admin, robot_identity):
        """Provision via HTTP, then authenticate via TCP."""
        robot_uuid, priv_hex, pub_hex = robot_identity
        admin.provision_robot(robot_uuid, pub_hex, "test_robot")

        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            jwt = client.authenticate()
            assert jwt is not None
        finally:
            client.disconnect()

    def test_provision_then_auth_udp(self, admin, robot_identity):
        """Provision via HTTP, then authenticate via UDP."""
        robot_uuid, priv_hex, pub_hex = robot_identity
        admin.provision_robot(robot_uuid, pub_hex, "test_robot")

        client = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            jwt = client.authenticate()
            assert jwt is not None
        finally:
            client.disconnect()

    @pytest.mark.skipif(not MQTT_AVAILABLE, reason="paho-mqtt not installed")
    def test_provision_then_auth_mqtt(self, admin, robot_identity):
        """Provision via HTTP, then authenticate via MQTT."""
        robot_uuid, priv_hex, pub_hex = robot_identity
        admin.provision_robot(robot_uuid, pub_hex, "test_robot")

        client = RobotMQTTClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            jwt = client.authenticate()
            assert jwt is not None
        finally:
            client.disconnect()
