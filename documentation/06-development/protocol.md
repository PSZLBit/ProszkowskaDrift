# Controller–Server–Game Protocol

Status: Draft
Last updated: 2026-10-26

Protocol version: `1` (MVP). The S0-04 server slice implements a subset of this
contract; see "Sprint 0 Implementation Notes" at the end. Changing a field or
message requires updating this file and, for breaking changes, bumping the
protocol version.

## Transport and Endpoints

- WebSocket over the local network. Plain `ws://` is acceptable because the
  network is local and carries no personal data.
- Controller endpoint: `ws://<laptop-ip>:8080/ws/controller`
- Game endpoint: `ws://127.0.0.1:8080/ws/game` (game token required)
- JSON text frames, UTF-8.

## Envelope

Every message uses the same envelope:

```json
{
  "v": 1,
  "type": "input",
  "seq": 42,
  "data": {}
}
```

| Field | Type | Meaning |
|-------|------|---------|
| `v` | integer | Protocol version. Server rejects unsupported versions. |
| `type` | string | Message type. |
| `seq` | integer | Per-connection sequence for diagnostics and stale-input detection. Optional for server→client messages. |
| `data` | object | Type-specific payload. |

Unknown fields are ignored. Unknown message types are logged and dropped.

## Controller → Server

### `hello`

Sent immediately after the WebSocket opens.

```json
{"v":1,"type":"hello","data":{"deviceToken":"<uuid>","appVersion":"0.1.0"}}
```

- `deviceToken`: random UUID generated and stored on the tablet. Never a name,
  Android ID, or serial number.
- Server replies with `welcome` or `error`.

### `input`

Sent at 20–30 Hz while the round is `Playing`.

```json
{"v":1,"type":"input","seq":128,"data":{"steer":-0.35,"throttle":0.8,"ts":1234567}}
```

- `steer`: `-1.0` (left) to `1.0` (right).
- `throttle`: `-1.0` (brake/reverse) to `1.0` (forward).
- `ts`: client monotonic milliseconds, diagnostics only. Server does not trust
  it for game timing.
- Server validates ranges and rate, drops invalid values, and forwards valid
  input to the game with the sender slot.

### `ping`

```json
{"v":1,"type":"ping","data":{"clientTime":1234567}}
```

Sent every 2 s. Server replies with `pong`.

### `leave`

```json
{"v":1,"type":"leave","data":{}}
```

Optional clean leave. The slot remains reserved for the session unless the host
kicks it.

## Server → Controller

### `welcome`

```json
{"v":1,"type":"welcome","data":{"slot":2,"label":"P2","color":"red","phase":"ready"}}
```

- Assigned slot is `1`–`4`; color and label are stable for the session.
- If the session is full: `error` with code `session_full`.

### `phase`

```json
{"v":1,"type":"phase","data":{"phase":"playing","timeLeftMs":154200,"roundId":7}}
```

Phases: `attract`, `ready`, `countdown`, `playing`, `paused`, `results`,
`ended`. `timeLeftMs` is authoritative from the game; clients must not run their
own countdown as the source of truth.

### `task`

```json
{"v":1,"type":"task","data":{"taskId":12,"spotLabel":"B-07","spotColor":"red","hint":"north","deadlineMs":30000}}
```

- Sent when a task is assigned.
- `spotLabel` is visible on the map; `hint` is a short direction aid.
- Do not send map geometry to controllers.

### `task_result`

```json
{"v":1,"type":"task_result","data":{"taskId":12,"points":86,"timeScore":62,"precisionScore":34,"collisionPenalty":10,"rating":"great","totalScore":240}}
```

### `round_result`

```json
{"v":1,"type":"round_result","data":{"roundId":7,"winnerSlot":2,"scores":[{"slot":1,"score":620,"tasks":5,"collisions":1},{"slot":2,"score":710,"tasks":5,"collisions":0}]}}
```

### `error`

```json
{"v":1,"type":"error","data":{"code":"session_full","message":"No free player slot."}}
```

Codes: `auth_failed`, `session_full`, `invalid_input`, `rate_limited`,
`protocol_mismatch`, `game_unavailable`, `kicked`, `internal_error`.

### `pong`

```json
{"v":1,"type":"pong","data":{"clientTime":1234567,"serverTime":7654321}}
```

## Server → Game

### `game_hello`

First message from the server after the game WebSocket opens. The game is
authenticated by a per-launch token passed at startup.

