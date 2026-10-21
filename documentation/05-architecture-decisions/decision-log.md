# Decision Log

Status: Decided
Last updated: 2026-10-21

Register of all team decisions. Newest entries on top.
For larger decisions, create an ADR file in this folder using `_templates/adr.md`
and link it here.

| Date       | ID   | Decision                                                                                                                                  | Status  | Owner |
| ---------- | ---- | ----------------------------------------------------------------------------------------------------------------------------------------- | ------- | ----- |
| 2026-10-21 | D-31 | The festival date is 2026-11-24, confirmed with the organizers. The internal festival-ready target is 2026-11-18, with 2026-11-19–23 reserved for fixes, setup, and final rehearsal. The Minimum Festival Demo (MFD) is the committed scope and the full MVP is a stretch goal on a five-sprint calendar ([roadmap](../06-development/roadmap.md)). | Decided | Team  |
| 2026-10-20 | D-30 | Project documentation is published as a read-only MkDocs site on GitHub Pages through GitHub Actions (first deployment pending Pages enablement); `documentation/` remains the source of truth and the in-repo router stays authoritative. MkDocs is pinned in `requirements-docs.txt` and CI runs `mkdocs build --strict`. | Decided | Team  |
| 2026-10-19 | D-29 | On a task branch, commit each coherent step without waiting for a separate instruction; pushing, merging, and opening pull requests still require an explicit request. | Decided | Team  |
| 2026-10-19 | D-28 | Engineering working agreement: English-only artifacts; small, regular commits using feat/fix/docs/test/chore/spike prefixes; test-first for pure logic; tests or documented manual hardware checks for every behavior change; no direct pushes to main and no history rewrites. | Decided | Team  |
| 2026-10-19 | D-27 | Delivery uses one-week sprints, a weekly demo and retrospective, GitHub Projects issues/milestones, documented DoR/DoD, peer review, CI, and explicit AI-assistance disclosure. | Decided | Team  |
| 2026-10-19 | D-26 | Delivery is split into Minimum Festival Demo, MVP, and Post-MVP; racing, drift, extra task types, and a web host panel are Post-MVP. | Decided | Team  |
| 2026-10-19 | D-25 | The MVP stores session state in memory only, with no accounts, player names, persistent profiles, or personal data; only anonymous in-memory event scores may be shown. | Decided | Team  |
| 2026-10-19 | D-24 | A disconnected controller's car stops and its slot is reserved for automatic reclaim; all-disconnected pauses the round, late join is between rounds, and game/server crashes restart to Attract with the current round lost. | Decided | Team  |
| 2026-10-19 | D-23 | Real-time targets are 60 FPS display, 20–30 Hz controller input, and at most 100 ms input-to-visible-response latency, to be measured in Sprint 0. | Decided | Team  |
| 2026-10-19 | D-22 | The MVP stack is Go for the server, Godot 4 for the game/display, and minimal Kotlin for the controller, connected by WebSocket + JSON; the server owns session/input authority and the game owns simulation/scoring, with in-memory state ([ADR-0001](adr-0001-mvp-architecture.md)). | Decided | Team  |
| 2026-10-19 | D-21 | The four fixed tablets connect automatically to a configured server address, register a random device token, and reclaim their slot after reconnect; QR or manual address entry is only a setup fallback. | Decided | Team  |
| 2026-10-19 | D-20 | The controller targets Android 4.4.2 (API 19) with a minimal Kotlin app: plain Views, no Compose, no OkHttp 4, and no AndroidX unless a library is proven API 19 compatible; compatibility is validated in Sprint 0, with a minimal Java app as the documented fallback ([ADR-0002](adr-0002-legacy-android-controller.md)). | Decided | Team  |
| 2026-10-19 | D-19 | The MVP is a top-down 2D parking time attack: 1–4 independent players, one fixed map, 3-minute rounds, five parking tasks per player, environment collisions only, and scoring for time, precision, and collisions ([game design](../02-game-domain/game-design.md)). | Decided | Team  |
| 2026-10-19 | D-18 | Timeboxed spikes of up to one week are allowed to resolve TBD items; spike code is throwaway, not merged as production, and results are recorded in the decision log. | Decided | Team  |
| 2026-10-16 | D-17 | Server language, game engine, and controller-to-server transport protocol are not selected yet. Do not introduce them without a decision. | Superseded by D-22 | Team  |
| 2026-10-16 | D-16 | Latency target and update/tick rate are TBD. Do not invent numbers in code or docs.                                                       | Superseded by D-23 | Team  |
| 2026-10-16 | D-15 | Disconnect, rejoin, late join, and host-crash handling are TBD.                                                                           | Superseded by D-24 | Team  |
| 2026-10-16 | D-14 | Controller input set and per-controller feedback are TBD.                                                                                 | TBD     | Team  |
| 2026-10-16 | D-13 | Game concept, rules, and win condition are TBD. Do not implement game logic until `02-game-domain/domain-overview.md` reaches `Decided`.  | Superseded by D-19 | Team  |
| 2026-10-16 | D-12 | No accounts and no retained personal data. Old account/role/18+/GDPR-DOB model is archived. Sessions hold transient state only.           | Decided | Team  |
| 2026-10-16 | D-11 | Server and game are separate so the game can be swapped via a stable game interface.                                                      | Decided | Team  |
| 2026-10-16 | D-10 | One session hosts 1–4 players, one Android tablet per player.                                                                             | Decided | Team  |
| 2026-10-16 | D-09 | Local-network only (e.g. shared WiFi). Tablets join via lobby code/QR. No internet play for MVP.                                          | Superseded by D-21 | Team  |
| 2026-10-16 | D-08 | Controller app is Android-only, Kotlin native, in `controller-android/`.                                                                  | Decided | Team  |
| 2026-10-16 | D-07 | Monorepo layout: `server/`, `game/`, `controller-android/`, `documentation/`.                                                             | Decided | Team  |
| 2026-10-16 | D-06 | Pivot from the Flanki tournament-management web app to the local real-time play system (controllers + server + swappable game + big-screen display). Old Flanki docs were removed from the working tree; decision history is retained. D-03 is superseded. | Decided | Team  |
| 2026-10-13 | D-05 | All project artifacts (docs, code, comments, commits) are in English.                                                                     | Decided | Team  |
| 2026-10-13 | D-04 | `documentation/` is the source of truth; agents use read-first / update-after protocol.                                                   | Decided | Team  |
| 2026-10-13 | D-03 | MVP bracket format is single-elimination only.                                                                                            | Superseded by D-06 | Team  |
| 2026-10-13 | D-02 | Date of birth is private; never exposed in public views, APIs, or logs.                                                                   | Superseded by D-12 | Team  |
| 2026-10-13 | D-01 | Tech stack, database, and hosting are not selected yet. Do not introduce them without a decision.                                         | Superseded by D-22 and D-25 | Team  |

## Rules

- A `TBD` row blocks implementation of that topic. Timeboxed spikes require an explicit decision entry.
- Changing a `Decided` row requires a new dated entry, not silent editing.
- Superseded rows stay in the log; never delete decision history.
- Link the ADR file in the Decision column when one exists.
