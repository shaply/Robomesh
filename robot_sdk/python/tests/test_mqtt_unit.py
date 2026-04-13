"""Unit tests for the Robomesh MQTT client.

These tests mock the paho-mqtt layer so they can run without a broker.
"""

import json
import threading
import pytest
from unittest.mock import patch, MagicMock

from robomesh_sdk import generate_ed25519_keypair
from robomesh_sdk.mqtt_client import RobotMQTTClient, MQTTAuthError, MQTTHeartbeatError


@pytest.fixture
def keypair():
    _, priv_hex, pub_hex = generate_ed25519_keypair()
    return priv_hex, pub_hex


@pytest.fixture
def client(keypair):
    """Create a RobotMQTTClient with mocked paho internals."""
    priv_hex, _ = keypair

    with patch("robomesh_sdk.mqtt_client.paho_mqtt") as mock_module:
        mock_paho_client = MagicMock()
        mock_module.Client.return_value = mock_paho_client
        mock_module.CallbackAPIVersion.VERSION2 = 2

        c = RobotMQTTClient(
            uuid="test-mqtt-001",
            private_key_hex=priv_hex,
            host="localhost",
            mqtt_port=1883,
        )

        # Capture the on_connect and on_message callbacks that were assigned
        on_connect = mock_paho_client.on_connect
        on_message = mock_paho_client.on_message

        # Simulate successful connection when connect() is called
        def do_connect(host, port, keepalive=60):
            on_connect(mock_paho_client, None, None, 0, None)

        mock_paho_client.connect.side_effect = do_connect

        yield c, mock_paho_client, on_connect, on_message


class TestMQTTClientInit:
    def test_create_client(self, keypair):
        priv_hex, pub_hex = keypair
        with patch("robomesh_sdk.mqtt_client.paho_mqtt") as mock_module:
            mock_module.CallbackAPIVersion.VERSION2 = 2
            mock_module.Client.return_value = MagicMock()
            c = RobotMQTTClient(uuid="test-001", private_key_hex=priv_hex)
            assert c.uuid == "test-001"
            assert c.public_key_hex == pub_hex
            assert c.jwt is None

    def test_topic_setup(self, keypair):
        priv_hex, _ = keypair
        with patch("robomesh_sdk.mqtt_client.paho_mqtt") as mock_module:
            mock_module.CallbackAPIVersion.VERSION2 = 2
            mock_module.Client.return_value = MagicMock()
            c = RobotMQTTClient(uuid="my-robot", private_key_hex=priv_hex)
            assert c._topic_auth == "robomesh/auth/my-robot"
            assert c._topic_auth_resp == "robomesh/auth/my-robot/response"
            assert c._topic_heartbeat == "robomesh/heartbeat/my-robot"
            assert c._topic_to_robot == "robomesh/to_robot/my-robot"


class TestMQTTConnect:
    def test_connect_success(self, client):
        c, mock_mqtt, on_connect, on_message = client
        c.connect()
        assert c.connected
        mock_mqtt.connect.assert_called_once_with("localhost", 1883, keepalive=60)
        mock_mqtt.loop_start.assert_called_once()

    def test_disconnect(self, client):
        c, mock_mqtt, on_connect, on_message = client
        c.connect()
        c.disconnect()
        assert not c.connected
        mock_mqtt.loop_stop.assert_called_once()
        mock_mqtt.disconnect.assert_called_once()


class TestMQTTAuth:
    def test_auth_success(self, client):
        c, mock_mqtt, on_connect, on_message = client
        nonce_hex = "abcdef1234567890" * 2

        def simulate_publish(topic, payload, **kwargs):
            data = json.loads(payload)

            msg = MagicMock()
            if data.get("signature"):
                msg.topic = c._topic_auth_resp
                msg.payload = json.dumps({
                    "status": "ok",
                    "jwt": "test.jwt.token",
                }).encode()
            else:
                msg.topic = c._topic_auth_resp
                msg.payload = json.dumps({
                    "status": "nonce",
                    "nonce": nonce_hex,
                }).encode()

            threading.Timer(0.01, on_message, args=(mock_mqtt, None, msg)).start()

        mock_mqtt.publish.side_effect = simulate_publish

        c.connect()
        jwt = c.authenticate()

        assert jwt == "test.jwt.token"
        assert c.jwt == "test.jwt.token"
        assert mock_mqtt.publish.call_count == 2

        # Verify step 1 publish
        step1_topic = mock_mqtt.publish.call_args_list[0][0][0]
        step1_data = json.loads(mock_mqtt.publish.call_args_list[0][0][1])
        assert step1_topic == "robomesh/auth/test-mqtt-001"
        assert step1_data["uuid"] == "test-mqtt-001"

        # Verify step 2 publish
        step2_data = json.loads(mock_mqtt.publish.call_args_list[1][0][1])
        assert "signature" in step2_data
        assert step2_data["nonce"] == nonce_hex

    def test_auth_error(self, client):
        c, mock_mqtt, on_connect, on_message = client

        def simulate_publish(topic, payload, **kwargs):
            msg = MagicMock()
            msg.topic = c._topic_auth_resp
            msg.payload = json.dumps({
                "status": "error",
                "error": "unknown robot",
            }).encode()
            threading.Timer(0.01, on_message, args=(mock_mqtt, None, msg)).start()

        mock_mqtt.publish.side_effect = simulate_publish

        c.connect()
        with pytest.raises(MQTTAuthError, match="unknown robot"):
            c.authenticate()

    def test_auth_timeout(self, client):
        c, mock_mqtt, on_connect, on_message = client
        mock_mqtt.publish.side_effect = None

        c.connect()
        with pytest.raises(MQTTAuthError, match="timed out"):
            c.authenticate(timeout=0.1)


