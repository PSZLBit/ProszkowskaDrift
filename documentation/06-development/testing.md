# Testing Strategy

Status: Draft
Last updated: 2026-10-22

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
record the result. A scratch scene or scratch input source is enough; do not
commit scratch edits with it.

### ServerInput spike (`game/scripts/server_input.gd`)

Recorded: 2026-10-22. Result: pass for the tested path.

Scope: Sprint 0 spike (D-18) for the single-car game input path. It does not
promote the protocol from `Draft`, does not verify the Android controller, and
does not replace the final host/start flow. `auto_start_round` is a spike
convenience, not a production host flow.

Purpose: verify that the Godot game can connect to the Go game endpoint,
complete the handshake, stay connected with heartbeats, start a round, receive
forwarded controller input, and drive the wired car.

Environment:

- Godot 4.7.2 stable.
- Server branch `feat/s0-04-go-server`, commit `ac166e2`.
- One `fakecontroller` client; no Android tablet and no TV.
- `game/scenes/test.tscn`, with `ServerInput` and `Car.input_source` wired.

Procedure:

1. Create a temporary worktree from the server branch and start the server:

   ```powershell
   git worktree add --detach <temp-dir> origin/feat/s0-04-go-server
   cd <temp-dir>/server
   go run ./cmd/server -game-token dev-token -log-level debug
   ```

2. In a second terminal, connect one fake controller:

   ```powershell
   cd <temp-dir>/server
   go run ./cmd/fakecontroller -clients 1 -duration 15s
   ```

3. Run `game/scenes/test.tscn` with F6 in Godot. The node connects to
   `ws://127.0.0.1:8080/ws/game?token=dev-token`, sends `game_ready`, and
   auto-sends `host_request start` after one second.

4. Check the Godot Output panel and watch the car. For this record, a temporary
   headless wrapper loaded the same scene and printed the car position; the
   wrapper was deleted after the run.

5. To check the 500 ms input timeout, restart the fake controller with
   `-silent-after 3s`. The connection stays open while input stops; the car
   should coast to a stop.

6. For the connection-timeout check, point `server_url` at an unused local port
   with no server running.

Observed:

- Handshake: `game_hello` with protocol 1, then `config` with a 180000 ms round,
  5 tasks, and 4 max players.
- `game_ready` was accepted and the server logged `game heartbeat` messages
  every second, so the heartbeat requirement was met.
- `player_joined slot=1`; `host_request start` produced `host_command start` and
  the server logged `host command accepted ... phase=playing`.
- First forwarded sample: `steer=0.150`, `throttle=0.600`.
- The wired car moved from `(559, 322)` at the start to `(725.7, 342.7)` after
  2 s and `(1222.7, 818.7)` after 5 s, so forwarded input reached the car through
  `Car.input_source`.
- Input timeout: after the fake controller went silent, `get_input()` returned
  `(0.15, 0.6)` at 1.5 s and `(0.0, 0.0)` at 4.0 s; the car velocity reached zero.
  This confirms the timeout path, not the exact 500 ms boundary.
- Negative check: with an unused port, the node logged `connect timed out;
  reconnecting` after about five seconds.

Conclusion: the single-car Godot WebSocket path works for the spike. Keep the
protocol and input decisions open; see `protocol.md` and D-14.

Not covered by this check: the exact 500 ms timeout boundary, reconnect after an
established server connection drops, multiple cars, the Android controller, and
visible latency on the TV.

### Car movement (`game/scripts/car.gd`)

Run when the car script, its input handling, or its tuning changes.
Environment: Godot editor on the target laptop.

The car reads `(steer, throttle)` from its assigned `input_source` through
`get_input()`. With no source assigned, it returns neutral input and coasts.

| # | Step | Pass condition |
|---|------|----------------|
| 1 | Open a scene with `car.gd` on a `CharacterBody2D` (Sprite2D + rectangular collision shape) and a `StaticBody2D` wall on the same collision layer. Leave `Input Source` empty and run the scene. | Car does not move. |
| 2 | Create a scratch `Node` with the script below and assign it to `Car.input_source`. Run the scene and hold Up. | Throttle accelerates to `max_speed` and holds; releasing Up coasts to a stop. |
| 3 | Hold Down while moving forward, then at a stop. | Brake decelerates to 0 first and only then reverses, capped at `max_reverse_speed`. |
| 4 | With the scratch source active, test steering at 0 speed / full speed / in reverse. | No turning at 0 speed; turn rate scales with speed; turning inverts in reverse; travel direction matches heading (no sliding). |
| 5 | Drive into the wall. | Car slides along it, speed visibly reduced, car never destroyed; repeated contact does not tunnel or jitter. |
| 6 | Remove the scratch source and set `Car.input_source` back to `../ServerInput`; save the scene. | Scene returns to server-driven input; no scratch files are committed. |

Scratch keyboard source:

```gdscript
extends Node

func get_input() -> Vector2:
	return Vector2(
		Input.get_axis("ui_left", "ui_right"),
		Input.get_axis("ui_down", "ui_up"),
	)
```

Record: date, pass/fail per step, Godot version. The scratch keyboard source
must not end up committed; the production path is `ServerInput`.

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
