# Bugs

Tracked bugs and issues in the Robomesh codebase. Fixed items are marked with ~~strikethrough~~.

---

## Backend (roboserver/)

### ~~BUG-001: Consumer group cancel function uses captured index — corrupts on removal~~
**File:** `comms/local.go`
**Severity:** Medium
**Status:** Fixed — cancel function now uses a stable auto-incrementing ID per entry instead of a slice index.

### ~~BUG-005: Login rate limiter never cleans up old entries~~
**File:** `http_server/auth.go`
**Severity:** Low
**Status:** Fixed — added a background goroutine that evicts stale IP entries every 10 minutes.

### ~~BUG-007: `handleConnection` returns after first HEARTBEAT without looping~~
**File:** `tcp_server/tcp_server.go`
**Severity:** Low
**Status:** Fixed — after the first HEARTBEAT, the connection enters `heartbeatLoop` to accept subsequent heartbeats on the same connection.

---

## Frontend (frontend_app/)

### ~~BUG-011: Robot detail log stream has no authentication~~
**File:** `http_server/handler.go`, `routes/(app)/robots/[device_id]/+page.svelte`
**Severity:** High
**Status:** Fixed — log stream endpoint moved to semi-public route with ticket-based auth (like main SSE). Frontend fetches a ticket via `POST /auth/ticket` before opening the EventSource.

---

## Configuration / Infrastructure

### ~~BUG-016: Default admin password `password1` is hardcoded and never enforced to change~~
**File:** `database/databases.go`
**Severity:** Medium (security)
**Status:** Fixed — admin password can now be set via `ADMIN_PASSWORD` env var. Falls back to `password1` with a warning if unset.
