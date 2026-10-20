# Controller–Server–Game Protocol

Status: Draft
Last updated: 2026-10-19

Protocol version: `1` (MVP). This is a design contract, not implemented yet.
Changing a field or message requires updating this file and, for breaking
changes, bumping the protocol version.

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
resets/kicks it.

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
authenticated by a one-time token passed at launch.

```json
{"v":1,"type":"game_hello","data":{"token":"<one-time-token>","protocolVersion":1}}
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
local key action until the server confirms.

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

Slots remain reserved for the session. The host can reset a slot to clear a lost
or replaced tablet.

## Versioning

- MVP uses `v: 1`.
- Additive fields are allowed within version 1; clients ignore unknown fields.
- Breaking changes bump the version and require a new decision/ADR.
- The server logs a protocol mismatch and rejects the connection with
  `protocol_mismatch`.
