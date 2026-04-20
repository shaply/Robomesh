"""Shared fixtures and configuration for Robomesh integration tests.

These tests require a running roboserver instance. Start one with:
    docker compose -f docker-compose.dev.yml up --build

Connection settings are read from environment variables (with defaults
matching docker-compose.dev.yml). Override them for different environments:
    ROBOMESH_HOST          (default: localhost)
    ROBOMESH_HTTP_PORT     (default: 8080)
    ROBOMESH_TCP_PORT      (default: 5002)
    ROBOMESH_UDP_PORT      (default: 5001)
    ROBOMESH_MQTT_PORT     (default: 1883)
    ROBOMESH_ADMIN_USER    (default: admin)
    ROBOMESH_ADMIN_PASS    (default: password1)
"""

import os
import sys
import uuid as uuid_lib

import pytest

# Add the Python SDK to the path so tests can import it without installation
SDK_PATH = os.path.join(os.path.dirname(__file__), "..", "..", "robot_sdk", "python")
sys.path.insert(0, SDK_PATH)

from robomesh_sdk import generate_ed25519_keypair
from robomesh_sdk.admin import AdminClient

# ---------------------------------------------------------------------------
# Connection settings (single source of truth for all test modules)
# ---------------------------------------------------------------------------
HOST = os.environ.get("ROBOMESH_HOST", "localhost")
HTTP_PORT = int(os.environ.get("ROBOMESH_HTTP_PORT", "8080"))
TCP_PORT = int(os.environ.get("ROBOMESH_TCP_PORT", "5002"))
UDP_PORT = int(os.environ.get("ROBOMESH_UDP_PORT", "5001"))
MQTT_PORT = int(os.environ.get("ROBOMESH_MQTT_PORT", "1883"))
ADMIN_USER = os.environ.get("ROBOMESH_ADMIN_USER", "admin")
ADMIN_PASS = os.environ.get("ROBOMESH_ADMIN_PASS", "password1")

# Pre-seeded robot from db/seed.sql (RFC 8032 test vector #1)
SEED_UUID = "example-001"
SEED_PRIVATE_KEY_HEX = "c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1"
SEED_PUBLIC_KEY_HEX = "b702036ee61847fdabecc07ce7da7b432c39aba98d1114c1c6f6f3f586ba98aa"


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def admin():
    """Admin client logged in for the entire test session."""
    client = AdminClient(host=HOST, http_port=HTTP_PORT)
    client.login(ADMIN_USER, ADMIN_PASS)
    return client


@pytest.fixture
def robot_identity():
    """Generate a fresh robot identity (uuid, private_key_hex, public_key_hex)."""
    _, priv_hex, pub_hex = generate_ed25519_keypair()
    robot_uuid = f"integ-{uuid_lib.uuid4().hex[:12]}"
    return robot_uuid, priv_hex, pub_hex


@pytest.fixture
def provisioned_robot(admin, robot_identity):
    """Provision a fresh robot and return (uuid, private_key_hex)."""
    robot_uuid, priv_hex, pub_hex = robot_identity
    admin.provision_robot(robot_uuid, pub_hex, "test_robot")
    return robot_uuid, priv_hex
