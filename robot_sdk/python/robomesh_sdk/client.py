"""Robomesh Robot TCP client - handles auth, heartbeat, and messaging."""

import json
import socket
import threading
import time
import logging
from typing import Callable

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey

from .keys import sign_message, load_private_key, load_public_key_hex

logger = logging.getLogger("robomesh_sdk")


class RobotClient:
    """TCP client for a robot connecting to Roboserver.

    Usage:
        client = RobotClient(
            uuid="my-robot-001",
            private_key_hex="abcdef...",
            host="localhost",
            tcp_port=5000,
        )
        client.authenticate()
        client.start_heartbeat(interval=30)
        client.send("hello from robot")
        client.on_message(lambda msg: print("Got:", msg))
    """

    def __init__(
        self,
        uuid: str,
        private_key_hex: str,
        host: str = "localhost",
        tcp_port: int = 5000,
        device_type: str | None = None,
    ):
        self.uuid = uuid
        self.private_key: Ed25519PrivateKey = load_private_key(private_key_hex)
        self.public_key_hex = load_public_key_hex(self.private_key)
        self.host = host
        self.tcp_port = tcp_port
        self.device_type = device_type

        self._sock: socket.socket | None = None
        self._jwt: str | None = None
        self._heartbeat_seq = 0
        self._heartbeat_thread: threading.Thread | None = None
        self._heartbeat_stop = threading.Event()
        self._recv_thread: threading.Thread | None = None
        self._recv_stop = threading.Event()
        self._message_callback: Callable[[str], None] | None = None
        self._connected = False

    @property
    def jwt(self) -> str | None:
        return self._jwt

    @property
    def connected(self) -> bool:
        return self._connected

    # ── Connection ──────────────────────────────────────────────

    def connect(self) -> None:
        """Open TCP connection to roboserver."""
        self._sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._sock.settimeout(30)
        self._sock.connect((self.host, self.tcp_port))
        self._connected = True
        logger.info("Connected to %s:%d", self.host, self.tcp_port)

    def disconnect(self) -> None:
        """Close the TCP connection and stop background threads."""
        self._heartbeat_stop.set()
        self._recv_stop.set()
        self._connected = False
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
        logger.info("Disconnected")

    def _send_line(self, line: str) -> None:
        """Send a newline-terminated line over TCP."""
        if not self._sock:
            raise ConnectionError("Not connected")
        self._sock.sendall((line + "\n").encode("utf-8"))

    def _recv_line(self) -> str:
        """Read a newline-terminated line from TCP."""
        if not self._sock:
            raise ConnectionError("Not connected")
        buf = b""
        while True:
            chunk = self._sock.recv(1)
            if not chunk:
                raise ConnectionError("Connection closed by server")
            if chunk == b"\n":
                break
            buf += chunk
            if len(buf) > 65536:
                raise ValueError("Line exceeds 64KB limit")
        return buf.decode("utf-8")

    # ── AUTH flow ───────────────────────────────────────────────

    def authenticate(self) -> str:
        """Perform the full AUTH challenge-response handshake.

        Returns the JWT session token.
        """
        if not self._connected:
            self.connect()

        # Step 1: Send AUTH command
        self._send_line("AUTH")

        # Step 2: Receive AUTH_CHALLENGE
        resp = self._recv_line()
        if resp != "AUTH_CHALLENGE":
            raise AuthError(f"Expected AUTH_CHALLENGE, got: {resp}")

        # Step 3: Send UUID
        self._send_line(self.uuid)

        # Step 4: Receive NONCE
        resp = self._recv_line()
        if not resp.startswith("NONCE "):
            raise AuthError(f"Expected NONCE, got: {resp}")
        nonce_hex = resp[6:]  # Strip "NONCE " prefix

        # Step 5: Sign the nonce (decode hex to bytes, then sign)
        nonce_bytes = bytes.fromhex(nonce_hex)
        signature_hex = sign_message(self.private_key, nonce_bytes)

        # Step 6: Send signature
        self._send_line(signature_hex)

        # Step 7: Receive AUTH_OK <JWT>
        resp = self._recv_line()
        if not resp.startswith("AUTH_OK "):
            raise AuthError(f"Authentication failed: {resp}")
        self._jwt = resp[8:]  # Strip "AUTH_OK " prefix
        logger.info("Authenticated successfully (session JWT received)")
        return self._jwt

    # ── REGISTER flow ──────────────────────────────────────────

    def register(self, timeout: float = 300) -> str:
        """Perform the REGISTER flow for a new robot.

        Blocks until admin approves/rejects or timeout.
        Returns the JWT session token on approval.
        """
        if not self._connected:
            self.connect()

        if not self.device_type:
            raise ValueError("device_type is required for registration")

        self._send_line("REGISTER")

        resp = self._recv_line()
        if resp != "REGISTER_CHALLENGE":
            raise AuthError(f"Expected REGISTER_CHALLENGE, got: {resp}")

        self._send_line(self.uuid)

        resp = self._recv_line()
        if not resp == "SEND_DEVICE_TYPE":
            raise AuthError(f"Expected SEND_DEVICE_TYPE, got: {resp}")

        self._send_line(self.device_type)

        resp = self._recv_line()
        if resp != "SEND_PUBLIC_KEY":
            raise AuthError(f"Expected SEND_PUBLIC_KEY, got: {resp}")

        self._send_line(self.public_key_hex)

        resp = self._recv_line()
        if resp != "REGISTER_PENDING":
            raise AuthError(f"Expected REGISTER_PENDING, got: {resp}")

        logger.info("Registration pending - waiting for admin approval...")

        # Wait for approval/rejection (server blocks until decision)
        self._sock.settimeout(timeout)
        try:
            resp = self._recv_line()
        except socket.timeout:
            raise AuthError("Registration timed out waiting for approval")

        if resp.startswith("REGISTER_OK "):
            self._jwt = resp[12:]
            logger.info("Registration approved")
            return self._jwt
        elif resp == "REGISTER_REJECTED":
            raise AuthError("Registration was rejected")
        else:
            raise AuthError(f"Unexpected registration response: {resp}")

    # ── PERSIST ─────────────────────────────────────────────────

    def persist(self) -> None:
        """Send PERSIST command to move from ephemeral Redis to permanent PostgreSQL."""
        self._send_line("PERSIST")
        resp = self._recv_line()
        if not resp.startswith("PERSIST_OK"):
            raise AuthError(f"Persist failed: {resp}")
        logger.info("Robot persisted to permanent storage")

    # ── Heartbeat ──────────────────────────────────────────────

    def send_heartbeat(self, extra_data: dict | None = None, ttl: int | None = None) -> None:
        """Send a single heartbeat on a separate TCP connection.

        Heartbeats are independent of the session connection and must be sent
        on their own connection (session mode forwards all lines to the handler).
        """
        self._heartbeat_seq += 1
        payload: dict = {"seq": self._heartbeat_seq}
        if ttl is not None:
            payload["ttl"] = ttl
        if extra_data is not None:
            payload["extra_data"] = extra_data

        payload_json = json.dumps(payload, separators=(",", ":"))
        signature_hex = sign_message(self.private_key, payload_json.encode("utf-8"))
        line = f"HEARTBEAT {self.uuid} {payload_json} {signature_hex}"

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(10)
        try:
            sock.connect((self.host, self.tcp_port))
            sock.sendall((line + "\n").encode("utf-8"))
            buf = b""
            while True:
                chunk = sock.recv(1)
                if not chunk or chunk == b"\n":
                    break
                buf += chunk
            resp = buf.decode("utf-8")
            if resp != "HEARTBEAT_OK":
                raise HeartbeatError(f"Heartbeat failed: {resp}")
            logger.debug("Heartbeat seq=%d OK", self._heartbeat_seq)
        finally:
            sock.close()

    def start_heartbeat(self, interval: float = 30, ttl: int | None = None) -> None:
        """Start a background thread that sends heartbeats at the given interval."""
        self._heartbeat_stop.clear()

        def _loop():
            while not self._heartbeat_stop.is_set():
                try:
                    self.send_heartbeat(ttl=ttl)
                except Exception as e:
                    logger.error("Heartbeat error: %s", e)
                    break
                self._heartbeat_stop.wait(interval)

        self._heartbeat_thread = threading.Thread(target=_loop, daemon=True)
        self._heartbeat_thread.start()
        logger.info("Heartbeat thread started (interval=%ds)", interval)

    def stop_heartbeat(self) -> None:
        """Stop the background heartbeat thread."""
        self._heartbeat_stop.set()
        if self._heartbeat_thread and self._heartbeat_thread.is_alive():
            self._heartbeat_thread.join(timeout=5)
        logger.info("Heartbeat thread stopped")

    # ── Messaging ──────────────────────────────────────────────

    def send(self, message: str) -> None:
        """Send a message to the server (in session mode, forwarded to handler)."""
        self._send_line(message)

    def on_message(self, callback: Callable[[str], None]) -> None:
        """Register a callback for incoming messages and start the receive loop."""
        self._message_callback = callback
        self._recv_stop.clear()

        def _loop():
            while not self._recv_stop.is_set():
                try:
                    line = self._recv_line()
                    if self._message_callback:
                        self._message_callback(line)
                except (ConnectionError, OSError):
                    if not self._recv_stop.is_set():
                        logger.warning("Connection lost")
                        self._connected = False
                    break
                except Exception as e:
                    logger.error("Receive error: %s", e)
                    break

        self._recv_thread = threading.Thread(target=_loop, daemon=True)
        self._recv_thread.start()



class AuthError(Exception):
    """Raised when authentication or registration fails."""
    pass


class HeartbeatError(Exception):
    """Raised when a heartbeat is rejected."""
    pass
