"""TCP protocol integration tests.

Tests AUTH, REGISTER, PERSIST, heartbeat, and messaging flows over TCP.
"""

import json
import socket
import threading
import time

import pytest

from conftest import (
    HOST, TCP_PORT, HTTP_PORT,
    SEED_UUID, SEED_PRIVATE_KEY_HEX,
)
from robomesh_sdk import RobotClient, generate_ed25519_keypair
from robomesh_sdk.client import AuthError, HeartbeatError
from robomesh_sdk.admin import AdminClient


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

class RawTCPClient:
    """Thin TCP helper for testing raw protocol interactions."""

    def __init__(self):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.settimeout(10)
        self.sock.connect((HOST, TCP_PORT))

    def send(self, msg: str):
        self.sock.sendall((msg + "\n").encode())

    def recv(self) -> str:
        buf = b""
        while not buf.endswith(b"\n"):
            chunk = self.sock.recv(1)
            if not chunk:
                raise ConnectionError("Connection closed")
            buf += chunk
        return buf.decode().strip()

    def close(self):
        self.sock.close()


def sign_nonce(priv_hex: str, nonce_hex: str) -> str:
    from robomesh_sdk.keys import sign_message, load_private_key
    key = load_private_key(priv_hex)
    return sign_message(key, bytes.fromhex(nonce_hex))


# ---------------------------------------------------------------------------
# AUTH flow tests
# ---------------------------------------------------------------------------

