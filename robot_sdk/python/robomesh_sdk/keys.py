"""Ed25519 key generation and management."""

from cryptography.hazmat.primitives.asymmetric.ed25519 import (
    Ed25519PrivateKey,
    Ed25519PublicKey,
)
from cryptography.hazmat.primitives import serialization


def generate_ed25519_keypair() -> tuple[Ed25519PrivateKey, str, str]:
    """Generate an Ed25519 keypair.

    Returns:
        (private_key_object, private_key_hex, public_key_hex)
    """
    private_key = Ed25519PrivateKey.generate()
    priv_bytes = private_key.private_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PrivateFormat.Raw,
        encryption_algorithm=serialization.NoEncryption(),
    )
    pub_bytes = private_key.public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw,
    )
    return private_key, priv_bytes.hex(), pub_bytes.hex()


def load_private_key(hex_key: str) -> Ed25519PrivateKey:
    """Load an Ed25519 private key from a 64-char hex string."""
    return Ed25519PrivateKey.from_private_bytes(bytes.fromhex(hex_key))


def load_public_key_hex(private_key: Ed25519PrivateKey) -> str:
    """Get the hex-encoded public key from a private key."""
    return private_key.public_key().public_bytes(
        encoding=serialization.Encoding.Raw,
        format=serialization.PublicFormat.Raw,
    ).hex()


def sign_message(private_key: Ed25519PrivateKey, message: bytes) -> str:
    """Sign a message and return the hex-encoded signature."""
    return private_key.sign(message).hex()
