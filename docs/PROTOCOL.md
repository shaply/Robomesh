# Robomesh Protocol Reference

Complete protocol documentation for the Robomesh robot management platform. Each protocol is documented in its own file.

## Documents

| Document | Description |
| --- | --- |
| [TCP.md](TCP.md) | Robot TCP protocol — AUTH, REGISTER, PERSIST flows, session mode |
| [HEARTBEAT.md](HEARTBEAT.md) | Signed heartbeat protocol (TCP + HTTP), replay protection, TTL |
| [HANDLER.md](HANDLER.md) | Handler script stdin/stdout JSON-RPC, reverse connections, lifecycle |
| [HTTP_API.md](HTTP_API.md) | All HTTP endpoints — auth, robots, handlers, SSE, plugins |
| [COMM_BUS.md](COMM_BUS.md) | `comms.Bus` interface, event topics, handler integration |
| [TERMINAL.md](TERMINAL.md) | Debug terminal CLI commands |
| [CONFIGURATION.md](CONFIGURATION.md) | config.yaml structure, env vars, Redis key schema, startup/shutdown |

## Quick Links

- **Adding a new robot type?** See [HANDLER.md](HANDLER.md) for directory structure and protocol
- **Integrating a robot client?** See [TCP.md](TCP.md) for AUTH/REGISTER flows, [HEARTBEAT.md](HEARTBEAT.md) for keepalive
- **Building a frontend plugin?** See [HTTP_API.md](HTTP_API.md) plugin system section
- **Deploying?** See [CONFIGURATION.md](CONFIGURATION.md) for env vars and Redis key schema
