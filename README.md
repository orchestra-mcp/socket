# Orchestra Socket Plugin

WebSocket pub/sub messaging system for real-time communication. Channel-based broadcasting, direct client messaging, and Redis bridge for multi-instance clustering.

## Features

- **Channel pub/sub** — clients subscribe to named channels and receive published messages
- **Direct messaging** — send to specific connected clients by ID
- **Redis bridge** — relay messages across server instances via Redis pub/sub (graceful fallback to standalone)
- **Connection hooks** — register callbacks for connect/disconnect events
- **3 MCP tools** — `list_ws_clients`, `ws_publish`, `list_ws_channels`

## Architecture

```
Client ──WebSocket──▶ Hub (event loop) ──▶ Channel subscribers
                       │                    │
                       ▼                    ▼
                   RedisBridge ◀──────▶ Other instances
```

- **Hub** — single-goroutine event loop managing clients, channels, and subscriptions
- **Client** — dual-pump (ReadPump + WritePump) per WebSocket connection
- **RedisBridge** — envelope-based pub/sub with instance-ID deduplication
- **Service** — high-level API wrapping Hub for dependency injection

## Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `MaxConnections` | 1000 | Maximum concurrent WebSocket connections |
| `PingInterval` | 30s | Keep-alive ping interval |
| `WriteTimeout` | 10s | Write deadline per message |
| `ReadBufferSize` | 1024 | WebSocket read buffer bytes |
| `WriteBufferSize` | 1024 | WebSocket write buffer bytes |

Redis bridge reads from environment: `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`, `REDIS_PUBSUB_PREFIX`.

## HTTP Routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/ws` | WebSocket upgrade endpoint |
| `GET` | `/ws/info` | Connection stats (clients, channels) |

## MCP Tools

| Tool | Description |
|------|-------------|
| `list_ws_clients` | Connected clients with metadata |
| `ws_publish` | Publish message to a channel |
| `list_ws_channels` | Active channels with subscriber counts |

## Package Structure

```
plugins/socket/
├── config/socket.go       # SocketConfig with defaults
├── providers/
│   ├── plugin.go          # SocketPlugin (activate, services, MCP tools)
│   ├── routes.go          # /ws and /ws/info endpoints
│   └── tools.go           # 3 MCP tool definitions
├── src/
│   ├── bridge/bridge.go   # Bridge interface + RedisBridge
│   ├── hub/
│   │   ├── hub.go         # Hub struct, event loop, client lifecycle
│   │   ├── pubsub.go      # Publish, Subscribe, broadcast, bridge relay
│   │   └── queries.go     # ConnectedClients, Channels, callbacks
│   ├── service/service.go # High-level Service API
│   └── types/types.go     # Message, ClientInfo, Conn, MessageHandler
├── tests/
│   ├── hub_test.go        # Mock infrastructure + hub-level tests
│   └── service_test.go    # Service-level + config tests
└── go.mod
```