```json
{"v":1,"type":"game_hello","data":{"token":"<per-launch-token>","protocolVersion":1}}
```

### `config`

```json
{"v":1,"type":"config","data":{"roundDurationMs":180000,"tasksPerPlayer":5,"maxPlayers":4}}
```

Sent once when the game connects. The game validates values.

### `player_joined` / `player_left`

```json
{"v":1,"type":"player_joined","data":{"slot":2,"label":"P2","color":"red"}}
{"v":1,"type":"player_left","data":{"slot":2,"reason":"disconnect"}}
```

The game does not receive device tokens.

### `input`

```json
{"v":1,"type":"input","data":{"slot":2,"seq":128,"steer":-0.35,"throttle":0.8}}
```

The server forwards only validated input. The game stores the latest input per
slot and applies it each frame.

### `host_command`

```json
{"v":1,"type":"host_command","data":{"command":"start"}}
{"v":1,"type":"host_command","data":{"command":"kick","slot":3}}
```

Commands: `start`, `pause`, `resume`, `reset`, `end`, `kick`.
The server is the authority. The game must apply commands only when received
from the server, not when the host key is pressed locally.

## Game → Server

### `game_ready`

```json
{"v":1,"type":"game_ready","data":{"gameVersion":"0.1.0","mapVersion":"proszkowska-1"}}
```

### `host_request`

The Godot window captures host keys and asks the server to execute a command.

```json
{"v":1,"type":"host_request","data":{"command":"start"}}
```

The server validates and responds with `host_command`. The game ignores its own
local key action until the server confirms. Until the server has received
`game_ready` on the current game connection, every `host_request` is answered
with `host_denied` code `invalid_phase`.

### `phase`

```json
{"v":1,"type":"phase","data":{"phase":"playing","timeLeftMs":179800,"roundId":7}}
```

The server broadcasts this to controllers.

### `player_feedback`

```json
{"v":1,"type":"player_feedback","data":{"slot":2,"task":{"taskId":12,"spotLabel":"B-07","spotColor":"red","hint":"north","deadlineMs":30000},"score":240,"state":"playing"}}
```

Used for task assignment, score changes, finished/reconnecting state, and
per-player HUD data.

### `task_result`

Same payload shape as `Server → Controller / task_result`, with `slot` added.

### `round_result`

Same payload shape as `Server → Controller / round_result`.

### `heartbeat`

```json
{"v":1,"type":"heartbeat","data":{"gameTimeMs":123456}}
```

Sent every second. Server restarts the game if heartbeat is missing for 3 s.

## Timing and Flow

```text
Controller            Server                         Game
    | hello               |                             |
    |-------------------->|                             |
    | welcome             |                             |
    |<--------------------|                             |
    |                     | game_hello / game_ready     |
    |                     |<--------------------------->|
    |                     | config                      |
    |                     |--------------------------->|
    |                     | host_request (start)        |
    |                     |<---------------------------|
    |                     | host_command (start)        |
    |                     |--------------------------->|
    |                     | phase(countdown/playing)    |
    |                     |<---------------------------|
    | phase(playing)      |                             |
    |<--------------------|                             |
    | input 20-30 Hz      | validated input             |
    |-------------------->|--------------------------->|
    | task / task_result  | player_feedback             |
    |<--------------------|<---------------------------|
    |                     | round_result                |
    |<--------------------|<---------------------------|
```

## Validation Rules

- `steer` and `throttle` are finite numbers in `[-1, 1]`; reject NaN/Infinity
  and out-of-range values.
- Accept at most 60 input messages per second per slot; expected rate is 20–30.
- Drop input with a sequence number older than the last accepted sequence for
  that slot, except after a reconnect where the sequence resets.
- Input is accepted only while the slot is assigned and the phase is `playing`.
- Invalid messages increment a per-connection error counter; repeated abuse
  closes the connection with `rate_limited` or `invalid_input`.
- The server never forwards raw device tokens to the game or other controllers.

## Reconnect

1. WebSocket drops or heartbeat expires.
2. Server marks the slot disconnected and tells the game to stop the car.
3. Controller reconnects with `hello` and the same `deviceToken`.
4. Server reclaims the same slot if the token matches and the session is still
   active; otherwise it returns `auth_failed` or assigns a free slot.
5. Server sends `welcome`, current `phase`, and the current task if the round is
   still running.
