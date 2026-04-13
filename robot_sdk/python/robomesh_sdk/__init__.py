"""Robomesh Robot SDK - Python client library for communicating with Roboserver."""

from .client import RobotClient
from .udp_client import RobotUDPClient
from .keys import generate_ed25519_keypair, load_private_key, load_public_key_hex

# MQTT client requires paho-mqtt — import lazily to avoid hard dependency
def _get_mqtt_client():
    from .mqtt_client import RobotMQTTClient
    return RobotMQTTClient

__all__ = [
    "RobotClient",
    "RobotUDPClient",
    "generate_ed25519_keypair",
    "load_private_key",
    "load_public_key_hex",
]
