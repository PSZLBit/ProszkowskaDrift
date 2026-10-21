# Proszkowska Drift Server

Go server for the local multiplayer parking game. It owns the session boundary
between the Android controllers and the Godot game: slots, device-token
reconnect, input validation, rate limiting, heartbeats, and the minimal host
command path.

This is the Sprint 0 slice (issue #12). It is intentionally small: session
management, game supervision, scoring, and round timing are later milestones.

## Requirements

- Go 1.23 or newer.
- No database, cloud service, or internet connection.

## Run

From `server/`:

```sh
go run ./cmd/server -game-token dev-token
```

Defaults:

- HTTP/WebSocket listen address: `:8080` (all interfaces, local network).
- Controller endpoint: `ws://<laptop-ip>:8080/ws/controller`
- Game endpoint: `ws://127.0.0.1:8080/ws/game?token=<game-token>`
- Controller heartbeat timeout: `6s`.
- Log level: `info`.

If `-game-token` and `PD_GAME_TOKEN` are empty, the server generates a random
token for this process and prints it once. The token is local and ephemeral; pass
it to Godot or the fake game.

### Configuration

| Flag | Default | Purpose |
|------|---------|---------|
| `-addr` | `:8080` | HTTP/WebSocket listen address. |
| `-game-token` | `PD_GAME_TOKEN` | One-time token for the game endpoint. Generated when empty. |
| `-controller-timeout` | `6s` | Close a controller after this long without any message. |
| `-log-level` | `info` | `debug`, `info`, `warn`, or `error`. |

## Protocol behavior in this slice

The wire contract is [documentation/06-development/protocol.md](../documentation/06-development/protocol.md).
This slice implements a subset and records the following choices:

- WebSocket library: [`github.com/coder/websocket`](https://github.com/coder/websocket)
  v1.8.15 (the final library is confirmed after the Sprint 0 spikes).
- Slots and colors: `P1` red, `P2` blue, `P3` green, `P4` yellow.
- A syntactically valid unknown device token gets a free slot. A malformed
  token, or a token whose slot is still connected, gets `auth_failed`. No free
  slot gets `session_full`.
- Reconnect with the same token reclaims the slot after the old connection has
  dropped. The input sequence counter restarts on reconnect; the per-slot rate
  window does not.
- Input limits: 60 messages per sliding second per slot. After 10 consecutive
  invalid messages the connection closes with `invalid_input`; after 20
  consecutive rate violations it closes with `rate_limited`.
- Input is accepted only for an assigned, connected slot while the phase is
  `playing`. Values must be finite and in `[-1, 1]`; sequence numbers must
  strictly increase after the first input on a connection.
- `ping` is answered with `pong`. Any message resets the 6 s controller
  watchdog. A silent connection closes with `heartbeat_timeout`.
- `leave` keeps the slot reserved for the session.
- The game endpoint accepts only loopback connections and the configured token.
  The token is presented as `?token=<value>` (the `X-Game-Token` header is also
  accepted). The server sends `game_hello`, then `config`; the game replies with
  `game_ready`.
- The server owns a minimal phase for Sprint 0: `host_request start` moves to
  `playing` (no countdown), and `pause`, `resume`, `reset`, `end`, and `kick`
  work from the game connection. The game may also send `phase`; the server
  stores it and broadcasts it to controllers. Full round timing and countdown
  remain game work.
- `player_joined` and `player_left` are sent to the game. Device tokens are never
  forwarded. Logs show only a truncated token reference.

### Implemented messages

Controller to server: `hello`, `input`, `ping`, `leave`.

Server to controller: `welcome`, `phase`, `error`, `pong`.

Server to game: `game_hello`, `config`, `player_joined`, `player_left`, `input`,
`host_command`.

Game to server: `game_ready`, `host_request`, `phase`, `heartbeat`.

Unknown message types are logged and dropped. Unknown JSON fields are ignored.
Unsupported envelope versions are rejected with `protocol_mismatch`.

## Test

```sh
cd server
gofmt -l .          # must print nothing
go vet ./...
go test -race ./...
```

The tests use table-driven cases for the protocol, slots, reconnect, input
validation, sequence, and rate limits, plus `httptest` integration tests with
fake controllers and a fake game.

## Manual check with fake clients

The fake clients are Go test tools; do not ship them as production binaries.

Terminal 1 — server:

```sh
cd server
go run ./cmd/server -game-token dev-token -log-level debug
```

Terminal 2 — fake game (starts the round after one second):

```sh
cd server
go run ./cmd/fakegame -token dev-token -auto-start
```

Terminal 3 — four fake controllers, one invalid input each, and one
disconnect/reconnect:

```sh
cd server
go run ./cmd/fakecontroller -clients 4 -duration 15s -invalid-after 3s -drop-after 5s
```

Expected:

- Four `welcome` messages with slots 1–4, labels `P1`–`P4`, and the four colors.
- The fake game reports `player_joined` for each slot and input counters for all
  four slots.
- Each fake controller receives one `invalid_input` error after the out-of-range
  sample and keeps running.
- Player 1 drops, reconnects with the same token, and receives the same slot
  again. Its sequence counter restarts.
- At the end, `leave` keeps the slots reserved on the server.

Heartbeat timeout check (run while the fake game is still in `playing`):

```sh
go run ./cmd/fakecontroller -clients 1 -duration 12s -silent-after 2s
```

The server closes the controller after about 6 s of silence with reason
`heartbeat_timeout`; the fake controller prints the close reason.

The fake game accepts interactive commands on stdin:

```text
start
pause
resume
reset
end
kick 2
phase playing 120000 7
quit
```

## Not implemented in this slice

- Full session lifecycle, late join, and all-disconnected pause.
- Game process supervision, crash restart, and missing-game-heartbeat handling
  (S0-07).
- Countdown, round timer ownership, tasks, scoring, and controller feedback.
- A kicked-token ban list. Kick clears the slot and closes the controller; an
  immediate reconnect with the same token can claim a free slot.
- TLS, persistence, accounts, or any internet-facing deployment.

## Privacy

All state is in memory. There is no database, account, player name, photo, or
personal profile. Device tokens are random client-generated identifiers; they
are compared in memory and never sent to the game or other controllers. Logs
truncate them.
