"""Admin HTTP helpers for provisioning robots and managing the server."""

import json
import urllib.request
import urllib.error
import logging

logger = logging.getLogger("robomesh_sdk")


class AdminClient:
    """HTTP client for admin operations (provisioning, login, robot status).

    Usage:
        admin = AdminClient(host="localhost", http_port=8080)
        admin.login("admin", "password1")
        admin.provision_robot("my-robot-001", public_key_hex, "example_robot")
    """

    def __init__(self, host: str = "localhost", http_port: int = 8080):
        self.base_url = f"http://{host}:{http_port}"
        self._token: str | None = None

    @property
    def token(self) -> str | None:
        return self._token

    def login(self, username: str, password: str) -> str:
        """Login as a user and store the JWT token. Returns the token."""
        data = json.dumps({"username": username, "password": password}).encode()
        req = urllib.request.Request(
            f"{self.base_url}/auth/login",
            data=data,
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        resp = self._do_request(req)
        self._token = resp["token"]
        logger.info("Admin logged in as %s", username)
        return self._token

    def provision_robot(self, uuid: str, public_key: str, device_type: str) -> dict:
        """Provision (register) a robot's public key in the server database."""
        data = json.dumps({
            "uuid": uuid,
            "public_key": public_key,
            "device_type": device_type,
        }).encode()
        req = urllib.request.Request(
            f"{self.base_url}/provision",
            data=data,
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self._token}",
            },
            method="POST",
        )
        resp = self._do_request(req)
        logger.info("Provisioned robot %s (type=%s)", uuid, device_type)
        return resp

    def get_robot_status(self, uuid: str) -> dict:
        """Get the online/offline status of a robot."""
        req = urllib.request.Request(
            f"{self.base_url}/provision/{uuid}/status",
            headers={"Authorization": f"Bearer {self._token}"},
        )
        return self._do_request(req)

    def get_all_robots(self) -> list:
        """Get all registered robots."""
        req = urllib.request.Request(
            f"{self.base_url}/provision",
            headers={"Authorization": f"Bearer {self._token}"},
        )
        return self._do_request(req)

    def _do_request(self, req: urllib.request.Request) -> dict | list:
        """Execute an HTTP request and return parsed JSON."""
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                return json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            body = e.read().decode() if e.fp else ""
            raise AdminError(f"HTTP {e.code}: {body.strip()}") from e
        except urllib.error.URLError as e:
            raise AdminError(f"Connection failed: {e.reason}") from e


class AdminError(Exception):
    """Raised on admin HTTP errors."""
    pass
