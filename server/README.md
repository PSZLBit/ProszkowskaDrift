# Proszkowska Drift Server

Go session server for the local multiplayer parking game. The server owns
session state, slots, input validation, reconnect handling, host-command
authority, and game-process supervision. The game owns simulation, task
assignment, parking detection, and scoring.

Status: baseline implemented (Sprint 0). The game and Android controller
clients are not implemented in this slice.

## Quick Start

Requirements: Go 1.27 or newer.

```sh
cd server
go test ./...
go vet ./...
go build ./cmd/server
```

Run in development mode without launching Godot. The static token is what a
local fake game or Godot editor client must put in the game WebSocket URL:

```sh
go run ./cmd/server -no-game -game-token dev-token
```

Run with a supervised game process:

```sh
go run ./cmd/server -config config.json
```

Copy `config.example.json` to `config.json` and edit paths locally. Do not commit
`config.json`; it is machine-specific.

## Endpoints

| Endpoint | Purpose |
|----------|---------|
| `ws://<laptop>:8080/ws/controller` | Android controller WebSocket |
| `ws://127.0.0.1:8080/ws/game?token=<one-time-token>` | Game WebSocket, loopback only |
| `GET /healthz` | Local health and readiness JSON |

The game endpoint rejects non-loopback clients, a missing/invalid token, and a
second simultaneous game connection. When the server supervises Godot, it
generates a new token for every launch and passes it as `PD_GAME_TOKEN`, with
`PD_SERVER_URL` and `PD_PROTOCOL_VERSION` in the process environment.

## Configuration

All values are loaded from defaults, then a JSON file, then command-line flags.
See `config.example.json`. The main defaults are:

- Listen on `0.0.0.0:8080`.
- Four slots: P1 blue, P2 red, P3 green, P4 yellow.
- 180 s rounds, five tasks per player.
- Controller idle timeout 6 s; all-disconnected pause after 10 s.
- Game heartbeat timeout 3 s.
- Input rate 60/s per slot with a burst of 60.
- Ten invalid or three rate-limit errors per 10 s closes a controller.
- JSON logs to stdout and an optional file; tokens are never logged in full.

## Tests

```sh
go test ./...
CGO_ENABLED=1 go test -race ./...   # requires a C compiler
go vet ./...
```

The test suite includes protocol parsing, session/slot rules, input validation,
rate limiting, reconnect and feedback caching, host authorization, the game
supervisor, and in-process WebSocket integration with a fake game and fake
controllers, including privacy of per-player feedback.

## Documentation

The source-of-truth design is in
[`documentation/06-development/server.md`](../documentation/06-development/server.md),
[`protocol.md`](../documentation/06-development/protocol.md), and
[`architecture.md`](../documentation/06-development/architecture.md).

## Current Limits

- State is in memory only; restarting the server loses the session.
- The host `end` command clears session state, closes controllers, and rejects
  new controller handshakes; it does not terminate the server process. Restart
  the server between events.
- The game/controller integration and real-hardware checks are still pending.
- The supervisor kills the direct Godot process. Process-tree handling is a
  festival-hardening task if Godot spawns helpers on the target OS.
