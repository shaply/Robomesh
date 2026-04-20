# Robomesh — System Design

A prep-oriented design document. Sections are ordered so you can tell the story top-down in an interview, then drill into any subsystem when follow-ups come.

---

## 1. Elevator Pitch

Robomesh is a centralized control plane for fleets of heterogeneous autonomous devices (home-assistants, rovers, trading bots, IoT sensors). A single Go binary (`roboserver`) authenticates robots cryptographically, spawns per-robot "handler" processes that encapsulate device-specific logic, and exposes a Svelte frontend for human operators.

The system is designed for **low-power Mini PC deployment** — which drove four core principles:

| Principle | What it means in practice |
|---|---|
| **Stateless** | Server restart loses no durable state. All permanent data in PostgreSQL, all session/ephemeral state in Redis with TTLs. |
| **Decoupled** | Robot ↔ handler ↔ frontend are wired through an event bus. Components don't hold references to each other. |
| **Cryptographically secure** | Every robot owns an Ed25519 keypair. Auth is challenge-response; messages are signed. No shared secrets on the wire. |
| **Zero-idle** | Handlers are OS processes that only exist while their robot is connected (or while work is pending). No always-on workers. |

---

## 2. Component Topology

```
                         ┌────────────────┐
                         │   Frontend     │   SvelteKit (auth'd humans)
                         │   (Svelte 5)   │
                         └───┬────────┬───┘
                    HTTP/SSE │        │ WebSocket
                         ┌───▼────────▼───────────────────────────┐
                         │                 roboserver             │
                         │  ┌──────────────────────────────────┐  │
                         │  │   5 concurrent server goroutines │  │
                         │  │  Terminal │ HTTP │ TCP │ UDP │MQTT│  │
                         │  └─────────────────┬────────────────┘  │
                         │           ┌────────▼────────┐          │
                         │           │   Comm Bus      │          │
                         │           │ (LocalBus)      │          │
                         │           │ ┌──────────────┐│          │
                         │           │ │ Event Bus    ││          │
                         │           │ │ Redis PubSub ││          │
                         │           │ └──────────────┘│          │
                         │           └────────┬────────┘          │
                         │       ┌────────────┴───────────┐       │
                         │       │    Handler Manager     │       │
                         │       │  (UUID → *Process)     │       │
                         │       └──────┬──────┬──────┬───┘       │
                         └──────────────┼──────┼──────┼───────────┘
                                        │      │      │
                                stdin   ▼      ▼      ▼  stdout (JSON-RPC)
                                   ┌────────┬────────┬────────┐
                                   │Handler1│Handler2│Handler3│   bash scripts
                                   └────────┴────────┴────────┘   (any language)
                                        ▲        ▲        ▲
                                 TCP/MQTT/UDP/HTTP  (signed)
                                   ┌────────┬────────┬────────┐
                                   │Robot 1 │Robot 2 │Robot 3 │
                                   └────────┴────────┴────────┘

                ┌──────────────┐   ┌──────────────┐
                │  PostgreSQL  │   │    Redis     │
                │ (permanent)  │   │ (ephemeral)  │
                └──────────────┘   └──────────────┘
```

### Five server goroutines (all wired to the same Bus + DBManager)

| Server | Role | Why it exists |
|---|---|---|
| **HTTP** | Operator API, SSE stream, WebSocket, plugin assets | Human-facing surface |
| **TCP** | Line-based robot protocol (`AUTH`/`REGISTER`/`HEARTBEAT`) | Default robot transport — persistent, ordered, simple |
| **MQTT** | Pub/sub broker (Mochi-mqtt embedded) on `robomesh/*` topics | Industry-standard for IoT fleets |
| **UDP** | JSON-packet protocol | Battery-sensitive / lossy-network devices |
| **Terminal** | Interactive CLI | Ops debugging without a browser |

All five are launched as goroutines from `main.go`, share the same `context.Context`, and coordinate shutdown via `sync.WaitGroup` with a 60s timeout.

---

## 3. Startup Sequence

`main.go` orders initialization so each layer sees its dependencies ready:

```
 ① Load config (config.yaml + .env overrides)
 ② Initialize event bus (in-process)
 ③ Initialize DB manager → PostgreSQL + Redis
      └─ seeds admin user if missing
 ④ Wrap event bus + Redis pub/sub in LocalBus (Comm Bus)
 ⑤ Spawn 5 server goroutines concurrently
 ⑥ Block on SIGINT/SIGTERM or ctx.Done()
 ⑦ Graceful shutdown: cancel ctx → HandlerManager.StopAll
      → wg.Wait (60s) → hard exit
```

