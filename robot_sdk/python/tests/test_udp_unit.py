"""Unit tests for the Robomesh UDP client.

These tests mock the socket layer so they can run without a server.
"""

import json
import socket
import pytest
from unittest.mock import patch, MagicMock

from robomesh_sdk import generate_ed25519_keypair, RobotUDPClient
from robomesh_sdk.udp_client import UDPAuthError, UDPHeartbeatError, UDPMessageError


@pytest.fixture
def keypair():
    _, priv_hex, pub_hex = generate_ed25519_keypair()
    return priv_hex, pub_hex


@pytest.fixture
def client(keypair):
    priv_hex, _ = keypair
    c = RobotUDPClient(
        uuid="test-udp-001",
        private_key_hex=priv_hex,
        host="localhost",
        udp_port=5001,
    )
    return c


class TestUDPClientInit:
    def test_create_client(self, keypair):
        priv_hex, pub_hex = keypair
        c = RobotUDPClient(uuid="test-001", private_key_hex=priv_hex)
        assert c.uuid == "test-001"
        assert c.public_key_hex == pub_hex
        assert c.jwt is None

    def test_default_port(self, keypair):
        priv_hex, _ = keypair
        c = RobotUDPClient(uuid="test-001", private_key_hex=priv_hex)
        assert c.udp_port == 5001


class TestUDPConnect:
    def test_connect_creates_socket(self, client):
        client.connect()
        assert client._sock is not None
        assert client._sock.type == socket.SOCK_DGRAM
        client.disconnect()

    def test_disconnect_cleans_up(self, client):
        client.connect()
        client.disconnect()
        assert client._sock is None


