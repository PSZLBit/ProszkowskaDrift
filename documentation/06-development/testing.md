# Testing Strategy

Status: Draft
Last updated: 2026-10-26

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

The Sprint 0 server slice (S0-04) lives in `server/` and is covered by
table-driven unit tests and `httptest` integration tests with fake controllers
and a fake game. Packages:

- `internal/validation`: finite and range-checked input, strict sequence,
  sliding-window rate limit, per-connection limiter state.
- `internal/protocol`: envelope parsing and version rejection, payload decoding,
  encoding, unknown fields, phase values.
- `internal/session`: slot assignment, invalid and duplicate tokens, kicked
  token blocklist, reconnect, leave, kick, host-command transitions and
  rejections (no players, phase guards), hard session end, input gating,
  per-slot rate limit, sequence reset on reconnect, idle-timeout and
  all-disconnected-pause timers with an injectable clock.
- `internal/gameserver`: four controllers, input forwarding, reconnect,
  heartbeat timeout, ping/pong, leave, invalid/rate closure, frozen-controller
  timeout, protocol mismatch, malformed messages, controller host-command
  rejection, kick with blocked token, host `end` with graceful controller
  close, `host_denied`, all-disconnected pause, game drop notification,
  game heartbeat timeout, game auth, game-driven phase, loopback checks,
  token truncation, `/healthz`, writer-goroutine shutdown, welcome ordering
  against phase broadcasts, timeout close after reconnect or leave, heartbeat
  expiry targeting the captured game connection, replacement-game readiness,
  stale-game message rejection, game-ready gating of `host_request`, ordered
  player notifications during game registration, kick identity, and atomic
  controller-slot access.

Commands, from `server/`:

```sh
gofmt -l .          # must print nothing
go vet ./...
go test -race ./...
```

CI runs the same commands on pull requests touching `server/`
(`.github/workflows/server.yml`) and `mkdocs build --strict` on documentation
pull requests (`.github/workflows/docs.yml`).