Each server goroutine owns its listener. On shutdown they each close their listener, which unblocks `Accept()` and lets the goroutine return.

---

## 4. Data Layer

Two stores with very different roles — **this is the split you want to nail in an interview**.

### PostgreSQL — Source of Truth (permanent)

Single table `robots`:

| Column | Purpose |
|---|---|
| `uuid` (PK) | Robot identity |
| `public_key` | Ed25519 pubkey (PEM or raw hex) |
| `device_type` | Routes to `handlers/{device_type}/start_handler.sh` |
| `is_blacklisted` | Soft-delete / revocation |
| `created_at` | Audit |

Migrations in `db/migrations/` use dbmate format.

### Redis — Ephemeral State (all TTL'd)

| Key pattern | TTL | Purpose |
|---|---|---|
| `robot:{uuid}:active` | config (~5min) | Active session (IP, JWT, PID, device_type) |
| `robot:{uuid}:heartbeat` | client-specified | Heartbeat state (LastSeq, LastSeen) — *independent of handler* |
| `robot:{uuid}:pending` | 5min | Awaiting operator approval |
| `robot:{uuid}:pubkey` | session TTL | Pubkey staged during REGISTER, copied to PG on PERSIST |
| `mqtt:nonce:{uuid}` | 30s | `nonce\|pubkey\|device_type` — cached to avoid a 2nd PG lookup in step-2 of MQTT auth |
| `udp:nonce:{uuid}` | 30s | Same pattern as MQTT |
| `handler:{uuid}:data:{k}` | no TTL | Handler-scoped custom storage |
| `user:{username}` | permanent | bcrypt-hashed password |
| `session:{token}` | config | User JWT sessions (server-side invalidation on logout) |
| `ticket:{id}` | 30s | Single-use SSE auth tickets |

### Why split this way?
- **Restart resilience**: blow away Redis, server recovers. Robots re-auth, sessions rebuild. PG never loses identity data.
- **Auto-expiry**: dead robot sessions disappear on their own — no reaper cron.
- **Hot-path reads**: session/heartbeat lookups don't hit PG.
- **Pub/sub**: Redis is doubled as the message substrate for `PublishRegistrationResponse` (operator approval flow blocks in a goroutine waiting on a Redis channel).

---

## 5. Authentication Model

### Robot identity
Ed25519 keypair generated by the SDK. The **public key** lives in PostgreSQL (provisioned out-of-band or via REGISTER). The **private key** never leaves the robot. There are no shared secrets.

### TCP challenge-response handshake (`auth/handshake.go`)

```
Robot                                     Roboserver
  │                                              │
  │ ─────────── "AUTH" ────────────────────────▶ │
  │ ◀─────── "AUTH_CHALLENGE" ─────────────────── │
  │                                              │
  │ ─────────── UUID ──────────────────────────▶ │  ┐
  │                                              │  │ PG lookup, blacklist check
  │ ◀─────────── "NONCE <hex>" ─────────────────  │  ┘
  │                                              │
  │ sign(nonce, privKey)                         │
  │ ─────────── <sig_hex> ─────────────────────▶ │  ┐
  │                                              │  │ Verify against stored pubkey
  │ ◀─────── "AUTH_OK <jwt>" ──────────────────── │  ┘
  │                                              │
  │           ─── enters session mode ───        │
```

The JWT (`HS256`, 30min default) encodes `{uuid, device_type, ip, session_id}`. Server uses it for subsequent HTTP/WS calls from the same robot.

