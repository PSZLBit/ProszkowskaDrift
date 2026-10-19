# Testing Strategy

Status: Draft
Last updated: 2026-10-19

## Goals

- Protect the game rules that affect fairness and fun: task assignment,
  parking detection, scoring, and round flow.
- Protect the session boundary: slots, input validation, reconnects, and host
  authority.
- Prove the system works on the real Android 4.4.2 tablets, target laptop, TV,
  router, and festival-like conditions.
- Keep tests cheap enough for a four-person team and free-tier CI.

## Test Layers

| Layer | What | Where | When |
|-------|------|-------|------|
| Unit | Game rules, scoring, task assignment, session/slot logic, validation | `game/`, `server/` | Every PR |
| Contract | Protocol parsing, versioning, error codes | `server/` and a fake game/controller | Every PR |
| Integration | Go server + fake game + four fake controllers; reconnect; host commands | `server/` | Every PR or nightly |
| Android build | Kotlin app compiles and APK is produced | CI | Every PR |
| Hardware | Install, touch, Wi-Fi, latency, charging, kiosk behavior | Real tablets | Sprint 0 and every controller change |
| Manual game | Map readability, controls, scoring feel, fun | Real laptop + TV + 4 tablets | Every playable increment |
| Festival rehearsal | Full setup, queue, failure drills | Real venue-like setup | Before the event |

## Go Server Tests

Planned packages and coverage:

- Session: create session, assign 1–4 slots, reject a full session, duplicate
  device token, kick/reset, session end clears state.
- Input validation: ranges, NaN/Infinity, rate limit, stale sequence, wrong slot,
  wrong round phase, kicked slot.
- Reconnect: heartbeat timeout, same token reclaims slot, unknown token
  rejected, all-disconnected pause trigger.
- Host commands: start/pause/resume/reset/end/kick authorization and
  broadcast behavior.
- Protocol: envelope version, unknown type, malformed JSON, error codes.
- Game supervision: start, crash/restart, missing heartbeat, game token
  mismatch.

Use table-driven tests and `go test -race ./...`. Do not require a real network
for unit tests; use an in-memory transport or `httptest` where practical.

## Godot Tests

Use a lightweight test addon (GUT or a small headless script runner) for pure
logic that can be separated from scenes:

- Time score decay and floor.
- Precision score for center offset and angle, including boundary values.
- Collision penalty cap.
- Finish bonus and cap.
- Task assignment: unique active targets, no repeats, pool return.
- Parking detection: all corners inside, speed threshold, 0.5 s hold.
- Round state transitions: ready → countdown → playing → results.
- Tie-breakers.
- Auto-skip at 45 s.

Scene-level smoke tests may verify that the main scene loads headless and the
map/spawn/target nodes exist. Visual quality is a manual test.

## Contract and Integration Tests

- A fake game client connects to the Go server, receives `config`, sends
  `game_ready`, and exchanges `host_command`/`phase` messages.
- Four fake controllers connect, receive slots, send input at 30 Hz, and verify
  that the server forwards only validated input.
- Reconnect test: drop one controller, reconnect with the same token, verify the
  same slot and resumed input.
- Host test: a controller cannot start/pause/reset; the server rejects it.
- Latency test: timestamp a round trip on the local network; record p50/p95.

## Android 4.4.2 Hardware Matrix

Run on all four actual tablets, not only one.

| Check | Pass condition |
|-------|----------------|
| Install APK | Installs and launches on API 19 |
| Boot behavior | App starts after reboot or is the default home |
| Touch | Buttons respond; multi-touch works; no stuck touches |
| Screen | Landscape, immersive, readable at arm's length |
| Keep awake | Screen stays on while charging for 30 minutes |
| Wi-Fi | Connects to the dedicated router; no drops in a 30-minute idle test |
| Reconnect | Wi-Fi off/on or app restart reclaims the same slot |
| Latency | Input-to-display ≤100 ms target; record actual values |
| Charging | Tablet charges while the app is running and screen is on |
| Heat/battery | No shutdown or severe throttling in a 30-minute session |

## Manual Game Tests

- Four-player round: all players can see their target, park, and score.
- One/two/three-player rounds: empty slots do not break the round.
- Target readability from the intended spectator distance.
- Target assignment is exclusive and understandable.
- Parking feels fair: a clearly correct park scores; a sloppy park scores lower.
- Collisions are noticeable but not punishing enough to ruin the round.
- Results screen shows the winner and scores correctly.
- Attract mode is clear and the host can start.
- Pause, reset, kick, and end behave as documented.
- No car-to-car collision; overlapping cars remain visually distinguishable.
- A finished player's state is clear.

## CI Plan

Free-tier GitHub Actions, planned after scaffolding:

- `server`: `go test -race ./...`, `go vet ./...`, build.
- `controller-android`: Gradle `assembleDebug`; cache Gradle; upload APK artifact.
- `game`: Godot headless tests where feasible; export is manual/tagged if CI is
  too slow.
- Docs: optional Markdown link check; keep it fast.
- Required for merge: Go tests/vet and Android build. Godot headless tests are
  required once configured.

Exact commands will be added to this file and `README.md` after the Sprint 0
scaffold. Until then, report checks that could not run.

## Festival Rehearsal

Before the event, run the full checklist from
`06-development/festival-operations.md`:

1. Cold start: power, router, laptop, TV, tablets, server, game.
2. Four-player round; measure setup and round turnover.
3. Disconnect one tablet mid-round; reconnect.
4. Kill the game process; verify auto-restart and reconnect.
5. Kill the server; verify supervisor restart and tablet reconnect.
6. Simulate router failure; execute the hotspot fallback.
7. Run for 30–60 minutes continuously; watch heat, battery, memory, logs.
8. Reset for the next group; confirm no personal data or scores leak between
   groups.

## Bug Triage

- `P0`: festival/demo cannot run; data loss; crash loop. Fix immediately.
- `P1`: core round or four-player flow broken; workaround is painful. Fix before
  the next demo.
- `P2`: polish, rare edge case, non-blocking visual issue. Schedule.
- `P3`: nice-to-have or post-MVP.

Every bug issue includes steps to reproduce, expected/actual, build/commit,
tablet model, and whether it reproduces on hardware.

## Privacy in Tests

Use fake tokens, fake slot labels, and no real names. Do not commit logs or
screenshots containing personal data. Session test data is ephemeral.
