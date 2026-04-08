# Bugs

Tracked bugs and issues in the Robomesh codebase. Fixed items are marked with ~~strikethrough~~.

---

## Backend (roboserver/)

### ~~BUG-001: Consumer group cancel function uses captured index — corrupts on removal~~
**File:** `comms/local.go`
**Severity:** Medium
**Status:** Fixed — cancel function now uses a stable auto-incrementing ID per entry instead of a slice index.

---

## Frontend (frontend_app/)

### ~~BUG-011: Robot detail log stream has no authentication~~
**File:** `http_server/handler.go`, `routes/(app)/robots/[device_id]/+page.svelte`
**Severity:** High
**Status:** Fixed — log stream endpoint moved to semi-public route with ticket-based auth (like main SSE). Frontend fetches a ticket via `POST /auth/ticket` before opening the EventSource.

---

## Configuration / Infrastructure
