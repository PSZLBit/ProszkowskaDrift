# Server Implementation

Status: Implemented
Last updated: 2026-10-21

The Go server in `server/` implements the session and network authority for the
MVP. It does not implement game simulation, task assignment, parking detection,
or scoring. Those stay in the Godot game (D-22, ADR-0001).

## Scope and Boundary

| Server owns | Game owns |
|-------------|----------|
| Configuration, HTTP/WebSocket endpoints, process supervision | Map, cars, physics, collisions |
| Slots, device-token mapping, reconnect reservation | Task assignment and parking detection |
| Phase cache and controller broadcast | Round timer and phase truth |
| Input validation, sequence, rate, and forwarding | Applying validated input |
| Host-command authorization and routing | Executing start/pause/reset/end/kick |
| Per-player feedback routing | Content of task, score, and result feedback |
| Health, structured logs, non-personal diagnostics | Display and audio |

Controllers never connect to the game. The game never receives device tokens.

## Package Layout

```text
server/
  cmd/server/            entry point, flags, signal handling
  internal/config/       defaults, JSON config, validation
  internal/logging/      JSON logging, token redaction
  internal/protocol/     v1 envelope, payloads, error codes
  internal/session/      slots, tokens, phase cache, validation, host rules
  internal/server/       WebSocket endpoints, routing, ticker, health
  internal/supervisor/   Godot launch, token rotation, restart/backoff
  config.example.json
```

## Runtime Flow

1. Load defaults, optional JSON config, and flags.
2. Start the HTTP listener.
3. If a game path is configured and `devNoGame` is false, start the Go
   supervisor. Each launch gets a fresh 32-byte hex game token.
4. Godot connects to `/ws/game?token=...` from loopback, receives `game_hello`,
   sends `game_ready`, and receives `config`.
5. Controllers connect to `/ws/controller`, send `hello`, and receive `welcome`,
   the current `phase`, and any cached `player_feedback`.
6. The game sends `host_request` for host keys. The server validates, sends
   `host_command`, and broadcasts `phase` changes to controllers.
7. Valid controller input is forwarded to the game with its slot. Game feedback
   is routed to the owning controller only.
8. If the game connection or heartbeat is lost, the server returns to
   `Attract`, tells controllers the game is unavailable, and asks the supervisor
   to restart it (when supervision is enabled).

## Defaults

| Setting | Default | Meaning |
|---------|---------|---------|
| Listen | `0.0.0.0:8080` | Controller and health listener |
| Game endpoint | `/ws/game` | Loopback-only, token required |
| Controller endpoint | `/ws/controller` | LAN WebSocket |
| Slots | P1 blue, P2 red, P3 green, P4 yellow | Stable labels and colors |
| Round duration | 180000 ms | Sent to the game as config |
| Tasks per player | 5 | Sent to the game as config |
| Controller idle timeout | 6000 ms | No message means disconnected |
| All-disconnected grace | 10000 ms | Then the server pauses the round |
| Game heartbeat timeout | 3000 ms | Then the game is treated as lost |
| Handshake timeout | 5000 ms | Hello/game_ready deadline |
| Input rate | 60/s, burst 60 | Per-slot token bucket |
| Invalid/rate limits | 10 / 3 per 10 s | Closes an abusive controller |
| Max message | 4096 bytes | WebSocket read limit |
| Write timeout | 2000 ms | One WebSocket write |

The defaults are configurable in JSON. Slot IDs must be ordered `1..maxPlayers`
with unique labels and colors.

## Session and Reconnect Rules

- A new token gets the lowest free slot. Labels and colors are fixed for the
  server run.
- A duplicate token is rejected with `auth_failed` while its slot is connected.
- A disconnected slot stays reserved. The same token reconnects to the same
  slot, clears the old input sequence, and receives the current task and score.
- A full session rejects a new token with `session_full`.
- A kicked token is blocked for the rest of the server run. A replacement tablet
  with a new token can take the freed slot.
- Input is accepted only while the slot is assigned, connected, and the phase is
  `playing`.
- A stale sequence is dropped silently; range, phase, and malformed-payload
  errors receive `error` and increment the per-connection counter.
- Late join is between rounds: a controller can attach at any time, but the
  game decides when a new player enters a round.
- If all controllers disconnect during `countdown` or `playing`, the server
  waits 10 s, sends `host_command pause`, and broadcasts `paused`.
- `end` clears slots, tokens, feedback, and round state, broadcasts `ended`, and
  gracefully closes controllers. New controller `hello` messages are rejected
  and game phase changes are ignored until the server restarts. It does not
  terminate the server process.

## Host Commands

| Command | Server check |
|---------|--------------|
| `start` | At least one connected controller; phase `attract`, `ready`, or `results` |
| `pause` | Phase `playing` |
| `resume` | Phase `paused` |
| `reset` | Phase `countdown`, `playing`, `paused`, or `results` |
| `end` | Phase is not already `ended` |
| `kick` | Valid assigned slot |

The game sends `host_request`; the server replies with `host_command` or
`host_denied`. Controllers cannot request host commands. A reset also sends
`config`. A kick sends `host_command` to the game and closes the controller with
`error code=kicked`.

## Game Link

- The game token is a query parameter. Missing or wrong tokens get HTTP 401.
  Non-loopback clients get HTTP 403, and a second game gets HTTP 409.
- `game_hello` carries the protocol version only; the token is not echoed.
- After `game_ready`, the server sends `config`, replays `player_joined` for all
  connected slots, and broadcasts the current phase.
- The game sends `phase` on every change and at least once per second while
  `playing`/`paused` so controllers receive authoritative time updates. The
  server forwards each phase to all controllers.
- `player_feedback` is cached per slot and sent only to that slot. It carries
  the task, score, and state. A reconnect receives the cached value.
- `task_result` is routed to its slot without the `slot` field. `round_result`
  is broadcast to all controllers.
- `heartbeat` is expected at least every second. A missing heartbeat for three
  seconds closes the game connection and triggers the restart path.
- On a supervised restart the token rotates. On a socket loss without process
  death, the supervisor is asked to restart the game.

## Logging and Privacy

- JSON logs go to stdout and optionally a file.
- Device tokens are not logged; a short diagnostic prefix is used when needed.
- The game token is passed through the process environment, not logged, and not
  forwarded to controllers.
- No accounts, names, photos, persistent profiles, or databases exist.
- Session data is in memory and is discarded on server exit or `end`.

## Verification

Automated:

```sh
cd server
go test ./...
CGO_ENABLED=1 go test -race ./...   # needs a C compiler
go vet ./...
go build ./cmd/server
```

The default Windows toolchain on the project laptop needs `CGO_ENABLED=1` and a
C compiler for `-race`; CI runs it on Ubuntu.

Manual checks still required on the target laptop and tablets:

1. Start the server with the real Godot export; confirm `game_ready`, `config`,
   and `Attract` in the game, and `gameReady=true` in `/healthz`.
2. Connect four tablets; confirm slots and colors, input movement, and visible
   input-to-display latency.
3. Disconnect one tablet; confirm its car stops, the slot is reserved, and the
   same tablet reclaims it.
4. Disconnect all four during `Playing`; confirm the round pauses after 10 s.
5. Kill Godot; confirm the supervisor restarts it and the server returns to
   `Attract`.
6. Kill the server; restart it and confirm the tablets reconnect.

## Known Limits

- Windows vs Linux festival laptop is still open; the supervisor uses `exec` and
  kills the direct child.
- The game/controller integrations are not implemented in this slice, so the
  protocol and hardware checks above are pending.
- The HTTP server does not terminate the process on host `end`; the operator
  restarts the server between events.
