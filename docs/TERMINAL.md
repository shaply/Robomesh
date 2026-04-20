# Terminal Commands

The debug terminal server (default port 6000, env var `TERMINAL_PORT`) accepts TCP connections with a line-based CLI. Useful for debugging and manual robot management without the web frontend.

**Security:** The terminal binds to `127.0.0.1` only (localhost). It provides full admin access (shutdown, accept/reject registrations, list robots) with no authentication, so it must not be exposed to the network.

Connect via: `telnet localhost 6000` or `nc localhost 6000`

## Commands

| Command | Description |
| --- | --- |
| `list` | List active robots (from Redis) |
| `robots` | List registered robots (from PostgreSQL) |
| `pending` | List pending robot registrations |
| `accept <uuid>` | Accept a pending registration |
| `reject <uuid>` | Reject a pending registration |
| `status <uuid>` | Get robot online status |
| `stop program` | Shut down the server |
| `subscribe <event>` | Subscribe to event type (prints events to terminal) |
| `unsubscribe <event>` | Unsubscribe from event type |
| `publish <event> <data>` | Publish an event on the comm bus |
| `help [command]` | Show available commands or help for a specific command |
| `exit` / `quit` | Close terminal session |
