"""UDP protocol integration tests.

Tests authentication, heartbeat, and messaging flows over UDP.
"""

import json
import socket
import time

import pytest

from conftest import (
    HOST, UDP_PORT,
    SEED_UUID, SEED_PRIVATE_KEY_HEX,
)
from robomesh_sdk import RobotUDPClient, generate_ed25519_keypair
from robomesh_sdk.udp_client import UDPAuthError, UDPHeartbeatError


# ---------------------------------------------------------------------------
# AUTH flow tests
# ---------------------------------------------------------------------------

class TestUDPAuth:
    """UDP challenge-response authentication tests."""

    def test_auth_with_seeded_robot(self):
        """Full two-step auth with the pre-seeded example-001."""
        client = RobotUDPClient(
            uuid=SEED_UUID,
            private_key_hex=SEED_PRIVATE_KEY_HEX,
            host=HOST,
            udp_port=UDP_PORT,
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

    def test_auth_unknown_robot(self):
        """Auth with unknown UUID returns error."""
        _, priv_hex, _ = generate_ed25519_keypair()
        client = RobotUDPClient(
            uuid="nonexistent-udp-robot",
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            with pytest.raises(UDPAuthError):
                client.authenticate()
        finally:
            client.disconnect()

    def test_auth_wrong_key(self, admin):
        """Auth with wrong private key should fail."""
        _, priv1_hex, pub1_hex = generate_ed25519_keypair()
        _, priv2_hex, _ = generate_ed25519_keypair()
        robot_uuid = f"integ-udp-wrongkey-{int(time.time())}"
        admin.provision_robot(robot_uuid, pub1_hex, "test_robot")

        client = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv2_hex,  # Wrong key
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            with pytest.raises(UDPAuthError):
                client.authenticate()
        finally:
            client.disconnect()

    def test_auth_raw_protocol(self):
        """Verify the raw JSON packet auth protocol."""
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        try:
            from robomesh_sdk.keys import sign_message, load_private_key

            # Step 1: Request nonce
            packet = json.dumps({"type": "auth", "uuid": SEED_UUID}).encode()
            sock.sendto(packet, (HOST, UDP_PORT))
            data, _ = sock.recvfrom(65535)
            resp = json.loads(data.decode())
            assert resp["status"] == "nonce"
            assert "nonce" in resp

            # Step 2: Sign and verify
            nonce_hex = resp["nonce"]
            key = load_private_key(SEED_PRIVATE_KEY_HEX)
            sig_hex = sign_message(key, bytes.fromhex(nonce_hex))

            packet = json.dumps({
                "type": "auth",
                "uuid": SEED_UUID,
                "nonce": nonce_hex,
                "signature": sig_hex,
            }).encode()
            sock.sendto(packet, (HOST, UDP_PORT))
            data, _ = sock.recvfrom(65535)
            resp = json.loads(data.decode())
            assert resp["status"] == "ok"
            assert "jwt" in resp
        finally:
            sock.close()


# ---------------------------------------------------------------------------
# Heartbeat tests
# ---------------------------------------------------------------------------

class TestUDPHeartbeat:
    """UDP signed heartbeat tests."""

    def test_heartbeat_after_auth(self, provisioned_robot):
        """Heartbeat succeeds after UDP authentication."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat()
        finally:
            client.disconnect()

    def test_heartbeat_with_extra_data(self, provisioned_robot):
        """Heartbeat with extra_data and custom TTL over UDP."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat(extra_data={"battery": 90}, ttl=120)
        finally:
            client.disconnect()

    def test_heartbeat_stale_sequence_rejected(self, provisioned_robot):
        """UDP server rejects stale heartbeat sequence."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat()  # seq=1
            client._heartbeat_seq = 0  # Reset to cause replay
            with pytest.raises(UDPHeartbeatError):
                client.send_heartbeat()  # seq=1 again
        finally:
            client.disconnect()

    def test_multiple_heartbeats(self, provisioned_robot):
        """Multiple sequential heartbeats succeed."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
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

class TestUDPMessaging:
    """UDP JWT-authenticated messaging tests."""

    def test_send_message(self, provisioned_robot):
        """Message can be sent after authentication."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotUDPClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            client.authenticate()
            client.send("hello from UDP integration test")
        finally:
            client.disconnect()

    def test_message_without_auth_fails(self):
        """Sending a message without authentication should fail."""
        _, priv_hex, _ = generate_ed25519_keypair()
        client = RobotUDPClient(
            uuid="no-auth-robot",
            private_key_hex=priv_hex,
            host=HOST,
            udp_port=UDP_PORT,
        )
        try:
            client.connect()
            with pytest.raises(UDPAuthError):
                client.send("should fail")
        finally:
            client.disconnect()


# ---------------------------------------------------------------------------
# Error handling tests
# ---------------------------------------------------------------------------

class TestUDPErrors:
    """UDP protocol error handling."""

    def test_malformed_packet(self):
        """Malformed JSON packet doesn't crash the server."""
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(3)
        try:
            sock.sendto(b"this is not json", (HOST, UDP_PORT))
            # Server should either respond with an error or ignore silently
            # Either way, it shouldn't crash
            try:
                data, _ = sock.recvfrom(65535)
                # If we get a response, it should be an error
                resp = json.loads(data.decode())
                assert resp.get("status") == "error"
            except socket.timeout:
                pass  # Server ignored the malformed packet — acceptable
        finally:
            sock.close()

    def test_missing_type_field(self):
        """Packet without 'type' field is rejected."""
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(3)
        try:
            sock.sendto(json.dumps({"uuid": "test"}).encode(), (HOST, UDP_PORT))
            try:
                data, _ = sock.recvfrom(65535)
                resp = json.loads(data.decode())
                assert resp.get("status") == "error"
            except socket.timeout:
                pass  # Ignored — acceptable
        finally:
            sock.close()
