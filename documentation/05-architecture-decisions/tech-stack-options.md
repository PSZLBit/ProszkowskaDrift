# Tech Stack Options — Analysis for Team Choice

Status: Draft
Last updated: 2026-10-16

Analysis-only document. No decision is made here.
Blocks implementation until the team moves items to `Decided`:
`D-01` (stack, database, hosting), `D-17` (server language, game engine,
transport), `D-16` (latency target, tick rate).
See `05-architecture-decisions/decision-log.md`.
Do not implement from this file.

## Constraints (from Decided docs)

- `controller-android/` is fixed: Kotlin native, Android tablets only (D-08, D-10).
- `server/` runs on a screen-connected PC, owns sessions, enforces all
  privileged actions server-side (D-11, `04-product/requirements.md` NFR-01).
- `game/` is swappable behind a stable server-driven interface (D-11, FR-S05).
- Local network only, 1–4 players, lobby code/QR join, no internet play (D-09).
- No accounts, transient session state only (slots, connection handles,
  game state), discarded at session end (D-12).
- English only (D-05).
- `server/`, `game/` are not scaffolded yet; no test runner configured
  (`06-development/workflow.md`).

## 1. Server language / runtime (D-17, D-01)

| Option | Pros | Cons | Fits when |
|--------|------|------|-----------|
| A. Kotlin/JVM + Ktor (WebSockets) + Compose for Desktop display | One language across controller + server; share slot/input message models; mature OkHttp ↔ Ktor WebSocket pairing; single Gradle monorepo; Compose for Desktop or JavaFX renders on the PC screen without a browser | JVM install / packaging overhead (jpackage/JAR); team must know Kotlin backend + Ktor; heavier than Go/Node for a tiny server | Team wants minimal language sprawl and easiest controller/server contract sharing |
| B. TypeScript + Node.js (ws / Socket.IO) | Fastest prototyping; largest WebSocket/QR/mDNS examples; easy hiring; display via browser kiosk, Electron, or Tauri | Second language besides Kotlin; need contract duplication (Kotlin data classes ↔ TypeScript types) or codegen (OpenAPI/JSON Schema); Node single-thread care for game loop (worker threads needed) | Team prioritizes iteration speed and web ecosystem over language unity |
| C. C# + .NET (ASP.NET Core SignalR / WebSockets) | First-class Windows PC support, single-file EXE; strong tick-loop and display options (WinForms/WPF/Avalonia/Unity host); good Android WebSocket clients | Second language; heaviest SDK if the game is not Unity; SignalR adds protocol weight vs plain WebSockets | Team already knows .NET, or Unity is chosen as the game engine |
| D. Go (net/http + gorilla/nhooyr WebSockets) | Single static binary, trivial PC distribution; excellent concurrency for 1–4 connections + tick loop; low memory | Second language; smaller Android-contract tooling than Kotlin/TS; display still needs a separate renderer (browser, SDL/Fyne, or engine window) | Team wants the smallest ops footprint and knows Go |

Not recommended for MVP: Python (GIL + packaging friction for a real-time
tick loop on a host PC), Ruby/PHP (poor persistent-connection story).

Test/build if chosen: A — Gradle + JUnit/Kotest; B — npm + Vitest/Jest;
C — dotnet test/xUnit; D — `go test`. Document exact commands in
`06-development/workflow.md` once decided.

## 2. Game engine / game module tech (D-17, D-13, FR-G01, FR-S05)

The game concept is TBD, so this choice is about rendering + loop + the
server↔game interface shape.

