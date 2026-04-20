"""HTTP API integration tests.

Tests provisioning, robot status, handler lifecycle, and auth endpoints.
"""

import json
import time
import urllib.request
import urllib.error

import pytest

from conftest import (
    HOST, HTTP_PORT, TCP_PORT,
    ADMIN_USER, ADMIN_PASS,
    SEED_UUID, SEED_PRIVATE_KEY_HEX,
)
from robomesh_sdk import RobotClient, generate_ed25519_keypair
from robomesh_sdk.admin import AdminClient, AdminError


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def http_request(method, path, body=None, token=None):
    """Make an HTTP request and return (status_code, parsed_body)."""
    url = f"http://{HOST}:{HTTP_PORT}{path}"
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    if token:
        req.add_header("Authorization", f"Bearer {token}")
    if data:
        req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        body_text = e.read().decode() if e.fp else ""
        try:
            return e.code, json.loads(body_text)
        except Exception:
            return e.code, body_text


# ---------------------------------------------------------------------------
# Auth endpoint tests
# ---------------------------------------------------------------------------

class TestHTTPAuth:
    """HTTP authentication endpoints."""

    def test_login_success(self):
        """POST /auth/login with valid credentials returns a token."""
        status, body = http_request("POST", "/auth/login", {
            "username": ADMIN_USER,
            "password": ADMIN_PASS,
        })
        assert status == 200
        assert "token" in body
        assert len(body["token"]) > 0

    def test_login_bad_password(self):
        """POST /auth/login with wrong password returns 401."""
        status, _ = http_request("POST", "/auth/login", {
            "username": ADMIN_USER,
            "password": "wrong_password_12345",
        })
        assert status == 401

    def test_auth_check_with_token(self, admin):
        """GET /auth with valid token returns 200."""
        status, _ = http_request("GET", "/auth", token=admin.token)
        assert status == 200

    def test_auth_check_without_token(self):
        """GET /auth without token returns 401."""
        status, _ = http_request("GET", "/auth")
        assert status == 401

    def test_auth_check_with_invalid_token(self):
        """GET /auth with garbage token returns 401."""
        status, _ = http_request("GET", "/auth", token="not-a-valid-jwt")
        assert status == 401


# ---------------------------------------------------------------------------
# Provisioning tests
# ---------------------------------------------------------------------------

class TestHTTPProvisioning:
    """HTTP provisioning endpoints."""

    def test_provision_robot(self, admin, robot_identity):
        """POST /provision creates a new robot."""
        robot_uuid, _, pub_hex = robot_identity
        status, body = http_request(
            "POST", "/provision",
            {"uuid": robot_uuid, "public_key": pub_hex, "device_type": "test_robot"},
            token=admin.token,
        )
        assert status == 201
        assert body["uuid"] == robot_uuid

    def test_provision_without_auth(self, robot_identity):
        """POST /provision without auth returns 401."""
        robot_uuid, _, pub_hex = robot_identity
        status, _ = http_request(
            "POST", "/provision",
            {"uuid": robot_uuid, "public_key": pub_hex, "device_type": "test_robot"},
        )
        assert status == 401

    def test_get_provisioned_robot(self, admin, robot_identity):
        """GET /provision/{uuid} returns robot details."""
        robot_uuid, _, pub_hex = robot_identity
        admin.provision_robot(robot_uuid, pub_hex, "test_robot")
        status, body = http_request(
            "GET", f"/provision/{robot_uuid}",
            token=admin.token,
        )
        assert status == 200
        assert body["UUID"] == robot_uuid

    def test_list_provisioned_robots(self, admin):
        """GET /provision returns a list."""
        status, body = http_request("GET", "/provision", token=admin.token)
        assert status == 200
        assert isinstance(body, list)

    def test_get_nonexistent_robot(self, admin):
        """GET /provision/{uuid} for unknown robot returns 404."""
        status, _ = http_request(
            "GET", "/provision/this-robot-does-not-exist",
            token=admin.token,
        )
        assert status == 404


