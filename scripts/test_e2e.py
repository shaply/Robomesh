#!/usr/bin/env python3
"""
End-to-end test for Robomesh server flows.

Tests:
  1. AUTH flow (cryptographic handshake with pre-registered robot)
  2. REGISTER flow (robot self-registration with user approval)
  3. Handler communication (echo test via handler script)
  4. PERSIST flow (ephemeral -> permanent storage)

Prerequisites:
  - Server running (docker compose -f docker-compose.dev.yml up --build)
  - Database seeded with example-001 robot (uses RFC 8032 test vector #1)

Usage:
  pip install cryptography
  python3 scripts/test_e2e.py
"""

import json
import os
import socket
import sys
import threading
import time
import urllib.request

from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives import serialization

HOST = os.environ.get("ROBOMESH_HOST", "localhost")
TCP_PORT = int(os.environ.get("ROBOMESH_TCP_PORT", "5002"))
HTTP_PORT = int(os.environ.get("ROBOMESH_HTTP_PORT", "8080"))
AUTH_TOKEN = "jwt-token-user-123"  # Matches the placeholder auth in auth.go

# Test keypair (matches seed.sql for example-001)
SEED_PRIVATE_KEY_HEX = "c55608b70c4a9f3b43bd1d23e86aaf4c3b2f4b823f54dc34ac668e85363ef2e1"
SEED_PUBLIC_KEY_HEX = "b702036ee61847fdabecc07ce7da7b432c39aba98d1114c1c6f6f3f586ba98aa"
SEED_UUID = "example-001"

# Counters
passed = 0
failed = 0


def log(msg):
    print(f"  {msg}")


def pass_test(name):
    global passed
    passed += 1
    print(f"  PASS: {name}")


def fail_test(name, reason=""):
    global failed
    failed += 1
    print(f"  FAIL: {name} — {reason}")


# --- TCP helpers ---

class TCPClient:
    def __init__(self, host=HOST, port=TCP_PORT):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.settimeout(10)
        self.sock.connect((host, port))

    def send(self, msg):
        self.sock.sendall((msg + "\n").encode())

    def recv(self):
        data = b""
        while not data.endswith(b"\n"):
            chunk = self.sock.recv(1)
            if not chunk:
                raise ConnectionError("Connection closed")
            data += chunk
        return data.decode().strip()

    def close(self):
        self.sock.close()


def sign_nonce(private_key_hex, nonce_hex):
    priv_bytes = bytes.fromhex(private_key_hex)
    key = Ed25519PrivateKey.from_private_bytes(priv_bytes)
    nonce_bytes = bytes.fromhex(nonce_hex)
    signature = key.sign(nonce_bytes)
    return signature.hex()


def generate_keypair():
    """Generate a fresh Ed25519 keypair for REGISTER flow tests."""
    key = Ed25519PrivateKey.generate()
    priv = key.private_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PrivateFormat.Raw,
        encryption_algorithm=serialization.NoEncryption(),
    ).hex()
    pub = key.public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw,
    ).hex()
    return priv, pub


# --- HTTP helpers ---

def http_request(method, path, body=None):
    url = f"http://{HOST}:{HTTP_PORT}{path}"
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Authorization", f"Bearer {AUTH_TOKEN}")
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
    except Exception as e:
        return 0, str(e)


# ========================================================================
# Test 1: AUTH flow (pre-registered robot from seed.sql)
# ========================================================================
def test_auth_flow():
    print("\n[Test 1] AUTH flow — pre-registered robot")
    try:
        c = TCPClient()

        # Send AUTH
        c.send("AUTH")
        resp = c.recv()
        if resp != "AUTH_CHALLENGE":
            fail_test("AUTH_CHALLENGE", f"expected AUTH_CHALLENGE, got: {resp}")
            c.close()
            return
        pass_test("AUTH_CHALLENGE received")

        # Send UUID
        c.send(SEED_UUID)
        resp = c.recv()
        if not resp.startswith("NONCE "):
            fail_test("NONCE", f"expected NONCE, got: {resp}")
            c.close()
            return
        nonce = resp.split(" ", 1)[1]
        pass_test(f"NONCE received ({nonce[:16]}...)")

        # Sign nonce and send signature
        sig = sign_nonce(SEED_PRIVATE_KEY_HEX, nonce)
        c.send(sig)
        resp = c.recv()
        if not resp.startswith("AUTH_OK"):
            fail_test("AUTH_OK", f"expected AUTH_OK, got: {resp}")
            c.close()
            return
        jwt = resp.split(" ", 1)[1] if " " in resp else ""
        pass_test(f"AUTH_OK received (JWT: {jwt[:30]}...)")

        # Test handler communication — send a message, expect echo
        c.send("hello world")
        time.sleep(0.5)
        # Read the WELCOME message first (from connect handler)
        resp = c.recv()
        log(f"Handler response: {resp}")
        if "WELCOME" in resp or "ECHO" in resp:
            pass_test("Handler communication works")
        else:
            # Could be the echo, try reading one more
            try:
                resp2 = c.recv()
                log(f"Handler response 2: {resp2}")
                if "ECHO" in resp2:
                    pass_test("Handler communication works")
                else:
                    fail_test("Handler communication", f"unexpected response: {resp2}")
            except Exception:
                fail_test("Handler communication", f"unexpected response: {resp}")

        c.close()

    except Exception as e:
        fail_test("AUTH flow", str(e))