class TestTCPAuth:
    """TCP AUTH handshake tests."""

    def test_auth_with_seeded_robot(self):
        """Full AUTH flow with the pre-seeded example-001 robot."""
        client = RobotClient(
            uuid=SEED_UUID,
            private_key_hex=SEED_PRIVATE_KEY_HEX,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            jwt = client.authenticate()
            assert jwt is not None
            assert len(jwt) > 0
            assert client.connected
        finally:
            client.disconnect()

    def test_auth_with_provisioned_robot(self, provisioned_robot):
        """AUTH flow with a freshly provisioned robot."""
        robot_uuid, priv_hex = provisioned_robot
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

    def test_auth_unknown_robot(self):
        """AUTH with an unknown UUID should fail."""
        _, priv_hex, _ = generate_ed25519_keypair()
        client = RobotClient(
            uuid="nonexistent-robot-xyz",
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            with pytest.raises(AuthError):
                client.authenticate()
        finally:
            client.disconnect()

    def test_auth_wrong_key(self, admin):
        """AUTH with wrong private key should fail (signature mismatch)."""
        _, priv1_hex, pub1_hex = generate_ed25519_keypair()
        _, priv2_hex, _ = generate_ed25519_keypair()
        robot_uuid = f"integ-wrongkey-{int(time.time())}"
        admin.provision_robot(robot_uuid, pub1_hex, "test_robot")

        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv2_hex,  # Wrong key
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            with pytest.raises(AuthError):
                client.authenticate()
        finally:
            client.disconnect()

    def test_auth_challenge_response_protocol(self):
        """Verify the raw AUTH protocol messages."""
        c = RawTCPClient()
        try:
            c.send("AUTH")
            assert c.recv() == "AUTH_CHALLENGE"

            c.send(SEED_UUID)
            resp = c.recv()
            assert resp.startswith("NONCE ")
            nonce = resp.split(" ", 1)[1]
            assert len(nonce) > 0

            sig = sign_nonce(SEED_PRIVATE_KEY_HEX, nonce)
            c.send(sig)
            resp = c.recv()
            assert resp.startswith("AUTH_OK ")
        finally:
            c.close()


# ---------------------------------------------------------------------------
# REGISTER flow tests
# ---------------------------------------------------------------------------

class TestTCPRegister:
    """TCP REGISTER flow tests."""

    def test_register_and_approve(self, admin):
        """Full REGISTER → approve → authenticated session flow."""
        _, priv_hex, pub_hex = generate_ed25519_keypair()
        reg_uuid = f"integ-reg-{int(time.time())}"

        c = RawTCPClient()
        try:
            c.send("REGISTER")
            assert c.recv() == "REGISTER_CHALLENGE"

            c.send(reg_uuid)
            assert c.recv() == "SEND_DEVICE_TYPE"

            c.send("test_robot")
            assert c.recv() == "SEND_PUBLIC_KEY"

            c.send(pub_hex)
            assert c.recv() == "REGISTER_PENDING"

            # Approve via HTTP
            from conftest import ADMIN_USER, ADMIN_PASS
            admin_client = AdminClient(host=HOST, http_port=HTTP_PORT)
            admin_client.login(ADMIN_USER, ADMIN_PASS)

            # Use raw HTTP to approve
            import urllib.request
            req_data = json.dumps({"uuid": reg_uuid, "accept": True}).encode()
            req = urllib.request.Request(
                f"http://{HOST}:{HTTP_PORT}/register",
                data=req_data,
                headers={
                    "Content-Type": "application/json",
                    "Authorization": f"Bearer {admin_client.token}",
                },
                method="POST",
            )
            urllib.request.urlopen(req, timeout=10)

            resp = c.recv()
            assert resp.startswith("REGISTER_OK")
        finally:
            c.close()

    def test_register_and_reject(self, admin):
        """REGISTER → reject flow sends REGISTER_REJECTED."""
        _, _, pub_hex = generate_ed25519_keypair()
        rej_uuid = f"integ-rej-{int(time.time())}"

        c = RawTCPClient()
        try:
            c.send("REGISTER")
            c.recv()  # REGISTER_CHALLENGE
            c.send(rej_uuid)
            c.recv()  # SEND_DEVICE_TYPE
            c.send("test_robot")
            c.recv()  # SEND_PUBLIC_KEY
            c.send(pub_hex)
            assert c.recv() == "REGISTER_PENDING"

            # Reject via HTTP
            from conftest import ADMIN_USER, ADMIN_PASS
            admin_client = AdminClient(host=HOST, http_port=HTTP_PORT)
            admin_client.login(ADMIN_USER, ADMIN_PASS)

            import urllib.request
            req_data = json.dumps({"uuid": rej_uuid, "accept": False}).encode()
            req = urllib.request.Request(
                f"http://{HOST}:{HTTP_PORT}/register",
                data=req_data,
                headers={
                    "Content-Type": "application/json",
                    "Authorization": f"Bearer {admin_client.token}",
                },
                method="POST",
            )
            urllib.request.urlopen(req, timeout=10)

            assert c.recv() == "REGISTER_REJECTED"
        finally:
            c.close()


# ---------------------------------------------------------------------------
# Heartbeat tests
# ---------------------------------------------------------------------------

class TestTCPHeartbeat:
    """TCP heartbeat tests."""

    def test_heartbeat_after_auth(self, provisioned_robot):
        """Heartbeat succeeds after authentication."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat()
        finally:
            client.disconnect()

    def test_heartbeat_with_extra_data(self, provisioned_robot):
        """Heartbeat with extra_data and custom TTL."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat(extra_data={"battery": 85, "temp": 42}, ttl=120)
        finally:
            client.disconnect()

    def test_heartbeat_stale_sequence_rejected(self, provisioned_robot):
        """Server rejects heartbeat with a stale (non-increasing) sequence number."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat()  # seq=1
            # Force sequence back to 0 to trigger replay rejection
            client._heartbeat_seq = 0
            with pytest.raises(HeartbeatError):
                client.send_heartbeat()  # seq=1 again — should fail
        finally:
            client.disconnect()

    def test_multiple_heartbeats(self, provisioned_robot):
        """Multiple sequential heartbeats succeed with increasing sequence."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
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

class TestTCPMessaging:
    """TCP message sending tests."""

    def test_send_message_in_session(self, provisioned_robot):
        """Messages can be sent during an authenticated session."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            client.send("hello from integration test")
            time.sleep(0.2)  # Let message propagate
        finally:
            client.disconnect()


# ---------------------------------------------------------------------------
# Invalid command tests
# ---------------------------------------------------------------------------

class TestTCPInvalidCommands:
    """TCP protocol error handling tests."""

    def test_unknown_command_rejected(self):
        """Sending an unknown command results in ERROR."""
        c = RawTCPClient()
        try:
            c.send("FOOBAR")
            resp = c.recv()
            assert resp.startswith("ERROR")
        finally:
            c.close()

    def test_empty_line(self):
        """Sending an empty line results in ERROR."""
        c = RawTCPClient()
        try:
            c.send("")
            resp = c.recv()
            assert resp.startswith("ERROR")
        finally:
            c.close()
