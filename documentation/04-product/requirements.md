# Product Requirements

Status: Draft
Last updated: 2026-10-19

All items below are planned and not implemented unless marked otherwise. Game
rules and tuning values are in `02-game-domain/game-design.md`; architecture and
protocol are in `06-development/architecture.md` and `06-development/protocol.md`.

## Controller (`controller-android/`, minimal Kotlin, Android 4.4.2)

- `FR-C01` Connect and register: on launch, connect automatically to the
  configured server address, send a random device token, and receive a player
  slot, color, and label.
- `FR-C02` Input send: while a round is active, send steering and throttle at
  20–30 Hz with a sequence number. Stop sending on pause, round end, or
  disconnect.
- `FR-C03` Feedback: show connection state, player slot/color, current parking
  task, score, round time, and round state (`Ready`, `Countdown`, `Playing`,
  `Paused`, `Finished`, `Results`).
- `FR-C04` Reconnect: reconnect automatically with backoff, reclaim the same
  slot with the device token, and show a reconnect overlay. No app restart
  required.
- `FR-C05` Legacy Android and kiosk: run on API 19; landscape; keep screen awake;
  immersive/fullscreen; register as a home launcher; relaunch after reboot.
  Screen pinning is unavailable on API 19 and is not required.
- `FR-C06` Controller only: do not render the game map or simulate car
  movement on the tablet.
- `FR-C07` Setup: provide a hidden team-only screen for the server address and
  diagnostics (connection, latency, app version). Visitors cannot reach it.

## Server (`server/`, Go)

- `FR-S01` Session startup: load configuration, start the Godot game process,
  expose controller and game WebSocket endpoints, and enter `Attract`.
- `FR-S02` Slots: assign one of four slots and a color to each connected device
  token, reject duplicate active tokens, and let the host reset or kick a slot.
- `FR-S03` Input validation and routing: verify the sender slot, numeric ranges,
  rate, and sequence; drop invalid or stale input; forward valid input to the
  game. Never trust controller UI.
- `FR-S04` Heartbeat and reconnect: detect dropped controllers, reserve their
  slots, notify the game, and accept a reconnect with the same device token.
- `FR-S05` Host commands: start, pause, resume, reset, end, and kick. Enforce
  every command server-side and reflect it on the display and controllers.
- `FR-S06` Game supervision: launch and restart the Godot process, pass a
  one-time game token, log exits, and return to `Attract` after a crash.
- `FR-S07` Session end: discard slots, connection handles, game state, and
  scores. Do not write personal data or persistent profiles.
- `FR-S08` Diagnostics: structured logs without personal data; configurable
  port, game executable, and allowed network interface.
- `FR-S09` Offline operation: work entirely on the local network with no
  internet access, cloud service, or database.

## Game (`game/`, Godot 4)

- `FR-G01` Map and display: one fixed top-down map, whole map visible at once,
  fullscreen on the connected TV/monitor, 1920×1080 reference resolution.
- `FR-G02` Round: support 1–4 cars, a shared 180 s timer, countdown, playing,
  pause, results, and reset.
- `FR-G03` Parking tasks: assign five unique target spots per player, highlight
  the target in the player's color, and auto-skip a task after 45 s.
- `FR-G04` Movement and collisions: arcade car movement; collide with static
  environment only; disable car-to-car collisions; stop a car when input is
  missing for 500 ms.
- `FR-G05` Parking detection and scoring: detect a parked car from corner
  containment, speed, and a 0.5 s hold; compute time, precision, and collision
  scores as defined in `02-game-domain/game-design.md`.
- `FR-G06` HUD and results: show timer, scores, task status, collision count,
  winner, and a results screen for about 15 s.
- `FR-G07` Attract mode: show title, one-line instructions, start prompt, control
  hint, and optional anonymous event scores.
- `FR-G08` Controller feedback: send current task, score, time, and round state
  to the server for each player.
- `FR-G09` Pause and disconnect behavior: stop accepting input while paused;
  stop a disconnected car; show the reconnect/finished states.
- `FR-G10` Future-map hook: keep the perimeter loop in the map for a future
  racing mode, without implementing racing in MVP.

## Post-MVP (Not Implemented)

- `FR-P01` Racing mode on the perimeter loop.
- `FR-P02` Drift mechanic and skid effects.
- `FR-P03` Additional task types: delivery/pickup, parallel parking, obstacle
  courses, timed bonus spots.
- `FR-P04` Web host panel for a second operator device.
- `FR-P05` Anonymous event leaderboard, only if it remains free of personal data.
- `FR-P06` Audio, richer art, and particle effects.

## Non-Functional Requirements

- `NFR-01` Server-authoritative session: slot assignment, kick, start/pause/
  reset/end, and input validity are enforced by the server, not the controller.
- `NFR-02` Local network only; no internet play, cloud relay, or remote accounts.
- `NFR-03` No accounts, player names, photos, or persistent personal data.
  Session state is in memory and discarded at session end. Only anonymous
  in-memory event scores may be shown (D-25).
- `NFR-04` Real-time targets: 60 FPS display, 20–30 Hz input, ≤100 ms
  input-to-visible response (D-23).
- `NFR-05` Controller compatibility: Android 4.4.2 (API 19); no Compose or
  OkHttp 4; avoid AndroidX unless a library is proven API 19 compatible;
  validated in Sprint 0 (D-20).
- `NFR-06` Festival robustness: auto-reconnect, auto-restart after game/server
  crash, clear recovery states, and a documented runbook.
- `NFR-07` English-only documentation, code, comments, commits, and UI strings
  (D-05).
- `NFR-08` Free-tier tooling only: GitHub, GitHub Actions within the free
  allowance, open-source libraries.
- `NFR-09` Full festival setup (power, network, tablets, display, game) in under
  15 minutes by two team members.
- `NFR-10` Readability: map, cars, targets, and scores must be legible from the
  intended spectator distance in the venue.

## Out of Scope for MVP

Internet play, iOS controllers, bring-your-own devices, accounts/profiles,
multiple simultaneous games, multiple maps, map editor, 3D, realistic vehicle
physics, car-to-car collisions, racing, drift, and persistent leaderboards.
See `01-vision-scope/vision-scope.md`.
