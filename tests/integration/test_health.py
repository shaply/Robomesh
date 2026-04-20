"""Server health and reachability tests.

Verifies that all server endpoints are reachable before running protocol tests.
These tests should pass first — if they fail, other tests are meaningless.
"""

import json
import socket
import urllib.request
import urllib.error

import pytest

from conftest import HOST, HTTP_PORT, TCP_PORT, UDP_PORT, MQTT_PORT


class TestServerReachability:
    """Verify all server ports are reachable."""

    def test_http_reachable(self):
        """HTTP API responds to requests."""
        req = urllib.request.Request(f"http://{HOST}:{HTTP_PORT}/auth", method="GET")
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                # 401 is expected (no token) — but it means the server is up
                pass
        except urllib.error.HTTPError as e:
            assert e.code in (401, 200), f"Unexpected HTTP status: {e.code}"
        except Exception as e:
            pytest.fail(f"HTTP server not reachable at {HOST}:{HTTP_PORT}: {e}")

    def test_tcp_reachable(self):
        """TCP server accepts connections."""
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5)
        try:
            sock.connect((HOST, TCP_PORT))
        except Exception as e:
            pytest.fail(f"TCP server not reachable at {HOST}:{TCP_PORT}: {e}")
        finally:
            sock.close()

    def test_udp_reachable(self):
        """UDP server responds to packets."""
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(5)
        try:
            # Send a minimal auth request — even if it fails, a response means the server is up
            packet = json.dumps({"type": "auth", "uuid": "health-check"}).encode()
            sock.sendto(packet, (HOST, UDP_PORT))
            data, _ = sock.recvfrom(65535)
            resp = json.loads(data.decode())
            # Any response (even error) means UDP server is alive
            assert "type" in resp or "status" in resp
        except socket.timeout:
            pytest.fail(f"UDP server not responding at {HOST}:{UDP_PORT}")
        finally:
            sock.close()

    def test_mqtt_reachable(self):
        """MQTT broker accepts connections."""
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5)
        try:
            sock.connect((HOST, MQTT_PORT))
        except Exception as e:
            pytest.fail(f"MQTT broker not reachable at {HOST}:{MQTT_PORT}: {e}")
        finally:
            sock.close()


class TestAdminAuth:
    """Verify admin login works (required for all other tests)."""

    def test_admin_login(self):
        """Admin can log in with seeded credentials."""
        from conftest import ADMIN_USER, ADMIN_PASS
        from robomesh_sdk.admin import AdminClient

        client = AdminClient(host=HOST, http_port=HTTP_PORT)
        token = client.login(ADMIN_USER, ADMIN_PASS)
        assert token is not None
        assert len(token) > 0

    def test_admin_login_bad_password(self):
        """Login with wrong password fails."""
        from robomesh_sdk.admin import AdminClient, AdminError

        client = AdminClient(host=HOST, http_port=HTTP_PORT)
        with pytest.raises(AdminError):
            client.login("admin", "wrong_password_12345")
