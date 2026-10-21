# Controller–Server–Game Protocol

Status: Decided
Last updated: 2026-10-21

Protocol version: `1` (MVP). The Go server baseline implements the controller
and game endpoints in `server/`. The Godot game and Android controller still
need to implement their sides. Changing a field or message requires updating
this file and, for breaking changes, bumping the protocol version.

## Transport and Endpoints

- WebSocket over the local network. Plain `ws://` is acceptable because the
  network is local and carries no personal data.
- Controller endpoint: `ws://<laptop-ip>:8080/ws/controller`
- Game endpoint: `ws://127.0.0.1:8080/ws/game?token=<one-time-token>`
- The game endpoint accepts loopback clients only. The token is generated per
  game launch by the supervisor, or supplied explicitly in development mode.
- JSON text frames, UTF-8. One message per WebSocket text frame.

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
| `seq` | integer | Per-connection sequence for diagnostics and stale-input detection. Required for controller `input`; optional elsewhere. |
| `data` | object | Type-specific payload. |

Unknown fields are ignored. Unknown message types are logged and dropped.
Malformed JSON or a missing data object is rejected with `invalid_input`;
controller errors are counted and repeated errors close the connection. A
version mismatch closes the connection with `protocol_mismatch`.

## Controller → Server

### `hello`

Sent immediately after the WebSocket opens.

```json
{"v":1,"type":"hello","data":{"deviceToken":"<uuid>","appVersion":"0.1.0"}}
```

- `deviceToken`: random token generated and stored on the tablet. Never a name,
  Android ID, or serial number. Maximum 128 characters.
- The server replies with `welcome` or `error`.
- A duplicate token is rejected while its slot is connected. A disconnected
  slot is reclaimed by the same token.
- After a session `end`, every `hello` is rejected with `auth_failed` until the
  server restarts.

### `input`

Sent at 20–30 Hz while the round is `playing`.

```json
{"v":1,"type":"input","seq":128,"data":{"steer":-0.35,"throttle":0.8,"ts":1234567}}
```

- `steer`: `-1.0` (left) to `1.0` (right).
- `throttle`: `-1.0` (brake/reverse) to `1.0` (forward).
- `ts`: client monotonic milliseconds, diagnostics only. The server does not
  trust it for game timing.
- The envelope `seq` is required and must increase after a reconnect reset.
- The server validates the slot, phase, finite ranges, sequence, and rate. Valid
  input is forwarded to the game with the sender slot; stale input is dropped
  silently.
- Invalid values receive `error` with `invalid_input`. Exceeding the rate
  receives `error` with `rate_limited`. Repeated errors close the connection.

### `ping`

```json
{"v":1,"type":"ping","data":{"clientTime":1234567}}
```

Sent every 2 s. The server replies with `pong`. Any controller message also
refreshes the connection idle timer.

### `leave`

```json
{"v":1,"type":"leave","data":{}}
```

Optional clean leave. The slot remains reserved for the session unless the host
kicks it or ends the session.

## Server → Controller

### `welcome`

```json
{"v":1,"type":"welcome","data":{"slot":2,"label":"P2","color":"red","phase":"ready"}}
```

- Assigned slot is `1`–`4`; label and color are stable for the server run.
- Default colors: P1 blue, P2 red, P3 green, P4 yellow.
- If the session is full: `error` with code `session_full`.
- A blocked token gets `error` with code `kicked`.

### `phase`

```json
{"v":1,"type":"phase","data":{"phase":"playing","timeLeftMs":154200,"roundId":7}}
```

Phases: `attract`, `ready`, `countdown`, `playing`, `paused`, `results`,
`ended`. `timeLeftMs` is authoritative from the game. The game sends `phase` on
every change and at least once per second while `playing`/`paused`; the server
broadcasts it to all controllers. Controllers may smooth the visible timer but
must correct it from these messages; they must not treat their own countdown as
the source of truth.

### `player_feedback`

```json
{"v":1,"type":"player_feedback","data":{"slot":2,"task":{"taskId":12,"spotLabel":"B-07","spotColor":"red","hint":"north","deadlineMs":30000},"score":240,"state":"playing"}}
```

- Carries the current task, score, and player state for one slot.
- The game sends it to the server with `slot`. The server caches the latest
  value and sends it only to that slot's controller.
- The game sends it on task assignment, score changes, and
  finished/reconnecting state. The server replays the cached value on reconnect.
- `state` is one of `ready`, `playing`, `finished`, `reconnecting`.
- Do not send map geometry, other players' tasks, or device tokens to a
  controller.

### `task_result`

```json
{"v":1,"type":"task_result","data":{"taskId":12,"points":86,"timeScore":62,"precisionScore":34,"collisionPenalty":10,"rating":"great","totalScore":240}}
```

Sent only to the owning controller after a completed or skipped task. The game
sends the same payload with `slot` to the server; the server removes `slot`
before forwarding.

### `round_result`

```json
{"v":1,"type":"round_result","data":{"roundId":7,"winnerSlot":2,"scores":[{"slot":1,"score":620,"tasks":5,"collisions":1},{"slot":2,"score":710,"tasks":5,"collisions":0}]}}
```

Broadcast to all controllers at the end of a round.

### `error`

```json
{"v":1,"type":"error","data":{"code":"session_full","message":"No free player slot."}}
```

Codes: `auth_failed`, `session_full`, `invalid_input`, `rate_limited`,
`protocol_mismatch`, `game_unavailable`, `kicked`, `internal_error`.