# ========================================================================
# Test 2: REGISTER flow (new robot, user approval)
# ========================================================================
def test_register_flow():
    print("\n[Test 2] REGISTER flow — new robot with user approval")
    reg_uuid = f"reg-test-{int(time.time())}"
    priv_hex, pub_hex = generate_keypair()

    try:
        c = TCPClient()

        # Send REGISTER
        c.send("REGISTER")
        resp = c.recv()
        if resp != "REGISTER_CHALLENGE":
            fail_test("REGISTER_CHALLENGE", f"got: {resp}")
            c.close()
            return
        pass_test("REGISTER_CHALLENGE received")

        # Send UUID
        c.send(reg_uuid)
        resp = c.recv()
        if resp != "SEND_DEVICE_TYPE":
            fail_test("SEND_DEVICE_TYPE", f"got: {resp}")
            c.close()
            return
        pass_test("SEND_DEVICE_TYPE prompt received")

        # Send device type
        c.send("example_robot")
        resp = c.recv()
        if resp != "SEND_PUBLIC_KEY":
            fail_test("SEND_PUBLIC_KEY", f"got: {resp}")
            c.close()
            return
        pass_test("SEND_PUBLIC_KEY prompt received")

        # Send public key
        c.send(pub_hex)
        resp = c.recv()
        if resp != "REGISTER_PENDING":
            fail_test("REGISTER_PENDING", f"got: {resp}")
            c.close()
            return
        pass_test("REGISTER_PENDING received — waiting for approval")

        # Check pending list via HTTP
        status, pending = http_request("GET", "/register/pending")
        if status != 200:
            fail_test("GET /register/pending", f"status={status}")
        else:
            found = any(r.get("uuid") == reg_uuid for r in pending) if isinstance(pending, list) else False
            if found:
                pass_test(f"Robot {reg_uuid} found in pending list")
            else:
                fail_test("Pending list", f"robot {reg_uuid} not found in pending: {pending}")

        # Accept the registration via HTTP
        status, body = http_request("POST", "/register", {"uuid": reg_uuid, "accept": True})
        if status == 200:
            pass_test(f"Registration accepted via HTTP: {body}")
        else:
            fail_test("Accept registration", f"status={status}, body={body}")

        # Robot should receive REGISTER_OK
        resp = c.recv()
        if resp.startswith("REGISTER_OK"):
            jwt = resp.split(" ", 1)[1] if " " in resp else ""
            pass_test(f"REGISTER_OK received (JWT: {jwt[:30]}...)")
        else:
            fail_test("REGISTER_OK", f"got: {resp}")
            c.close()
            return

        # Verify robot is active via HTTP
        time.sleep(0.5)
        status, body = http_request("GET", f"/robot/{reg_uuid}")
        if status == 200 and body.get("online"):
            pass_test(f"Robot {reg_uuid} is active in Redis")
        else:
            fail_test("Active check", f"status={status}, body={body}")

        # Test handler echo
        c.send("ping from registered robot")
        time.sleep(0.5)
        # Read responses (might get WELCOME first, then ECHO)
        try:
            resp = c.recv()
            log(f"Handler: {resp}")
            if "ECHO" not in resp:
                resp = c.recv()
                log(f"Handler: {resp}")
            if "ECHO" in resp:
                pass_test("Handler echo works for registered robot")
            else:
                fail_test("Handler echo", f"got: {resp}")
        except Exception as e:
            fail_test("Handler echo", str(e))

        # Test PERSIST flow
        c.send("PERSIST")
        resp = c.recv()
        if resp == "PERSIST_OK":
            pass_test("PERSIST_OK received — robot stored in PostgreSQL")
        else:
            fail_test("PERSIST", f"got: {resp}")

        # Verify robot is now in PostgreSQL
        status, body = http_request("GET", f"/provision/{reg_uuid}")
        if status == 200 and body.get("UUID") == reg_uuid:
            pass_test(f"Robot {reg_uuid} found in PostgreSQL registry")
        else:
            fail_test("PostgreSQL check", f"status={status}, body={body}")

        c.close()

    except Exception as e:
        fail_test("REGISTER flow", str(e))


