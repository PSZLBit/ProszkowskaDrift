# Proszkowska Drift

Working title for a local multiplayer arcade parking game. One to four players
use Android tablets as controllers; a laptop runs the game and shows it on a TV
or monitor. Each round is a short parking time attack on a stylized map of the
campus area on Proszkowska Street.

The project is built by a student team for a science-club festival stand. It is
local-only, account-free, and designed to run without internet access.

## Game in One Paragraph

Players drive around a fixed top-down parking lot, receive a sequence of target
spots, and score points for completing each parking task quickly, precisely, and
without hitting the environment. Players compete independently on the same map;
cars do not collide with each other. Rounds last three minutes. The whole map,
all players, and the score are visible on the big screen at once.

Drift and a racing mode are post-MVP ideas. The MVP is the parking core.

## Components

- `server/` — Go session server: slots, input validation, reconnects, host
  commands, session timers, and recovery signals. Sprint 0 slice: controller
  and game WebSocket endpoints, in-memory slots, validation, all-disconnected
  pause, `/healthz`, and fake clients. Game process supervision is not
  implemented yet. See [`server/README.md`](server/README.md).
- `game/` — Godot 4 game and fullscreen display: map, movement, collisions,
  parking detection, scoring, HUD. (Scaffolded.)
- `controller-android/` — minimal Kotlin Android controller for Android 4.4.2.
  (Planned; not scaffolded.)
- `documentation/` — source of truth: vision, game design, requirements,
  decisions, architecture, protocol, testing, roadmap, festival runbook.

## Technology Direction

- Go server, Godot 4 game, minimal Kotlin controller.
- WebSocket + JSON for controller↔server and server↔game.
- In-memory state only: no database, accounts, or personal data.
- Android 4.4.2 (API 19) compatibility: no Compose, no OkHttp 4, and no
  AndroidX unless a library is proven API 19 compatible.
- Display: fixed full-map view, 1920×1080 reference, 60 FPS target.
- Input: 20–30 Hz, target ≤100 ms input-to-visible response.

These choices are recorded in
[`documentation/05-architecture-decisions/adr-0001-mvp-architecture.md`](documentation/05-architecture-decisions/adr-0001-mvp-architecture.md)
and [`adr-0002-legacy-android-controller.md`](documentation/05-architecture-decisions/adr-0002-legacy-android-controller.md).

## Project Status

Planning and Sprint 0. The Go session server slice exists: controller and game
WebSocket endpoints, four slots, input validation, reconnects, heartbeats, the
minimal host command path, and fake client tools. Godot, the Android controller,
scoring, and game supervision are not implemented yet. The festival is
2026-11-24 and the internal festival-ready target is 2026-11-18. The committed
scope is the Minimum Festival Demo (MFD); the full MVP is a stretch goal.

Work board: <https://github.com/orgs/PSZLBit/projects/1>.

See [`documentation/06-development/roadmap.md`](documentation/06-development/roadmap.md).

## Getting Started

### Prerequisites

- Git. On Windows, install Git for Windows. Git Bash is optional for setup; the
  `pre-push` hook runs through Git's bundled shell.
- Go 1.23 or newer for the `server/` component.

Godot and the Android SDK are not required yet; their setup steps will be added
when those components are scaffolded.

### Clone and set up

```sh
git clone https://github.com/PSZLBit/ProszkowskaDrift.git
cd ProszkowskaDrift
```

Then run the small, idempotent setup script for your platform. It sets the
repository-local `core.hooksPath` to `.githooks` and installs the `pre-push` hook
that blocks direct pushes to `main`.

Linux, macOS, or Git Bash:

```sh
./scripts/setup.sh
```

Windows PowerShell (the command uses a one-off execution-policy bypass):

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\setup.ps1
```

Windows Git Bash users can also run `sh scripts/setup.sh`. No general task
runner is required; extend the scripts only when more setup steps are needed.

Verify the hook path:

```sh
git config --local --get core.hooksPath
# .githooks
```

The local hook catches accidental direct pushes. It is not a security boundary:
GitHub rules for `main` must still require a pull request and review.

For a local documentation preview, follow the pinned toolchain and commands in
[`documentation/06-development/testing.md`](documentation/06-development/testing.md#documentation-site-checks).

## Documentation

Published site: <https://pszlbit.github.io/ProszkowskaDrift/>.

Start with [`documentation/README.md`](documentation/README.md), then read:

- [`documentation/01-vision-scope/vision-scope.md`](documentation/01-vision-scope/vision-scope.md)
- [`documentation/02-game-domain/game-design.md`](documentation/02-game-domain/game-design.md)
- [`documentation/04-product/requirements.md`](documentation/04-product/requirements.md)
- [`documentation/05-architecture-decisions/decision-log.md`](documentation/05-architecture-decisions/decision-log.md)
- [`documentation/06-development/architecture.md`](documentation/06-development/architecture.md)
- [`documentation/06-development/protocol.md`](documentation/06-development/protocol.md)
- [`documentation/06-development/workflow.md`](documentation/06-development/workflow.md)

## Repository History

The repository started as a Flanki tournament-management web app. On
2026-10-16 the team pivoted to the local real-time play system (D-06). The old
concept is superseded; the decision history is kept in the decision log.

## License

To be determined.