6. Controller resumes sending input. The game applies the latest input.

Slots remain reserved for the session. The host can kick a slot to clear a lost
or replaced tablet.

## Versioning

- MVP uses `v: 1`.
- Additive fields are allowed within version 1; clients ignore unknown fields.
- Breaking changes bump the version and require a new decision/ADR.
- The server logs a protocol mismatch and rejects the connection with
  `protocol_mismatch`.

## Sprint 0 Implementation Notes (2026-10-22)

The `server/` slice for S0-04 implements the envelope and a subset of this
contract, using `github.com/coder/websocket` v1.8.15. This section records the
choices and deviations. The document stays `Draft`; these are not final protocol
decisions.

Implemented now:

- Controller endpoint, game endpoint, envelope validation, unknown-field
  tolerance, and unknown-type dropping.
- Controller `hello`, `input`, `ping`, and `leave`; server `welcome`, `phase`,
  `error`, and `pong`.
- Game `game_hello`, `config`, `player_joined`, `player_left`, forwarded
  `input`, and `host_command`; game `game_ready`, `host_request`, `phase`,
  `heartbeat`, and server `host_denied`.
- Range, finite-number, phase, slot, sequence, and 60-messages-per-second
  validation. Invalid values are dropped and reported; repeated abuse closes
  the connection.
- Device-token slot assignment and reconnect, with the slot reserved for the
  session.
- Server-side timers on a 250 ms tick: controller idle timeout, all-disconnected
  pause, and game heartbeat monitoring.

Implementation choices:

- Slot colors are `P1` red, `P2` blue, `P3` green, `P4` yellow. The example in
  this document is illustrative, not a color mapping.
- Controller heartbeat timeout is 6 s. Any controller message resets it; the
  controller is expected to ping every 2 s. Close reason: `heartbeat_timeout`.
  Timed-out controllers are closed with `player_left.reason` `timeout`.
- A connection closes with `invalid_input` after 10 rejected input messages
  without an accepted input between them and with `rate_limited` after 20 rate
  violations without an accepted input between them. Only an accepted input
  resets both counters; pings and other message types do not. Input that
  arrives outside the `playing` phase is dropped silently: no error, no
  counter, and the sequence number stays unconsumed, so a controller that keeps
  sending during a pause does not lose its sequence position.
- A connection whose outbound queue stays full is closed with the
  `slow_client` close reason; broadcasts never block on it.
- A malformed device token, or a token whose slot is still connected, receives
  `auth_failed`. A syntactically valid unknown token gets a free slot. A full
  session receives `session_full`; the connection stays open and may retry.
- The game token is presented as `?token=<value>` on `/ws/game` (the
  `X-Game-Token` header is also accepted). The endpoint rejects non-loopback
  connections before the WebSocket handshake.

Session-boundary choices (this sprint, additive):

- `host_request` that the server rejects is answered with `host_denied`
  carrying `command`, `slot`, `code`, and `message`. Codes:
  `no_players` (start with no connected controller), `invalid_phase` (command
  not allowed in the current phase, including any command before the current
  game connection sends `game_ready`), and `not_authorized` (unknown command or
  slot). `reset` is only valid from `countdown`, `playing`, `paused`, and
  `results`.
- Kick blocks the kicked device token for the rest of the session: a reconnect
  with the same token receives `error` `kicked`. A replacement token can claim
  the freed slot immediately. There is no cross-session ban list.
- `player_joined` and `player_left` are emitted in the same order as the
  session slot transitions they describe. A game that connects while
  controllers are joining or leaving receives the current snapshot without
  duplicate or inverted events.
- Host `end` clears slots, tokens, and the round identifier. New `hello`s and
  `phase` changes are rejected until the server restarts; connected
  controllers receive `phase` `ended` and a graceful close with reason
  `session_ended`. Host commands are serialized with controller registration
  and slot transitions, so a controller cannot join a session that just ended
  and the game cannot receive `player_joined` after host `end`.
- If every controller disconnects during `countdown` or `playing`, the server
  waits 10 s (configurable) and then pauses the round: it sends
  `host_command` `pause` to the game and broadcasts `phase` `paused`. A
  reconnecting controller stops the grace timer. This is the only case where
  the server changes the phase without a host request. Game `phase` messages
  share the same transition lock as the timer and slot changes, so a
  concurrent game phase cannot overwrite the paused broadcast.