# ========================================================================
# Test 3: REGISTER rejection flow
# ========================================================================
def test_register_rejection():
    print("\n[Test 3] REGISTER flow — rejection")
    rej_uuid = f"rej-test-{int(time.time())}"
    _, pub_hex = generate_keypair()

    try:
        c = TCPClient()

        c.send("REGISTER")
        c.recv()  # REGISTER_CHALLENGE
        c.send(rej_uuid)
        c.recv()  # SEND_DEVICE_TYPE
        c.send("example_robot")
        c.recv()  # SEND_PUBLIC_KEY
        c.send(pub_hex)
        resp = c.recv()
        if resp != "REGISTER_PENDING":
            fail_test("Rejection: REGISTER_PENDING", f"got: {resp}")
            c.close()
            return
        pass_test("Robot pending, sending rejection")

        # Reject via HTTP
        status, _ = http_request("POST", "/register", {"uuid": rej_uuid, "accept": False})
        if status == 200:
            pass_test("Rejection sent via HTTP")
        else:
            fail_test("Rejection HTTP", f"status={status}")

        resp = c.recv()
        if resp == "REGISTER_REJECTED":
            pass_test("REGISTER_REJECTED received")
        else:
            fail_test("REGISTER_REJECTED", f"got: {resp}")

        c.close()

    except Exception as e:
        fail_test("Rejection flow", str(e))


# ========================================================================
# Test 4: HTTP provisioning
# ========================================================================
def test_http_provisioning():
    print("\n[Test 4] HTTP provisioning — add robot via API")
    prov_uuid = f"prov-test-{int(time.time())}"
    _, pub_hex = generate_keypair()

    # Provision a robot
    status, body = http_request("POST", "/provision", {
        "uuid": prov_uuid,
        "public_key": pub_hex,
        "device_type": "example_robot",
    })
    if status == 201:
        pass_test(f"Robot {prov_uuid} provisioned via HTTP")
    else:
        fail_test("Provision", f"status={status}, body={body}")
        return

    # Get robot record
    status, body = http_request("GET", f"/provision/{prov_uuid}")
    if status == 200 and body.get("UUID") == prov_uuid:
        pass_test("Robot record retrieved from PostgreSQL")
    else:
        fail_test("Get record", f"status={status}, body={body}")

    # List all robots
    status, body = http_request("GET", "/provision")
    if status == 200 and isinstance(body, list):
        found = any(r.get("UUID") == prov_uuid for r in body)
        if found:
            pass_test("Robot found in full registry list")
        else:
            fail_test("Registry list", f"robot {prov_uuid} not in list")
    else:
        fail_test("Registry list", f"status={status}")


# ========================================================================
# Test 5: Invalid commands
# ========================================================================
def test_invalid_commands():
    print("\n[Test 5] Invalid TCP commands")
    try:
        c = TCPClient()
        c.send("FOOBAR")
        resp = c.recv()
        if resp.startswith("ERROR"):
            pass_test(f"Unknown command rejected: {resp}")
        else:
            fail_test("Unknown command", f"expected ERROR, got: {resp}")
        c.close()
    except Exception as e:
        fail_test("Invalid commands", str(e))


# ========================================================================
# Main
# ========================================================================
if __name__ == "__main__":
    print("=" * 60)
    print("Robomesh E2E Test Suite")
    print("=" * 60)

    # Check server is reachable
    try:
        s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        s.settimeout(3)
        s.connect((HOST, TCP_PORT))
        s.close()
    except Exception:
        print(f"\nERROR: Cannot connect to TCP server at {HOST}:{TCP_PORT}")
        print("Make sure the server is running:")
        print("  docker compose -f docker-compose.dev.yml up --build")
        sys.exit(1)

    test_auth_flow()
    test_register_flow()
    test_register_rejection()
    test_http_provisioning()
    test_invalid_commands()

    print("\n" + "=" * 60)
    print(f"Results: {passed} passed, {failed} failed")
    print("=" * 60)
    sys.exit(1 if failed > 0 else 0)