Integration tests use `httptest` and fake WebSocket clients; they do not need a
real game, tablet, or external network. The manual four-client check and the
heartbeat check are documented in `server/README.md` and summarized in
[Manual Server Checks](#manual-server-checks).

## Godot Tests

The game scaffold uses a small headless GDScript runner
(`game/tests/run_tests.gd`) with no addon dependency. It covers:

- Protocol envelope parsing, version rejection, payload validation, and fatal
  error classification.
- `game_hello` and `config` validation, including rejected ranges.
- Slot-keyed input mapping, per-slot timeouts, sequence reset, and clearing one
  slot without affecting the others.
- Car throttle, brake/reverse, coast, and steering math.
- Car input-source wiring, including passing the car's own slot.
- Main-scene structure: four cars, slots 1–4, collision layers, floating motion.
- Project settings: main scene, 1920×1080 viewport, and fullscreen mode.

Commands, from the repository root:

```sh
godot --headless --path game --import
godot --headless --path game --script res://tests/run_tests.gd
```

CI runs the same commands on pull requests touching `game/`
(`.github/workflows/game.yml`).

Still future: time/precision scoring, task assignment, parking detection, round
transitions, tie-breakers, auto-skip, and a physics-level map/target test.

## Contract and Integration Tests

Automated in S0-04:

- A fake game client connects to the Go server, receives `game_hello` and
  `config`, sends `game_ready`, and exchanges `host_command`/`phase` messages.
- Four fake controllers connect, receive slots, send input, and verify that the
  server forwards only validated input.
- Reconnect test: drop one controller, reconnect with the same token, verify the
  same slot and resumed input.
- Host test: a controller cannot start/pause/reset; the server rejects it.
- Host `end` ordering: an in-flight `hello` either registers before `end` and
  is closed with `session_ended`, or is rejected after it; the controller
  cannot join an ended session and the game sees `player_joined` before
  `host_command` `end`.
- Timer/phase ordering: a game `phase` message cannot overwrite the
  all-disconnected paused broadcast.
- Abuse tests: invalid input and rate violations close the connection after the
  configured threshold; a protocol version mismatch closes it immediately.

Still future: the latency test (timestamp a round trip on the local network and
record p50/p95), late join, game supervision, and the target-display/TV run.

## Manual Server Checks

These checks use the S0-04 fake clients. Details and expected output are in
`server/README.md`.

1. Start the server with a known token:
   `go run ./cmd/server -game-token dev-token -log-level debug` from `server/`.
2. Controllers must exist before the round starts: `start` is denied with
   `host_denied` `no_players` otherwise. Start
   `go run ./cmd/fakecontroller -clients 4 -duration 15s -invalid-after 3s -drop-after 5s`
   first, then `go run ./cmd/fakegame -token dev-token -auto-start`. The fake
   game should receive `game_hello`, reply with `game_ready`, accept the
   delayed `start`, and report `player_joined` as controllers connect.
3. Expect four `welcome` messages with slots 1–4, labels `P1`–`P4`, and the four
   colors; one `invalid_input` error per controller; player 1 drops and
   reconnects into the same slot with a reset sequence.
4. Send `leave` from the fake controllers and confirm the slots stay reserved.
5. Run `go run ./cmd/fakecontroller -clients 1 -duration 12s -silent-after 2s`
   while the game is in `playing`. The server must close the controller after
   about 6 s with `heartbeat_timeout`.
6. Close the fake game with `quit` while controllers are connected. Every
   controller must receive `game_unavailable` and a `phase` back to `attract`;
   a restarted fake game reclaims the connection with the same token.
7. A controller that stops responding (frozen tablet process) is covered by an
   automated regression test (`TestFrozenControllersDoNotStallTickLoop`): the
   server tears the connection down after a bounded wait and the tick loop
   keeps enforcing timers. A manual check with a suspended real tablet is part
   of the controller hardware checks (S0-02 and later).
8. `curl http://127.0.0.1:8080/healthz` must report `phase`, the number of
   connected controllers, and `gameReady`.
9. Use `go run ./cmd/fakegame` interactive commands to check `start`, `pause`,
   `resume`, `reset`, `end`, and `kick 2`; a kicked controller receives `error`
   `kicked`, the slot is freed, and the game receives `player_left` with reason
   `kicked`.

Record the date, commit, commands, and observed result for these checks. A real
Android controller and the target TV run are not available yet; those hardware
checks remain open.

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

### Game scaffold input path (`game/scripts/server_input.gd`)

Recorded: 2026-10-26. Result: pass for the automated and headless path; the
target-display/TV run is still pending in issue #13.

Scope: the game scaffold promoted from S0-05 (D-32). It verifies one game
WebSocket connection, the handshake, heartbeats, four slot-keyed cars,
per-slot input, timeout, and host start. It does not verify the Android
controller, TV readability, or measured latency.

Environment:

- Godot 4.7.2 stable.
- Server at `main` (`fbc1dc6`; PR #30 merged).
- Four `fakecontroller` clients; no Android tablet and no TV.
- `game/scenes/main.tscn`; no token is committed; `PD_GAME_TOKEN=dev-token`.

Procedure:

1. Start the server from `server/`:

   ```powershell
   go run ./cmd/server -game-token dev-token -log-level debug
   ```

2. In a second terminal, connect four fake controllers:

   ```powershell
   cd server
   go run ./cmd/fakecontroller -clients 4 -duration 15s
   ```

3. Set the token and run the project from Godot (F5), or run the headless
   smoke path:

   ```powershell
   $env:PD_GAME_TOKEN = "dev-token"
   godot --headless --path game --quit-after 600 -- --dev-auto-start
   ```

   `--dev-auto-start` is a development-only flag; the normal host path is
   `Enter` in the game window.

4. In the window path, press `Enter` when the four controllers have joined. The
   server must accept `host_request start` and forward input only after the
   phase becomes `playing`.

5. Check that each car moves independently and that stopping one fake
   controller or leaving its slot stops only that car. Restart the server and
   confirm the game reconnects and sends `game_ready` again.

6. For the auth-failure check, start the game with a wrong `PD_GAME_TOKEN` and
   confirm it logs the fatal error and does not keep reconnecting.

Observed:

- The headless runner passes 96 checks on 2026-10-26.
- Handshake: `game_hello` protocol 1, then `config` with a 180000 ms round,
  5 tasks, and 4 max players; `game_ready` was accepted.
- Four `player_joined` messages for slots 1–4 with labels `P1`–`P4` and colors
  red, blue, green, and yellow.
- `host_request start` produced `host_command start`; forwarded input arrived
  for all four slots independently (for example slot 1 `steer=0.600`,
  `throttle=0.600` and slot 3 `steer=0.150`, `throttle=0.600`).
- The main scene loads headless with four cars on slots 1–4, collision layer 2,
  mask 1, floating motion, and each car wired to the shared `ServerInput`.
- A temporary headless wrapper loaded `main.tscn` and ran it for 6 s with the
  four fake controllers. All four cars moved from their spawn points (for
  example slot 2 from `(350, 500)` to `(1021, 1020)`), so forwarded input reached
  the scene. The wrapper was deleted after the run.
- Wrong `PD_GAME_TOKEN`: the game logged `Fatal auth_failed` once and did not
  reconnect, so the close-reason path works even when the final error frame is
  not delivered.
- Not covered: visible movement, per-slot leave/kick on the real display, TV
  readability, FPS/tearing, latency, and the Android controller. Per-slot
  clearing and timeout are covered by the headless unit tests.

### Four-car fullscreen display

Run before claiming the S0-05 display criteria. Result: pending (issue #13).

Environment: target laptop and TV, fullscreen 1920×1080, four fake controllers.

| # | Step | Pass condition |
|---|------|----------------|
| 1 | Start the server, connect four fake controllers, and run the project. | The window is fullscreen 1920×1080; four cars labelled P1–P4 are visible with distinct colors. |
| 2 | Press `Enter` to start. | The server accepts the start; all four cars move independently. |
| 3 | Drive each car into the environment and into another car. | Environment collisions reduce speed; cars pass through/overlap each other without colliding; boundary walls keep cars on-screen. |
| 4 | Stop one fake controller without leaving, then reconnect it. | Only that car stops; the other cars continue; the reconnected car resumes in the same slot. |
| 5 | Press `P` and `R`. | Pause/resume and reset affect all cars and the server phase correctly. |
| 6 | Restart the server. | The game reconnects, sends `game_ready`, and does not duplicate cars. |
| 7 | Watch for at least one minute and record measurements. | 60 FPS target; note stutter, tearing, input-to-visible latency, and readability from the spectator distance. |

Record the date, Godot version, server commit, FPS/tearing observations, and
pass/fail per step. Do not mark the display criteria as passed until this check
has been run on the target laptop/TV.

### Car movement (`game/scripts/car.gd`)

Run when the car script, its input handling, or its tuning changes.
Environment: Godot editor on the target laptop.

The car reads `(steer, throttle)` from its assigned `input_source` through
`get_input(slot)`. With no source assigned, it returns neutral input and coasts.

| # | Step | Pass condition |
|---|------|----------------|
| 1 | Open `game/scenes/sprites/car.tscn` (CharacterBody2D + Sprite2D + collision shape) and add a `StaticBody2D` wall on collision layer 1. Leave `Input Source` empty and run the scene. | Car does not move. |
| 2 | Create a scratch `Node` with the script below and assign it to `Car.input_source`. Run the scene and hold Up. | Throttle accelerates to `max_speed` and holds; releasing Up coasts to a stop. |
| 3 | Hold Down while moving forward, then at a stop. | Brake decelerates to 0 first and only then reverses, capped at `max_reverse_speed`. |
| 4 | With the scratch source active, test steering at 0 speed / full speed / in reverse. | No turning at 0 speed; turn rate scales with speed; turning inverts in reverse; travel direction matches heading (no sliding). |
| 5 | Drive into the wall. | Car slides along it, speed visibly reduced, car never destroyed; repeated contact does not tunnel or jitter. |
| 6 | Remove the scratch source and discard the local scene changes. | No scratch files or scratch wiring are committed; `main.tscn` already wires each car to `../ServerInput`. |

Scratch keyboard source:

```gdscript
extends Node

func get_input(_slot: int) -> Vector2:
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
- `game`: import the project headless and run `game/tests/run_tests.gd`; export
  is manual/tagged if CI is too slow.
- Docs: `mkdocs build --strict` when documentation, `mkdocs.yml`, or
  `requirements-docs.txt` change.
- Required for merge when branch protection is enabled: Go tests/vet and Godot
  headless tests; the Android build joins when `controller-android/` is
  scaffolded. Branch protection is still a repository-settings task.

Server commands are documented above and in `server/README.md`. Godot commands
are documented above. Report any check that could not run.

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
