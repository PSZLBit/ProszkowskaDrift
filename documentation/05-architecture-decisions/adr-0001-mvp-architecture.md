# ADR-0001: MVP Architecture and Stack

Status: Decided
Last updated: 2026-10-19
Date: 2026-10-19
Owner: Team

## Context

The MVP is a local multiplayer 2D parking time attack for 1–4 players. Four
team-owned Samsung tablets running Android 4.4.2 act as controllers; a laptop
runs the server and game and drives a TV/monitor. There is no internet play, no
account system, and no persistence requirement. The game must be reliable enough
for a festival stand and must be built by a four-person team with backend,
C/C++, web, and product/testing skills, but no prior Android or game-dev
experience. AI assistance is expected, with peer review.

The existing documentation left the stack, transport, and simulation boundary
open (D-01, D-17). The team selected a direction to unblock a one-week spike and
the first vertical slice.

## Options Considered

### Server and game stack

1. **Go server + Godot 4 game + minimal Kotlin controller (selected).**
   Go matches the strongest backend skill on the team and produces a simple
   single binary. Godot provides 2D rendering, collisions, a scene editor, and
   fast iteration. Kotlin is a small, isolated controller app.
2. **Kotlin/Ktor server + Compose Desktop/LibGDX game.**
   One language across server and controller, but no one on the team has Kotlin
   backend experience, and Compose Desktop/LibGDX would add another learning
   curve.
3. **Node/TypeScript server + browser game.**
   Strong web skill on the team, but the game needs a reliable local rendering
   and physics loop, and the controller still needs a native Android app for
   API 19. This would split effort across three unfamiliar runtimes.
4. **Go server + custom 2D loop without an engine.**
   Fewer runtimes, but the team would have to build rendering, collision,
   camera, HUD, and asset handling from scratch.

### Simulation ownership

- **A. Godot owns simulation, collisions, parking detection, and scoring
  (selected).** The server owns session, slots, input validation, reconnects,
  host commands, and process supervision. This uses Godot's strengths and gets
  to a playable prototype fastest. Cheating is not a concern on team-owned
  tablets on a local network.
- **B. Go owns the simulation; Godot is a renderer.** Cleaner server authority,
  but requires implementing car physics, collisions, and parking detection in Go
  and streaming state snapshots to Godot. More work for no festival benefit.
- **C. Godot owns the server too.** Removes Go and the server/game boundary, but
  loses the team's backend strength and the session/ops layer.

### Transport and persistence

- **WebSocket + JSON (selected).** Simple, debuggable, works with Go, Godot, and
  Android 4.4.2 clients. Four players at 20–30 Hz is a tiny load. TCP head-of-line
  blocking is acceptable for a local party game.
- WebRTC, raw UDP, and HTTP polling were rejected for MVP: signalling complexity,
  custom reliability work, or the wrong traffic shape.
- **In-memory state (selected).** Matches D-12/D-25. No database, no accounts, no
  personal data.

### Connection model

- **Auto-connect with a stored device token and slot reclaim (selected).**
  The tablets are fixed team hardware. QR scanning on old cameras and per-player
  join flows add no value at the stand.
- QR or manual address entry remains a setup fallback only.

## Decision

Adopt the following MVP architecture:

- `server/`: **Go** process. Owns session state, player slots, input validation,
  rate limiting, heartbeat/reconnect, host commands, the WebSocket endpoints,
  and the Godot process supervisor.
- `game/`: **Godot 4** project. Owns the fixed map, arcade car movement,
  environment collisions, parking detection, scoring, HUD, attract mode,
  results, and fullscreen rendering.
- `controller-android/`: **minimal Kotlin** app for Android 4.4.2 (API 19).
  Sends input and displays task/score/time. No game rendering. See ADR-0002.
- **WebSocket + JSON** between controllers and server, and between server and
  game. The game connects to the server on localhost; controllers connect over
  the local network.
- **In-memory** session and score state. No database.
- The server is authoritative for session actions and input authenticity. The
  game is the trusted local simulation/scoring module. This boundary is recorded
  in D-22 and detailed in `06-development/architecture.md` and
  `06-development/protocol.md`.
- Fixed tablets auto-connect to the configured server and reclaim slots using a
  random device token (D-21).
- Real-time targets: 60 FPS display, 20–30 Hz input, ≤100 ms input-to-visible
  response (D-23).

## Consequences

- Three runtimes/languages: Go, GDScript (Godot), and Kotlin. Each is isolated
  behind a narrow protocol, and the controller is intentionally small.
- The team must maintain a protocol contract and reconnect behavior across three
  components.
- Godot must be packaged and supervised by Go; the festival launcher is part of
  the server's responsibility.
- The game logic is tested with Godot tests and manual playtests, while session
  and networking logic are tested in Go.
- Android 4.4.2 limits the controller libraries and UI toolkit (ADR-0002).
- No database or cloud service is needed. The laptop is the only host.
- A future game swap is possible through the server↔game protocol, but the MVP
  does not build a plugin system or a second game.

## Validation Gate (Sprint 0)

The decision is accepted, but feasibility must be proven in a one-week,
throwaway spike (D-18):

1. Build and install a minimal Kotlin app on all four tablets; send WebSocket
   input to the Go server.
2. Start Godot on the target laptop, connect it to Go, and show four moving
   rectangles on the TV.
3. Measure input-to-display latency, Wi-Fi stability, tablet charging, and
   readability at the intended viewing distance.
4. Confirm the Go supervisor can start and restart the Godot process.

If any of these fail, record a new dated decision and supersede the affected
parts of this ADR. Do not silently replace the stack.

## Links

- D-22 (stack and boundary), D-23 (latency targets), D-20/D-21 (controller and
  connection constraints)
- `05-architecture-decisions/adr-0002-legacy-android-controller.md`
- `06-development/architecture.md`
- `06-development/protocol.md`
- `06-development/roadmap.md`