The server may close the connection after a fatal error. `game_unavailable` is
sent to connected controllers when the game connection is lost; the server
returns to `Attract` and waits for the supervisor to restart the game.

### `pong`

```json
{"v":1,"type":"pong","data":{"clientTime":1234567,"serverTime":7654321}}
```

## Server → Game

### `game_hello`

First message from the server after an authenticated game WebSocket opens. The
game proves possession of the one-time token in the connection URL; the token is
not echoed back.

```json
{"v":1,"type":"game_hello","data":{"protocolVersion":1}}
```

### `config`

```json
{"v":1,"type":"config","data":{"roundDurationMs":180000,"tasksPerPlayer":5,"maxPlayers":4}}
```

Sent on connect and after a reset. The game validates values.

### `player_joined` / `player_left`

```json
{"v":1,"type":"player_joined","data":{"slot":2,"label":"P2","color":"red"}}
{"v":1,"type":"player_left","data":{"slot":2,"reason":"disconnect"}}
```

Reasons: `disconnect`, `leave`, `timeout`. A host kick is sent as
`host_command` with `command: "kick"` instead.

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

### `host_denied`

```json
{"v":1,"type":"host_denied","data":{"command":"start","code":"no_players","message":"At least one controller must be connected."}}
```

Codes: `no_players`, `invalid_phase`, `not_authorized`. The game should keep its
local host action pending or discarded and must not apply the command.

## Game → Server

### `game_ready`

```json
{"v":1,"type":"game_ready","data":{"gameVersion":"0.1.0","mapVersion":"proszkowska-1"}}
```

Sent as the first application message, within the handshake timeout. A second
simultaneous game connection is rejected.

### `host_request`

The Godot window captures host keys and asks the server to execute a command.

```json
{"v":1,"type":"host_request","data":{"command":"start"}}
```

The server validates and responds with `host_command` or `host_denied`. The game
ignores its own local key action until the server confirms.

### `phase`

```json
{"v":1,"type":"phase","data":{"phase":"playing","timeLeftMs":179800,"roundId":7}}
```

The server broadcasts this to controllers. Send it on every change and at least
once per second while `playing`/`paused`.

### `player_feedback`

```json
{"v":1,"type":"player_feedback","data":{"slot":2,"task":{"taskId":12,"spotLabel":"B-07","spotColor":"red","hint":"north","deadlineMs":30000},"score":240,"state":"playing"}}
```

Used for task assignment, score changes, finished/reconnecting state, and
per-player HUD data. The server routes it to the owning controller only.

### `task_result`

Same payload shape as `Server → Controller / task_result`, with `slot` added.

### `round_result`

Same payload shape as `Server → Controller / round_result`. The server
broadcasts it to all controllers.

### `heartbeat`

```json
{"v":1,"type":"heartbeat","data":{"gameTimeMs":123456}}
```

Sent every second. The server closes the game connection if no game message
arrives for 3 s and, when supervision is configured, asks the supervisor to
restart Godot.

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
    | player_feedback     | player_feedback             |
    |<--------------------|<---------------------------|
    |                     | round_result                |
    |<--------------------|<---------------------------|
```

## Validation Rules

- `steer` and `throttle` are finite numbers in `[-1, 1]`; reject NaN/Infinity
  and out-of-range values.
- Accept at most 60 input messages per second per slot with a burst of 60;
  expected rate is 20–30.
- Drop input with a sequence number older than the last accepted sequence for
  that slot, except after a reconnect where the sequence resets.
- Input is accepted only while the slot is assigned, connected, and the phase is
  `playing`.
- Invalid messages increment a per-connection error counter. Ten invalid
  messages or three rate-limit errors within 10 s closes the connection with
  `invalid_input`/`rate_limited` and a WebSocket policy close.
- The host commands are authorized server-side: `start` needs at least one
  connected controller and `attract`/`ready`/`results`; `pause` needs `playing`;
  `resume` needs `paused`; `reset` needs an active round; `kick` needs an
  assigned slot.
- The server never forwards raw device tokens to the game or other controllers.

## Reconnect

1. WebSocket drops or the controller idle timeout expires.
2. Server marks the slot disconnected, tells the game to stop the car, and keeps
   the token-to-slot reservation.
3. Controller reconnects with `hello` and the same `deviceToken`.
4. Server reclaims the same slot if the token matches and the session is still
   active; otherwise it returns `auth_failed` or assigns the lowest free slot.
5. Server sends `welcome`, current `phase`, the cached `player_feedback` task and
   score, and the last `round_result` when in `results`.
6. Controller resets its input sequence and resumes sending input. The game
   applies the latest input.

Slots remain reserved for the session. A host kick clears the slot, blocks the
old token for the rest of the server run, and lets a replacement tablet claim the
free slot. A session `end` clears all slots and tokens.

## Session End

The host sends `end` through a `host_request`. The server:

1. Sends `host_command` with `command: "end"` to the game.
2. Clears slots, tokens, feedback, and round state.
3. Broadcasts `phase` with `phase: "ended"`.
4. Gracefully closes controller connections.

The server process keeps running until the operator stops it. After `end`, new
controller `hello` messages are rejected with `auth_failed` and game phase
changes are ignored. Restart the server to begin a new event session.

## Versioning

- MVP uses `v: 1`.
- Additive fields are allowed within version 1; clients ignore unknown fields.
- Breaking changes bump the version and require a new decision/ADR.
- The server logs a protocol mismatch and rejects the connection with
  `protocol_mismatch`.