### User auth (operators)
- `POST /auth/login` → bcrypt verify → HS256 JWT + `session:{token}` key in Redis.
- Middleware validates **both** JWT signature AND session-exists-in-Redis → logout can kill a session server-side (JWTs alone can't be revoked).
- Password constraint: 8–72 chars (bcrypt's hard limit).

### Two unusual details to volunteer
1. **MQTT/UDP cache the robot record in Redis alongside the nonce** (`nonce|pubkey|device_type` pipe-delimited, 30s TTL). Step 1 of the two-step auth reads PG; step 2 reads from Redis only. One PG hit per auth instead of two.
2. **Registration is operator-gated.** An un-provisioned robot calling `REGISTER` lands in `robot:{uuid}:pending`, publishes `robot.registering` on the bus, and blocks on a Redis pub/sub channel `robot:{uuid}:reg_response`. The frontend issues accept/reject, which publishes to that channel and unblocks the goroutine.

---

## 6. Handler Engine — The Core Idea

Each robot type has a directory `handlers/{device_type}/` with a `start_handler.sh` entry point. When a robot authenticates, the engine:

1. Resolves the script path from the registry
2. `exec.CommandContext("/bin/bash", scriptPath)` — captures stdin/stdout/stderr
3. Puts the handler in its own **process group** (`Setpgid: true`) so we can SIGKILL the entire tree on cleanup
4. Passes robot metadata via env vars (`ROBOT_UUID`, `ROBOT_DEVICE_TYPE`, `ROBOT_IP`, `ROBOT_SESSION_ID`)
5. Registers the `*HandlerProcess` in a global, mutex-guarded map keyed by UUID

### JSON-RPC over stdin/stdout

The handler script and the Go sidecar speak newline-delimited JSON:

**Go → handler (stdin):**
```json
{"type":"connect",   "uuid":"…","device_type":"…","ip":"…","session_id":"…"}
{"type":"incoming",  "uuid":"…","payload":"…"}
{"type":"disconnect","uuid":"…","reason":"tcp_closed"}
{"type":"event",     "event_type":"price.tick","data":{…}}
{"type":"heartbeat", "event_type":"robot.abc.heartbeat","data":{…}}
```

**Handler → Go (stdout) — JSON-RPC envelope:**
```json
{"target":"robot",        "id":"1","data":{…}}                        // send to robot
{"target":"database",     "id":"2","method":"get_robot","data":"uuid"}
{"target":"event_bus",    "method":"price.alert","data":{…}}
{"target":"config",       "method":"forward_heartbeats","data":true}
{"target":"config",       "method":"subscribe","data":"price.tick"}
{"target":"connect_robot","data":{"port":8888,"protocol":"tcp"}}      // reverse connect
```

Responses come back as `{"target":"response","id":"2","data":…,"error":""}`. Correlation via `id`.

### Concurrency details worth knowing (and being asked about)

- **Dedicated stdin writer goroutine** drains a buffered channel (`writeCh`, cap 256). This was a specific bug fix (BUG-013): if a handler script stalls on reading stdin, a naive mutex-locked write would freeze any goroutine trying to send to it. Now senders enqueue to the channel and return immediately; the writer goroutine blocks alone.
- **Drop on overflow**, don't block. If `writeCh` is full, the message is dropped and logged. This keeps the event bus and TCP server goroutines from ever being held hostage by one bad handler.
- **Stop() ordering is subtle.** The disconnect message must be enqueued *before* `close(writeCh)`, both under the same mutex with `closed=true`, or the disconnect gets silently dropped (a prior bug). After `close`, the writer drains the channel and exits.
- **Subprocess kill is a group-kill.** `syscall.Kill(-PID, SIGKILL)` on timeout — negative PID targets the whole process group. Without this, a bash wrapper whose child is a Python interpreter leaves the child orphaned.

### Handlers survive TCP disconnects

This is non-obvious. If the robot's TCP connection drops, the handler process stays alive. Its `RobotSend` callback gets nil'd and it receives a `disconnect` message. It may keep running background tasks, serving reverse connections, or waiting for the robot to reconnect (via `Reattach`).

This supports two real patterns:
1. Flaky-network robots that reconnect frequently — no cold-start cost per reconnect.
2. Handlers that do work independent of the robot being online (scheduled jobs, processing queued data).

Operators can kill a handler explicitly via `POST /handler/{uuid}/kill`.

### TryStartSpawning — a small but important lock

Multiple servers (TCP + MQTT + UDP) can all authenticate the same robot concurrently. We never want two handlers for the same UUID. The manager exposes a two-step atomic guard:

```go
if m.TryStartSpawning(uuid) { // sets spawning[uuid]=true if safe
    defer m.FinishSpawning(uuid)
    // ... SpawnHandlerProcess ...
} else if existing, ok := m.Get(uuid); ok {
    existing.Reattach(...)
} else {
    // another connection is mid-spawn — poll the map
}
```

The polling fallback has a 50ms tick and respects context cancellation.

---

## 7. Comm Bus — Abstraction Layer

`comms.Bus` is the interface that services call. `LocalBus` is the current impl; future `KafkaBus` or `GRPCBus` would let you horizontally scale by dropping in a different implementation.

```go
type Bus interface {
    PublishEvent(eventType string, data any) error
    SubscribeEvent(eventType string, h EventHandler) (cancel, error)
    PublishToGroup(group, eventType string, data any) error  // competing consumers
    SubscribeAsGroup(group, eventType string, h EventHandler) (cancel, error)
    PublishRegistrationResponse(ctx, uuid, accepted bool) error
    WaitForRegistrationResponse(ctx, uuid) (bool, error)
}
```

### Internal event bus (`shared/event_bus/`)

- Typed pub/sub, SafeMap-backed (`map[eventType] → set[subscriber]`, `map[subscriber] → map[eventType]handler`).
- **Non-blocking backpressure**: there's a global `inFlight` atomic counter. If it exceeds `EVENT_BUS_BUFFER_SIZE` (1000), new publishes are *dropped* rather than queued. Rationale: the publisher is almost always a network goroutine we must not stall.
- Each handler fires in its own goroutine with a `recover()` — a panicking subscriber doesn't take down its neighbors.

### Consumer groups (competing consumers)

Beyond standard broadcast pub/sub, `LocalBus` supports a "group" pattern: N subscribers register under the same `group::eventType` key, and each publish is delivered to exactly one via round-robin (atomic counter mod N). That's the seed of a point-to-point semantic for future microservice decomposition — work doesn't duplicate across replicas.

---

## 8. Key Flow Diagrams

### Flow A — Robot first-time registration (operator-gated)

```
Robot                   TCP Server              Redis           EventBus          Frontend (SSE)
  │ "REGISTER"              │                      │                │                    │
  ├────────────────────────▶│                      │                │                    │
  │ UUID / devtype / pubkey │                      │                │                    │
  ├────────────────────────▶│ SET pending:{uuid} ─▶│                │                    │
  │                         │───────── publish "robot.registering" ▶│                    │
  │                         │                      │                │ SSE event ────────▶│ "Accept? y/n"
  │ "REGISTER_PENDING"      │                      │                │                    │
  │◀────────────────────────┤                      │                │                    │
  │                         │ SUB reg_response     │                │                    │
  │                         │◀─── blocks on ────── │                │                    │
  │                         │                      │                │        POST /register/{uuid}/accept
  │                         │                      │◀──────────── PUBLISH ─────────── ───│
  │                         │◀── "accept" ─────────│                │                    │
  │                         │ DEL pending:{uuid}   │                │                    │
  │                         │ SET active:{uuid}    │                │                    │
  │ "REGISTER_OK <jwt>"     │                      │                │                    │
  │◀────────────────────────┤                      │                │                    │
  │                         │ spawns handler ──────┼────────────────┼────────────────────┤
  │         enters session mode (incoming → handler stdin)          │                    │
```

Later the robot can send `PERSIST` → server copies pubkey from `robot:{uuid}:pubkey` → `robots` table in PG.

### Flow B — Authenticated session & lifecycle

```
Robot       TCP/MQTT/UDP    HandlerManager       HandlerProcess    Bus    Frontend
  │ AUTH ───▶ challenge/sign     │                      │            │        │
  │◀── JWT ──┤                   │                      │            │        │
  │          │ TryStartSpawning  │                      │            │        │
  │          │──────────────────▶│                      │            │        │
  │          │                   │ exec.Command         │            │        │
  │          │                   │─────────────────────▶│ (bash)     │        │
  │          │                   │ register in map      │            │        │
  │          │                   │                      │ "connect" ─┼──────▶ │
  │          │ forward robot msg │                      │            │        │
  │ payload ─┼──────────────────▶│── SendIncoming ─────▶│ stdin      │        │
  │          │                   │                      │ stdout ────┤        │
  │          │                   │  ◀── JSON-RPC ────── │ route to:  │        │
  │          │                   │   {target:"robot"}   │  robot     │        │
  │◀── data ─┼───────────────────┼──────────────────────┤            │        │
  │                                                        publish   │        │
  │                                                     {target:"event_bus"}──▶│ SSE
  │                                                                            │
  │  (TCP closes)                 SendDisconnect → handler stays alive         │
  │                               RobotSend = nil                              │
  │  (reconnect) AUTH ──────────▶ Reattach → handler gets new "connect"        │
```

### Flow C — Reverse connect (roboserver dials the robot)

Use case: handler wants to push large/streaming data to the robot without inverting the protocol. E.g., a robot listens on port 8888 and the handler pushes a firmware image.

```
Handler script            Go sidecar          Robot
  │ {target:"connect_robot",    │                   │
  │  data:{port:8888}} ────────▶│                   │
  │                             │ resolve IP from   │
  │                             │ heartbeat state   │
  │                             │────── TCP dial ──▶│
  │                             │◀── "CONNECT_OK" ──│
  │                             │── NONCE ─────────▶│
  │                             │◀── signature ─────│  (same crypto as auth)
  │                             │ verify vs PG      │
  │                             │── "AUTH_OK" ─────▶│
  │ ◀── resp: {data:"connected"}│                   │
  │ stdin (incoming messages)   │◀── bridge ───────▶│
  │ stdout (robot messages)     │                   │
```

UDP variant skips the handshake and just bridges — trust boundary is the handler's request.

### Flow D — Frontend real-time updates (SSE)

```
Browser             HTTP server          EventsManager          EventBus
  │ POST /auth/login ──▶ bcrypt + JWT + session:{token}         │
  │◀── JWT ─────────────┤                                       │
  │ POST /events/subscribe {topics:[...]} ──▶ register in mgr   │
  │                                        ─── SubscribeEvent ─▶│
  │                                                             │
  │ GET /events?ticket=… ──▶ ConsumeTicket(Redis)  (GetDel atomic)
  │◀── text/event-stream ─────┤                                 │
  │                           │◀──── event published ───────────│
  │◀── data: {id,type,data}   │                                 │
```

Single-use tickets exist because `EventSource` can't send `Authorization` headers. Ticket is issued by a JWT-authed POST, consumed by the GET, deleted atomically.

---

## 9. Security Model

| Threat | Mitigation |
|---|---|
| Robot impersonation | Ed25519 signature over random per-session nonce. Private key never leaves device. |
| Replay | Nonces are one-time (30s TTL, GetDel consumes atomically). Heartbeats carry monotonic `seq` — server rejects `seq <= last_seq`. |
| Credential theft | No shared secrets. User passwords bcrypt-hashed. JWTs use `HS256` with `JWT_SECRET` from `.env`. |
| Post-logout JWT reuse | Middleware requires both valid JWT AND live `session:{token}` key in Redis. Logout deletes the Redis key. |
| Oversized payloads | 64KB cap on TCP lines; `MaxBytesReader` on HTTP bodies. |
| Cross-robot MQTT eavesdropping | Custom ACL hook restricts `robomesh/*/response` and `robomesh/to_robot/*` to matching UUID. |
| Orphaned child processes (resource leak) | Handlers spawned with `Setpgid: true`; `SIGKILL` targets the whole process group on cleanup timeout. |
| Event-bus saturation | Non-blocking drop when `inFlight >= 1000`. Per-handler panics recovered. |
| PII in logs | `RedactIP` helper for sensitive log lines. |
| CORS | Explicit whitelist from config — no `*`. |

---

## 10. Concurrency Model — Cheat Sheet

Where Go's tooling shows up:

- **`context.Context`** is the cancellation backbone. Root ctx is created in `main`; every server, every handler spawn, every DB call receives a derived ctx. Graceful shutdown is just `cancel()` + `WaitGroup`.
- **`sync.Mutex` / `sync.RWMutex`** on anything mutable across goroutines: HandlerManager map, per-handler `RobotSend` pointer, SafeMap internals.
- **Channels as work queues** — `HandlerProcess.writeCh` decouples sender mutex from pipe writes.
- **Atomic counters** — `inFlight` for backpressure, consumer-group round-robin.
- **Non-blocking selects with `default:`** to avoid priority inversions on shutdown-sensitive paths.
- **Goroutine-per-connection** on all network servers (Accept loops spawn a handler goroutine). No worker pool yet; Go's scheduler handles it.
- **Panic recovery** at process boundaries — one malformed MQTT payload or stderr line never takes down the broker or the handler.

---

## 11. Frontend

SvelteKit 2.16, Svelte 5 runes mode, TypeScript strict, Tailwind 4, Vite 6.

- Route groups: `(auth)/` (unauthenticated), `(app)/` (protected).
- JWT in `localStorage['auth-token']`; `fetchBackend()` wrapper auto-injects `Authorization: Bearer`.
- `backendBaseUrl()` derives protocol from `window.location` — HTTP and HTTPS both work without config.
- **Plugin system**: every robot type can ship its own UI. `handlers/{type}/frontend/` builds to `dist/{robot_card,robot_handler}.js`. Backend serves them at `/plugins/{type}/…`. Frontend `plugin-loader.ts` does dynamic `import()` and caches the module.
- **Registry pattern** maps device_type → Svelte component. Built-ins registered statically, plugins loaded on demand.
- **`EventSourceManager` singleton** holds a single SSE connection; subscribes/unsubscribes via POST API.
- **WebSocket `/ws`** for bidirectional ops: `subscribe`, `unsubscribe`, `send_to_robot`, `send_to_handler`.

---

## 12. Deployment Model

- Target: **low-power Mini PC** (cost constraint drove the zero-idle design).
- `docker-compose.yml` stands up `roboserver`, Postgres, Redis.
- `defaults.env` at project root centralizes all port defaults; Compose autoloads it. Non-Docker runs default from `config.yaml`.
- TLS is opt-in per server (`server.tls.enabled`). Same cert/key for TCP and HTTP.
- Config precedence: **env > config.yaml**. Secrets (DB passwords, JWT secret) live in `.env`, never committed.

---

## 13. Known Tradeoffs & Future Scale-Out Path

Be ready to volunteer these — Optiver will push on tradeoffs.

### Things we explicitly chose

| Choice | Why | Cost |
|---|---|---|
| Go monolith, not microservices | Single binary to deploy on a Mini PC; low ops overhead | Harder to horizontally scale one component in isolation |
| Handlers as OS processes (not goroutines) | Language-agnostic (Python, Node, Rust); fault-isolated (one handler panic doesn't kill server) | ~1–5MB RSS per handler; process spawn is ~tens of ms |
| PostgreSQL for durable, Redis for ephemeral | Each tool to its strength; avoids Redis-persistence config | Two dependencies to run |
| JSON-RPC on stdin/stdout | Zero setup, debuggable with `cat` | No flow-control signaling; reliant on OS pipe backpressure |
| Drop events on bus saturation | Network goroutines must never stall | Subscribers can miss events — need application-level reconciliation if that matters |
| Line-based TCP protocol | Trivial to implement in any SDK | 64KB message ceiling; no multiplexing (you'd need framing) |
| JWT + Redis session for users | Server-side revocation without giving up stateless JWTs | Middleware must hit Redis on every request |

### Scale-out path

If Optiver asks "how would you scale this to 10,000 robots / 10 servers?":

1. **Replace `LocalBus` with `KafkaBus` (or NATS).** The `Bus` interface is already the seam. Consumer groups are already modeled.
2. **Run `N` replicas of roboserver** behind a load balancer. Robots hash to a replica by UUID (sticky) or auth returns a redirect. MQTT brokers can cluster; TCP can be fronted by HAProxy.
3. **Move handlers off the Go box.** The `HandlerProcess` abstraction is thin — `Reattach`, `SendIncoming`, `Stop`, `SendToRobot`. Put each handler in a container (K8s pod per robot) and have the sidecar talk gRPC instead of pipes. Handler spawn cost goes up (container start ~100s of ms) but you get per-handler resource limits for free.
4. **PostgreSQL**: single primary is fine to ~100k writes/day. Read replicas for dashboards.
5. **Redis**: cluster mode; UUID-sharded keys already hash-compatible. Session/pubsub needs a single-primary broadcast channel, so either pin pubsub to one shard or move to NATS.
6. **Observability gap** (acknowledge it honestly): no metrics/tracing yet. Would add Prometheus counters for event-bus drops, handler spawns, auth failures, per-server QPS; OpenTelemetry traces across auth → spawn → message.

---

## 14. Interview Talking Points (likely follow-ups)

Rehearsed answers to the questions you'll probably get:

**Q: Why OS processes for handlers instead of goroutines or threads?**
Three reasons in order of importance: (1) **language-agnostic** — anyone writing a new robot type can use Python, Node, Rust, whatever; (2) **fault isolation** — a segfault or OOM in one handler doesn't touch `roboserver`; (3) **killability** — SIGKILL on a process group is a guarantee; cancelling a goroutine is a request that a well-behaved goroutine chooses to honor.

**Q: What happens if a handler script misbehaves and blocks reading stdin?**
The `writeCh` channel (256 msgs) absorbs bursts. When it fills, subsequent sends drop the message and log. The server's network goroutines never block. If the script is permanently stuck, the operator kills it via API (or Stop's 60s timer SIGKILLs the process group on shutdown).

**Q: How do you prevent two handlers spawning for the same robot?**
`HandlerManager.TryStartSpawning` is an atomic check-and-set under a mutex: it returns `true` only if (a) no handler exists and (b) no other caller is currently spawning. Losers either reattach to the existing handler or poll the map until the winner finishes. The `spawning` flag is always cleared in a `defer` so a panic during spawn doesn't wedge future attempts.

**Q: Why Redis pub/sub for registration approval instead of a channel or WebSocket?**
The server process blocked on registration approval is a goroutine inside *a specific replica*. The approval comes in via HTTP to *any replica*. Redis pub/sub is the cross-replica fan-out that makes this work even before we horizontally scale. Doing it with channels would lock approvals to the same process the robot is connected to.

**Q: Heartbeat sequence numbers — what problem do they solve?**
Replay protection. An attacker capturing a signed heartbeat packet (even though TLS prevents this in practice) cannot resend it — the server stores `last_seq` in Redis and rejects anything `<= last_seq`. Each heartbeat is cryptographically fresh.

**Q: Why a separate `Bus` interface on top of the event bus?**
Two reasons: (1) it abstracts cross-process communication (Redis pub/sub for registration) that the in-process event bus can't do; (2) it's the seam for swapping to Kafka/NATS when we need horizontal scale. Services depend on `comms.Bus`, not on the concrete implementation.

**Q: What's the single biggest architectural risk?**
The event bus dropping events under load. It's a deliberate design choice (the alternative is stalling the publisher, which means stalling a network goroutine, which means the whole server), but it means subscribers need to handle missed events — either by reconciling against Redis/PG state on reconnect or by using the `PublishToGroup` path for anything that must be delivered exactly once.

**Q: Why JSON-RPC instead of gRPC for handlers?**
Handler scripts are bash + whatever runtime. gRPC would mean every new handler brings in protobuf + a gRPC runtime in whatever language. Newline-delimited JSON on stdin/stdout works from bash with `jq`, from Python with `json`, from anything. The tradeoff is no schema enforcement — handlers can send malformed JSON and we just log and move on.

**Q: How do you handle a flaky network where a robot reconnects every few seconds?**
Handlers survive TCP disconnects. `SendDisconnect` nils `RobotSend` but keeps the process alive. On reconnect, TCP server calls `Reattach` with the new `conn.Write` closure. Handler sees a new `connect` message but never lost state. Only the operator (or server shutdown) kills handlers.

**Q: Why embed the MQTT broker instead of using an external one?**
Single-binary deployment. No operator has to install and configure Mosquitto separately. The Mochi-mqtt library gives us hooks (`OnPublished`, ACL) that let us implement the robot protocol as a library-level concern — topic-based auth and message routing are just `strings.HasPrefix` switches. For scale-out we'd replace with EMQX/HiveMQ and move protocol handling to a separate service that subscribes to `robomesh/#`.

---

## 15. Glossary (quick lookup)

- **ActiveRobot** — JSON blob in `robot:{uuid}:active` describing a live session.
- **Handler** — an OS process running `handlers/{type}/start_handler.sh`, one per connected robot.
- **HandlerManager** — global mutex-guarded map `UUID → *HandlerProcess`.
- **LocalBus** — default `comms.Bus` impl (in-proc event bus + Redis pub/sub).
- **Nonce** — hex-encoded random bytes used once in challenge-response.
- **PERSIST** — TCP command that moves a REGISTER-flow robot's pubkey from Redis to PostgreSQL.
- **Reattach** — swap the `RobotSend` callback on an existing handler when a robot reconnects.
- **Reverse connect** — handler-initiated TCP/UDP dial from server to robot.
- **Session JWT** — HS256 token issued on AUTH success; encodes `{uuid, device_type, ip, session_id}`.
- **SSE ticket** — single-use token that lets `EventSource` authenticate (which can't send headers).
