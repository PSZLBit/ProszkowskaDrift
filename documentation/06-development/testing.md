# Testing Strategy

Status: Draft
Last updated: 2026-10-21

## Goals

- Protect the game rules that affect fairness and fun: task assignment,
  parking detection, scoring, and round flow.
- Protect the session boundary: slots, input validation, reconnects, and host
  authority.
- Prove the system works on the real Android 4.4.2 tablets, target laptop, TV,
  router, and festival-like conditions.
- Keep tests cheap enough for a four-person team and free-tier CI.

## Test-First Practice

- **Pure logic:** write the test first for game rules, scoring, task
  assignment, session/slot logic, input validation, and protocol parsing.
  Red → green → refactor.
- **UI, visual, and hardware behavior:** write the manual check first in the
  issue or in this file, then implement and run it on the real hardware.
- **Bug fixes:** add a regression test first when practical; otherwise document
  the manual reproduction and fix verification.
- **Spikes:** exempt from test-first, but must record what was measured and the
  conclusion (D-18).
- **Coverage:** cover decisions, boundaries, and edge cases that can break a
  round or the festival stand. Do not chase a coverage percentage.
- **Determinism:** unit and contract tests must not depend on real time, a real
  network, or a real tablet. Use fakes and fixed clocks.
- **Speed:** keep the PR test suite fast enough to run on every pull request.

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

## Component Manual Checks

First checks for single scripts/scenes, outside a full round. Write the check
here (or in the issue) before running it, run it on the target laptop, and
record the result. A scratch scene is enough; do not commit scratch edits
(temporary input map, uncommented fallback) with it.

### Car movement stub (`game/scripts/car.gd`)

Run when the car script, its input handling, or its tuning changes.
Environment: Godot editor on the target laptop.

| # | Step | Pass condition |
|---|------|----------------|
| 1 | Attach `car.gd` to a `CharacterBody2D` (Sprite2D + rectangular collision shape), add a `StaticBody2D` wall on the same collision layer, run the scene. | Car does not move; input stays stubbed (`Vector2.ZERO`). |
| 2 | Locally and temporarily enable the commented keyboard fallback in `_gather_input()` (add temporary input actions in project.godot if needed). | Throttle accelerates to `max_speed` and holds; brake decelerates to 0 first and only then reverses, capped at `max_reverse_speed`; releasing input coasts to a stop. |
| 3 | While step 2 is active, test steering at 0 speed / full speed / in reverse. | No turning at 0 speed; turn rate scales with speed; turning inverts in reverse; travel direction matches heading (no sliding). |
| 4 | Drive into the wall. | Car slides along it, speed visibly reduced, car never destroyed; repeated contact does not tunnel or jitter. |
| 5 | Revert fallback input and temp input map; re-run step 1 before committing. | Stub-neutral car does not move. |

Record: date, pass/fail per step, Godot version. The keyboard fallback must
not end up committed unless a decision adds it.

## CI Plan

Free-tier GitHub Actions; component jobs start after scaffolding. Tooling
details, triggers, and the "do not add yet" list live in
`06-development/tooling-plan.md`.

- `server`: `go test -race ./...`, `go vet ./...`, build.
- `controller-android`: Gradle `assembleDebug`; cache Gradle; upload APK artifact.
- `game`: Godot headless tests where feasible; export is manual/tagged if CI is
  too slow.
- Docs: `mkdocs build --strict` when documentation, `mkdocs.yml`, or
  `requirements-docs.txt` change.
- Required for merge: Go tests/vet and Android build. Godot headless tests are
  required once configured.

Exact commands for the component jobs will be added to this file and `README.md`
after the Sprint 0 scaffold. Until then, report checks that could not run.

## Developer Setup Check

Run after changing `scripts/setup.sh`, `scripts/setup.ps1`, or the Git hooks:

1. In a fresh clone, `test -x scripts/setup.sh` succeeds on Linux/macOS. From
   the repository root, `./scripts/setup.sh` exits 0 on Linux/macOS and Windows
   Git Bash (`sh scripts/setup.sh` is equivalent). On Windows PowerShell, run
   `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\setup.ps1`
   and confirm it exits 0.
2. `git config --local --get core.hooksPath` prints `.githooks`.
3. On Linux/macOS, `test -x .githooks/pre-push` succeeds. From Windows
   PowerShell, `git ls-files -s .githooks/pre-push` reports the mode `100755`.
4. On Linux/macOS or Git Bash, a synthetic push to `main` is blocked by the
   hook; a push to another branch is allowed:

   ```sh
   printf 'refs/heads/topic %s refs/heads/main %s\n' \
     "$(git rev-parse HEAD)" \
     "0000000000000000000000000000000000000000" | .githooks/pre-push
   ```

   The first command must exit 1. Repeat it with `refs/heads/topic` as the
   remote ref and expect exit 0. Run this check in Git Bash on Windows;
   PowerShell has no POSIX `printf`/`sh`.

5. On Windows, confirm that `setup.ps1` (or `setup.sh` under Git Bash) sets
   the hook path and that Git for Windows executes the `pre-push` hook and
   rejects a push to `main`. The hook is a POSIX shell script and runs through
   Git's bundled shell. This check must be run on a Windows machine before
   relying on the hook there.

## Documentation Site Checks

Run before merging a documentation or site-config change:

1. Install the pinned toolchain: `pip install -r requirements-docs.txt`.
2. `mkdocs build --strict` succeeds.
3. The generated site has the router as its home page, no `_templates/` pages,
   and no broken relative links reported by `--strict`.
4. After a deployment from `main`, open
   <https://pszlbit.github.io/ProszkowskaDrift/> and check the home page, a deep
   page, navigation, search, and the 404 page with styles loaded.

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
