# Tech Stack Options — Analysis and Outcome

Status: Draft
Last updated: 2026-10-19

This file preserves the analysis that led to D-22 and ADR-0001. The decision is
recorded there; the options below are context, not open choices.

## Selected Outcome (2026-10-19)

| Area | Selection | Decision |
|------|-----------|----------|
| Server runtime | Go | D-22, ADR-0001 |
| Game/display | Godot 4 | D-22, ADR-0001 |
| Controller | Minimal Kotlin for Android 4.4.2 | D-20, ADR-0002 |
| Transport | WebSocket + JSON | D-22 |
| Simulation owner | Godot (server owns session/input authority) | D-22 |
| Persistence | In-memory only | D-25 |
| Hosting | Self-contained laptop app | D-22 |
| Connection | Auto-connect + device token slot reclaim | D-21 |

The Sprint 0 spike validates compatibility. If it fails, a new dated decision
supersedes the affected part; do not silently change the stack.

## Constraints (from Decided docs)

- `controller-android/` is fixed: Kotlin native, Android tablets only, API 19
  (D-08, D-10, D-20).
- `server/` runs on a screen-connected PC, owns sessions, enforces all
  privileged actions server-side (D-11, D-22, `04-product/requirements.md` NFR-01).
- `game/` is swappable behind a stable server-driven interface (D-11, D-22).
- Local network only, 1–4 players, fixed tablets auto-connect, no internet play
  (D-21).
- No accounts, transient session state only (slots, connection handles,
  game state), discarded at session end (D-12, D-25).
- English only (D-05).
- `server/`, `game/` are not scaffolded yet; no test runner configured
  (`06-development/workflow.md`).

## 1. Server language / runtime (D-17, D-01 — resolved by D-22)

| Option                                                          | Pros                                                                                                                                                                                                               | Cons                                                                                                                                                                                               | Fits when                                                                         |
| --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| A. Kotlin/JVM + Ktor (WebSockets) + Compose for Desktop display | One language across controller + server; share slot/input message models; mature OkHttp ↔ Ktor WebSocket pairing; single Gradle monorepo; Compose for Desktop or JavaFX renders on the PC screen without a browser | JVM install / packaging overhead (jpackage/JAR); team must know Kotlin backend + Ktor; heavier than Go/Node for a tiny server                                                                      | Team wants minimal language sprawl and easiest controller/server contract sharing |
| B. TypeScript + Node.js (ws / Socket.IO)                        | Fastest prototyping; largest WebSocket/QR/mDNS examples; easy hiring; display via browser kiosk, Electron, or Tauri                                                                                                | Second language besides Kotlin; need contract duplication (Kotlin data classes ↔ TypeScript types) or codegen (OpenAPI/JSON Schema); Node single-thread care for game loop (worker threads needed) | Team prioritizes iteration speed and web ecosystem over language unity            |
| C. C# + .NET (ASP.NET Core SignalR / WebSockets)                | First-class Windows PC support, single-file EXE; strong tick-loop and display options (WinForms/WPF/Avalonia/Unity host); good Android WebSocket clients                                                           | Second language; heaviest SDK if the game is not Unity; SignalR adds protocol weight vs plain WebSockets                                                                                           | Team already knows .NET, or Unity is chosen as the game engine                    |
| D. Go (net/http + gorilla/nhooyr WebSockets)                    | Single static binary, trivial PC distribution; excellent concurrency for 1–4 connections + tick loop; low memory                                                                                                   | Second language; smaller Android-contract tooling than Kotlin/TS; display still needs a separate renderer (browser, SDL/Fyne, or engine window)                                                    | Team wants the smallest ops footprint and knows Go                                |

Not recommended for MVP: Python (GIL + packaging friction for a real-time
tick loop on a host PC), Ruby/PHP (poor persistent-connection story).

Test/build if chosen: A — Gradle + JUnit/Kotest; B — npm + Vitest/Jest;
C — dotnet test/xUnit; D — `go test`. Document exact commands in
`06-development/workflow.md` once decided.

## 2. Game engine / game module tech (D-17, D-13 — resolved by D-19/D-22)

The game concept is now a top-down 2D parking time attack (D-19), so this
choice is about rendering, collision, and the server↔game interface shape.

