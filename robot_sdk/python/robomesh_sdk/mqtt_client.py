"""Robomesh Robot MQTT client - handles auth, heartbeat, and messaging over MQTT."""

import json
import threading
import logging
from typing import Callable

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from .keys import sign_message, load_private_key, load_public_key_hex

try:
    import paho.mqtt.client as paho_mqtt
except ImportError:
    raise ImportError(
        "paho-mqtt is required for MQTT support. "
        "Install it with: pip install paho-mqtt>=2.0"
    )

logger = logging.getLogger("robomesh_sdk.mqtt")


class RobotMQTTClient:
    """MQTT client for a robot connecting to Roboserver.

    Uses topic-based protocol with two-step challenge-response auth.

    Topics:
        robomesh/auth/{uuid}              - Robot publishes auth requests
        robomesh/auth/{uuid}/response     - Server responds with nonce/JWT
        robomesh/heartbeat/{uuid}         - Robot publishes signed heartbeats
        robomesh/heartbeat/{uuid}/response - Server heartbeat acknowledgements
        robomesh/message/{uuid}           - Robot publishes messages to handler
        robomesh/to_robot/{uuid}          - Server sends messages to robot

    Usage:
        client = RobotMQTTClient(
            uuid="my-robot-001",
            private_key_hex="abcdef...",
            host="localhost",
            mqtt_port=1883,
        )
        client.connect()
        client.authenticate()
        client.start_heartbeat(interval=30)
        client.send("hello from robot")
    """

    def __init__(
        self,
        uuid: str,
        private_key_hex: str,
        host: str = "localhost",
        mqtt_port: int = 1883,
    ):
        self.uuid = uuid
        self.private_key: Ed25519PrivateKey = load_private_key(private_key_hex)
        self.public_key_hex = load_public_key_hex(self.private_key)
        self.host = host
        self.mqtt_port = mqtt_port

        self._jwt: str | None = None
        self._heartbeat_seq = 0
        self._heartbeat_lock = threading.Lock()
        self._heartbeat_thread: threading.Thread | None = None
        self._heartbeat_stop = threading.Event()
        self._message_callback: Callable[[str], None] | None = None
        self._connected = False

        # Auth synchronization
        self._auth_response: dict | None = None
        self._auth_event = threading.Event()

        # Heartbeat synchronization
        self._hb_response: dict | None = None
        self._hb_event = threading.Event()

        # MQTT client setup
        self._mqtt = paho_mqtt.Client(
            paho_mqtt.CallbackAPIVersion.VERSION2,
            client_id=f"robomesh-{uuid}",
        )
        self._mqtt.on_connect = self._on_connect
        self._mqtt.on_message = self._on_message

        # Topic constants
        self._topic_auth = f"robomesh/auth/{uuid}"
        self._topic_auth_resp = f"robomesh/auth/{uuid}/response"
        self._topic_heartbeat = f"robomesh/heartbeat/{uuid}"
        self._topic_heartbeat_resp = f"robomesh/heartbeat/{uuid}/response"
        self._topic_message = f"robomesh/message/{uuid}"
        self._topic_to_robot = f"robomesh/to_robot/{uuid}"

    @property
    def jwt(self) -> str | None:
        return self._jwt

    @property
    def connected(self) -> bool:
        return self._connected

    # ── Connection ──────────────────────────────────────────────

    def _on_connect(self, client, userdata, flags, reason_code, properties=None):
        """Called when MQTT connection is established."""
        if reason_code == 0:
            self._connected = True
            # Subscribe to response topics and incoming messages
            client.subscribe(self._topic_auth_resp)
            client.subscribe(self._topic_heartbeat_resp)
            client.subscribe(self._topic_to_robot)
            logger.info("MQTT connected and subscribed to response topics")
        else:
            logger.error("MQTT connection failed: %s", reason_code)

    def _on_message(self, client, userdata, msg):
        """Route incoming MQTT messages to the appropriate handler."""
        topic = msg.topic
        try:
            payload = json.loads(msg.payload.decode("utf-8"))
        except (json.JSONDecodeError, UnicodeDecodeError):
            payload = {"status": "error", "error": "invalid JSON from server"}

        if not isinstance(payload, dict):
            payload = {"status": "error", "error": "unexpected response format"}

        if topic == self._topic_auth_resp:
            self._auth_response = payload
            self._auth_event.set()
        elif topic == self._topic_heartbeat_resp:
            self._hb_response = payload
            self._hb_event.set()
        elif topic == self._topic_to_robot:
            if self._message_callback:
                # Pass the raw payload bytes as string for to_robot messages
                self._message_callback(msg.payload.decode("utf-8", errors="replace"))

    def connect(self) -> None:
        """Connect to the MQTT broker and start the network loop."""
        self._mqtt.connect(self.host, self.mqtt_port, keepalive=60)
        self._mqtt.loop_start()
        # Wait briefly for connection to establish
        for _ in range(50):
            if self._connected:
                break
            import time
            time.sleep(0.1)
        if not self._connected:
            raise ConnectionError("Failed to connect to MQTT broker")
        logger.info("Connected to MQTT broker at %s:%d", self.host, self.mqtt_port)

    def disconnect(self) -> None:
        """Disconnect from the MQTT broker and stop background threads."""
        self._heartbeat_stop.set()
        self._mqtt.loop_stop()
        self._mqtt.disconnect()
        self._connected = False
        if self._heartbeat_thread and self._heartbeat_thread.is_alive():
            self._heartbeat_thread.join(timeout=5)
        logger.info("MQTT disconnected")

    # ── AUTH flow ───────────────────────────────────────────────

    def authenticate(self, timeout: float = 10) -> str:
        """Perform two-step challenge-response auth over MQTT.

        Returns the JWT session token.
        """
        if not self._connected:
            self.connect()

        # Step 1: Request nonce
        self._auth_event.clear()
        self._auth_response = None
        self._mqtt.publish(
            self._topic_auth,
            json.dumps({"uuid": self.uuid}, separators=(",", ":")),
        )

        if not self._auth_event.wait(timeout=timeout):
            raise MQTTAuthError("Auth step 1 timed out waiting for nonce")

        resp = self._auth_response
        if resp.get("status") == "error":
            raise MQTTAuthError(f"Auth step 1 failed: {resp.get('error', 'unknown')}")
        if resp.get("status") != "nonce":
            raise MQTTAuthError(f"Expected nonce response, got: {resp}")

        nonce_hex = resp["nonce"]

        # Step 2: Sign nonce and send back
        nonce_bytes = bytes.fromhex(nonce_hex)
        signature_hex = sign_message(self.private_key, nonce_bytes)

        self._auth_event.clear()
        self._auth_response = None
        self._mqtt.publish(
            self._topic_auth,
            json.dumps({
                "uuid": self.uuid,
                "signature": signature_hex,
                "nonce": nonce_hex,
            }, separators=(",", ":")),
        )

        if not self._auth_event.wait(timeout=timeout):
            raise MQTTAuthError("Auth step 2 timed out waiting for JWT")

        resp = self._auth_response
        if resp.get("status") == "error":
            raise MQTTAuthError(f"Auth step 2 failed: {resp.get('error', 'unknown')}")
        if resp.get("status") != "ok":
            raise MQTTAuthError(f"Expected ok response, got: {resp}")

        self._jwt = resp["jwt"]
        logger.info("MQTT authenticated successfully")
        return self._jwt

    # ── Heartbeat ──────────────────────────────────────────────

    def send_heartbeat(self, extra_data: dict | None = None, ttl: int | None = None) -> None:
        """Send a signed heartbeat over MQTT."""
        if not self._connected:
            raise ConnectionError("Not connected to MQTT broker")

        with self._heartbeat_lock:
            self._heartbeat_seq += 1
            seq = self._heartbeat_seq
        payload: dict = {"seq": seq}
        if ttl is not None:
            payload["ttl"] = ttl
        if extra_data is not None:
            payload["extra_data"] = extra_data

        # MQTT heartbeat sends payload as a JSON string (not object)
        payload_json = json.dumps(payload, separators=(",", ":"))
        signature_hex = sign_message(self.private_key, payload_json.encode("utf-8"))

        self._hb_event.clear()
        self._hb_response = None
        self._mqtt.publish(
            self._topic_heartbeat,
            json.dumps({
                "payload": payload_json,
                "signature": signature_hex,
            }, separators=(",", ":")),
        )

        if not self._hb_event.wait(timeout=5):
            raise MQTTHeartbeatError("Heartbeat timed out")

        resp = self._hb_response
        if resp.get("status") == "error":
            raise MQTTHeartbeatError(f"Heartbeat failed: {resp.get('error', 'unknown')}")
        logger.debug("MQTT heartbeat seq=%d OK", self._heartbeat_seq)

    def start_heartbeat(self, interval: float = 30, ttl: int | None = None) -> None:
        """Start a background thread that sends heartbeats at the given interval."""
        self._heartbeat_stop.clear()

        def _loop():
            while not self._heartbeat_stop.is_set():
                try:
                    self.send_heartbeat(ttl=ttl)
                except Exception as e:
                    logger.error("MQTT heartbeat error: %s", e)
                    break
                self._heartbeat_stop.wait(interval)

        self._heartbeat_thread = threading.Thread(target=_loop, daemon=True)
        self._heartbeat_thread.start()
        logger.info("MQTT heartbeat thread started (interval=%ds)", interval)

    def stop_heartbeat(self) -> None:
        """Stop the background heartbeat thread."""
        self._heartbeat_stop.set()
        if self._heartbeat_thread and self._heartbeat_thread.is_alive():
            self._heartbeat_thread.join(timeout=5)
        logger.info("MQTT heartbeat thread stopped")

    # ── Messaging ──────────────────────────────────────────────

    def send(self, message: str) -> None:
        """Send a message to the handler via MQTT.

        Unlike UDP, MQTT messages don't require JWT — access is controlled
        by the broker's ACL hook at the topic level.
        """
        if not self._connected:
            raise ConnectionError("Not connected to MQTT broker")
        self._mqtt.publish(self._topic_message, message.encode("utf-8"))

    def on_message(self, callback: Callable[[str], None]) -> None:
        """Register a callback for incoming messages from the handler.

        Messages arrive on robomesh/to_robot/{uuid} and are passed to the
        callback as strings.
        """
        self._message_callback = callback


class MQTTAuthError(Exception):
    """Raised when MQTT authentication fails."""
    pass


class MQTTHeartbeatError(Exception):
    """Raised when an MQTT heartbeat is rejected."""
    pass
