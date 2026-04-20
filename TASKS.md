# Tasks

Unfinished features, TODOs, and implementation gaps in the Robomesh codebase.

---

## High Priority

### ~~TASK-023: Implement UDP server~~
**Files:** `udp_server/udp_server.go`
**Status:** Complete — UDP server implemented with JSON packet-based protocol. Supports two-step challenge-response auth, signed heartbeats, and JWT-authenticated messaging. Wired into `main.go`, configurable via `udp_port` in config.yaml (default 5001).

---

## Medium Priority

---

## Low Priority

### ~~TASK-022: Integration testing with robot~~
**Files:** `tests/integration/`
**Status:** Complete — Comprehensive pytest-based integration testing framework. Tests all protocols (TCP, UDP, MQTT, HTTP), auth flows (success/failure), heartbeat (including replay rejection), handler lifecycle (start/kill/restart), provisioning, cross-protocol interactions, and server health checks. Run with `./tests/integration/run.sh` or `cd tests/integration && pytest -v`.
