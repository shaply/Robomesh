"""Integration tests for the Robomesh Python SDK.

Requires a running roboserver (use docker-compose.dev.yml).
Default dev server: TCP on localhost:5001, HTTP on localhost:8080.

Run:
    cd robot_sdk/python
    pip install -e .
    pytest tests/test_integration.py -v
"""

import os
import time
import uuid as uuid_lib
import pytest

from robomesh_sdk import RobotClient, generate_ed25519_keypair
from robomesh_sdk.admin import AdminClient, AdminError
from robomesh_sdk.client import AuthError, HeartbeatError

# Connection settings (override with env vars for different setups)
HOST = os.environ.get("ROBOMESH_HOST", "localhost")
TCP_PORT = int(os.environ.get("ROBOMESH_TCP_PORT", "5001"))
HTTP_PORT = int(os.environ.get("ROBOMESH_HTTP_PORT", "8080"))
ADMIN_USER = os.environ.get("ROBOMESH_ADMIN_USER", "admin")
ADMIN_PASS = os.environ.get("ROBOMESH_ADMIN_PASS", "password1")


@pytest.fixture(scope="module")
def admin():
    """Admin client logged in and ready."""
    client = AdminClient(host=HOST, http_port=HTTP_PORT)
    client.login(ADMIN_USER, ADMIN_PASS)
    return client


@pytest.fixture
def robot_identity():
    """Generate a fresh robot identity (UUID + keypair)."""
    _, priv_hex, pub_hex = generate_ed25519_keypair()
    robot_uuid = f"test-{uuid_lib.uuid4().hex[:12]}"
    return robot_uuid, priv_hex, pub_hex


@pytest.fixture
def provisioned_robot(admin, robot_identity):
    """Provision a fresh robot and return (uuid, private_key_hex)."""
    robot_uuid, priv_hex, pub_hex = robot_identity
    admin.provision_robot(robot_uuid, pub_hex, "test_robot")
    return robot_uuid, priv_hex


class TestAdminLogin:
    def test_login_success(self):
        admin = AdminClient(host=HOST, http_port=HTTP_PORT)
        token = admin.login(ADMIN_USER, ADMIN_PASS)
        assert token is not None
        assert len(token) > 0

    def test_login_bad_password(self):
        admin = AdminClient(host=HOST, http_port=HTTP_PORT)
        with pytest.raises(AdminError, match="401"):
            admin.login(ADMIN_USER, "wrong_password")


class TestProvision:
    def test_provision_robot(self, admin, robot_identity):
        robot_uuid, _, pub_hex = robot_identity
        result = admin.provision_robot(robot_uuid, pub_hex, "test_robot")
        assert result["status"] == "provisioned"
        assert result["uuid"] == robot_uuid

    def test_get_all_robots(self, admin):
        robots = admin.get_all_robots()
        assert isinstance(robots, list)


class TestAuth:
    def test_auth_flow(self, provisioned_robot):
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
            assert len(jwt) > 0
            assert client.connected
        finally:
            client.disconnect()

    def test_auth_unknown_robot(self):
        _, priv_hex, _ = generate_ed25519_keypair()
        client = RobotClient(
            uuid="nonexistent-robot",
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            with pytest.raises(AuthError, match="UNKNOWN_ROBOT|ERROR"):
                client.authenticate()
        finally:
            client.disconnect()

    def test_auth_wrong_key(self, admin):
        """Provision with one key, try to auth with a different one."""
        _, priv1_hex, pub1_hex = generate_ed25519_keypair()
        _, priv2_hex, _ = generate_ed25519_keypair()
        robot_uuid = f"test-wrongkey-{uuid_lib.uuid4().hex[:8]}"
        admin.provision_robot(robot_uuid, pub1_hex, "test_robot")

        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv2_hex,  # Wrong key!
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            with pytest.raises(AuthError, match="INVALID_SIGNATURE|ERROR"):
                client.authenticate()
        finally:
            client.disconnect()

    def test_auth_with_seeded_robot(self):
        """Test auth using the pre-seeded example-001 robot from seed.sql."""
        priv_hex = "c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1"
        client = RobotClient(
            uuid="example-001",
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            jwt = client.authenticate()
            assert jwt is not None
        finally:
            client.disconnect()


class TestHeartbeat:
    def test_heartbeat_after_auth(self, provisioned_robot):
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            # Heartbeats on the session connection
            client.send_heartbeat()
            client.send_heartbeat(extra_data={"battery": 85})
            client.send_heartbeat(ttl=120)
        finally:
            client.disconnect()

    def test_heartbeat_with_extra_data(self, provisioned_robot):
        """Test heartbeat with extra data and custom TTL."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            client.send_heartbeat(extra_data={"temp": 42}, ttl=120)
        finally:
            client.disconnect()

    def test_heartbeat_stale_sequence(self, provisioned_robot):
        """Verify the server rejects stale heartbeat sequences."""
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
            # Force sequence back to 0
            client._heartbeat_seq = 0
            with pytest.raises(HeartbeatError, match="stale|ERROR"):
                client.send_heartbeat()  # seq=1 again - should fail
        finally:
            client.disconnect()


class TestMessaging:
    def test_send_message_in_session(self, provisioned_robot):
        """After auth, we should be able to send messages (handler receives them)."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            # Send a message - even if no handler echoes back, this shouldn't error
            client.send("hello from test robot")
            # Small delay to let message propagate
            time.sleep(0.1)
        finally:
            client.disconnect()


class TestRobotStatus:
    def test_robot_shows_online_after_auth(self, admin, provisioned_robot):
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            status = admin.get_robot_status(robot_uuid)
            assert status["uuid"] == robot_uuid
            assert status["online"] is True
        finally:
            client.disconnect()