- The game heartbeat is live: any valid game message resets it. After 3 s
  without a game message on a ready connection, the server closes it with
  reason `game_heartbeat_timeout`, broadcasts `error` `game_unavailable`,
  resets the session to `attract` (slots stay reserved), and waits for the
  game to reconnect with the configured token. Token rotation per game launch
  arrives with S0-07 game supervision.
- `/healthz` (plain HTTP, not WebSocket) returns
  `{status, phase, roundId, controllers, gameReady}` for the launcher and
  manual checks.

Deviations and open points:

- This document does not say how the game presents its token. The query
  parameter is the Sprint 0 choice. `game_hello` still carries the token as
  documented, but there is no game-to-server authentication message.
- The server owns a minimal phase for S0-04 so input gating can be tested
  without a full Godot round: `start` moves directly to `playing` (no
  `countdown`), `pause`/`resume` toggle `playing`/`paused`, `reset` moves to
  `ready`, and `end` moves to `ended`. Server-generated `phase` messages omit
  `timeLeftMs` because the server does not own a round timer. The game may send
  `phase` (including `timeLeftMs` and `roundId`); the server stores it and
  broadcasts it to controllers. Full countdown and timer ownership remain game
  work. The all-disconnected pause is the exception: the server sets `paused`.
- The server sends `phase` only to controllers. The game learns the accepted
  host command from `host_command`; there is no server-to-game phase message.
- `player_left.reason` may be `disconnect`, `leave`, `kicked`, or `timeout`
  (idle timeout). The extra values are additive.
- On reconnect the server sends `welcome` with the current phase, but not the
  current task, because task data is not implemented yet.
- Server-to-client messages omit `seq`; it is optional in the envelope.
- A controller's `host_request` is an unknown type on the controller endpoint,
  so it is logged and dropped by design. There is no host request message for
  controllers.

## Sprint 0 Implementation Notes (S0-05)

The `game/` scaffold promoted from S0-05 (D-32) implements the contract above
with these choices and deviations. This document stays `Draft`.

Implemented now:

- One `/ws/game` connection per game process. The server replaces the previous
  game connection, so one `ServerInput` node owns the connection and all four
  cars read from its slot-keyed input store.
- Each slot has its own 500 ms input timeout, last sequence, and clear path.
  `player_left` and `host_command` `kick` clear one slot; pause, reset, end, and
  a connection loss clear every slot.
- The game requires the protocol envelope and rejects a bare `InputData` object.
- `game_hello.protocolVersion` and `config` values are validated before the game
  sends `game_ready`; invalid values close the connection instead of starting a
  partially configured round.
- Input requires `slot` and `seq` and validates `steer`/`throttle` as finite
  numbers in `[-1, 1]`. A duplicate or older sequence is ignored. `player_joined`
  resets the slot, which handles the sequence reset after a controller
  reconnect.
- The game reads its token from the launcher/configuration; `PD_GAME_TOKEN` is
  the development path. The token is not committed to the repository. S0-07
  will inject a per-launch token.
- `auth_failed`, `kicked`, and `protocol_mismatch` are fatal: the game stops
  reconnecting. The same values are checked as WebSocket close reasons, because
  the server can reject the game before or during the handshake and the final
  error frame may not be delivered. Other disconnects retry with exponential
  backoff from 1 s to 30 s. A missing token also fails fast.
- Host keys `Enter` (start or resume), `P` (pause or resume), and `R` (reset)
  send `host_request`. `dev_auto_start_round` is a development flag, default
  `false`.
- A main scene, 1920×1080 viewport, and fullscreen mode are configured in
  `game/project.godot`.

Deviations and open points:

- The game does not send an `error` message for a protocol or config failure; it
  logs the reason and closes the connection. The current contract has no
  game-to-server error message.
- The server remains the phase authority. The game tracks the last accepted
  `host_command` only to choose the correct local host-key action. The game does
  not send `phase` messages yet, so the server cannot get `timeLeftMs` or task
  state from this scaffold.
- `host_command` `kick` carries the slot and clears that car. The server also
  emits `player_left` with reason `kicked`, so the slot is cleared twice; the
  operation is idempotent.
- The four cars use placeholder art and a single background with boundary walls.
  Map collision geometry, tasks, scoring, HUD, and the final assets are later
  work.
- Four-car display, TV readability, and FPS/tearing measurements are not
  claimed by the scaffold; they are recorded in `testing.md` after the target
  hardware run (issue #13).
