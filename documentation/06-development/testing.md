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
  heartbeat timeout, ping/pong, leave, invalid/rate closure, slow-client
  close, protocol mismatch, malformed messages, controller host-command
  rejection, kick with blocked token, host `end` with graceful controller
  close, `host_denied`, all-disconnected pause, game drop notification,
  game heartbeat timeout, game auth, game-driven phase, loopback checks,
  token truncation, and `/healthz`.

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

Automated in S0-04:

- A fake game client connects to the Go server, receives `game_hello` and
  `config`, sends `game_ready`, and exchanges `host_command`/`phase` messages.
- Four fake controllers connect, receive slots, send input, and verify that the
  server forwards only validated input.
- Reconnect test: drop one controller, reconnect with the same token, verify the
  same slot and resumed input.
- Host test: a controller cannot start/pause/reset; the server rejects it.
- Abuse tests: invalid input and rate violations close the connection after the
  configured threshold; a protocol version mismatch closes it immediately.

Still future: the latency test (timestamp a round trip on the local network and
record p50/p95), the all-disconnected pause, late join, game supervision, and
real Godot integration.

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
7. A controller that stops reading (frozen tablet process) is covered by an
   automated regression test (`TestFrozenControllersDoNotStallTickLoop`): the
   server tears the connection down after a bounded wait and the tick loop
   keeps enforcing timers. A manual check with a suspended real tablet is part
   of the controller hardware checks (S0-02 and later).
8. `curl http://127.0.0.1:8080/healthz` must report `phase`, the number of
   connected controllers, and `gameReady`.
6. Use `go run ./cmd/fakegame` interactive commands to check `start`, `pause`,
   `resume`, `reset`, `end`, and `kick 2`; a kicked controller receives `error`
   `kicked`, the slot is freed, and the game receives `player_left` with reason
   `kicked`.

Record the date, commit, commands, and observed result for these checks. A real
Android controller and the real Godot client are not available yet; those
hardware checks remain open.

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

Server commands are documented above and in `server/README.md`. Android and
Godot commands will be added here and in `README.md` when those components are
scaffolded. Until then, report checks that could not run.

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