| Option                                                                      | Pros                                                                                                                                                                                     | Cons                                                                                                                                  | Fits when                                                                    |
| --------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| A. Godot 4 (GDScript or C#)                                                 | Open source, light for 2D party games; fast iteration; Windows/Linux export; separate game window driven by the server over localhost (socket, subprocess, or GDExtension/service)       | Two runtimes (server + Godot); interface must be defined (state in, frame/raster out); GDScript is another language unless C# is used | First game is 2D real-time and the team wants an editor without Unity weight |
| B. Unity (C#)                                                               | Best tooling for richer 2D/3D; pairs naturally with a C# server; Asset Store                                                                                                             | Heavyweight, license/overhead for a simple MVP; two runtimes; overkill if the first game is minimal                                   | Team knows Unity or the TBD game needs 3D/physics/polish                     |
| C. Custom lightweight loop in the server language (no engine)               | No second runtime; game is a plain module behind `GameInterface.update(inputs, dt) → state`; rendering via Compose Desktop / Avalonia / HTML Canvas; fastest path to "one playable game" | No editor/physics/particles; custom work if the game grows beyond simple 2D                                                           | MVP-first: unknown simple game, keep swap interface small                    |
| D. Web rendering (Canvas/WebGL, e.g. Phaser, plain TS) served by the server | Big-screen output is just a kiosk browser; easy effects; same tech if server is Node                                                                                                     | Still needs a game loop home (server or browser); browser kiosk management on the PC                                                  | Display must be portable and the team prefers web graphics                   |

Interface note (FR-S05): regardless of engine, keep the contract narrow and
engine-agnostic, e.g. `init(config) / update(slotInputs, dt) → gameState /
renderSnapshot / reset`. JSON is adopted for MVP; binary is only a post-MVP
optimisation if measurements require it (D-22, D-23).

## 3. Controller-to-server transport (D-17, D-16 — resolved by D-22/D-23)

| Option                                                | Pros                                                                                                                                                                             | Cons                                                                                                                 | Fits when                                                               |
| ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| A. WebSockets + JSON (upgrade to CBOR/Protobuf later) | Simple over shared WiFi; works on all server options; OkHttp on Android ↔ Ktor/Node/.NET/Go servers; easy slot-auth, kick, heartbeat; enough for 1–4 players at party-game rates | TCP head-of-line blocking; not the lowest possible latency; needs app-level sequence/ack for lost inputs             | Selected for MVP (D-22); revisit only if measured latency requires it |
| B. WebRTC DataChannel                                 | Low-latency, NAT-friendly; built-in encryption                                                                                                                                   | Needs signaling (still WebSockets/HTTP) + STUN/TURN complexity; overkill for same-WiFi MVP; harder on Android native | Only if later play goes beyond local WiFi                               |
| C. Raw UDP / ENet / KCP (custom reliability)          | Lowest latency, full tick control                                                                                                                                                | Must reimplement ordering, reliability, reconnect, and slot-auth; firewall/WiFi quirks; most custom code             | Only if measured latency (D-23) demands it and the team can own netcode |
| D. HTTP polling / SSE only                            | Trivial to debug                                                                                                                                                                 | Wrong shape for bidirectional real-time input; latency/jitter worse than WebSockets                                  | Not recommended; useful only as fallback join/lobby channel             |

Connection/join (FR-C01/FR-S01) is separate from transport: the four fixed
tablets auto-connect to the configured server address and reclaim slots by
device token (D-21). QR or manual address entry is only a setup fallback.
mDNS (`_proszkowska._tcp`) is optional convenience and is not part of the MVP
connection model.

Adopted MVP pairing: WebSocket + JSON, 20–30 Hz input, 60 FPS display, target
≤100 ms input-to-visible response (D-22, D-23).

## 4. Database / persistence (D-01, constrained by D-12)

No accounts and no retained personal data are Decided. Viable options:

| Option                                                  | Pros                                                      | Cons                                              | Fits when                        |
| ------------------------------------------------------- | --------------------------------------------------------- | ------------------------------------------------- | -------------------------------- |
| A. In-memory only, discard at session end (selected) | Matches D-12 and D-25; zero ops; no migration/GDPR surface | No history/stats across sessions | MVP default (D-25) |
| B. SQLite file for local diagnostics only (optional)    | Zero-service crash logs / score history on the host PC    | Must prove no personal data is stored; extra code | Team explicitly wants local logs |
| C. No embedded DB + flat JSON log (optional)            | Simplest inspectable diagnostics                          | Same privacy care as B; no querying               | Light alternative to B           |

No server-side database (Postgres/MySQL/Mongo) and no cloud store are needed
for MVP. If proposed later, they require a new ADR against D-12.

## 5. Hosting / distribution / display PC (D-01)

Local-only MVP: no cloud host, no container orchestrator.

| Option                                                                                                                      | Pros                                                              | Cons                                                                           |
| --------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| A. Self-contained PC app (jpackage EXE/MSI, .NET single-file, Go binary, or Node pkg) + fullscreen window on the big screen | One artifact the host double-clicks; works offline on shared WiFi | Team owns installer/updater per OS                                             |
| B. Docker image for the server                                                                                              | Reproducible env                                                  | Docker on a presentation PC is friction for MVP; display forwarding is awkward |
| C. Cloud relay / internet hosting                                                                                           | —                                                                 | Explicit non-goal for MVP (D-09); do not choose                                |

## 6. Outcome (2026-10-19)

1. Server: Go (Section 1, option D). Chosen for the team's strongest backend
   skill, simple single-binary deployment, and easy supervision of the game.
2. Game/display: Godot 4 (Section 2, option A). Chosen for 2D rendering,
   collisions, scene editing, and fast iteration.
3. Transport: WebSocket + JSON (Section 3, option A). Revisit only if the
   Sprint 0 latency measurement misses D-23.
4. Persistence: in-memory only (Section 4, option A), matching D-12/D-25.
5. Hosting: self-contained laptop app (Section 5, option A): Go binary + Godot
   export + supervisor/launcher.
6. Controller: minimal Kotlin app for Android 4.4.2 (ADR-0002).
7. The Sprint 0 spike validates the stack and the controller toolchain. A
   failure produces a new dated decision, not a silent change.

## Open questions

- Controller input scheme and per-controller feedback (D-14) — resolve by
  playtesting buttons, slider, and virtual joystick on the real tablets.
- Exact map layout and tuning values — `02-game-domain/game-design.md` is
  `Draft` and will be updated after playtests.
- Festival date, laptop/TV specifications, and budget — needed for the delivery
  roadmap and operations plan.
- Disconnect/rejoin/late-join/host-crash handling (D-15).