# ---------------------------------------------------------------------------
# Robot status tests
# ---------------------------------------------------------------------------

class TestHTTPRobotStatus:
    """HTTP robot status endpoints."""

    def test_robot_shows_online_after_tcp_auth(self, admin, provisioned_robot):
        """Robot appears online after TCP authentication."""
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
            assert status["uuid"] == robot_uuid
            assert status["online"] is True
        finally:
            client.disconnect()

    def test_list_active_robots(self, admin):
        """GET /robot returns a list of active robots."""
        status, body = http_request("GET", "/robot", token=admin.token)
        assert status == 200
        assert isinstance(body, list)


# ---------------------------------------------------------------------------
# Handler lifecycle tests
# ---------------------------------------------------------------------------

class TestHTTPHandlerLifecycle:
    """HTTP handler start/kill/status endpoints."""

    def test_handler_spawned_on_auth(self, admin, provisioned_robot):
        """Handler is automatically started when a robot authenticates."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            time.sleep(0.5)
            status, body = http_request(
                "GET", f"/handler/{robot_uuid}",
                token=admin.token,
            )
            assert status == 200
            assert body.get("active") is True
        finally:
            client.disconnect()

    def test_handler_kill(self, admin, provisioned_robot):
        """Handler can be killed via HTTP."""
        robot_uuid, priv_hex = provisioned_robot
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            time.sleep(0.5)

            # Kill the handler
            status, _ = http_request(
                "POST", f"/handler/{robot_uuid}/kill",
                token=admin.token,
            )
            assert status == 200

            time.sleep(0.5)
            # Verify handler is no longer active
            status, body = http_request(
                "GET", f"/handler/{robot_uuid}",
                token=admin.token,
            )
            assert status == 200
            assert body.get("active") is False
        finally:
            client.disconnect()

    def test_handler_manual_start(self, admin, provisioned_robot):
        """Handler can be manually started via HTTP (even without TCP connection)."""
        robot_uuid, priv_hex = provisioned_robot
        # Authenticate to create a session, then disconnect
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()
            time.sleep(0.5)
            # Kill any existing handler
            http_request("POST", f"/handler/{robot_uuid}/kill", token=admin.token)
            time.sleep(0.5)
        finally:
            client.disconnect()

        # Now start handler manually (no TCP connection)
        status, _ = http_request(
            "POST", f"/handler/{robot_uuid}/start",
            token=admin.token,
        )
        assert status == 200

        time.sleep(0.5)
        status, body = http_request(
            "GET", f"/handler/{robot_uuid}",
            token=admin.token,
        )
        assert status == 200
        assert body.get("active") is True

        # Cleanup
        http_request("POST", f"/handler/{robot_uuid}/kill", token=admin.token)

    def test_list_handlers(self, admin):
        """GET /handler/ returns handler list."""
        status, body = http_request("GET", "/handler/", token=admin.token)
        assert status == 200


# ---------------------------------------------------------------------------
# Heartbeat endpoint tests
# ---------------------------------------------------------------------------

class TestHTTPHeartbeat:
    """HTTP heartbeat endpoint tests."""

    def test_heartbeat_via_http(self, admin, provisioned_robot):
        """POST /heartbeat with valid signature succeeds."""
        robot_uuid, priv_hex = provisioned_robot
        from robomesh_sdk.keys import sign_message, load_private_key

        # Authenticate first to create a session
        client = RobotClient(
            uuid=robot_uuid,
            private_key_hex=priv_hex,
            host=HOST,
            tcp_port=TCP_PORT,
        )
        try:
            client.authenticate()

            # Send heartbeat via HTTP
            payload = json.dumps({"seq": 1}, separators=(",", ":"))
            key = load_private_key(priv_hex)
            sig = sign_message(key, payload.encode())

            status, body = http_request("POST", "/heartbeat", {
                "uuid": robot_uuid,
                "payload": payload,
                "signature": sig,
            })
            assert status == 200
        finally:
            client.disconnect()