class TestMQTTHeartbeat:
    def test_heartbeat_success(self, client):
        c, mock_mqtt, on_connect, on_message = client

        def simulate_publish(topic, payload, **kwargs):
            if "heartbeat" not in topic:
                return
            msg = MagicMock()
            msg.topic = c._topic_heartbeat_resp
            msg.payload = json.dumps({"status": "ok"}).encode()
            threading.Timer(0.01, on_message, args=(mock_mqtt, None, msg)).start()

        mock_mqtt.publish.side_effect = simulate_publish

        c.connect()
        c.send_heartbeat()

        hb_data = json.loads(mock_mqtt.publish.call_args[0][1])
        assert "payload" in hb_data
        assert "signature" in hb_data
        # MQTT heartbeat payload is a JSON string, not an object
        assert isinstance(hb_data["payload"], str)
        payload = json.loads(hb_data["payload"])
        assert payload["seq"] == 1

    def test_heartbeat_with_extras(self, client):
        c, mock_mqtt, on_connect, on_message = client

        def simulate_publish(topic, payload, **kwargs):
            if "heartbeat" not in topic:
                return
            msg = MagicMock()
            msg.topic = c._topic_heartbeat_resp
            msg.payload = json.dumps({"status": "ok"}).encode()
            threading.Timer(0.01, on_message, args=(mock_mqtt, None, msg)).start()

        mock_mqtt.publish.side_effect = simulate_publish

        c.connect()
        c.send_heartbeat(extra_data={"battery": 95}, ttl=120)

        hb_data = json.loads(mock_mqtt.publish.call_args[0][1])
        payload = json.loads(hb_data["payload"])
        assert payload["ttl"] == 120
        assert payload["extra_data"] == {"battery": 95}

    def test_heartbeat_error(self, client):
        c, mock_mqtt, on_connect, on_message = client

        def simulate_publish(topic, payload, **kwargs):
            if "heartbeat" not in topic:
                return
            msg = MagicMock()
            msg.topic = c._topic_heartbeat_resp
            msg.payload = json.dumps({"status": "error", "error": "stale sequence"}).encode()
            threading.Timer(0.01, on_message, args=(mock_mqtt, None, msg)).start()

        mock_mqtt.publish.side_effect = simulate_publish

        c.connect()
        with pytest.raises(MQTTHeartbeatError, match="stale sequence"):
            c.send_heartbeat()


class TestMQTTMessaging:
    def test_send_message(self, client):
        c, mock_mqtt, on_connect, on_message = client
        mock_mqtt.publish.side_effect = None

        c.connect()
        c.send("hello from robot")

        mock_mqtt.publish.assert_called_once_with(
            "robomesh/message/test-mqtt-001",
            b"hello from robot",
        )

    def test_send_not_connected(self, keypair):
        priv_hex, _ = keypair
        with patch("robomesh_sdk.mqtt_client.paho_mqtt") as mock_module:
            mock_module.CallbackAPIVersion.VERSION2 = 2
            mock_module.Client.return_value = MagicMock()
            c = RobotMQTTClient(uuid="test-001", private_key_hex=priv_hex)
            with pytest.raises(ConnectionError, match="Not connected"):
                c.send("hello")

    def test_on_message_callback(self, client):
        c, mock_mqtt, on_connect, on_message = client
        received = []

        c.connect()
        c.on_message(lambda msg: received.append(msg))

        # Simulate incoming message from handler
        msg = MagicMock()
        msg.topic = c._topic_to_robot
        msg.payload = b'{"action":"move","x":10}'
        on_message(mock_mqtt, None, msg)

        assert len(received) == 1
        assert '"action"' in received[0]


class TestMQTTSignatureIntegrity:
    def test_heartbeat_signature_matches_payload_string(self, client):
        """Verify MQTT heartbeat signs the payload JSON string (not object)."""
        from robomesh_sdk.keys import sign_message as verify_sign

        c, mock_mqtt, on_connect, on_message = client

        def simulate_publish(topic, payload, **kwargs):
            if "heartbeat" not in topic:
                return
            msg = MagicMock()
            msg.topic = c._topic_heartbeat_resp
            msg.payload = json.dumps({"status": "ok"}).encode()
            threading.Timer(0.01, on_message, args=(mock_mqtt, None, msg)).start()

        mock_mqtt.publish.side_effect = simulate_publish

        c.connect()
        c.send_heartbeat(extra_data={"temp": 42}, ttl=60)

        hb_data = json.loads(mock_mqtt.publish.call_args[0][1])
        payload_str = hb_data["payload"]
        expected_sig = verify_sign(c.private_key, payload_str.encode("utf-8"))
        assert hb_data["signature"] == expected_sig
