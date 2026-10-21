# Game Domain Overview

Status: Decided
Last updated: 2026-10-21

## Concept

Proszkowska Drift is a top-down 2D parking time attack. Players drive around a
fixed parking lot, receive a sequence of target spots, and score points for
completing each task quickly, precisely, and without hitting the environment.
Players compete independently on the same visible map; there are no car-to-car
collisions. The detailed rules and tuning values live in
`02-game-domain/game-design.md` (D-19).

The MVP deliberately contains one core task: park in the assigned spot. Parking
task variants are expressed through spot shape, tightness, and required facing,
not through separate game systems. Drift and racing are post-MVP.

## System Shape (Decided)

Three separate parts in one monorepo, plus the display:

- `controller-android/` — minimal Kotlin native Android app on the tablets.
  Sends input and displays task/score/time. No game rendering. Constrained by
  Android 4.4.2 (D-20).
- `server/` — Go process on the laptop. Owns sessions, player slots, input
  validation, heartbeats, reconnects, host commands, and game process
  supervision (D-22).
- `game/` — Godot 4 game module. Owns the simulation, collisions, parking
  detection, scoring, HUD, attract mode, and fullscreen display (D-22).
- `Display / Screen` — TV or monitor connected to the laptop. Shows the game,
  scores, and results.

The server and game are separate processes. The server is authoritative for who
may send input and for session actions; the game is the trusted local simulation
and scoring module. The boundary is a WebSocket JSON protocol documented in
`06-development/protocol.md`.

## Session Lifecycle (Decided)

1. `Server starting` — the server loads configuration and, when supervision is
   configured, starts the game process.
2. `Attract` — display shows the title, instructions, and a prompt to start.
3. `Ready` — tablets connect automatically, register their device tokens, and
   receive slots and colors. The host sees the connected player count.
4. `Countdown` — host starts a round; display counts down 3–2–1.
5. `Playing` — three-minute round. Inputs are validated and forwarded to the
   game; the game simulates, scores, and renders.
6. `Results` — scores, winner, and per-player task summary for about 15 seconds.
7. `Next round or end` — host starts the next round or ends the session.

State transitions must be explicit in code. No silent transitions.

## Edge Cases (Decided)

- A tablet disconnects mid-round: its car stops, the controller shows
  "Reconnecting", and the same slot can be reclaimed automatically. If it
  reconnects during the round, control resumes.
- All tablets disconnect: the round pauses after a short grace period and the
  host can resume or reset.
- Late join: allowed between rounds; not in the middle of a running round.
- Game process crash: the server restarts it, returns to `Attract`, and the
  current round is lost. Tablets reconnect automatically.
- Server crash: the festival launcher (M3) restarts the server; the Go
  supervisor restarts the game. Tablets reconnect and the current round is
  lost.
- Host pause/resume/reset is always explicit and shown on the display.
- A player who finishes all tasks waits for the round to end; the display shows
  a finished state.

Detailed reconnect and host-crash rules are in D-24 and
`03-players-sessions/roles-sessions.md`.

## Real-Time Constraints (Decided)

- Display: target 60 FPS.
- Controller input: 20–30 Hz while a round is active.
- Target input-to-visible-response latency: at most 100 ms on the local network.
- Transport: WebSocket + JSON for MVP. Binary encoding is a post-MVP
  optimisation if measurements require it (D-22, D-23).
- No client-side prediction in MVP. The game applies the latest validated input
  every frame; the controller does not simulate the car.

Numbers are targets, not measured guarantees. The Sprint 0 spike must measure
them on the real tablets, router, laptop, and TV. If the target is not reachable,
record a new decision before changing the architecture.
