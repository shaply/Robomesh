#!/usr/bin/env python3
"""
Test client that simulates an example_robot connecting to the Robomesh TCP server.

This script performs the full cryptographic handshake (Ed25519) and enters session
mode for interactive communication with the handler script.

Prerequisites:
  1. Generate a key pair:
       python3 test_client.py --generate-keys
  2. Register the robot via HTTP API:
       curl -X POST http://localhost:8080/provision \
         -H "Authorization: Bearer <token>" \
         -H "Content-Type: application/json" \
         -d '{"uuid":"example-001","public_key":"<hex_public_key>","device_type":"example_robot"}'
  3. Run the client:
       python3 test_client.py --host localhost --port 5000

Usage:
  python3 test_client.py --generate-keys         # Generate and save Ed25519 key pair
  python3 test_client.py --host HOST --port PORT  # Connect to server
"""

import argparse
import os
import socket
import sys

try:
    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
    from cryptography.hazmat.primitives import serialization
    HAS_CRYPTO = True
except ImportError:
    HAS_CRYPTO = False

KEY_DIR = os.path.dirname(os.path.abspath(__file__))
PRIVATE_KEY_FILE = os.path.join(KEY_DIR, "private_key.hex")
PUBLIC_KEY_FILE = os.path.join(KEY_DIR, "public_key.hex")
UUID_FILE = os.path.join(KEY_DIR, "uuid.txt")

DEFAULT_UUID = "example-001"


def generate_keys():
    if not HAS_CRYPTO:
        print("Error: 'cryptography' package required. Install with: pip install cryptography")
        sys.exit(1)

    private_key = Ed25519PrivateKey.generate()
    private_bytes = private_key.private_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PrivateFormat.Raw,
        encryption_algorithm=serialization.NoEncryption(),
    )
    public_bytes = private_key.public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw,
    )

    priv_hex = private_bytes.hex()
    pub_hex = public_bytes.hex()

    with open(PRIVATE_KEY_FILE, "w") as f:
        f.write(priv_hex)
    with open(PUBLIC_KEY_FILE, "w") as f:
        f.write(pub_hex)
    with open(UUID_FILE, "w") as f:
        f.write(DEFAULT_UUID)

    print(f"Keys generated and saved to {KEY_DIR}/")
    print(f"  Private key: {PRIVATE_KEY_FILE}")
    print(f"  Public key:  {PUBLIC_KEY_FILE}")
    print(f"  UUID:        {DEFAULT_UUID}")
    print()
    print("Register this robot with the server:")
    print(f'  curl -X POST http://localhost:8080/provision \\')
    print(f'    -H "Authorization: Bearer <token>" \\')
    print(f'    -H "Content-Type: application/json" \\')
    print(f'    -d \'{{"uuid":"{DEFAULT_UUID}","public_key":"{pub_hex}","device_type":"example_robot"}}\'')


def load_keys():
    if not os.path.exists(PRIVATE_KEY_FILE):
        print(f"Error: {PRIVATE_KEY_FILE} not found. Run with --generate-keys first.")
        sys.exit(1)

    with open(PRIVATE_KEY_FILE) as f:
        priv_hex = f.read().strip()

    uuid = DEFAULT_UUID
    if os.path.exists(UUID_FILE):
        with open(UUID_FILE) as f:
            uuid = f.read().strip()

    return uuid, priv_hex


def sign_nonce(private_key_hex: str, nonce_hex: str) -> str:
    if not HAS_CRYPTO:
        print("Error: 'cryptography' package required. Install with: pip install cryptography")
        sys.exit(1)

    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
    priv_bytes = bytes.fromhex(private_key_hex)
    key = Ed25519PrivateKey.from_private_bytes(priv_bytes)
    nonce_bytes = bytes.fromhex(nonce_hex)
    signature = key.sign(nonce_bytes)
    return signature.hex()


def connect(host: str, port: int):
    uuid, priv_hex = load_keys()

    print(f"Connecting to {host}:{port} as {uuid}...")
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((host, port))
    sock.settimeout(30)

    def recv_line():
        data = b""
        while not data.endswith(b"\n"):
            chunk = sock.recv(1)
            if not chunk:
                raise ConnectionError("Connection closed")
            data += chunk
        return data.decode().strip()

    def send_line(msg):
        sock.sendall((msg + "\n").encode())

    # Step 1: Send AUTH
    send_line("AUTH")

    # Step 2: Expect AUTH_CHALLENGE
    resp = recv_line()
    print(f"<< {resp}")
    if resp != "AUTH_CHALLENGE":
        print(f"Unexpected response: {resp}")
        sock.close()
        return

    # Step 3: Send UUID
    send_line(uuid)
    print(f">> {uuid}")

    # Step 4: Expect NONCE
    resp = recv_line()
    print(f"<< {resp}")
    if not resp.startswith("NONCE "):
        print(f"Unexpected response: {resp}")
        sock.close()
        return
    nonce = resp.split(" ", 1)[1]

    # Step 5: Sign and send signature
    signature = sign_nonce(priv_hex, nonce)
    send_line(signature)
    print(f">> {signature[:32]}...")

    # Step 6: Expect AUTH_OK
    resp = recv_line()
    print(f"<< {resp}")
    if not resp.startswith("AUTH_OK"):
        print(f"Authentication failed: {resp}")
        sock.close()
        return

    jwt = resp.split(" ", 1)[1] if " " in resp else ""
    print(f"\nAuthenticated! JWT: {jwt[:40]}...")
    print("Entering session mode. Type messages to send, Ctrl+C to quit.\n")

    # Session mode: interactive send/receive
    import threading

    def reader():
        try:
            while True:
                line = recv_line()
                print(f"<< {line}")
        except Exception:
            print("\nConnection closed by server.")

    t = threading.Thread(target=reader, daemon=True)
    t.start()

    try:
        while True:
            msg = input(">> ")
            if msg:
                send_line(msg)
    except (KeyboardInterrupt, EOFError):
        print("\nDisconnecting...")
    finally:
        sock.close()


def main():
    parser = argparse.ArgumentParser(description="Example robot test client")
    parser.add_argument("--generate-keys", action="store_true", help="Generate Ed25519 key pair")
    parser.add_argument("--host", default="localhost", help="Server host (default: localhost)")
    parser.add_argument("--port", type=int, default=5000, help="Server TCP port (default: 5000)")
    args = parser.parse_args()

    if args.generate_keys:
        generate_keys()
    else:
        connect(args.host, args.port)


if __name__ == "__main__":
    main()
