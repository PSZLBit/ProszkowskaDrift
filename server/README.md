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
- Health endpoint: `http://<laptop-ip>:8080/healthz` (plain HTTP JSON).
- Controller heartbeat timeout: `6s`.
- All-disconnected pause grace: `10s`.
- Game heartbeat timeout: `3s`.
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
| `-all-disconnected-grace` | `10s` | Pause the round after every controller has been disconnected for this long during a round. |
| `-game-heartbeat-timeout` | `3s` | Close the game connection when no game message arrives for this long after `game_ready`. |
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
  strictly increase after the first input on a connection. Input outside the
  `playing` phase is dropped silently: no error, no abuse counter, and the
  sequence number stays unconsumed.
- `ping` is answered with `pong`. Any message resets the 6 s controller
  heartbeat. A silent connection closes with `heartbeat_timeout`, and the game
  receives `player_left` with reason `timeout`.
- `leave` keeps the slot reserved for the session.
- Writes are queued per connection (controller queue 64, game queue 256). A
  connection that does not drain its queue is closed with `slow_client`; a slow
  client never delays other connections.
- Rejected host commands are answered to the game with `host_denied`
  (`no_players`, `invalid_phase`, or `not_authorized`). `start` needs at least
  one connected controller; `reset` only works from an active round phase.
- A kicked device token is blocked for the rest of the session: reconnecting
  with it returns `error` `kicked`, and a replacement token takes the freed
  slot immediately.
- Host `end` clears slots and tokens, sends `phase` `ended`, closes controllers
  gracefully (`session_ended`), and rejects further `hello`s and `phase`
  changes until the server restarts.
- If every controller disconnects during a round, the server pauses after the
  10 s grace: `host_command` `pause` to the game and `phase` `paused` to
  controllers. A reconnecting controller stops the timer.
- The game endpoint accepts only loopback connections and the configured token.
  The token is presented as `?token=<value>` (the `X-Game-Token` header is also
  accepted). The server sends `game_hello`, then `config`; the game replies with
  `game_ready`.
- The game heartbeat starts after `game_ready`; any game message resets it.
  After 3 s of silence the server closes the connection with
  `game_heartbeat_timeout`, broadcasts `game_unavailable` to controllers,
  returns the session to `attract` (slots stay reserved), and accepts a new
  game connection with the configured token.
- The server owns a minimal phase for Sprint 0: `host_request start` moves to
  `playing` (no countdown), and `pause`, `resume`, `reset`, `end`, and `kick`
  come from the game connection. The game may also send `phase`; the server
  stores it and broadcasts it to controllers. Full round timing and countdown
  remain game work.
- `/healthz` returns `{status, phase, roundId, controllers, gameReady}` for the
  launcher and manual checks.
- `player_joined` and `player_left` are sent to the game. Device tokens are never
  forwarded. Logs show only a truncated token reference.

### Implemented messages

Controller to server: `hello`, `input`, `ping`, `leave`.

Server to controller: `welcome`, `phase`, `error`, `pong`.

Server to game: `game_hello`, `config`, `player_joined`, `player_left`, `input`,
`host_command`, `host_denied`.

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

Server-timer checks:

- All-disconnected pause: with the round in `playing`, stop all fake
  controllers (`Ctrl+C`). After about 10 s the fake game receives
  `host_command` `pause`; controllers reconnecting later get a `welcome` with
  phase `paused`.
- Game drop and recovery: stop the fake game (`Ctrl+C` or `quit`) while
  controllers are connected. Every controller receives `game_unavailable` and
  a `phase` back to `attract`; a restarted fake game reclaims the connection
  with the same token.
- Health: `curl http://127.0.0.1:8080/healthz` prints the phase, connected
  controller count, and game readiness.

The game heartbeat timeout close (3 s without a game message after
`game_ready`) is covered by the automated `TestGameHeartbeatTimeoutClosesGame`
test; a manual check needs a game binary that stops heartbeats and arrives with
the S0-05 game client.

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

- Full session lifecycle, late join.
- Game process supervision and per-launch token rotation (S0-07). A game crash
  or a silent game connection already notifies controllers and accepts a
  reconnect; the server does not launch or restart Godot itself.
- Countdown, round timer ownership, tasks, scoring, and controller feedback.
- TLS, persistence, accounts, or any internet-facing deployment.

## Privacy

All state is in memory. There is no database, account, player name, photo, or
personal profile. Device tokens are random client-generated identifiers; they
are compared in memory and never sent to the game or other controllers. Logs
truncate them.
