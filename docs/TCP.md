# TCP Protocol

Robots connect via TCP (default port 5002, env var `TCP_PORT`) and send either `AUTH` (pre-registered) or `REGISTER` (new robot). Both flows end in an authenticated session where the handler script is spawned.

Line-based protocol with a maximum message size of 64KB.

## AUTH Flow (Pre-Registered Robots)

For robots already stored in PostgreSQL via `POST /provision` or the PERSIST flow.

```text
    Robot                          Server (Go)
      |                               |
      |---- TCP Connect ------------->|
      |---- AUTH -------------------->|
      |<--- AUTH_CHALLENGE -----------|
      |                               |
      |---- UUID -------------------->|
      |                               | Look up PublicKey in PostgreSQL
      |                               | Check IsBlacklisted = false
      |<--- NONCE {random_hex} -------|
      |                               |
      |  Sign(nonce, PrivateKey)      |
      |---- {signature_hex} --------->|
      |                               | Verify(signature, PublicKey, nonce)
      |                               | Issue JWT + store in Redis
      |<--- AUTH_OK {jwt} ------------|
      |                               |
      |  (Authenticated session)      | Spawn handler process
```

**Supported signature algorithms:** Ed25519, ECDSA (PEM and raw hex formats).

**Error responses:** `ERROR NO_DATABASE`, `ERROR UNKNOWN_ROBOT`, `ERROR BLACKLISTED`, `ERROR INVALID_SIGNATURE`

## REGISTER Flow (New Robots)

For robots not yet in PostgreSQL. Stored ephemerally in Redis pending user approval. Duplicate UUIDs rejected (checked against Redis active, Redis pending, and PostgreSQL).

Device types are validated against `[a-zA-Z0-9_-]{1,64}` to prevent path traversal.

```text
    Robot                          Server (Go)            User (Frontend/Terminal)
      |                               |                          |
      |---- TCP Connect ------------->|                          |
      |---- REGISTER ---------------->|                          |
      |<--- REGISTER_CHALLENGE -------|                          |
      |                               |                          |
      |---- UUID -------------------->|                          |
      |<--- SEND_DEVICE_TYPE ---------|                          |
      |---- {device_type} ----------->|                          |
      |<--- SEND_PUBLIC_KEY ----------|                          |
      |---- {public_key_hex} -------->|                          |
      |                               | Store in Redis (pending) |
      |<--- REGISTER_PENDING ---------|                          |
      |                               |                          |
      |  (Robot blocks, waiting...)   |   GET /register/pending  |
      |                               |<-------------------------|
      |                               |  POST /register          |
      |                               |<-- {uuid, accept: true} -|
      |                               |                          |
      |                               | comms.Bus pub/sub notify |
      |<--- REGISTER_OK {jwt} --------|                          |
      |                               | Spawn handler process    |
```

**On rejection:** `REGISTER_REJECTED` is sent and the connection closes.

**Timeout:** Pending registrations expire after 5 minutes. Robot receives `ERROR REGISTRATION_TIMEOUT`.

## PERSIST Flow (Ephemeral to Permanent)

A registered (ephemeral) robot can promote itself to the PostgreSQL registry during an active session.

```text
    Robot                          Server (Go)
      |                               |
      |  (In active session)          |
      |---- PERSIST ----------------->|
      |                               | Copy public key from Redis -> PostgreSQL
      |<--- PERSIST_OK ---------------|
```

If already persisted: `PERSIST_OK ALREADY_PERSISTED`. If no public key found in Redis: `ERROR NO_PUBLIC_KEY`.

## Session Mode

After AUTH or REGISTER succeeds, the connection enters session mode:

- All subsequent lines are forwarded to the handler script as `incoming` messages
- The `PERSIST` command is intercepted before reaching the handler (for REGISTER-originated sessions)
- **Handlers survive TCP disconnect** — when the TCP connection closes, the handler is notified with a `disconnect` message but continues running
- Handlers can be manually killed via `POST /handler/{uuid}/kill`
- Handlers can be manually started via `POST /handler/{uuid}/start` (even without a TCP connection)

## Error Format

All errors follow: `ERROR <CODE>`

Error codes are generic identifiers (e.g., `HANDLER_SPAWN_FAILED`, `HEARTBEAT_REJECTED`). Internal error details are logged server-side only and never sent to the client.
