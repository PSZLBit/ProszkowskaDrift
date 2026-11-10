# Decision Log

Status: Decided
Last updated: 2026-11-10

Register of all team decisions. Newest entries on top.
For larger decisions, create an ADR file in this folder using `_templates/adr.md`
and link it here. The ADR threshold is defined in
[`documentation-rules.md`](../00-meta/documentation-rules.md#adr-threshold).

| Date       | ID   | Decision                                                                                                                                  | Status  | Owner |
| ---------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------- | ------- | ----- |
| 2026-11-10 | D-37 | Game process supervision for the MVP is implemented as an optional in-process Go supervisor (`server/internal/supervisor`, enabled with `--game-binary`): it starts Godot with a per-launch token and restarts it after unexpected exits, returning the session to `Attract`. Server restart and the festival launcher/wrapper remain future work. The shell-script launcher option is not used. | Decided | Team  |
| 2026-11-10 | D-36 | ADRs are required when a decision selects or changes the stack, transport, an architecture boundary, a cross-component contract, or a build/runtime interface constraint. Product tuning, performance targets, privacy rules, delivery plans, and process agreements are recorded in their module documents and dated decision-log rows without an ADR. ADR links in decision rows, the ADR index, the decision map, and the open TBD/validation register stay current. | Decided | Team |
| 2026-11-09 | D-35 | The S0-05 controller WebSocket spike is promoted to the production `controller-android/` scaffold after cleanup, following the D-32 precedent. The scaffold uses OkHttp 3.12.x, a device-token `hello`, 25 Hz phase-gated input, ping/pong liveness with a bounded watchdog, malformed-frame tolerance, and reconnect with capped backoff; duplicate-token `token_in_use` is retryable while invalid-token `auth_failed` stays fatal. Real-tablet install, latency, and stability evidence remains tracked in issues #11 and #48 until recorded. D-18 continues to apply to throwaway prototype logic, not this scaffold. | Decided | Team  |
| 2026-11-04 | D-34 | Purchased third-party art assets are kept out of the public Git repository. The "Large Parking Lot" tileset is ignored by `.gitignore`; each team member installs the purchased copy locally, and provenance stays recorded in `game/assets/CREDITS.md`. The ignore rule applies to new commits; existing commits are not rewritten for the MVP. | Decided | Team  |
| 2026-11-03 | D-33 | Sprint 1 (2026-10-26 – 2026-11-01) did not meet its M1 first-half goal; only the game scaffold landed. Sprint 2 is re-baselined to a single-tablet end-to-end vertical slice, and four-player validation moves to Sprint 3 as the M2/MFD acceptance run. The M1 exit is one complete round with one task type on one real tablet. The MFD (2026-11-15) and festival-ready (2026-11-18) dates do not change; the full MVP remains a stretch. Non-MFD work moves to Sprint 3/4 or Post-MVP. | Decided | Team |
| 2026-10-26 | D-32 | The S0-05 Godot integration is promoted to the production `game/` scaffold after cleanup. The scaffold keeps one `/ws/game` connection, maps slots 1–4 to four cars with a slot-keyed input store, and validates protocol/config before play. Four-car/TV/FPS evidence remains tracked in issue #13 until the hardware check is recorded. D-18 continues to apply to throwaway prototype logic, not this scaffold. | Decided | Team  |
| 2026-10-21 | D-31 | The festival date is 2026-11-24, confirmed with the organizers. The internal festival-ready target is 2026-11-18, with 2026-11-19–23 reserved for fixes, setup, and final rehearsal. The Minimum Festival Demo (MFD) is the committed scope and the full MVP is a stretch goal on a five-sprint calendar ([roadmap](../06-development/roadmap.md)). | Decided | Team  |
| 2026-10-20 | D-30 | Project documentation is published as a read-only MkDocs site on GitHub Pages through GitHub Actions (deployment enabled 2026-11-09; see [Status Notes](#status-notes)); `documentation/` remains the source of truth and the in-repo router stays authoritative. MkDocs is pinned in `requirements-docs.txt` and CI runs `mkdocs build --strict`. | Decided | Team  |
| 2026-10-19 | D-29 | On a task branch, commit each coherent step without waiting for a separate instruction; pushing, merging, and opening pull requests still require an explicit request. | Decided | Team  |
| 2026-10-19 | D-28 | Engineering working agreement: English-only artifacts; small, regular commits using feat/fix/docs/test/chore/spike prefixes; test-first for pure logic; tests or documented manual hardware checks for every behavior change; no direct pushes to main and no history rewrites. | Decided | Team  |
| 2026-10-19 | D-27 | Delivery uses one-week sprints, a weekly demo and retrospective, GitHub Projects issues/milestones, documented DoR/DoD, peer review, CI, and explicit AI-assistance disclosure. | Decided | Team  |
| 2026-10-19 | D-26 | Delivery is split into Minimum Festival Demo, MVP, and Post-MVP; racing, drift, extra task types, and a web host panel are Post-MVP. | Decided | Team  |
| 2026-10-19 | D-25 | The MVP stores session state in memory only, with no accounts, player names, persistent profiles, or personal data; only anonymous in-memory event scores may be shown. | Decided | Team  |
| 2026-10-19 | D-24 | A disconnected controller's car stops and its slot is reserved for automatic reclaim; all-disconnected pauses the round, late join is between rounds, and game/server crashes restart to Attract with the current round lost ([ADR-0003](adr-0003-session-lifecycle-and-recovery.md)). | Decided | Team  |
| 2026-10-19 | D-23 | Real-time targets are 60 FPS display, 20–30 Hz controller input, and at most 100 ms input-to-visible-response latency, to be measured in Sprint 0. | Decided | Team  |
| 2026-10-19 | D-22 | The MVP stack is Go for the server, Godot 4 for the game/display, and minimal Kotlin for the controller, connected by WebSocket + JSON; the server owns session/input authority and the game owns simulation/scoring, with in-memory state ([ADR-0001](adr-0001-mvp-architecture.md)). | Decided | Team  |
| 2026-10-19 | D-21 | The four fixed tablets connect automatically to a configured server address, register a random device token, and reclaim their slot after reconnect; QR or manual address entry is only a setup fallback ([ADR-0003](adr-0003-session-lifecycle-and-recovery.md)). | Decided | Team  |
| 2026-10-19 | D-20 | The controller targets Android 4.4.2 (API 19) with a minimal Kotlin app: plain Views, no Compose, no OkHttp 4, and no AndroidX unless a library is proven API 19 compatible; compatibility is validated in Sprint 0, with a minimal Java app as the documented fallback ([ADR-0002](adr-0002-legacy-android-controller.md)). | Decided | Team  |
| 2026-10-19 | D-19 | The MVP is a top-down 2D parking time attack: 1–4 independent players, one fixed map, 3-minute rounds, five parking tasks per player, environment collisions only, and scoring for time, precision, and collisions ([game design](../02-game-domain/game-design.md)). | Decided | Team  |
| 2026-10-19 | D-18 | Timeboxed spikes of up to one week are allowed to resolve TBD items; spike code is throwaway, not merged as production, and results are recorded in the decision log. | Decided | Team  |
| 2026-10-16 | D-17 | Server language, game engine, and controller-to-server transport protocol are not selected yet. Do not introduce them without a decision. | Superseded by D-22 | Team  |
| 2026-10-16 | D-16 | Latency target and update/tick rate are TBD. Do not invent numbers in code or docs.                                                       | Superseded by D-23 | Team  |
| 2026-10-16 | D-15 | Disconnect, rejoin, late join, and host-crash handling are TBD.                                                                           | Superseded by D-24 | Team  |
| 2026-10-16 | D-14 | Controller input set and per-controller feedback are TBD.                                                                                 | TBD     | TracerDS |
| 2026-10-16 | D-13 | Game concept, rules, and win condition are TBD. Do not implement game logic until `02-game-domain/domain-overview.md` reaches `Decided`.  | Superseded by D-19 | Team  |
| 2026-10-16 | D-12 | No accounts and no retained personal data. Old account/role/18+/GDPR-DOB model is archived. Sessions hold transient state only.           | Decided | Team  |
| 2026-10-16 | D-11 | Server and game are separate so the game can be swapped via a stable game interface ([ADR-0001](adr-0001-mvp-architecture.md)).                                                      | Decided | Team  |
| 2026-10-16 | D-10 | One session hosts 1–4 players, one Android tablet per player.                                                                             | Decided | Team  |
| 2026-10-16 | D-09 | Local-network only (e.g. shared WiFi). Tablets join via lobby code/QR. No internet play for MVP.                                          | Superseded by D-21 | Team  |
| 2026-10-16 | D-08 | Controller app is Android-only, Kotlin native, in `controller-android/` ([ADR-0002](adr-0002-legacy-android-controller.md)).                                                                  | Decided | Team  |
| 2026-10-16 | D-07 | Monorepo layout: `server/`, `game/`, `controller-android/`, `documentation/`.                                                             | Decided | Team  |
| 2026-10-16 | D-06 | Pivot from the Flanki tournament-management web app to the local real-time play system (controllers + server + swappable game + big-screen display). Old Flanki docs were removed from the working tree; decision history is retained. D-03 is superseded. | Decided | Team  |
| 2026-10-13 | D-05 | All project artifacts (docs, code, comments, commits) are in English.                                                                     | Decided | Team  |
| 2026-10-13 | D-04 | `documentation/` is the source of truth; agents use read-first / update-after protocol.                                                   | Decided | Team  |
| 2026-10-13 | D-03 | MVP bracket format is single-elimination only.                                                                                            | Superseded by D-06 | Team  |
| 2026-10-13 | D-02 | Date of birth is private; never exposed in public views, APIs, or logs.                                                                   | Superseded by D-12 | Team  |
| 2026-10-13 | D-01 | Tech stack, database, and hosting are not selected yet. Do not introduce them without a decision.                                         | Superseded by D-22 and D-25 | Team  |

## Spike Results

| Date | Spike | Result | Evidence |
|------|-------|--------|----------|
| 2026-11-09 | Controller WebSocket spike (`controller-android/`) | Pass for the host-JVM scope: endpoint/hello, 25 Hz phase-gated input, ping/pong watchdog, reconnect and duplicate-token retry, `game_unavailable` staying connected, sequence reset, stop/start, and fatal error handling are covered by fake-server tests plus debug build and lint. Real-tablet install, venue Wi-Fi, and latency evidence are pending (#11, #48). | [PR #63](https://github.com/PSZLBit/ProszkowskaDrift/pull/63), [`testing.md`](../06-development/testing.md), "Android Controller Build Checks" |
| 2026-10-26 | Godot four-car game scaffold | Partial: four-car slot mapping, protocol/config validation, token/fatal-error handling, reconnect backoff, and headless tests implemented. Target laptop/TV run and FPS/tearing notes still pending (#13). | [`testing.md`](../06-development/testing.md), "Game scaffold input path"; `game/tests/run_tests.gd` |
| 2026-10-22 | Godot `ServerInput` single-car WebSocket path | Pass for the tested path: handshake, heartbeat, auto-start, forwarded input, input-timeout path, and wired-car movement. Exact timeout boundary and established-connection reconnect were not exercised. | [PR #31](https://github.com/PSZLBit/ProszkowskaDrift/pull/31), single-car spike record |

## Open TBD and Validation Items

Open items are listed with a named owner, the next step, and the tracking issue.
The list does not change a decision; it records what is still unproven or
undecided. Hardware checks stay pending until they are run on the real devices.

| Open item | Owner | Next step | Tracking |
|---|---|---|---|
| Controller input scheme and feedback (D-14) | TracerDS | Playtest input options on T1–T4 (#18); implement feedback in #48; record the chosen scheme here | #18, #48 |
| MFD map, task skip, parking tolerance, and tuning | Brzana | Land the MFD baseline in #45; implement #46/#47; full GDD remains #32 | #45, #46, #47, #32 |
| Finished-car rendering (hidden, ghosted, or finish area) | Brzana | Decide with the MFD map/task baseline in #45; implement with the game slice in #47 | #45, #47 |
| Target-finding UI (minimap vs on-map beacons) | Brzana | Decide in the MFD display baseline in #45; implement in #47 | #45, #47 |
| WebSocket libraries | TracerDS + GreenDude5 | Confirm Go `coder/websocket` v1.8.15 and OkHttp 3.12.x on API 19 in the architecture; production in #48 | #11, #48 |
| Supervisor and festival laptop OS | GreenDude5 + ZofiaDan | Game-only supervision implemented in PR #59 (merged, D-37); manual crash drill still pending (#15). Server restart and festival launcher remain #50. Decide Windows vs Linux in #49/#50 | #15, #49, #50 |
| Router model and static-IP plan | ZofiaDan | Choose the model and address plan during #49; document it in #50 | #49, #50 |
| Hardware checks and spares | ZofiaDan | S0-02 short Wi-Fi/charging check passed 2026-11-10 (#10); run the 30-minute Wi-Fi, reconnect, latency, touch/readability, and spare checks in #14/#49; update the inventory and ADR-0002 | #14, #49 |
| Gamepad fallback input (last resort) | ZofiaDan | Xbox 360/PC gamepad is the identified last-resort option only; do not implement without a decision, because the server accepts controller WebSocket connections only | #49 |
| Latency measurement (D-23) | ZofiaDan | Measure p50/p95 in #14 and record the evidence in `testing.md` | #14 |
| Protocol task/score messages | GreenDude5 + Brzana | Implement with #46/#47; update protocol coverage and tests | #46, #47 |

The second host control device is Post-MVP (D-26) and is not an open item.

## ADR Index

| ADR | Decisions | Scope |
|---|---|---|
| [ADR-0001](adr-0001-mvp-architecture.md) | D-11, D-22 | Stack, transport, and server/game boundary |
| [ADR-0002](adr-0002-legacy-android-controller.md) | D-08, D-20 | Android 4.4.2 / API 19 controller |
| [ADR-0003](adr-0003-session-lifecycle-and-recovery.md) | D-21, D-24 | Session lifecycle, reconnect, and recovery |

## Decision Map

This map links active decisions to their authoritative artifact. Superseded
decisions D-01, D-02, D-03, D-09, D-13, D-15, D-16, and D-17 remain in the
table above for history and have no active artifact.

| Artifact | Decisions |
|---|---|
| `documentation/README.md`, `00-meta/` | D-04, D-05, D-07, D-36 |
| [Decision log](decision-log.md) | D-06 |
| [Vision and scope](../01-vision-scope/vision-scope.md) | D-26 |
| [Game design](../02-game-domain/game-design.md) | D-14 (TBD), D-19 |
| [Players, sessions, and permissions](../03-players-sessions/roles-sessions.md) | D-10, D-12, D-21, D-24, D-25 |
| [Product requirements](../04-product/requirements.md) | D-19, D-26 |
| [ADR-0001](adr-0001-mvp-architecture.md) | D-11, D-22 |
| [ADR-0002](adr-0002-legacy-android-controller.md) | D-08, D-20 |
| [ADR-0003](adr-0003-session-lifecycle-and-recovery.md) | D-21, D-24 |
| [Runtime architecture](../06-development/architecture.md) | D-11, D-21, D-22, D-23, D-24, D-32, D-37 |
| [Protocol](../06-development/protocol.md) | D-21, D-23, D-24, D-32 |
| [Testing strategy](../06-development/testing.md) | D-23, D-24, D-32 |
| [Roadmap](../06-development/roadmap.md) | D-26, D-31, D-33 |
| [Workflow](../06-development/workflow.md) | D-18, D-27, D-28, D-29, D-30 |
| [Sprint reviews](../06-development/sprint-reviews.md) | D-27, D-31, D-33 |
| `.gitignore`, `game/assets/CREDITS.md` | D-34 |

## Status Notes

| Date | Related | Note |
|---|---|---|
| 2026-11-09 | D-14 | Owner set to TracerDS; the input scheme remains TBD in #18 and controller feedback is tracked in #48. |
| 2026-11-09 | D-19 | The MFD gameplay baseline is still being defined in #45; the full GDD is #32. The decision is not reopened; `game-design.md` stays Draft until #45 lands. |
| 2026-11-09 | D-21 | The server token/slot reclaim path is implemented. Controller auto-connect is merged (PR #63); real-tablet proof remains pending in #11/#49. |
| 2026-11-09 | D-20 / ADR-0002 | API 19 compatibility validation is pending. Static inventory is merged (#10/PR #44); APK build is merged (PR #63); install, input, reconnect, and hardware checks remain #11/#48/#49. |
| 2026-11-09 | D-23 | Latency has not been measured. The targets remain decided, but the evidence is pending in #14. |
| 2026-11-09 | D-24 / ADR-0003 | The server reconnect and all-disconnected pause path is implemented. The Godot supervisor is implemented (PR #59, D-37); the server-crash launcher remains #50; real-hardware proof remains #49. |
| 2026-11-09 | D-30 | GitHub Pages deployment is live (HTTP 200; Docs run 41847329287). The earlier pending status is resolved; no decision change. |

## Rules

- A `TBD` row blocks implementation of that topic. Timeboxed spikes require an explicit decision entry.
- Changing a `Decided` row requires a new dated entry, not silent editing.
- Superseded rows stay in the log; never delete decision history.
- Link the ADR file in the Decision column when one exists.
- Keep the Open TBD and Validation Items register current; do not leave an open
  item without an owner and a next step.
