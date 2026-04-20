"""MQTT protocol integration tests.

Tests authentication, heartbeat, and messaging flows over MQTT.
Requires paho-mqtt>=2.0.
"""

import time

import pytest

from conftest import (
    HOST, MQTT_PORT,
    SEED_UUID, SEED_PRIVATE_KEY_HEX,
)
from robomesh_sdk import generate_ed25519_keypair

try:
    from robomesh_sdk.mqtt_client import (
        RobotMQTTClient,
        MQTTAuthError,
        MQTTHeartbeatError,
    )
    MQTT_AVAILABLE = True
except ImportError:
    MQTT_AVAILABLE = False

pytestmark = pytest.mark.skipif(
    not MQTT_AVAILABLE,
    reason="paho-mqtt not installed (pip install paho-mqtt>=2.0)",
)


# ---------------------------------------------------------------------------
# AUTH flow tests
# ---------------------------------------------------------------------------

class TestMQTTAuth:
    """MQTT topic-based challenge-response authentication tests."""

    def test_auth_with_seeded_robot(self):
        """Full two-step auth with the pre-seeded example-001."""
        client = RobotMQTTClient(
            uuid=SEED_UUID,
            private_key_hex=SEED_PRIVATE_KEY_HEX,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            jwt = client.authenticate()
            assert jwt is not None
            assert len(jwt) > 0
        finally:
            client.disconnect()

    def test_auth_with_provisioned_robot(self, provisioned_robot):
        """Auth with a freshly provisioned robot."""
        robot_uuid, priv_hex = provisioned_robot
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

    def test_auth_unknown_robot(self):
        """Auth with unknown UUID returns error."""
        _, priv_hex, _ = generate_ed25519_keypair()
        client = RobotMQTTClient(
            uuid="nonexistent-mqtt-robot",
            private_key_hex=priv_hex,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            with pytest.raises(MQTTAuthError):
                client.authenticate()
        finally:
            client.disconnect()

    def test_auth_wrong_key(self, admin):
        """Auth with wrong private key should fail."""
        _, priv1_hex, pub1_hex = generate_ed25519_keypair()
        _, priv2_hex, _ = generate_ed25519_keypair()
        robot_uuid = f"integ-mqtt-wrongkey-{int(time.time())}"
        admin.provision_robot(robot_uuid, pub1_hex, "test_robot")

        client = RobotMQTTClient(
            uuid=robot_uuid,
            private_key_hex=priv2_hex,  # Wrong key
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            with pytest.raises(MQTTAuthError):
                client.authenticate()
        finally:
            client.disconnect()


# ---------------------------------------------------------------------------
# Heartbeat tests
# ---------------------------------------------------------------------------

class TestMQTTHeartbeat:
    """MQTT signed heartbeat tests."""

    def test_heartbeat_after_auth(self, provisioned_robot):
        """Heartbeat succeeds after MQTT authentication."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotMQTTClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat()
        finally:
            client.disconnect()

    def test_heartbeat_with_extra_data(self, provisioned_robot):
        """Heartbeat with extra_data and custom TTL over MQTT."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotMQTTClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat(extra_data={"battery": 95}, ttl=120)
        finally:
            client.disconnect()

    def test_heartbeat_stale_sequence_rejected(self, provisioned_robot):
        """MQTT broker rejects stale heartbeat sequence."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotMQTTClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat()  # seq=1
            client._heartbeat_seq = 0
            with pytest.raises(MQTTHeartbeatError):
                client.send_heartbeat()  # seq=1 again
        finally:
            client.disconnect()

    def test_multiple_heartbeats(self, provisioned_robot):
        """Multiple sequential heartbeats succeed."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotMQTTClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            client.authenticate()
            for _ in range(5):
                client.send_heartbeat()
        finally:
            client.disconnect()


# ---------------------------------------------------------------------------
# Messaging tests
# ---------------------------------------------------------------------------

class TestMQTTMessaging:
    """MQTT messaging tests."""

    def test_send_message(self, provisioned_robot):
        """Message can be sent to handler after authentication."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotMQTTClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            mqtt_port=MQTT_PORT,
        )
        try:
            client.authenticate()
            client.send("hello from MQTT integration test")
            time.sleep(0.2)
        finally:
            client.disconnect()
