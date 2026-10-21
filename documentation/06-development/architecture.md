# Runtime Architecture

Status: Decided
Last updated: 2026-10-21

This document describes the MVP runtime. Decisions are recorded in D-22,
ADR-0001, and ADR-0002. Message schemas are in `06-development/protocol.md`.

## Overview

```text
┌──────────────────────┐        WebSocket + JSON        ┌──────────────────────┐
│ 4 Android tablets    │  ws://<laptop-ip>:8080/ws/...   │ Go server            │
│ minimal Kotlin app   │ ─────────────────────────────► │ session, slots,      │
│ input + feedback     │ ◄───────────────────────────── │ validation, host,    │
└──────────────────────┘                                │ supervision          │
                                                        └──────────┬───────────┘
                                                                   │ localhost
                                                                   │ WebSocket + JSON
                                                        ┌──────────▼───────────┐
                                                        │ Godot 4 game         │
                                                        │ map, physics, score, │
                                                        │ HUD, fullscreen      │
                                                        └──────────┬───────────┘
                                                                   │ HDMI
                                                        ┌──────────▼───────────┐
                                                        │ TV / monitor         │
                                                        └──────────────────────┘
```

The host uses keyboard shortcuts in the Godot window. Godot sends host requests
to the server; the server validates them and broadcasts the resulting command.
The game never accepts controller input directly, and controllers never render
or simulate the game.

## Components

| Component | Process | Responsibilities |
|-----------|---------|------------------|
| Controller | Kotlin app on each tablet | Connect, authenticate with device token, send input, display task/score/time/state, reconnect |
| Server | Go binary on laptop | Session state, slots, colors, input validation, heartbeats, reconnect, host authority, game supervision, logs. Sprint 0 slice implemented in `server/`; supervision is later work |
| Game | Godot 4 on laptop | Map, car movement, environment collisions, parking detection, scoring, HUD, attract/results, fullscreen display |
| Display | TV/monitor | Shows the game only; no host UI in MVP |
| Router/hotspot | Dedicated local network | Connects tablets and laptop; no internet dependency |

## Sprint 0 Server Slice

Issue #12 scaffolded the server on 2026-10-21. It is a Go 1.23 module,
`github.com/PSZLBit/ProszkowskaDrift/server`, using
`github.com/coder/websocket` v1.8.15. The library choice is provisional until
the Sprint 0 spikes finish.

- `/ws/controller` accepts controller connections on the configured listen
  address (default `:8080`).
- `/ws/game` accepts only loopback connections and requires the per-launch game
  token as `?token=<value>` or the `X-Game-Token` header.
- The server owns slots, colors, device-token reconnect, input range/sequence/
  rate validation, controller heartbeats, and a minimal phase path for
  `start`, `pause`, `resume`, `reset`, `end`, and `kick`.
- A server tick loop (250 ms) enforces the controller idle timeout, the
  all-disconnected pause (10 s grace), and game heartbeat monitoring. Writing
  is serialized per connection through a bounded queue; a connection that
  stops draining its queue is closed as `slow_client` instead of blocking
  broadcasts.
- Session boundaries are enforced server-side: `host_denied` answers rejected
  host requests, a kicked device token is blocked for the session, host `end`
  clears slots and closes controllers gracefully, and `/healthz` reports
  phase, slot count, and game readiness.
- Session state is in memory. Device tokens are never forwarded to the game.
- `cmd/fakecontroller` and `cmd/fakegame` provide manual end-to-end checks. The
  protocol details and current deviations are in `06-development/protocol.md`.

This slice does not launch or supervise Godot (the game heartbeat closes a
silent game connection already; launching, token rotation per launch, and
restarts are S0-07), assign tasks, score, implement late join, or provide a
production controller UI. Those remain later sprint items.

## Authority and Trust Boundary

- **Server authority:** who may connect, which slot they own, whether input is
  valid, whether a round may start/pause/reset/end, and whether a slot is kicked.
- **Game authority:** simulation, collision response, parking detection, task
  assignment, scoring, and rendering. The game is a trusted local process; this
  is a deliberate trade-off for MVP speed (ADR-0001).
- **Controller:** untrusted input source. The server validates slot, ranges,
  rate, and sequence. The controller UI is never an authorization boundary.
- No remote clients, no internet exposure, no account system. The laptop is the
  trust anchor.

## Process Model and Supervision

1. The host starts the server (Go binary or festival launcher).
2. The server loads configuration: port, bind address, Godot executable path,
   round defaults, log path.
3. The server starts Godot with a per-launch game token and the local WebSocket
   URL.
