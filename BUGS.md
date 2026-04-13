# Bugs

Tracked bugs and issues in the Robomesh codebase. Fixed items are marked with ~~strikethrough~~.

---

## Backend (roboserver/)

### ~~BUG-001: Consumer group cancel function uses captured index — corrupts on removal~~
**File:** `comms/local.go`
**Severity:** Medium
**Status:** Fixed — cancel function now uses a stable auto-incrementing ID per entry instead of a slice index.