| Option | Pros | Cons | Fits when |
|--------|------|------|-----------|
| A. Godot 4 (GDScript or C#) | Open source, light for 2D party games; fast iteration; Windows/Linux export; separate game window driven by the server over localhost (socket, subprocess, or GDExtension/service) | Two runtimes (server + Godot); interface must be defined (state in, frame/raster out); GDScript is another language unless C# is used | First game is 2D real-time and the team wants an editor without Unity weight |
| B. Unity (C#) | Best tooling for richer 2D/3D; pairs naturally with a C# server; Asset Store | Heavyweight, license/overhead for a simple MVP; two runtimes; overkill if the first game is minimal | Team knows Unity or the TBD game needs 3D/physics/polish |
| C. Custom lightweight loop in the server language (no engine) | No second runtime; game is a plain module behind `GameInterface.update(inputs, dt) → state`; rendering via Compose Desktop / Avalonia / HTML Canvas; fastest path to "one playable game" | No editor/physics/particles; custom work if the game grows beyond simple 2D | MVP-first: unknown simple game, keep swap interface small |
| D. Web rendering (Canvas/WebGL, e.g. Phaser, plain TS) served by the server | Big-screen output is just a kiosk browser; easy effects; same tech if server is Node | Still needs a game loop home (server or browser); browser kiosk management on the PC | Display must be portable and the team prefers web graphics |

Interface note (FR-S05): regardless of engine, keep the contract narrow and
engine-agnostic, e.g. `init(config) / update(slotInputs, dt) → gameState /
renderSnapshot / reset`. Decide binary vs JSON snapshot when D-16 is set.

## 3. Controller-to-server transport (D-17, D-16, FR-C02/FR-S03)

| Option | Pros | Cons | Fits when |
|--------|------|------|-----------|
| A. WebSockets + JSON (upgrade to CBOR/Protobuf later) | Simple over shared WiFi; works on all server options; OkHttp on Android ↔ Ktor/Node/.NET/Go servers; easy slot-auth, kick, heartbeat; enough for 1–4 players at party-game rates | TCP head-of-line blocking; not the lowest possible latency; needs app-level sequence/ack for lost inputs | Default for MVP; choose unless latency tests prove otherwise |
| B. WebRTC DataChannel | Low-latency, NAT-friendly; built-in encryption | Needs signaling (still WebSockets/HTTP) + STUN/TURN complexity; overkill for same-WiFi MVP; harder on Android native | Only if later play goes beyond local WiFi |
| C. Raw UDP / ENet / KCP (custom reliability) | Lowest latency, full tick control | Must reimplement ordering, reliability, reconnect, and slot-auth; firewall/WiFi quirks; most custom code | Only if a latency target (D-16) demands it and the team can own netcode |
| D. HTTP polling / SSE only | Trivial to debug | Wrong shape for bidirectional real-time input; latency/jitter worse than WebSockets | Not recommended; useful only as fallback join/lobby channel |

Discovery/join (FR-C01/FR-S01) is orthogonal: QR encodes
`ws://<pc-ip>:<port>/join?code=XXXX` (or mDNS name); manual code entry as
fallback. mDNS (`_flanki._tcp`) is optional convenience, not transport.

Recommended MVP pairing: WebSockets + JSON, 20–30 Hz input send with
server tick to be set under D-16, binary encoding only if measured need.

## 4. Database / persistence (D-01, constrained by D-12)

No accounts and no retained personal data are Decided. Viable options:

| Option | Pros | Cons | Fits when |
|--------|------|------|-----------|
| A. In-memory only, discard at session end (recommended) | Matches D-12 exactly; zero ops; no migration/GDPR surface | No history/stats across sessions | MVP default |
| B. SQLite file for local diagnostics only (optional) | Zero-service crash logs / score history on the host PC | Must prove no personal data is stored; extra code | Team explicitly wants local logs |
| C. No embedded DB + flat JSON log (optional) | Simplest inspectable diagnostics | Same privacy care as B; no querying | Light alternative to B |

No server-side database (Postgres/MySQL/Mongo) and no cloud store are needed
for MVP. If proposed later, they require a new ADR against D-12.

## 5. Hosting / distribution / display PC (D-01)

Local-only MVP: no cloud host, no container orchestrator.

| Option | Pros | Cons |
|--------|------|------|
| A. Self-contained PC app (jpackage EXE/MSI, .NET single-file, Go binary, or Node pkg) + fullscreen window on the big screen | One artifact the host double-clicks; works offline on shared WiFi | Team owns installer/updater per OS |
| B. Docker image for the server | Reproducible env | Docker on a presentation PC is friction for MVP; display forwarding is awkward |
| C. Cloud relay / internet hosting | — | Explicit non-goal for MVP (D-09); do not choose |

## 6. Suggested decision path for the team

1. Pick server language (Section 1): Kotlin+Ktor if language unity matters,
   Node+TS if prototype speed matters, .NET if Unity/C# skill matters.
2. Pick game tech (Section 2): custom loop for fastest MVP, Godot 4 for
   2D editor support, Unity only if game scope demands it.
3. Confirm transport (Section 3): WebSockets+JSON default; set numeric
   latency target and tick rate to close D-16.
4. Confirm persistence (Section 4): in-memory only unless diagnostics file
   is explicitly wanted.
5. Record each as `Decided` rows in `decision-log.md` with ADR links, then
   scaffold `server/`, `game/`, and the `GameInterface` (FR-S05).

## Open questions left for ADRs

- Numeric latency budget and tick/update rate (D-16).
- Input set and per-controller feedback (D-14).
- Game concept/rules/win condition (D-13).
- Disconnect/rejoin/late-join/host-crash handling (D-15).
