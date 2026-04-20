#!/usr/bin/env bash
# Robomesh Integration Test Runner
# =================================
# Deployment readiness test suite. Run this before pushing to production.
#
# Prerequisites:
#   docker compose -f docker-compose.dev.yml up --build
#
# Usage:
#   ./tests/integration/run.sh              # Run all tests
#   ./tests/integration/run.sh -k tcp       # Run only TCP tests
#   ./tests/integration/run.sh -k health    # Run only health checks
#   ./tests/integration/run.sh --no-mqtt    # Skip MQTT tests (no paho-mqtt)
#
# Environment overrides:
#   ROBOMESH_HOST=192.168.1.50 ./tests/integration/run.sh
#   ROBOMESH_TCP_PORT=5002 ROBOMESH_HTTP_PORT=8080 ./tests/integration/run.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Load defaults if available
if [ -f "$PROJECT_ROOT/defaults.env" ]; then
    # Export defaults (won't override existing env vars)
    set -a
    source "$PROJECT_ROOT/defaults.env"
    set +a
    # Map to ROBOMESH_ prefixed vars (only if not already set)
    export ROBOMESH_HOST="${ROBOMESH_HOST:-localhost}"
    export ROBOMESH_HTTP_PORT="${ROBOMESH_HTTP_PORT:-${HTTP_PORT:-8080}}"
    export ROBOMESH_TCP_PORT="${ROBOMESH_TCP_PORT:-${TCP_PORT:-5002}}"
    export ROBOMESH_UDP_PORT="${ROBOMESH_UDP_PORT:-${UDP_PORT:-5001}}"
    export ROBOMESH_MQTT_PORT="${ROBOMESH_MQTT_PORT:-${MQTT_PORT:-1883}}"
fi

echo "=============================================="
echo "  Robomesh Integration Test Suite"
echo "=============================================="
echo "  Host:      ${ROBOMESH_HOST:-localhost}"
echo "  HTTP:      ${ROBOMESH_HTTP_PORT:-8080}"
echo "  TCP:       ${ROBOMESH_TCP_PORT:-5002}"
echo "  UDP:       ${ROBOMESH_UDP_PORT:-5001}"
echo "  MQTT:      ${ROBOMESH_MQTT_PORT:-1883}"
echo "=============================================="
echo ""

# Install dependencies if needed
if ! python3 -c "import pytest" 2>/dev/null; then
    echo "Installing test dependencies..."
    pip install -r "$SCRIPT_DIR/requirements.txt"
fi

# Ensure the SDK is importable
if ! python3 -c "import robomesh_sdk" 2>/dev/null; then
    echo "Installing Robomesh Python SDK..."
    pip install -e "$PROJECT_ROOT/robot_sdk/python[all]"
fi

# Run tests
cd "$SCRIPT_DIR"
exec python3 -m pytest "$@" -v --tb=short