class TestUDPAuth:
    def test_auth_success(self, client):
        """Test successful two-step auth flow."""
        nonce_hex = "abcdef1234567890" * 2

        def mock_recvfrom(bufsize):
            if not hasattr(mock_recvfrom, "call_count"):
                mock_recvfrom.call_count = 0
            mock_recvfrom.call_count += 1

            if mock_recvfrom.call_count == 1:
                # Step 1 response: nonce
                return (json.dumps({
                    "type": "auth_response",
                    "status": "nonce",
                    "nonce": nonce_hex,
                }).encode(), ("127.0.0.1", 5001))
            else:
                # Step 2 response: JWT
                return (json.dumps({
                    "type": "auth_response",
                    "status": "ok",
                    "jwt": "test.jwt.token",
                }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            jwt = client.authenticate()

            assert jwt == "test.jwt.token"
            assert client.jwt == "test.jwt.token"
            # Verify two packets were sent (step 1 and step 2)
            assert mock_sock.sendto.call_count == 2

            # Verify step 1 packet
            step1_data = json.loads(mock_sock.sendto.call_args_list[0][0][0])
            assert step1_data["type"] == "auth"
            assert step1_data["uuid"] == "test-udp-001"
            assert "signature" not in step1_data

            # Verify step 2 packet
            step2_data = json.loads(mock_sock.sendto.call_args_list[1][0][0])
            assert step2_data["type"] == "auth"
            assert step2_data["uuid"] == "test-udp-001"
            assert "signature" in step2_data
            assert step2_data["nonce"] == nonce_hex

    def test_auth_error_step1(self, client):
        """Test auth failure at step 1 (unknown robot)."""
        def mock_recvfrom(bufsize):
            return (json.dumps({
                "type": "auth_response",
                "status": "error",
                "error": "unknown robot",
            }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            with pytest.raises(UDPAuthError, match="unknown robot"):
                client.authenticate()

    def test_auth_error_step2(self, client):
        """Test auth failure at step 2 (bad signature)."""
        nonce_hex = "abcdef1234567890" * 2

        def mock_recvfrom(bufsize):
            if not hasattr(mock_recvfrom, "call_count"):
                mock_recvfrom.call_count = 0
            mock_recvfrom.call_count += 1

            if mock_recvfrom.call_count == 1:
                return (json.dumps({
                    "type": "auth_response",
                    "status": "nonce",
                    "nonce": nonce_hex,
                }).encode(), ("127.0.0.1", 5001))
            else:
                return (json.dumps({
                    "type": "auth_response",
                    "status": "error",
                    "error": "signature verification failed",
                }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            with pytest.raises(UDPAuthError, match="signature verification failed"):
                client.authenticate()


class TestUDPHeartbeat:
    def test_heartbeat_success(self, client):
        """Test sending a heartbeat."""
        def mock_recvfrom(bufsize):
            return (json.dumps({
                "type": "heartbeat_response",
                "status": "ok",
            }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            client.send_heartbeat()

            # Verify heartbeat packet
            sent_data = json.loads(mock_sock.sendto.call_args[0][0])
            assert sent_data["type"] == "heartbeat"
            assert sent_data["uuid"] == "test-udp-001"
            assert "signature" in sent_data
            assert sent_data["payload"]["seq"] == 1

    def test_heartbeat_with_extras(self, client):
        """Test heartbeat with extra data and TTL."""
        def mock_recvfrom(bufsize):
            return (json.dumps({
                "type": "heartbeat_response",
                "status": "ok",
            }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            client.send_heartbeat(extra_data={"battery": 85}, ttl=120)

            sent_data = json.loads(mock_sock.sendto.call_args[0][0])
            assert sent_data["payload"]["ttl"] == 120
            assert sent_data["payload"]["extra_data"] == {"battery": 85}

    def test_heartbeat_seq_increments(self, client):
        """Test that heartbeat sequence number increments."""
        call_count = 0

        def mock_recvfrom(bufsize):
            return (json.dumps({
                "type": "heartbeat_response",
                "status": "ok",
            }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            client.send_heartbeat()
            client.send_heartbeat()
            client.send_heartbeat()

            calls = mock_sock.sendto.call_args_list
            for i, call in enumerate(calls):
                data = json.loads(call[0][0])
                assert data["payload"]["seq"] == i + 1

    def test_heartbeat_error(self, client):
        """Test heartbeat error response."""
        def mock_recvfrom(bufsize):
            return (json.dumps({
                "type": "heartbeat_response",
                "status": "error",
                "error": "stale sequence",
            }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            with pytest.raises(UDPHeartbeatError, match="stale sequence"):
                client.send_heartbeat()


class TestUDPMessaging:
    def test_send_requires_auth(self, client):
        """Test that sending requires authentication."""
        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_socket_cls.return_value = mock_sock

            client.connect()
            with pytest.raises(UDPAuthError, match="Not authenticated"):
                client.send("hello")

    def test_send_message(self, client):
        """Test sending an authenticated message."""
        def mock_recvfrom(bufsize):
            return (json.dumps({
                "type": "message_response",
                "status": "ok",
            }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            client._jwt = "test.jwt.token"  # Simulate auth
            client.send("hello from robot")

            sent_data = json.loads(mock_sock.sendto.call_args[0][0])
            assert sent_data["type"] == "message"
            assert sent_data["uuid"] == "test-udp-001"
            assert sent_data["jwt"] == "test.jwt.token"
            assert sent_data["payload"] == "hello from robot"

    def test_send_message_error(self, client):
        """Test message error response."""
        def mock_recvfrom(bufsize):
            return (json.dumps({
                "type": "message_response",
                "status": "error",
                "error": "no handler running",
            }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            client._jwt = "test.jwt.token"
            with pytest.raises(UDPMessageError, match="no handler running"):
                client.send("hello")


class TestUDPSignatureIntegrity:
    def test_heartbeat_signature_matches_payload(self, client):
        """Verify the signature is computed over the exact payload JSON bytes."""
        from robomesh_sdk.keys import sign_message as verify_sign

        def mock_recvfrom(bufsize):
            return (json.dumps({
                "type": "heartbeat_response",
                "status": "ok",
            }).encode(), ("127.0.0.1", 5001))

        with patch("socket.socket") as mock_socket_cls:
            mock_sock = MagicMock()
            mock_sock.recvfrom = mock_recvfrom
            mock_socket_cls.return_value = mock_sock

            client.connect()
            client.send_heartbeat(extra_data={"temp": 42}, ttl=60)

            sent_data = json.loads(mock_sock.sendto.call_args[0][0])
            # Re-serialize the payload to get what the server would see
            payload_json = json.dumps(sent_data["payload"], separators=(",", ":"))
            # The signature should match signing the payload JSON bytes
            expected_sig = verify_sign(client.private_key, payload_json.encode("utf-8"))
            assert sent_data["signature"] == expected_sig
