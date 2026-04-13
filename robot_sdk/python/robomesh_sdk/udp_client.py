"""Robomesh Robot UDP client - handles auth, heartbeat, and messaging over UDP."""

import json
import socket
import threading
import logging
from typing import Callable

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from .keys import sign_message, load_private_key, load_public_key_hex

logger = logging.getLogger("robomesh_sdk.udp")


class RobotUDPClient:
    """UDP client for a robot connecting to Roboserver.

    Uses JSON packets over UDP with two-step challenge-response auth.

    Usage:
        client = RobotUDPClient(
            uuid="my-robot-001",
            private_key_hex="abcdef...",
            host="localhost",
            udp_port=5001,
        )
        client.authenticate()
        client.start_heartbeat(interval=30)
        client.send("hello from robot")
    """

    def __init__(
        self,
        uuid: str,
        private_key_hex: str,
        host: str = "localhost",
        udp_port: int = 5001,
    ):
        self.uuid = uuid
        self.private_key: Ed25519PrivateKey = load_private_key(private_key_hex)
        self.public_key_hex = load_public_key_hex(self.private_key)
        self.host = host
        self.udp_port = udp_port

        self._sock: socket.socket | None = None
        self._server_addr: tuple[str, int] = (host, udp_port)
        self._jwt: str | None = None
        self._heartbeat_seq = 0
        self._heartbeat_lock = threading.Lock()
        self._heartbeat_thread: threading.Thread | None = None
        self._heartbeat_stop = threading.Event()
        self._recv_thread: threading.Thread | None = None
        self._recv_stop = threading.Event()
        self._message_callback: Callable[[dict], None] | None = None

    @property
    def jwt(self) -> str | None:
        return self._jwt

    # ── Connection ──────────────────────────────────────────────

    def connect(self) -> None:
        """Create a UDP socket bound to the server address."""
        self._sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self._sock.settimeout(10)
        logger.info("UDP socket created for %s:%d", self.host, self.udp_port)

    def disconnect(self) -> None:
        """Close the UDP socket and stop background threads."""
        self._heartbeat_stop.set()
        self._recv_stop.set()
        if self._sock:
            try:
                self._sock.close()
            except OSError:
                pass
            self._sock = None
        if self._heartbeat_thread and self._heartbeat_thread.is_alive():
            self._heartbeat_thread.join(timeout=5)
        if self._recv_thread and self._recv_thread.is_alive():
            self._recv_thread.join(timeout=5)
        logger.info("UDP client disconnected")

    def _send_packet(self, packet: dict) -> None:
        """Send a JSON packet to the server."""
        if not self._sock:
            raise ConnectionError("Not connected")
        data = json.dumps(packet, separators=(",", ":")).encode("utf-8")
        self._sock.sendto(data, self._server_addr)

    def _recv_packet(self, timeout: float | None = None) -> dict:
        """Receive a JSON packet from the server."""
        if not self._sock:
            raise ConnectionError("Not connected")
        if timeout is not None:
            self._sock.settimeout(timeout)
        try:
            data, _ = self._sock.recvfrom(65535)
            return json.loads(data.decode("utf-8"))
        finally:
            if timeout is not None:
                self._sock.settimeout(10)

    # ── AUTH flow ───────────────────────────────────────────────

    def authenticate(self) -> str:
        """Perform two-step challenge-response auth over UDP.

        Returns the JWT session token.
        """
        if not self._sock:
            self.connect()

        # Step 1: Request nonce
        self._send_packet({"type": "auth", "uuid": self.uuid})

        resp = self._recv_packet(timeout=10)
        if resp.get("status") == "error":
            raise UDPAuthError(f"Auth step 1 failed: {resp.get('error', 'unknown')}")
        if resp.get("status") != "nonce":
            raise UDPAuthError(f"Expected nonce response, got: {resp}")

        nonce_hex = resp["nonce"]

        # Step 2: Sign nonce and send back
        nonce_bytes = bytes.fromhex(nonce_hex)
        signature_hex = sign_message(self.private_key, nonce_bytes)

        self._send_packet({
            "type": "auth",
            "uuid": self.uuid,
            "nonce": nonce_hex,
            "signature": signature_hex,
        })

        resp = self._recv_packet(timeout=10)
        if resp.get("status") == "error":
            raise UDPAuthError(f"Auth step 2 failed: {resp.get('error', 'unknown')}")
        if resp.get("status") != "ok":
            raise UDPAuthError(f"Expected ok response, got: {resp}")

        self._jwt = resp["jwt"]
        logger.info("UDP authenticated successfully")
        return self._jwt

    # ── Heartbeat ──────────────────────────────────────────────

    def send_heartbeat(self, extra_data: dict | None = None, ttl: int | None = None) -> None:
        """Send a signed heartbeat over UDP."""
        if not self._sock:
            self.connect()

        with self._heartbeat_lock:
            self._heartbeat_seq += 1
            seq = self._heartbeat_seq
        payload: dict = {"seq": seq}
        if ttl is not None:
            payload["ttl"] = ttl
        if extra_data is not None:
            payload["extra_data"] = extra_data

        payload_json = json.dumps(payload, separators=(",", ":"))
        signature_hex = sign_message(self.private_key, payload_json.encode("utf-8"))

        # Payload is sent as a raw JSON object (json.RawMessage on server side).
        # _send_packet uses the same compact separators, so the re-serialized
        # nested dict matches what we signed.
        self._send_packet({
            "type": "heartbeat",
            "uuid": self.uuid,
            "payload": payload,
            "signature": signature_hex,
        })

        resp = self._recv_packet(timeout=5)
        if resp.get("status") == "error":
            raise UDPHeartbeatError(f"Heartbeat failed: {resp.get('error', 'unknown')}")
        logger.debug("UDP heartbeat seq=%d OK", self._heartbeat_seq)

    def start_heartbeat(self, interval: float = 30, ttl: int | None = None) -> None:
        """Start a background thread that sends heartbeats at the given interval."""
        self._heartbeat_stop.clear()

        def _loop():
            while not self._heartbeat_stop.is_set():
                try:
                    self.send_heartbeat(ttl=ttl)
                except Exception as e:
                    logger.error("UDP heartbeat error: %s", e)
                    break
                self._heartbeat_stop.wait(interval)

        self._heartbeat_thread = threading.Thread(target=_loop, daemon=True)
        self._heartbeat_thread.start()
        logger.info("UDP heartbeat thread started (interval=%ds)", interval)

    def stop_heartbeat(self) -> None:
        """Stop the background heartbeat thread."""
        self._heartbeat_stop.set()
        if self._heartbeat_thread and self._heartbeat_thread.is_alive():
            self._heartbeat_thread.join(timeout=5)
        logger.info("UDP heartbeat thread stopped")

    # ── Messaging ──────────────────────────────────────────────

    def send(self, message: str) -> None:
        """Send a JWT-authenticated message to the handler via UDP."""
        if not self._jwt:
            raise UDPAuthError("Not authenticated — call authenticate() first")
        if not self._sock:
            raise ConnectionError("Not connected")

        self._send_packet({
            "type": "message",
            "uuid": self.uuid,
            "jwt": self._jwt,
            "payload": message,
        })

        resp = self._recv_packet(timeout=5)
        if resp.get("status") == "error":
            raise UDPMessageError(f"Message failed: {resp.get('error', 'unknown')}")

    def on_message(self, callback: Callable[[dict], None]) -> None:
        """Start a background listener for incoming UDP packets from the server.

        Note: UDP is connectionless, so this listens for any packets on the
        bound socket. The callback receives the parsed JSON dict.
        """
        self._message_callback = callback
        self._recv_stop.clear()

        def _loop():
            while not self._recv_stop.is_set():
                try:
                    if not self._sock:
                        break
                    self._sock.settimeout(1)
                    data, _ = self._sock.recvfrom(65535)
                    msg = json.loads(data.decode("utf-8"))
                    if self._message_callback:
                        self._message_callback(msg)
                except socket.timeout:
                    continue
                except (ConnectionError, OSError):
                    if not self._recv_stop.is_set():
                        logger.warning("UDP socket error")
                    break
                except Exception as e:
                    logger.error("UDP receive error: %s", e)
                    break

        self._recv_thread = threading.Thread(target=_loop, daemon=True)
        self._recv_thread.start()


class UDPAuthError(Exception):
    """Raised when UDP authentication fails."""
    pass


class UDPHeartbeatError(Exception):
    """Raised when a UDP heartbeat is rejected."""
    pass


class UDPMessageError(Exception):
    """Raised when a UDP message send fails."""
    pass
