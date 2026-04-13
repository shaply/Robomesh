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

### TASK-022: Integration testing with robot
**Files:** TBD
**Description:** There should be an integration testing framework that utilizes test robot and connects it and ensures that all flows (including failures) are properly managed. This is like a last minute test suite before deployment.
