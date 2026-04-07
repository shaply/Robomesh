# Tasks

Unfinished features, TODOs, and implementation gaps in the Robomesh codebase.

---

## High Priority

### TASK-001: Implement MQTT server
**Files:** `mqtt_server/mqtt_server.go`, `mqtt_server/mqtt_server_interface.go`
**Description:** The MQTT server is a stub — `Start()` just logs and waits on context. The interface and config are defined but no actual MQTT broker logic exists. This is needed for IoT devices that speak MQTT natively.

---

## Medium Priority

### TASK-004: Implement search and filter on robots page
**File:** `routes/(app)/robots/+page.svelte:58-59`
**Description:** The Search and Filter+ buttons exist in the UI but are non-functional. `SearchBar` component renders but isn't wired up to filter the robot list.

### TASK-005: Implement settings page
**File:** `routes/(app)/+layout.svelte:21`
**Description:** The sidebar links to `/settings` but no settings route/page exists. Clicking it will 404.

### TASK-007: Add WebSocket command handling
**File:** `http_server/http_websocket/websocket.go`
**Description:** The WebSocket manager handles connections and event subscriptions, but there's no bidirectional command protocol. Currently it's read-only (events from server to client). The infrastructure is there for sending commands from the frontend but no command routing exists.

### TASK-008: Add database method extensibility for handlers
**File:** `handler_engine/process.go:382-401`
**Description:** `handleDatabaseRequest` only supports `get_robot`. Handlers need more database operations — list robots, query by type, store custom data, etc. The switch statement needs more cases or a plugin-style query system.

---

## Low Priority

### TASK-012: Add handler type listing endpoint
**Description:** `handler_engine.ListHandlerTypes()` exists but isn't exposed via HTTP. The frontend could use this to show available handler types in the provision form's device type dropdown.

### TASK-016: Add tests for http_events, mqtt_server, shared, terminal packages
**Description:** Several packages have no test files:
- `http_server/http_events/` — SSE client/session/manager logic untested
- `mqtt_server/` — no tests (stub)
- `shared/` — config loading untested
- `terminal/` — interactive CLI untested

### ~~TASK-018: Add `ADMIN_PASSWORD` env var override for default admin~~
**Status:** Done — see BUG-016.

### TASK-019: Add robot message endpoint
**File:** `routes/(app)/robots/[device_id]/+page.svelte:275`
**Description:** The handler component's `sendToHandler` prop calls `POST /robot/{uuid}/message` but this endpoint doesn't exist in the HTTP server. Handler communication currently only works over TCP.

### TASK-020: Refactor SSE event double-encoding
**Files:** `http_server/http_events/eventClient.go:127-162`, `frontend_app/src/lib/backend/event_source/EventSourceManager.ts:222-262`
**Description:** Events are JSON-encoded, then base64-encoded, then wrapped in another JSON struct, then base64-encoded again. The frontend decodes base64 → JSON → base64 → data. This double encoding adds unnecessary complexity and CPU overhead. A single JSON encode with proper SSE formatting would suffice.

### TASK-021: Ensure SDKs are updated
**Files:** `robot_sdk/c`, `robot_sdk/python`
**Description:** The SDKs should be updated to match the current state of roboserver.

### TASK-022: Integration testing with robot
**Files:** TBD
**Description:** There should be an integration testing framework that utilizes test robot and connects it and ensures that all flows (including failures) are properly managed. This is like a last minute test suite before deployment.


### TASK-023: Add password change / user management API
**Description:** Currently only the seeded admin user exists. There's no endpoint to change passwords, create users, or manage accounts. The Redis `User` schema supports it but the HTTP API doesn't expose it.