4. Godot connects to `/ws/game`, sends `game_ready`, and enters `Attract`.
5. Tablets connect to `/ws/controller`, send `hello`, and receive slots.
6. The host starts a round from Godot. Godot sends `host_request`; the server
   validates and broadcasts `host_command`.
7. During play, controllers send input to the server; the server validates and
   forwards it to Godot. Godot sends task/score/phase updates back through the
   server.
8. On game exit or crash, the server restarts Godot and returns to `Attract`.
   The current round is lost.
9. On server exit, the festival launcher restarts the server and game. Tablets
   reconnect automatically.

Development mode may run Go and Godot in separate terminals; the festival mode
must use the supervisor/launcher so a crash does not require a developer. The
Sprint 0 server slice implements the server side of steps 4–7 with fake clients;
launching and supervising Godot (steps 2–3 and 8–9) is not implemented yet.

## Data Flow

```text
Controller --input(seq, steer, throttle)--> Server
Server --validated input(slot, seq, steer, throttle)--> Game
Game --task/score/phase feedback--> Server
Server --task/score/phase--> Controller
Game --fullscreen render--> Display
Host keyboard --> Game --host_request--> Server --host_command--> Game
```

- Controllers send input at 20–30 Hz while `Playing`.
- The game applies the latest input per player each physics frame.
- No client-side prediction in MVP.
- The game sends feedback on change (task assigned, task result, phase change),
  not a full state snapshot every frame. This keeps the protocol small.
- The server does not need to understand map geometry or scoring; it routes and
  validates.

## Network Topology

- Dedicated Wi-Fi router or laptop hotspot. Do not rely on venue Wi-Fi.
- Laptop uses a stable local address (static IP or DHCP reservation).
- Tablets use DHCP reservations or a fixed address pool.
- 2.4 GHz is the compatibility baseline for the old tablets; use 5 GHz only if
  all four tablets support it.
- Disable AP/client isolation on the router.
- No internet required. If the router provides internet, it is irrelevant and
  should not be required by the game.
- Allow the server port (default `8080`) through the laptop firewall on the
  private network only.

## Persistence and Privacy

- All session and score state is in memory (D-25).
- No database, no accounts, no player names, no photos.
- Device tokens are random identifiers stored on the tablet; do not use Android
  ID, serial number, or account data.
- Logs are local diagnostics only. Do not log personal data; truncate tokens.
- Optional event scores are anonymous and in-memory; clear them between groups.

## Failure and Recovery

| Failure | Behavior |
|---------|----------|
| Tablet disconnects | Car stops; slot reserved; controller reconnects and resumes if the round is still running |
| All tablets disconnect | Round pauses after 10 s; host resumes, resets, or ends |
| Game crashes | Server restarts Godot; returns to `Attract`; current round lost |
| Server crashes | Launcher restarts server and game; tablets reconnect; current round lost |
| Router/Wi-Fi fails | No play until network returns; use the runbook's hotspot fallback |
| TV/monitor fails | Switch to the backup monitor; game keeps running |
| Tablet battery dies | Keep tablets plugged in; replace with spare if available |

## Packaging and Deployment

- `server/`: Go binary, config file, logs directory.
- `game/`: Godot export (Windows and/or Linux, decided by the laptop OS).
- `server/ops/`: launch script or small supervisor that starts the server,
  waits for health, starts/stops the game, and restarts on failure. It stays
  inside `server/` to respect the D-07 layout.
- Development: separate commands for Go and Godot, documented after scaffolding.
- Festival: one folder, one launch action, fullscreen, no developer tools.
- Version and build info visible in a diagnostics screen/log, not in the player
  UI.

## Performance Budget

| Metric | Target | Measured in |
|--------|--------|-------------|
| Display frame rate | 60 FPS | Sprint 0 and playtests |
| Controller input rate | 20–30 Hz | Sprint 0 |
| Input-to-visible response | ≤100 ms | Sprint 0 |
| Players | 4 | Sprint 0 and rehearsal |
| Setup time | <15 min | Festival rehearsal |

If the budget is missed, record a new decision before changing transport,
engine, or architecture. Do not silently add prediction, UDP, or a new runtime.

## Open Architecture Items

- Final confirmation of the Go WebSocket library after the Sprint 0 spikes.
  The S0-04 slice uses `github.com/coder/websocket` v1.8.15; the Android
  WebSocket library is still open.
- Exact process supervisor implementation (Go or shell script).
- Windows vs Linux festival laptop.
- Router model and static-IP plan.
- Map data format and Godot scene structure.
- Whether the host has a second control device; web host panel is post-MVP.
